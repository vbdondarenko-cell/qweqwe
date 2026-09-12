package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
)

// TestV11CityContextStoreApplyWritesRoundedPoint is this session's own
// added coverage: a real citycontext.Service.Resolve() call (not a direct
// row insert, unlike this file's sibling test helpers) must both create
// the city_context_locks row (pre-existing behavior, previously only
// exercised indirectly through direct-insert test helpers rather than a
// real Resolve() call — a genuine, pre-existing integration-coverage gap
// noted here rather than silently left alone, though fixing that gap in
// full is out of this block's own scope) and, new in this block, a
// coarsened city_context_points row sharing its exact expires_at — never
// the raw observed coordinate.
func TestV11CityContextStoreApplyWritesRoundedPoint(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires LINKUP_TEST_DATABASE_URL and LINKUP_TEST_DATABASE_DESTRUCTIVE=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatal(err)
	}

	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	localityID := mustTestUUID(t)
	// A deliberately distant, non-overlapping polygon (Sydney-ish, far
	// from the 49-51N/29-33E region every other locality fixture in this
	// package uses) so this test's resolution can never be shadowed by
	// another test's same-region locality left over in this shared
	// disposable/accumulating database, regardless of cleanup ordering
	// elsewhere.
	_, err = pool.Exec(ctx, `
		INSERT INTO localities (
			id,source,source_locality_id,name,country_code,timezone_name,
			centroid_latitude_e6,centroid_longitude_e6,boundary_wkt,active
		) VALUES ($1,'integration',$2,'Test City Apply','AU','Australia/Sydney',-33000000,151000000,'POLYGON((150 -34,152 -34,152 -32,150 -34))',true)`,
		localityID, "city-context-apply-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM localities WHERE id=$1`, localityID)
	})

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	user := registerIntegrationUser(t, ctx, accountService, "ccp", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{user.User.ID}) })

	store, err := NewCityContextStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	service, err := citycontext.NewService(store, citycontext.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}

	const rawLat, rawLng = -33123450, 151087650
	out, err := service.Resolve(ctx, user.User.ID, citycontext.Observation{
		LatitudeE6: rawLat, LongitudeE6: rawLng, AccuracyM: 30,
		PermissionClass: citycontext.PermissionPrecise, CapturedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Locality.ID != localityID {
		t.Fatalf("expected resolution into the seeded locality, got %#v", out.Locality)
	}

	var latitudeE6, longitudeE6 int
	var expiresAt time.Time
	if err := pool.QueryRow(ctx, `SELECT latitude_e6,longitude_e6,expires_at FROM city_context_points WHERE user_id=$1`, user.User.ID).
		Scan(&latitudeE6, &longitudeE6, &expiresAt); err != nil {
		t.Fatal(err)
	}
	if latitudeE6 == rawLat && longitudeE6 == rawLng {
		t.Fatalf("expected a coarsened point, got the exact raw observation stored verbatim: lat=%d lng=%d", latitudeE6, longitudeE6)
	}
	if abs(latitudeE6-rawLat) > pointGridE6/2 || abs(longitudeE6-rawLng) > pointGridE6/2 {
		t.Fatalf("coarsened point too far from the raw observation: lat=%d lng=%d (raw lat=%d lng=%d)", latitudeE6, longitudeE6, rawLat, rawLng)
	}
	// Postgres timestamptz truncates to microsecond precision, while Go's
	// time.Time carries nanoseconds — compare with a generous tolerance
	// rather than exact equality, which would spuriously fail on that
	// truncation alone rather than a real expiry mismatch.
	if diff := expiresAt.Sub(out.ExpiresAt); diff > time.Millisecond || diff < -time.Millisecond {
		t.Fatalf("expected city_context_points.expires_at to match the lock's own expiry, got point=%v lock=%v", expiresAt, out.ExpiresAt)
	}

	// A second Resolve() call upserts in place rather than accumulating
	// rows: exactly one city_context_points row per user, always.
	if _, err := service.Resolve(ctx, user.User.ID, citycontext.Observation{
		LatitudeE6: rawLat + 5000, LongitudeE6: rawLng, AccuracyM: 30,
		PermissionClass: citycontext.PermissionPrecise, CapturedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	var pointCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM city_context_points WHERE user_id=$1`, user.User.ID).Scan(&pointCount); err != nil {
		t.Fatal(err)
	}
	if pointCount != 1 {
		t.Fatalf("expected exactly one city_context_points row per user after a second Resolve(), got %d", pointCount)
	}
}
