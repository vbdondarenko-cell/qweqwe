package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citymap"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func TestV11CanonicalPlaceSlotAndMapIntegration(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires LINKUP_TEST_DATABASE_URL and LINKUP_TEST_DATABASE_DESTRUCTIVE=1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil { t.Fatal(err) }
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil { t.Fatal(err) }
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatalf("apply canonical migrations: %v", err)
	}
	assertMigrationCount(t, ctx, pool, 15)

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil { t.Fatal(err) }
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	viewer := registerIntegrationUser(t, ctx, accountService, "map", suffix)
	placeID, err := identifier.NewUUID()
	if err != nil { t.Fatal(err) }

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM slots WHERE host_id=$1`, viewer.User.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM app_users WHERE id=$1`, viewer.User.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM canonical_places WHERE id=$1`, placeID)
	})

	_, err = pool.Exec(ctx, `
		INSERT INTO canonical_places (
			id,source,source_place_id,name,category,locality,country_code,
			latitude_e6,longitude_e6,precision_m
		) VALUES ($1,'integration',$2,'Integration Place','cafe','Kyiv','UA',50500000,30500000,50)`,
		placeID, "integration-"+suffix,
	)
	if err != nil { t.Fatal(err) }

	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil { t.Fatal(err) }
	v11Store, err := NewV11SlotStore(baseStore)
	if err != nil { t.Fatal(err) }
	slotService, err := slot.NewService(v11Store)
	if err != nil { t.Fatal(err) }

	start := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	created, err := slotService.Create(ctx, viewer.User.ID, slot.CreateInput{
		Title: "Map integration", Activity: "coffee", PlaceText: "Integration Place",
		CanonicalPlaceID: &placeID, StartAt: &start, Capacity: 2,
	}, "v11-map-create-integration-0001")
	if err != nil { t.Fatal(err) }
	if created.CanonicalPlaceID == nil || *created.CanonicalPlaceID != placeID {
		t.Fatalf("canonical place missing from created slot: %#v", created)
	}

	fetched, err := slotService.Get(ctx, viewer.User.ID, created.ID)
	if err != nil { t.Fatal(err) }
	if fetched.CanonicalPlaceID == nil || *fetched.CanonicalPlaceID != placeID {
		t.Fatalf("canonical place missing from fetched slot: %#v", fetched)
	}

	mapStore, err := NewCityMapStore(pool)
	if err != nil { t.Fatal(err) }
	mapService, err := citymap.NewService(mapStore)
	if err != nil { t.Fatal(err) }
	clusters, err := mapService.Viewport(ctx, viewer.User.ID, citymap.Viewport{
		WestE6: 30000000, SouthE6: 50000000, EastE6: 31000000, NorthE6: 51000000,
		Zoom: 16, From: start.Add(-time.Hour), To: start.Add(time.Hour), Limit: 20,
	})
	if err != nil { t.Fatal(err) }
	if len(clusters) != 1 || clusters[0].PlaceID == nil || *clusters[0].PlaceID != placeID || clusters[0].SlotCount != 1 {
		t.Fatalf("unexpected map clusters: %#v", clusters)
	}

	cleared, err := slotService.Edit(ctx, viewer.User.ID, created.ID, slot.EditInput{
		ExpectedVersion: created.Version, ClearCanonicalPlaceID: true,
	}, "v11-map-clear-place-0001")
	if err != nil { t.Fatal(err) }
	if cleared.CanonicalPlaceID != nil {
		t.Fatalf("canonical place was not cleared: %#v", cleared)
	}
}
