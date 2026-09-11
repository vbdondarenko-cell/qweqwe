package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citymap"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type CityMapStore struct {
	pool *pgxpool.Pool
	// waitlistRequestTTL mirrors V11SlotStore's own field (see its doc
	// comment on waitlistExpiryCutoff): it bounds how long a WAITLIST
	// slot_requests row still reads as the viewer's PENDING relationship
	// in PlaceSlots, the same display-only staleness the read side
	// (Get/ListPulse/ListMine) was already fixed for. This never changes
	// which Slots are visible on the Map — only what PENDING/NONE a
	// viewer sees for one they already have visibility into.
	waitlistRequestTTL time.Duration
}

func NewCityMapStore(pool *pgxpool.Pool, waitlistRequestTTL time.Duration) (*CityMapStore, error) {
	if pool == nil || waitlistRequestTTL <= 0 {
		return nil, errors.New("invalid city map store dependency")
	}
	return &CityMapStore{pool: pool, waitlistRequestTTL: waitlistRequestTTL}, nil
}

func (s *CityMapStore) waitlistExpiryCutoff() time.Time {
	return time.Now().UTC().Add(-s.waitlistRequestTTL)
}

func (s *CityMapStore) Viewport(ctx context.Context, viewerID, localityID string, query citymap.Viewport) ([]citymap.Cluster, error) {
	if viewerID == "" || localityID == "" || !query.Valid() {
		return nil, citymap.ErrInvalidViewport
	}
	rows, err := s.pool.Query(ctx, `
		WITH visible_places AS (
			SELECT p.id,p.name,p.latitude_e6,p.longitude_e6,count(*)::int AS slot_count
			FROM canonical_places p
			JOIN slots s ON s.canonical_place_id=p.id
			WHERE p.active
			  AND p.locality_id=$10::uuid
			  AND (
				s.visibility='PUBLIC'
				OR (
					s.visibility='LINKS'
					AND EXISTS(
						SELECT 1 FROM friendships f
						WHERE f.user_lo_id=LEAST(s.host_id,$6::uuid) AND f.user_hi_id=GREATEST(s.host_id,$6::uuid)
					)
				)
			  )
			  AND s.state IN ('PUBLISHED','FILLING','FULL')
			  AND s.start_at IS NOT NULL
			  AND s.start_at >= $7::timestamptz AND s.start_at < $8::timestamptz
			  AND p.latitude_e6 BETWEEN $2::integer AND $4::integer
			  AND (
				($1::integer < $3::integer AND p.longitude_e6 BETWEEN $1::integer AND $3::integer)
				OR ($1::integer > $3::integer AND (p.longitude_e6 >= $1::integer OR p.longitude_e6 <= $3::integer))
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM user_blocks b
				WHERE (b.blocker_id=$6::uuid AND b.blocked_id=s.host_id)
				   OR (b.blocker_id=s.host_id AND b.blocked_id=$6::uuid)
			  )
			GROUP BY p.id,p.name,p.latitude_e6,p.longitude_e6
		), bucketed AS (
			SELECT
				floor(latitude_e6::numeric / $5::numeric)::bigint AS bucket_lat,
				floor(longitude_e6::numeric / $5::numeric)::bigint AS bucket_lng,
				id,name,latitude_e6,longitude_e6,slot_count
			FROM visible_places
		), clusters AS (
			SELECT
				bucket_lat,bucket_lng,
				round(avg(latitude_e6))::int AS latitude_e6,
				round(avg(longitude_e6))::int AS longitude_e6,
				count(*)::int AS place_count,
				sum(slot_count)::int AS slot_count,
				CASE WHEN count(*)=1 THEN min(id::text) END AS place_id,
				CASE WHEN count(*)=1 THEN min(name) END AS place_name
			FROM bucketed
			GROUP BY bucket_lat,bucket_lng
		)
		SELECT bucket_lat::text || ':' || bucket_lng::text,
		       latitude_e6,longitude_e6,place_count,slot_count,place_id,place_name
		FROM clusters
		ORDER BY slot_count DESC,bucket_lat,bucket_lng
		LIMIT $9::integer`,
		query.WestE6, query.SouthE6, query.EastE6, query.NorthE6, query.BucketE6(),
		viewerID, query.From, query.To, query.Limit, localityID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]citymap.Cluster, 0, query.Limit)
	for rows.Next() {
		var item citymap.Cluster
		if err := rows.Scan(
			&item.Key, &item.LatitudeE6, &item.LongitudeE6,
			&item.PlaceCount, &item.SlotCount, &item.PlaceID, &item.PlaceName,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *CityMapStore) PlaceSlots(ctx context.Context, viewerID, localityID string, query citymap.PlaceSlotsQuery) ([]slot.Slot, error) {
	if viewerID == "" || localityID == "" || !query.Valid() {
		return nil, citymap.ErrInvalidViewport
	}
	rows, err := s.pool.Query(ctx, `SELECT `+v11SlotColumns+`,
		CASE
			WHEN s.host_id=$1::uuid THEN 'HOST'
			WHEN EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$1::uuid) THEN 'ACCEPTED'
			WHEN EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$1::uuid
				AND (s.access_mode<>'WAITLIST' OR r.created_at>$7::timestamptz)) THEN 'PENDING'
			ELSE 'NONE'
		END
	FROM slots s
	JOIN app_users u ON u.id=s.host_id
	JOIN canonical_places p ON p.id=s.canonical_place_id
	WHERE p.id=$2::uuid
	  AND p.active
	  AND p.locality_id=$6::uuid
	  AND (
		s.visibility='PUBLIC'
		OR (
			s.visibility='LINKS'
			AND EXISTS(
				SELECT 1 FROM friendships f
				WHERE f.user_lo_id=LEAST(s.host_id,$1::uuid) AND f.user_hi_id=GREATEST(s.host_id,$1::uuid)
			)
		)
	  )
	  AND s.state IN ('PUBLISHED','FILLING','FULL')
	  AND s.start_at IS NOT NULL
	  AND s.start_at >= $3::timestamptz AND s.start_at < $4::timestamptz
	  AND NOT EXISTS(
		SELECT 1 FROM user_blocks b
		WHERE (b.blocker_id=$1::uuid AND b.blocked_id=s.host_id)
		   OR (b.blocker_id=s.host_id AND b.blocked_id=$1::uuid)
	  )
	ORDER BY s.start_at,s.id
	LIMIT $5::integer`, viewerID, query.PlaceID, query.From, query.To, query.Limit, localityID, s.waitlistExpiryCutoff())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]slot.Slot, 0)
	for rows.Next() {
		item, err := scanV11Slot(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
