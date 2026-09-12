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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestV11PrivateVisibilitySlot closes README §4.3's first non-PUBLIC
// visibility mode: PRIVATE. It proves the full contract in one place:
//
//  1. A PRIVATE Slot is invisible to a stranger on every existing PUBLIC-gated
//     discovery surface (ListPulse, Map PlaceSlots, Get by a viewer with no
//     relationship yet) — automatically, because those queries already
//     require visibility='PUBLIC' and were never relaxed for PRIVATE.
//  2. A stranger who is handed the Slot ID directly can still call Request()
//     on it — this is the deliberate "shared ID = invite" design: Request()/
//     Join()/Approve() never reference visibility at all.
//  3. Once that stranger has a pending request, Get() becomes visible to
//     them via the existing slot_requests-EXISTS branch, same as any other
//     Slot.
//  4. The host always has full access regardless of visibility.
func TestV11PrivateVisibilitySlot(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "pvh", suffix)
	stranger := registerIntegrationUser(t, ctx, accountService, "pvs", suffix)

	localityID, err := identifier.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	placeID, err := identifier.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	// t.Cleanup runs LIFO: canonical_places carries ON DELETE RESTRICT from
	// slots.canonical_place_id (migration 000015), so the users/slots cleanup
	// (registered second, deletes slots first) must run before this one
	// (registered first) actually deletes the place.
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM canonical_places WHERE id=$1::uuid`, placeID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM localities WHERE id=$1::uuid`, localityID)
	})
	t.Cleanup(func() {
		cleanupIntegrationRows(pool, []string{host.User.ID, stranger.User.ID})
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO localities (
			id,source,source_locality_id,name,country_code,timezone_name,
			centroid_latitude_e6,centroid_longitude_e6,boundary_wkt
		) VALUES ($1::uuid,'integration',$2,'Kyiv','UA','Europe/Kyiv',
			50500000,30500000,'MULTIPOLYGON(((30 50,31 50,31 51,30 51,30 50)))')`,
		localityID, "private-visibility-"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO canonical_places (
			id,source,source_place_id,name,category,locality,country_code,
			latitude_e6,longitude_e6,precision_m,locality_id
		) VALUES ($1::uuid,'integration',$2,'Private Visibility Place','cafe','Kyiv','UA',50500000,30500000,50,$3::uuid)`,
		placeID, "private-visibility-place-"+suffix, localityID); err != nil {
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
	visibility := slot.VisibilityPrivate
	draft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Private Meetup", Activity: "coffee", PlaceText: "Private Visibility Place",
		CanonicalPlaceID: &placeID, StartAt: &start, Capacity: 3, Visibility: &visibility,
	}, "v11-private-visibility-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	if draft.Visibility != slot.VisibilityPrivate {
		t.Fatalf("draft did not preserve PRIVATE visibility: %#v", draft)
	}
	published, err := slotService.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "v11-private-visibility-publish-0001")
	if err != nil {
		t.Fatal(err)
	}
	if published.Visibility != slot.VisibilityPrivate || published.State != slot.StateFilling {
		t.Fatalf("published slot lost PRIVATE visibility or wrong state: %#v", published)
	}

	// (1a) ListPulse must never surface a PRIVATE slot to a stranger.
	pulse, err := slotService.ListPulse(ctx, stranger.User.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range pulse {
		if item.ID == published.ID {
			t.Fatalf("PRIVATE slot must not appear in a stranger's Pulse feed: %#v", item)
		}
	}

	// (1b) Map PlaceSlots must never surface a PRIVATE slot to a stranger.
	mapStore, err := NewCityMapStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	mapService, err := citymap.NewService(mapStore)
	if err != nil {
		t.Fatal(err)
	}
	placeSlots, err := mapService.PlaceSlots(ctx, stranger.User.ID, localityID, citymap.PlaceSlotsQuery{
		PlaceID: placeID, From: start.Add(-time.Hour), To: start.Add(time.Hour), Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range placeSlots {
		if item.ID == published.ID {
			t.Fatalf("PRIVATE slot must not appear on the Map for a stranger: %#v", item)
		}
	}

	// (1c) Get() for a stranger with no relationship yet must 404, not leak
	// the PRIVATE slot's existence.
	if _, err := slotService.Get(ctx, stranger.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a stranger reading a PRIVATE slot by ID, got %v", err)
	}

	// (4) The host always retains full access regardless of visibility.
	hostView, err := slotService.Get(ctx, host.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if hostView.ViewerState != slot.ViewerHost || hostView.Visibility != slot.VisibilityPrivate {
		t.Fatalf("host must retain full access to their own PRIVATE slot: %#v", hostView)
	}

	// (2) A stranger who is handed the Slot ID directly can still Request()
	// on it — visibility is a discoverability gate, not an access-control
	// gate; Request()/Join()/Approve() never reference visibility.
	afterRequest, err := slotService.Request(ctx, stranger.User.ID, published.ID, "v11-private-visibility-request-0001")
	if err != nil {
		t.Fatalf("a stranger with a known PRIVATE slot ID must still be able to Request(): %v", err)
	}
	if afterRequest.ViewerState != slot.ViewerPending {
		t.Fatalf("expected PENDING after Request() on a PRIVATE slot, got %s", afterRequest.ViewerState)
	}

	// (3) Once a pending request exists, Get() becomes visible to that
	// requester via the existing slot_requests-EXISTS branch.
	requesterView, err := slotService.Get(ctx, stranger.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if requesterView.ViewerState != slot.ViewerPending || requesterView.Visibility != slot.VisibilityPrivate {
		t.Fatalf("requester should now see the PRIVATE slot as PENDING: %#v", requesterView)
	}

	// After approval, the accepted member keeps full access too.
	approved, err := slotService.Approve(ctx, host.User.ID, published.ID, stranger.User.ID, "v11-private-visibility-approve-0001")
	if err != nil {
		t.Fatal(err)
	}
	if approved.AcceptedCount != 1 {
		t.Fatalf("expected 1 accepted member after approval: %#v", approved)
	}
	memberView, err := slotService.Get(ctx, stranger.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if memberView.ViewerState != slot.ViewerAccepted {
		t.Fatalf("accepted member must retain access to a PRIVATE slot: %#v", memberView)
	}
}

// TestV11PrivateVisibilityRejectedForLegacyCreate is the regression check
// for the v1.0 create endpoint: it must keep rejecting any visibility other
// than PUBLIC, since v1.0 clients have no UI concept of PRIVATE and the
// v1.0 Store implementation was never audited for non-PUBLIC visibility.
func TestV11PrivateVisibilityRejectedForLegacyCreate(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "pvl", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID}) })

	store, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := slot.VisibilityPrivate
	if _, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Legacy Private Attempt", Activity: "coffee", PlaceText: "Center", Capacity: 3, Visibility: &visibility,
	}, "v11-private-visibility-legacy-0001"); !errors.Is(err, slot.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState rejecting PRIVATE on the v1.0 create endpoint, got %v", err)
	}
}

// TestV11EditVisibilityAllowedOnlyWhileDraft closes the Edit-time gap §52
// deliberately left open: a host can now flip a draft between PUBLIC and
// PRIVATE before publishing, but once a Slot is PUBLISHED/FILLING/FULL,
// changing who can discover it is a bigger decision than a plain Edit
// should silently make (it could yank visibility out from under strangers
// who already found it, or reveal a Slot from under a shared-ID-only
// arrangement) — so V11SlotStore.Edit rejects a Visibility patch there,
// exactly mirroring the existing AccessMode-after-DRAFT restriction.
func TestV11EditVisibilityAllowedOnlyWhileDraft(t *testing.T) {
	ctx, _, service, host, _ := newV11WaitlistFixture(t)

	draft, err := service.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Edit Visibility Draft", Activity: "coffee", PlaceText: "Center", Capacity: 3,
	}, "v11-edit-visibility-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	if draft.Visibility != slot.VisibilityPublic {
		t.Fatalf("expected default PUBLIC draft: %#v", draft)
	}

	private := slot.VisibilityPrivate
	edited, err := service.Edit(ctx, host.User.ID, draft.ID, slot.EditInput{
		ExpectedVersion: draft.Version, Visibility: &private,
	}, "v11-edit-visibility-draft-edit-0001")
	if err != nil {
		t.Fatalf("expected DRAFT to allow a visibility edit: %v", err)
	}
	if edited.Visibility != slot.VisibilityPrivate {
		t.Fatalf("expected edited draft to be PRIVATE: %#v", edited)
	}

	published, err := service.PublishDraft(ctx, host.User.ID, edited.ID, edited.Version, "v11-edit-visibility-publish-0001")
	if err != nil {
		t.Fatal(err)
	}
	if published.Visibility != slot.VisibilityPrivate {
		t.Fatalf("expected published slot to keep the edited PRIVATE visibility: %#v", published)
	}

	public := slot.VisibilityPublic
	if _, err := service.Edit(ctx, host.User.ID, published.ID, slot.EditInput{
		ExpectedVersion: published.Version, Visibility: &public,
	}, "v11-edit-visibility-published-edit-0001"); !errors.Is(err, slot.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState editing visibility after publish, got %v", err)
	}

	// The rejected edit must not have silently applied anyway.
	unchanged, err := service.Get(ctx, host.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Visibility != slot.VisibilityPrivate {
		t.Fatalf("visibility must remain unchanged after a rejected post-publish edit: %#v", unchanged)
	}
}
