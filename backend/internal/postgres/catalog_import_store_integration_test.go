package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/catalog"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
)

func TestCatalogImportIsIdempotentAndAtomic(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires explicit destructive opt-in")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatal(err)
	}
	store, err := NewCatalogImportStore(pool)
	if err != nil {
		t.Fatal(err)
	}

	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	source := "catalog_test"
	localityKey := "locality-" + suffix
	placeKey := "place-" + suffix
	rollbackKey := "rollback-" + suffix
	active := true
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM canonical_places WHERE source=$1 AND source_place_id IN ($2,$3)`, source, placeKey, "missing-place-"+suffix)
		_, _ = pool.Exec(context.Background(), `DELETE FROM localities WHERE source=$1 AND source_locality_id IN ($2,$3)`, source, localityKey, rollbackKey)
	})

	doc := catalog.Document{
		Localities: []catalog.LocalityRecord{{
			Source: source, SourceLocalityID: localityKey, Name: "Test Locality", CountryCode: "UA", Timezone: "Europe/Kyiv",
			CentroidLatitudeE6: 49_050_000, CentroidLongitudeE6: 32_050_000,
			BoundaryWKT: "POLYGON((32 49,32.1 49,32.1 49.1,32 49.1,32 49))", Active: &active,
		}},
		Places: []catalog.PlaceRecord{{
			Source: source, SourcePlaceID: placeKey, Name: "Test Place", Category: "park",
			LocalitySource: source, LocalitySourceID: localityKey, CountryCode: "UA",
			LatitudeE6: 49_050_000, LongitudeE6: 32_050_000, PrecisionM: 50, Active: &active,
		}},
	}
	if _, err := store.Import(ctx, doc); err != nil {
		t.Fatalf("first import: %v", err)
	}
	var localityID1, placeID1 string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM localities WHERE source=$1 AND source_locality_id=$2`, source, localityKey).Scan(&localityID1); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id::text FROM canonical_places WHERE source=$1 AND source_place_id=$2`, source, placeKey).Scan(&placeID1); err != nil {
		t.Fatal(err)
	}
	doc.Localities[0].Name = "Test Locality Updated"
	doc.Places[0].Name = "Test Place Updated"
	if _, err := store.Import(ctx, doc); err != nil {
		t.Fatalf("second import: %v", err)
	}
	var localityID2, placeID2 string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM localities WHERE source=$1 AND source_locality_id=$2`, source, localityKey).Scan(&localityID2); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id::text FROM canonical_places WHERE source=$1 AND source_place_id=$2`, source, placeKey).Scan(&placeID2); err != nil {
		t.Fatal(err)
	}
	if localityID1 != localityID2 || placeID1 != placeID2 {
		t.Fatalf("canonical identities changed locality %s→%s place %s→%s", localityID1, localityID2, placeID1, placeID2)
	}

	bad := catalog.Document{
		Localities: []catalog.LocalityRecord{{
			Source: source, SourceLocalityID: rollbackKey, Name: "Must Roll Back", CountryCode: "UA", Timezone: "Europe/Kyiv",
			CentroidLatitudeE6: 49_440_000, CentroidLongitudeE6: 32_060_000,
			BoundaryWKT: "POLYGON((32 49,32.1 49,32.1 49.1,32 49.1,32 49))", Active: &active,
		}},
		Places: []catalog.PlaceRecord{{
			Source: source, SourcePlaceID: "missing-place-" + suffix, Name: "Broken Place", CountryCode: "UA",
			LocalitySource: source, LocalitySourceID: "does-not-exist-" + suffix,
			LatitudeE6: 49_440_000, LongitudeE6: 32_060_000, PrecisionM: 50, Active: &active,
		}},
	}
	if _, err := store.Import(ctx, bad); !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("broken reference should reject whole import, got %v", err)
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM localities WHERE source=$1 AND source_locality_id=$2)`, source, rollbackKey).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("failed import left a partially committed locality")
	}
}
