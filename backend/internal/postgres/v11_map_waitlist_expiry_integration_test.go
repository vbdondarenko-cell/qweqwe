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

// TestV11MapPlaceSlotsWaitlistExpiryAwareness closes the Map-side half of the
// gap IMPLEMENTATION_STATUS.md §44 deliberately left open ("citymap_store.go
// ... deliberately left for a follow-up"): CityMapStore.PlaceSlots computed
// PENDING from slot_requests without the same WAITLIST-TTL awareness the
// read side (Get/ListPulse/ListMine, closed in §44) already has, so a Map
// viewer could keep seeing themselves as PENDING on a queue position that
// promoteOldestWaitlistTx/waitlistRequest already treat as expired.
func TestV11MapPlaceSlotsWaitlistExpiryAwareness(t *testing.T) {
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

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accountService, "mwh", suffix)
	seatA := registerIntegrationUser(t, ctx, accountService, "mwa", suffix)
	seatB := registerIntegrationUser(t, ctx, accountService, "mwb", suffix)
	queued := registerIntegrationUser(t, ctx, accountService, "mwq", suffix)

	localityID, err := identifier.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	placeID, err := identifier.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	// t.Cleanup runs LIFO: canonical_places carries `ON DELETE RESTRICT` from
	// slots.canonical_place_id (migration 000015), so the users/slots cleanup
	// (registered second, deletes slots first) must run before this one
	// (registered first) actually deletes the place — otherwise the RESTRICT
	// would reject deleting a place a still-live slot references.
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM canonical_places WHERE id=$1::uuid`, placeID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM localities WHERE id=$1::uuid`, localityID)
	})
	t.Cleanup(func() {
		cleanupIntegrationRows(pool, []string{host.User.ID, seatA.User.ID, seatB.User.ID, queued.User.ID})
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO localities (
			id,source,source_locality_id,name,country_code,timezone_name,
			centroid_latitude_e6,centroid_longitude_e6,boundary_wkt
		) VALUES ($1::uuid,'integration',$2,'Kyiv','UA','Europe/Kyiv',
			50500000,30500000,'MULTIPOLYGON(((30 50,31 50,31 51,30 51,30 50)))')`,
		localityID, "map-waitlist-expiry-"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO canonical_places (
			id,source,source_place_id,name,category,locality,country_code,
			latitude_e6,longitude_e6,precision_m,locality_id
		) VALUES ($1::uuid,'integration',$2,'Map Waitlist Place','cafe','Kyiv','UA',50500000,30500000,50,$3::uuid)`,
		placeID, "map-waitlist-place-"+suffix, localityID); err != nil {
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
	slotService, err := slot.NewService(v11Store)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	waitlistMode := slot.AccessWaitlist
	draft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Map waitlist expiry", Activity: "coffee", PlaceText: "Map Waitlist Place",
		CanonicalPlaceID: &placeID, StartAt: &start, Capacity: 2, AccessMode: &waitlistMode,
	}, "v11-map-waitlist-expiry-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	published, err := slotService.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "v11-map-waitlist-expiry-publish-0001")
	if err != nil {
		t.Fatal(err)
	}
	if published.AccessMode != slot.AccessWaitlist || published.State != slot.StateFilling {
		t.Fatalf("waitlist draft did not publish into FILLING: %#v", published)
	}

	// Fill both seats so the next requester queues instead of auto-admitting.
	if _, err := slotService.Request(ctx, seatA.User.ID, published.ID, "v11-map-waitlist-expiry-seat-a-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, seatB.User.ID, published.ID, "v11-map-waitlist-expiry-seat-b-0001"); err != nil {
		t.Fatal(err)
	}
	queuedResult, err := slotService.Request(ctx, queued.User.ID, published.ID, "v11-map-waitlist-expiry-queue-0001")
	if err != nil || queuedResult.ViewerState != slot.ViewerPending {
		t.Fatalf("expected queued requester: viewer=%v err=%v", queuedResult.ViewerState, err)
	}

	mapStore, err := NewCityMapStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	mapService, err := citymap.NewService(mapStore)
	if err != nil {
		t.Fatal(err)
	}

	before, err := mapService.PlaceSlots(ctx, queued.User.ID, localityID, citymap.PlaceSlotsQuery{
		PlaceID: placeID, From: start.Add(-time.Hour), To: start.Add(time.Hour), Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 1 || before[0].ViewerState != slot.ViewerPending {
		t.Fatalf("live WAITLIST request should read PENDING on the Map before expiry: %#v", before)
	}

	backdateWaitlistRequest(t, ctx, pool, published.ID, queued.User.ID, -2*time.Hour)

	after, err := mapService.PlaceSlots(ctx, queued.User.ID, localityID, citymap.PlaceSlotsQuery{
		PlaceID: placeID, From: start.Add(-time.Hour), To: start.Add(time.Hour), Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || after[0].ViewerState != slot.ViewerNone {
		t.Fatalf("expired WAITLIST request must read as NONE on the Map, not PENDING: %#v", after)
	}

	// The Slot is still visible on the Map at all (PUBLIC/FILLING), only the
	// viewer's own relationship display changed — losing visibility outright
	// would be a different, larger change this fix does not make.
	if after[0].ID != published.ID {
		t.Fatalf("expired WAITLIST request must not lose Map visibility of the Slot itself: %#v", after)
	}
}
