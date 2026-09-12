package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
)

type CityContextStore struct {
	pool *pgxpool.Pool
}

func NewCityContextStore(pool *pgxpool.Pool) (*CityContextStore, error) {
	if pool == nil {
		return nil, citycontext.ErrResolverUnavailable
	}
	return &CityContextStore{pool: pool}, nil
}

func (s *CityContextStore) Apply(
	ctx context.Context,
	userID string,
	observation citycontext.Observation,
	policy citycontext.Policy,
	now time.Time,
) (citycontext.Context, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return citycontext.Context{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialize lock creation and boundary switching even when this user has no
	// city_context_locks row yet. The hash is transaction-scoped and contains no location.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, userID); err != nil {
		return citycontext.Context{}, err
	}

	resolved, err := resolveLocalityTx(ctx, tx, observation.LatitudeE6, observation.LongitudeE6)
	if err != nil {
		return citycontext.Context{}, err
	}
	current, err := currentCityLockTx(ctx, tx, userID, false, now)
	if err != nil && !errors.Is(err, citycontext.ErrNotFound) {
		return citycontext.Context{}, err
	}
	if errors.Is(err, citycontext.ErrNotFound) {
		current = nil
	}

	next, err := citycontext.NextLock(current, resolved, observation, now, policy)
	if err != nil {
		return citycontext.Context{}, err
	}
	next.UserID = userID

	_, err = tx.Exec(ctx, `
		INSERT INTO city_context_locks (
			user_id, locality_id, permission_class, accuracy_m, observed_at, expires_at,
			candidate_locality_id, candidate_count, candidate_observed_at, updated_at
		) VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7::uuid,$8,$9,$10)
		ON CONFLICT (user_id) DO UPDATE SET
			locality_id=EXCLUDED.locality_id,
			permission_class=EXCLUDED.permission_class,
			accuracy_m=EXCLUDED.accuracy_m,
			observed_at=EXCLUDED.observed_at,
			expires_at=EXCLUDED.expires_at,
			candidate_locality_id=EXCLUDED.candidate_locality_id,
			candidate_count=EXCLUDED.candidate_count,
			candidate_observed_at=EXCLUDED.candidate_observed_at,
			updated_at=EXCLUDED.updated_at`,
		next.UserID, next.Locality.ID, string(next.PermissionClass), next.AccuracyM,
		next.ObservedAt, next.ExpiresAt, nullableStringValue(next.CandidateLocalityID),
		next.CandidateCount, nullableTimeValue(next.CandidateObservedAt), now.UTC(),
	)
	if err != nil {
		return citycontext.Context{}, err
	}

	// LASSO/TRAVEL_CORRIDOR visibility (README §4.3) needs some form of
	// the viewer's own approximate position to test containment against
	// an arbitrary host-drawn shape -- a locality id alone (all
	// city_context_locks stores) cannot express that. Per the user's own
	// explicit choice of how to close this gap: persist only a COARSENED
	// point, at the identical freshness/expiry discipline the lock above
	// already uses (same expires_at, same upsert-per-user shape), never
	// the raw observation. This runs unconditionally on every successful
	// resolve, in the same transaction as the lock -- the two can never
	// drift out of sync with each other.
	roundedLat, roundedLng := roundToPointGrid(observation.LatitudeE6, observation.LongitudeE6)
	_, err = tx.Exec(ctx, `
		INSERT INTO city_context_points (
			user_id, latitude_e6, longitude_e6, permission_class, accuracy_m, observed_at, expires_at, updated_at
		) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (user_id) DO UPDATE SET
			latitude_e6=EXCLUDED.latitude_e6,
			longitude_e6=EXCLUDED.longitude_e6,
			permission_class=EXCLUDED.permission_class,
			accuracy_m=EXCLUDED.accuracy_m,
			observed_at=EXCLUDED.observed_at,
			expires_at=EXCLUDED.expires_at,
			updated_at=EXCLUDED.updated_at`,
		next.UserID, roundedLat, roundedLng, string(next.PermissionClass), next.AccuracyM,
		next.ObservedAt, next.ExpiresAt, now.UTC(),
	)
	if err != nil {
		return citycontext.Context{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return citycontext.Context{}, err
	}
	return next.Public(), nil
}

// pointGridE6 coarsens a raw coordinate to a ~111m grid (1000 E6 units is
// 0.001 degrees of latitude; longitude is coarser than this away from the
// equator, which is an acceptable, stated approximation -- the goal is
// "coarse," not "precisely 111m everywhere"). Never store the raw
// observation this rounds away.
const pointGridE6 = 1000

// roundToPointGrid rounds toward the nearest grid line rather than always
// down, so the coarsening error is bounded at +/- half a grid cell
// instead of up to a full cell.
func roundToPointGrid(latitudeE6, longitudeE6 int) (int, int) {
	return roundToGrid(latitudeE6, pointGridE6), roundToGrid(longitudeE6, pointGridE6)
}

func roundToGrid(value, grid int) int {
	if value >= 0 {
		return ((value + grid/2) / grid) * grid
	}
	return -((-value + grid/2) / grid) * grid
}

func (s *CityContextStore) Current(ctx context.Context, userID string, now time.Time) (citycontext.Context, error) {
	lock, err := currentCityLockRow(s.pool.QueryRow(ctx, currentCityLockSQL+` AND c.expires_at > $2 AND l.active`, userID, now.UTC()))
	if errors.Is(err, pgx.ErrNoRows) {
		return citycontext.Context{}, citycontext.ErrNotFound
	}
	if err != nil {
		return citycontext.Context{}, err
	}
	return lock.Public(), nil
}

func resolveLocalityTx(ctx context.Context, tx pgx.Tx, latitudeE6, longitudeE6 int) (citycontext.Locality, error) {
	var available bool
	if err := tx.QueryRow(ctx, `
		SELECT to_regtype('public.geometry') IS NOT NULL
		   AND to_regprocedure('public.st_covers(public.geometry,public.geometry)') IS NOT NULL
		   AND to_regprocedure('public.st_makepoint(double precision,double precision)') IS NOT NULL
		   AND EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name='localities' AND column_name='boundary'
		   )`).Scan(&available); err != nil {
		return citycontext.Locality{}, err
	}
	if !available {
		return citycontext.Locality{}, citycontext.ErrResolverUnavailable
	}

	var locality citycontext.Locality
	err := tx.QueryRow(ctx, `
		SELECT id::text,name,country_code,timezone_name,centroid_latitude_e6,centroid_longitude_e6
		FROM localities
		WHERE active
		  AND boundary IS NOT NULL
		  AND public.st_covers(
			boundary,
			public.st_setsrid(
				public.st_makepoint($2::double precision / 1000000.0, $1::double precision / 1000000.0),
				4326
			)
		  )
		ORDER BY public.st_area(boundary) ASC, id ASC
		LIMIT 1`, latitudeE6, longitudeE6).Scan(
		&locality.ID, &locality.Name, &locality.CountryCode, &locality.Timezone,
		&locality.CentroidLatitudeE6, &locality.CentroidLongitudeE6,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return citycontext.Locality{}, citycontext.ErrNoLocality
	}
	if err != nil {
		return citycontext.Locality{}, err
	}
	return locality, nil
}

func currentCityLockTx(ctx context.Context, tx pgx.Tx, userID string, onlyFresh bool, now time.Time) (*citycontext.Lock, error) {
	query := currentCityLockSQL + ` AND l.active`
	args := []any{userID}
	if onlyFresh {
		query += ` AND c.expires_at > $2`
		args = append(args, now.UTC())
	}
	query += ` FOR UPDATE OF c`
	lock, err := currentCityLockRow(tx.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, citycontext.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

const currentCityLockSQL = `
	SELECT c.user_id::text,l.id::text,l.name,l.country_code,l.timezone_name,
	       l.centroid_latitude_e6,l.centroid_longitude_e6,
	       c.permission_class,c.accuracy_m,c.observed_at,c.expires_at,
	       c.candidate_locality_id::text,c.candidate_count,c.candidate_observed_at
	FROM city_context_locks c
	JOIN localities l ON l.id=c.locality_id
	WHERE c.user_id=$1::uuid`

func currentCityLockRow(row scanner) (citycontext.Lock, error) {
	var out citycontext.Lock
	var permission string
	var candidate pgtype.Text
	var candidateAt pgtype.Timestamptz
	if err := row.Scan(
		&out.UserID, &out.Locality.ID, &out.Locality.Name, &out.Locality.CountryCode, &out.Locality.Timezone,
		&out.Locality.CentroidLatitudeE6, &out.Locality.CentroidLongitudeE6,
		&permission, &out.AccuracyM, &out.ObservedAt, &out.ExpiresAt,
		&candidate, &out.CandidateCount, &candidateAt,
	); err != nil {
		return citycontext.Lock{}, err
	}
	out.PermissionClass = citycontext.PermissionClass(permission)
	if candidate.Valid {
		value := candidate.String
		out.CandidateLocalityID = &value
	}
	if candidateAt.Valid {
		value := candidateAt.Time
		out.CandidateObservedAt = &value
	}
	return out, nil
}

func nullableStringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTimeValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}
