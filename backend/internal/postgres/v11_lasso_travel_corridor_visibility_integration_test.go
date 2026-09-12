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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// setCityContextPoint directly inserts a city_context_points row,
// bypassing the citycontext.Service/Resolve flow — this file exercises
// only Slot visibility's own consumption of an existing point, not City
// Context resolution or the rounding city_context_store.go's Apply
// applies before ever persisting one (covered by
// TestRoundToGridNeverStoresRawValue/TestRoundToPointGridBoundsErrorToHalfAGridCell
// and, end-to-end, by exercising Resolve directly rather than here).
func setCityContextPoint(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID string, latitudeE6, longitudeE6 int, now time.Time) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO city_context_points (user_id,latitude_e6,longitude_e6,permission_class,accuracy_m,observed_at,expires_at)
		VALUES ($1,$2,$3,'PRECISE',50,$4,$5)
		ON CONFLICT (user_id) DO UPDATE SET latitude_e6=EXCLUDED.latitude_e6, longitude_e6=EXCLUDED.longitude_e6, expires_at=EXCLUDED.expires_at`,
		userID, latitudeE6, longitudeE6, now.Add(-time.Minute), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
}

// TestV11LassoVisibilitySlot closes the discovery half of README §4.3's
// "Lasso/geo-scoped" mode: a LASSO Slot is invisible to a viewer with no
// live point at all, invisible to a viewer whose live point falls outside
// the host-drawn polygon, visible to a viewer currently inside it, and a
// block still wins even for a viewer geometrically inside the shape.
func TestV11LassoVisibilitySlot(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "lvh", suffix)
	inside := registerIntegrationUser(t, ctx, accountService, "lvi", suffix)
	outside := registerIntegrationUser(t, ctx, accountService, "lvo", suffix)
	noPoint := registerIntegrationUser(t, ctx, accountService, "lvn", suffix)
	t.Cleanup(func() {
		cleanupIntegrationRows(pool, []string{host.User.ID, inside.User.ID, outside.User.ID, noPoint.User.ID})
	})

	now := time.Now().UTC()
	setCityContextPoint(t, ctx, pool, inside.User.ID, 50500000, 30500000, now)
	setCityContextPoint(t, ctx, pool, outside.User.ID, 52000000, 32000000, now)
	// noPoint deliberately has no city_context_points row at all.

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

	visibility := slot.VisibilityLasso
	wkt := "POLYGON((30 50,31 50,31 51,30 51,30 50))"
	draft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Lasso Meetup", Activity: "coffee", PlaceText: "Center", Capacity: 3,
		Visibility: &visibility, LassoPolygonWKT: &wkt,
	}, "v11-lasso-visibility-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	if draft.LassoPolygonWKT == nil || *draft.LassoPolygonWKT != wkt {
		t.Fatalf("draft did not echo the lasso polygon back to its host: %#v", draft)
	}
	published, err := slotService.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "v11-lasso-visibility-publish-0001")
	if err != nil {
		t.Fatal(err)
	}

	// (1) A viewer with no live point at all: excluded.
	if _, err := slotService.Get(ctx, noPoint.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a viewer with no city context point, got %v", err)
	}

	// (2) A viewer whose live point is outside the polygon: excluded.
	if _, err := slotService.Get(ctx, outside.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a viewer outside the lasso polygon, got %v", err)
	}
	outsidePulse, err := slotService.ListPulse(ctx, outside.User.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range outsidePulse {
		if item.ID == published.ID {
			t.Fatalf("LASSO slot must not appear in an outside viewer's Pulse feed: %#v", item)
		}
	}

	// (3) A viewer whose live point is inside the polygon: included, but
	// the polygon itself is never echoed back to a non-host viewer.
	insideView, err := slotService.Get(ctx, inside.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if insideView.LassoPolygonWKT != nil {
		t.Fatalf("a non-host viewer must never see the host's lasso polygon, got %#v", insideView.LassoPolygonWKT)
	}
	insidePulse, err := slotService.ListPulse(ctx, inside.User.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range insidePulse {
		if item.ID == published.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the inside viewer to see the LASSO slot on Pulse: %#v", insidePulse)
	}

	// (4) A block wins even for a viewer geometrically inside the lasso.
	if _, err := pool.Exec(ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1,$2)`, host.User.ID, inside.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Get(ctx, inside.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected a block to override being inside the lasso, got %v", err)
	}

	// (5) A stranger with no point at all, handed the Slot ID directly,
	// can still Request() it — the same discoverability-gate contract
	// every other mode here has.
	afterRequest, err := slotService.Request(ctx, noPoint.User.ID, published.ID, "v11-lasso-visibility-request-0001")
	if err != nil {
		t.Fatalf("a viewer with no city context point and a known LASSO slot id must still be able to Request(): %v", err)
	}
	if afterRequest.ViewerState != slot.ViewerPending {
		t.Fatalf("expected PENDING after Request() on a LASSO slot, got %s", afterRequest.ViewerState)
	}
}

// TestV11LassoVisibilityRejectedForLegacyCreate is the regression check
// for the v1.0 create endpoint, mirroring every other non-PUBLIC mode's own.
func TestV11LassoVisibilityRejectedForLegacyCreate(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "lvl", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID}) })

	store, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := slot.VisibilityLasso
	wkt := "POLYGON((30 50,31 50,31 51,30 51,30 50))"
	if _, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Legacy Lasso Attempt", Activity: "coffee", PlaceText: "Center", Capacity: 3,
		Visibility: &visibility, LassoPolygonWKT: &wkt,
	}, "v11-lasso-visibility-legacy-0001"); !errors.Is(err, slot.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState rejecting LASSO on the v1.0 create endpoint, got %v", err)
	}
}

// TestV11TravelCorridorVisibilitySlot mirrors TestV11LassoVisibilitySlot
// for README §4.3's "Travel corridor" mode: a buffered route instead of a
// polygon, tested with a distance predicate instead of containment.
func TestV11TravelCorridorVisibilitySlot(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "tvh", suffix)
	onRoute := registerIntegrationUser(t, ctx, accountService, "tvi", suffix)
	farAway := registerIntegrationUser(t, ctx, accountService, "tvo", suffix)
	t.Cleanup(func() {
		cleanupIntegrationRows(pool, []string{host.User.ID, onRoute.User.ID, farAway.User.ID})
	})

	now := time.Now().UTC()
	// The route runs along latitude 50 from longitude 30 to 31.
	// onRoute sits ~111m off the route (well inside a 5km buffer);
	// farAway sits roughly 220km away (well outside it).
	setCityContextPoint(t, ctx, pool, onRoute.User.ID, 50001000, 30500000, now)
	setCityContextPoint(t, ctx, pool, farAway.User.ID, 52000000, 32000000, now)

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

	visibility := slot.VisibilityTravelCorridor
	line := "LINESTRING(30 50,31 50)"
	radius := 5000
	draft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Travel Corridor Meetup", Activity: "coffee", PlaceText: "Center", Capacity: 3,
		Visibility: &visibility, CorridorLineWKT: &line, CorridorRadiusM: &radius,
	}, "v11-corridor-visibility-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	published, err := slotService.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "v11-corridor-visibility-publish-0001")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := slotService.Get(ctx, farAway.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a viewer far from the travel corridor, got %v", err)
	}
	onRouteView, err := slotService.Get(ctx, onRoute.User.ID, published.ID)
	if err != nil {
		t.Fatalf("expected a viewer on the travel corridor to see the slot: %v", err)
	}
	if onRouteView.CorridorLineWKT != nil || onRouteView.CorridorRadiusM != nil {
		t.Fatalf("a non-host viewer must never see the host's corridor route, got %#v", onRouteView)
	}

	hostView, err := slotService.Get(ctx, host.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if hostView.CorridorLineWKT == nil || *hostView.CorridorLineWKT != line || hostView.CorridorRadiusM == nil || *hostView.CorridorRadiusM != radius {
		t.Fatalf("expected the host's own read to show the configured corridor, got %#v", hostView)
	}
}

// TestV11TravelCorridorVisibilityRejectedForLegacyCreate mirrors every
// other non-PUBLIC mode's legacy-create regression check.
func TestV11TravelCorridorVisibilityRejectedForLegacyCreate(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "tvl", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID}) })

	store, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := slot.VisibilityTravelCorridor
	line := "LINESTRING(30 50,31 50)"
	radius := 5000
	if _, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Legacy Corridor Attempt", Activity: "coffee", PlaceText: "Center", Capacity: 3,
		Visibility: &visibility, CorridorLineWKT: &line, CorridorRadiusM: &radius,
	}, "v11-corridor-visibility-legacy-0001"); !errors.Is(err, slot.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState rejecting TRAVEL_CORRIDOR on the v1.0 create endpoint, got %v", err)
	}
}
