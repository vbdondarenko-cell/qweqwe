package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func TestV11CityRealtimeFeedIsolationAndTransitions(t *testing.T) {
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
	defer pool.Close()
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatalf("apply canonical migrations: %v", err)
	}
	assertMigrationCount(t, ctx, pool, 23)

	accounts, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	v11Store, err := NewV11SlotStore(baseStore, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slots, err := slot.NewService(v11Store)
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := blocklist.NewService(NewBlockStore(pool, time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	viewerStore, err := NewRealtimeViewerStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	cityFeed, err := realtime.NewCityFeedService(viewerStore)
	if err != nil {
		t.Fatal(err)
	}

	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accounts, "crh", suffix)
	viewerA := registerIntegrationUser(t, ctx, accounts, "cra", suffix)
	viewerB := registerIntegrationUser(t, ctx, accounts, "crb", suffix)
	localityA := mustTestUUID(t)
	localityB := mustTestUUID(t)
	placeA := mustTestUUID(t)
	placeB := mustTestUUID(t)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM slots WHERE host_id=$1`, host.User.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM canonical_places WHERE id=ANY($1::uuid[])`, []string{placeA, placeB})
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM app_users WHERE id=ANY($1::uuid[])`, []string{host.User.ID, viewerA.User.ID, viewerB.User.ID})
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM localities WHERE id=ANY($1::uuid[])`, []string{localityA, localityB})
	})

	_, err = pool.Exec(ctx, `
		INSERT INTO localities (
			id,source,source_locality_id,name,country_code,timezone_name,
			centroid_latitude_e6,centroid_longitude_e6,boundary_wkt,active
		) VALUES
			($1,'integration',$2,'City A','UA','Europe/Kyiv',49400000,32000000,'POLYGON((31 49,33 49,33 50,31 49))',true),
			($3,'integration',$4,'City B','UA','Europe/Kyiv',50400000,30000000,'POLYGON((29 50,31 50,31 51,29 50))',true)`,
		localityA, "city-a-"+suffix, localityB, "city-b-"+suffix,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO canonical_places (
			id,source,source_place_id,name,category,locality,country_code,
			latitude_e6,longitude_e6,precision_m,active,locality_id
		) VALUES
			($1,'integration',$2,'Place A','park','City A','UA',49400000,32000000,50,true,$3),
			($4,'integration',$5,'Place B','park','City B','UA',50400000,30000000,50,true,$6)`,
		placeA, "place-a-"+suffix, localityA, placeB, "place-b-"+suffix, localityB,
	)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	_, err = pool.Exec(ctx, `
		INSERT INTO city_context_locks (
			user_id,locality_id,permission_class,accuracy_m,observed_at,expires_at
		) VALUES
			($1,$2,'PRECISE',50,$5,$6),
			($3,$4,'PRECISE',50,$5,$6)`,
		viewerA.User.ID, localityA, viewerB.User.ID, localityB, now.Add(-time.Minute), now.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}

	beforeCreate := realtimeMaxSequence(t, ctx, pool)
	start := now.Add(2 * time.Hour)
	created, err := slots.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "City realtime", Activity: "walk", PlaceText: "Place A",
		CanonicalPlaceID: &placeA, StartAt: &start, Capacity: 3,
	}, "city-realtime-create-"+suffix)
	if err != nil {
		t.Fatal(err)
	}

	aCreate, err := cityFeed.Pull(ctx, viewerA.User.ID, beforeCreate, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(aCreate.Invalidations) == 0 {
		t.Fatalf("City A viewer missing create invalidation: %#v", aCreate)
	}
	bCreate, err := cityFeed.Pull(ctx, viewerB.User.ID, beforeCreate, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(bCreate.Invalidations) != 0 || bCreate.Cursor <= beforeCreate {
		t.Fatalf("City B viewer must skip City A event while advancing cursor: %#v", bCreate)
	}

	moved, err := slots.Edit(ctx, host.User.ID, created.ID, slot.EditInput{
		ExpectedVersion: created.Version, CanonicalPlaceID: &placeB,
	}, "city-realtime-move-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	aMove, err := cityFeed.Pull(ctx, viewerA.User.ID, aCreate.Cursor, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(aMove.Invalidations) == 0 {
		t.Fatalf("old locality missing move-away invalidation: %#v", aMove)
	}
	bMove, err := cityFeed.Pull(ctx, viewerB.User.ID, bCreate.Cursor, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(bMove.Invalidations) == 0 {
		t.Fatalf("new locality missing move-in invalidation: %#v", bMove)
	}

	cancelled, err := slots.Cancel(ctx, host.User.ID, moved.ID, moved.Version, "city-realtime-cancel-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != slot.StateCancelled {
		t.Fatalf("unexpected cancelled state: %s", cancelled.State)
	}
	bCancel, err := cityFeed.Pull(ctx, viewerB.User.ID, bMove.Cursor, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(bCancel.Invalidations) == 0 {
		t.Fatalf("City B viewer missing cancel invalidation: %#v", bCancel)
	}

	if err := blocks.Block(ctx, host.User.ID, viewerB.User.ID); err != nil {
		t.Fatal(err)
	}
	bBlock, err := cityFeed.Pull(ctx, viewerB.User.ID, bCancel.Cursor, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(bBlock.Invalidations) == 0 {
		t.Fatalf("blocked viewer missing revocation invalidation: %#v", bBlock)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM city_context_locks WHERE user_id=$1`, viewerA.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cityFeed.Pull(ctx, viewerA.User.ID, aMove.Cursor, realtime.MaxViewerBatch); !errors.Is(err, citycontext.ErrNotFound) {
		t.Fatalf("missing/freshness-invalid City Context must fail closed, got %v", err)
	}
}

func mustTestUUID(t *testing.T) string {
	t.Helper()
	id, err := identifier.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	return id
}
