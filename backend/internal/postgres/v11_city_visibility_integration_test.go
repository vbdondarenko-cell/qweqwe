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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citymap"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// lockCityContext directly inserts a city_context_locks row for userID,
// bypassing the citycontext.Service/Resolve flow entirely — this file is
// only exercising Slot visibility's own consumption of an existing lock,
// not City Context resolution itself (already covered by
// internal/citycontext's own tests and v11_city_context_privileges_
// integration_test.go), so a direct row insert keeps these tests focused.
func lockCityContext(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID, localityID string, now time.Time) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO city_context_locks (user_id,locality_id,permission_class,accuracy_m,observed_at,expires_at)
		VALUES ($1,$2,'PRECISE',50,$3,$4)
		ON CONFLICT (user_id) DO UPDATE SET locality_id=EXCLUDED.locality_id, expires_at=EXCLUDED.expires_at`,
		userID, localityID, now.Add(-time.Minute), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
}

func insertTestLocality(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, sourceID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO localities (
			id,source,source_locality_id,name,country_code,timezone_name,
			centroid_latitude_e6,centroid_longitude_e6,boundary_wkt,active
		) VALUES ($1,'integration',$2,'Test City','UA','Europe/Kyiv',49400000,32000000,'POLYGON((31 49,33 49,33 50,31 49))',true)`,
		id, sourceID)
	if err != nil {
		t.Fatal(err)
	}
}

// TestV11CityVisibilitySlot closes the discovery half of README §4.3's
// "City-only" mode: a CITY Slot is invisible to a stranger with no live
// City Context lock at all, invisible to a viewer locked to a *different*
// locality than the host, visible to a viewer currently locked to the
// *same* locality as the host, and a block still wins even between two
// people in the same city — matching every other non-PUBLIC mode's own
// "block always wins" invariant.
func TestV11CityVisibilitySlot(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "cvh", suffix)
	sameCity := registerIntegrationUser(t, ctx, accountService, "cvs", suffix)
	otherCity := registerIntegrationUser(t, ctx, accountService, "cvo", suffix)
	noLock := registerIntegrationUser(t, ctx, accountService, "cvn", suffix)

	localityA := mustTestUUID(t)
	localityB := mustTestUUID(t)
	insertTestLocality(t, ctx, pool, localityA, "city-a-"+suffix)
	insertTestLocality(t, ctx, pool, localityB, "city-b-"+suffix)
	// Registered before cleanupIntegrationRows below on purpose (t.Cleanup
	// runs LIFO): city_context_locks rows (inserted via lockCityContext
	// below) reference these localities with ON DELETE RESTRICT, and only
	// cascade away once the owning app_users row is deleted -- so the user
	// cleanup must run FIRST (i.e. be registered LAST) or this delete
	// silently no-ops on that FK restriction, permanently orphaning the
	// locality row in this shared disposable database.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM localities WHERE id=ANY($1::uuid[])`, []string{localityA, localityB})
	})
	t.Cleanup(func() {
		cleanupIntegrationRows(pool, []string{host.User.ID, sameCity.User.ID, otherCity.User.ID, noLock.User.ID})
	})

	now := time.Now().UTC()
	lockCityContext(t, ctx, pool, host.User.ID, localityA, now)
	lockCityContext(t, ctx, pool, sameCity.User.ID, localityA, now)
	lockCityContext(t, ctx, pool, otherCity.User.ID, localityB, now)
	// noLock deliberately has no city_context_locks row at all.

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

	visibility := slot.VisibilityCity
	draft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "City Meetup", Activity: "coffee", PlaceText: "Center", Capacity: 3,
		Visibility: &visibility,
	}, "v11-city-visibility-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	published, err := slotService.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "v11-city-visibility-publish-0001")
	if err != nil {
		t.Fatal(err)
	}
	if published.Visibility != slot.VisibilityCity || published.State != slot.StateFilling {
		t.Fatalf("published slot lost CITY visibility or wrong state: %#v", published)
	}

	// (1) A viewer with no live City Context lock at all: excluded.
	if _, err := slotService.Get(ctx, noLock.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a viewer with no city lock, got %v", err)
	}

	// (2) A viewer locked to a different locality: excluded.
	if _, err := slotService.Get(ctx, otherCity.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a viewer locked to a different city, got %v", err)
	}
	otherPulse, err := slotService.ListPulse(ctx, otherCity.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range otherPulse {
		if item.ID == published.ID {
			t.Fatalf("CITY slot must not appear in a different-city viewer's Pulse feed: %#v", item)
		}
	}

	// (3) A viewer locked to the SAME current locality as the host: included.
	sameView, err := slotService.Get(ctx, sameCity.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sameView.ViewerState != slot.ViewerNone {
		t.Fatalf("a same-city viewer with no other relationship should read ViewerNone, got %s", sameView.ViewerState)
	}
	samePulse, err := slotService.ListPulse(ctx, sameCity.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range samePulse {
		if item.ID == published.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the same-city viewer to see the CITY slot on Pulse: %#v", samePulse)
	}

	// (4) A block wins even between two people currently in the same city.
	if _, err := pool.Exec(ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1,$2)`, host.User.ID, sameCity.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Get(ctx, sameCity.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected a block to override same-city CITY visibility, got %v", err)
	}

	// (5) A stranger with no city lock at all, handed the Slot ID
	// directly, can still Request() it — the same "discoverability gate,
	// not access-control gate" contract every other mode here has.
	afterRequest, err := slotService.Request(ctx, noLock.User.ID, published.ID, "v11-city-visibility-request-0001")
	if err != nil {
		t.Fatalf("a viewer with no city lock and a known CITY slot id must still be able to Request(): %v", err)
	}
	if afterRequest.ViewerState != slot.ViewerPending {
		t.Fatalf("expected PENDING after Request() on a CITY slot, got %s", afterRequest.ViewerState)
	}
}

// TestV11CityVisibilityRejectedForLegacyCreate is the regression check for
// the v1.0 create endpoint, mirroring PRIVATE/LINKS/SELECTED's own.
func TestV11CityVisibilityRejectedForLegacyCreate(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "cvl", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID}) })

	store, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := slot.VisibilityCity
	if _, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Legacy City Attempt", Activity: "coffee", PlaceText: "Center", Capacity: 3,
		Visibility: &visibility,
	}, "v11-city-visibility-legacy-0001"); !errors.Is(err, slot.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState rejecting CITY on the v1.0 create endpoint, got %v", err)
	}
}

// TestV11CityVisibilityMapAndRealtimeViewer proves CITY reaches the same
// discovery/realtime parity LINKS/SELECTED both have: Map's Viewport/
// PlaceSlots and the per-viewer realtime feed all honor the same live
// same-locality-lock rule as Get/ListPulse above.
func TestV11CityVisibilityMapAndRealtimeViewer(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "cvm", suffix)
	sameCity := registerIntegrationUser(t, ctx, accountService, "cvw", suffix)
	stranger := registerIntegrationUser(t, ctx, accountService, "cvx", suffix)

	localityA := mustTestUUID(t)
	place := mustTestUUID(t)
	insertTestLocality(t, ctx, pool, localityA, "city-map-"+suffix)
	_, err = pool.Exec(ctx, `
		INSERT INTO canonical_places (
			id,source,source_place_id,name,category,locality,country_code,
			latitude_e6,longitude_e6,precision_m,active,locality_id
		) VALUES ($1,'integration',$2,'Test Place','park','Test City','UA',49400000,32000000,50,true,$3)`,
		place, "place-"+suffix, localityA)
	if err != nil {
		t.Fatal(err)
	}
	// Registered before cleanupIntegrationRows below on purpose: t.Cleanup
	// runs its funcs LIFO, and slots (deleted by cleanupIntegrationRows,
	// keyed on host_id) reference canonical_places.id, which references
	// localities.id — so the slot cleanup must run FIRST (i.e. be
	// registered LAST) or this delete silently no-ops on an FK violation,
	// permanently orphaning both rows in this shared disposable database.
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM canonical_places WHERE id=$1`, place)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM localities WHERE id=$1`, localityA)
	})
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, sameCity.User.ID, stranger.User.ID}) })

	now := time.Now().UTC()
	lockCityContext(t, ctx, pool, host.User.ID, localityA, now)
	lockCityContext(t, ctx, pool, sameCity.User.ID, localityA, now)
	// stranger deliberately has no lock at all.

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
	mapStore, err := NewCityMapStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	before := realtimeMaxSequence(t, ctx, pool)
	visibility := slot.VisibilityCity
	start := now.Add(2 * time.Hour)
	draft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "City Map Meetup", Activity: "walk", PlaceText: "Test Place",
		CanonicalPlaceID: &place, StartAt: &start, Capacity: 3, Visibility: &visibility,
	}, "v11-city-map-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "v11-city-map-publish-0001"); err != nil {
		t.Fatal(err)
	}

	viewport := citymap.Viewport{
		WestE6: 31000000, SouthE6: 49000000, EastE6: 33000000, NorthE6: 50000000,
		Zoom: 12, From: now, To: now.Add(4 * time.Hour), Limit: citymap.MaxClusters,
	}
	strangerClusters, err := mapStore.Viewport(ctx, stranger.User.ID, localityA, viewport)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range strangerClusters {
		if c.SlotCount > 0 {
			t.Fatalf("a viewer with no city lock must not see the CITY slot's place cluster: %#v", strangerClusters)
		}
	}
	sameCityClusters, err := mapStore.Viewport(ctx, sameCity.User.ID, localityA, viewport)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, c := range sameCityClusters {
		total += c.SlotCount
	}
	if total == 0 {
		t.Fatalf("expected the same-city viewer to see the CITY slot's place cluster: %#v", sameCityClusters)
	}

	strangerPlaceSlots, err := mapStore.PlaceSlots(ctx, stranger.User.ID, localityA, citymap.PlaceSlotsQuery{
		PlaceID: place, From: now, To: now.Add(4 * time.Hour), Limit: citymap.MaxPlaceSlots,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(strangerPlaceSlots) != 0 {
		t.Fatalf("a viewer with no city lock must not see the CITY slot via PlaceSlots: %#v", strangerPlaceSlots)
	}
	sameCityPlaceSlots, err := mapStore.PlaceSlots(ctx, sameCity.User.ID, localityA, citymap.PlaceSlotsQuery{
		PlaceID: place, From: now, To: now.Add(4 * time.Hour), Limit: citymap.MaxPlaceSlots,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sameCityPlaceSlots) != 1 {
		t.Fatalf("expected the same-city viewer to see the CITY slot via PlaceSlots: %#v", sameCityPlaceSlots)
	}

	viewerStore, err := NewRealtimeViewerStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	feed, err := realtime.NewFeedService(viewerStore)
	if err != nil {
		t.Fatal(err)
	}
	strangerBatch, err := feed.Pull(ctx, stranger.User.ID, before, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if hasRealtimeType(strangerBatch.Events, "slot.created") {
		t.Fatalf("a viewer with no city lock must not see a CITY slot.created event: %#v", strangerBatch.Events)
	}
	sameCityBatch, err := feed.Pull(ctx, sameCity.User.ID, before, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRealtimeType(sameCityBatch.Events, "slot.created") {
		t.Fatalf("expected the same-city viewer to see a CITY slot.created event: %#v", sameCityBatch.Events)
	}
}
