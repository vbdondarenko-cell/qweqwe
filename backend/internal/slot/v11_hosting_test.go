package slot

import (
	"context"
	"errors"
	"testing"
	"time"
)

type hostingStoreStub struct {
	Store
	created      Slot
	publishedID  string
	publishedVer int64
}

func (s *hostingStoreStub) CreateDraft(_ context.Context, _ string, candidate Slot, _ string, _ []byte) (Slot, error) {
	s.created = candidate
	return candidate, nil
}

func (s *hostingStoreStub) PublishDraft(_ context.Context, _ string, slotID string, expectedVersion int64, _ string, _ []byte, _ time.Time) (Slot, error) {
	s.publishedID = slotID
	s.publishedVer = expectedVersion
	return Slot{ID: slotID, State: StateFilling, Version: expectedVersion + 1}, nil
}

func TestCreateDraftUsesNonDiscoverableDraftState(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC) }

	out, err := service.CreateDraft(context.Background(), "host-id", CreateInput{
		Title:     "Coffee later",
		Activity:  "coffee",
		PlaceText: "Central Cafe",
		Capacity:  4,
	}, "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if out.State != StateDraft || store.created.State != StateDraft {
		t.Fatalf("draft must stay non-discoverable: out=%s stored=%s", out.State, store.created.State)
	}
	if out.AccessMode != AccessApproval || out.Visibility != VisibilityPublic || out.AcceptedCount != 0 {
		t.Fatalf("unexpected draft authority defaults: %#v", out)
	}
}

func TestCreateDraftPreservesConfiguredAccessMode(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	mode := AccessWaitlist
	out, err := service.CreateDraft(context.Background(), "host-id", CreateInput{
		Title: "Coffee later", Activity: "coffee", PlaceText: "Central Cafe", Capacity: 4, AccessMode: &mode,
	}, "00000000-0000-0000-0000-000000000003")
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessMode != AccessWaitlist || store.created.AccessMode != AccessWaitlist {
		t.Fatalf("draft access mode lost: out=%s stored=%s", out.AccessMode, store.created.AccessMode)
	}
}

func TestCreateDraftPreservesConfiguredVisibility(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := VisibilityPrivate
	out, err := service.CreateDraft(context.Background(), "host-id", CreateInput{
		Title: "Coffee later", Activity: "coffee", PlaceText: "Central Cafe", Capacity: 4, Visibility: &visibility,
	}, "00000000-0000-0000-0000-000000000004")
	if err != nil {
		t.Fatal(err)
	}
	if out.Visibility != VisibilityPrivate || store.created.Visibility != VisibilityPrivate {
		t.Fatalf("draft visibility lost: out=%s stored=%s", out.Visibility, store.created.Visibility)
	}
}

func TestCreateDraftPreservesConfiguredLinksVisibility(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := VisibilityLinks
	out, err := service.CreateDraft(context.Background(), "host-id", CreateInput{
		Title: "Coffee later", Activity: "coffee", PlaceText: "Central Cafe", Capacity: 4, Visibility: &visibility,
	}, "00000000-0000-0000-0000-000000000006")
	if err != nil {
		t.Fatal(err)
	}
	if out.Visibility != VisibilityLinks || store.created.Visibility != VisibilityLinks {
		t.Fatalf("draft visibility lost: out=%s stored=%s", out.Visibility, store.created.Visibility)
	}
}

func TestCreateDraftPreservesConfiguredCityVisibility(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := VisibilityCity
	out, err := service.CreateDraft(context.Background(), "host-id", CreateInput{
		Title: "City meetup", Activity: "coffee", PlaceText: "Central Cafe", Capacity: 4, Visibility: &visibility,
	}, "00000000-0000-0000-0000-000000000007")
	if err != nil {
		t.Fatal(err)
	}
	if out.Visibility != VisibilityCity || store.created.Visibility != VisibilityCity {
		t.Fatalf("draft visibility lost: out=%s stored=%s", out.Visibility, store.created.Visibility)
	}
	if len(store.created.SelectedUserIDs) != 0 {
		t.Fatalf("CITY visibility has no allow-list; expected none stored, got %v", store.created.SelectedUserIDs)
	}
}

func TestCreateDraftPreservesSelectedVisibilityAndAllowList(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := VisibilitySelected
	selected := []string{"00000000-0000-0000-0000-0000000000aa", "00000000-0000-0000-0000-0000000000bb"}
	out, err := service.CreateDraft(context.Background(), "host-id", CreateInput{
		Title: "Coffee later", Activity: "coffee", PlaceText: "Central Cafe", Capacity: 4,
		Visibility: &visibility, SelectedUserIDs: selected,
	}, "00000000-0000-0000-0000-000000000007")
	if err != nil {
		t.Fatal(err)
	}
	if out.Visibility != VisibilitySelected || store.created.Visibility != VisibilitySelected {
		t.Fatalf("draft visibility lost: out=%s stored=%s", out.Visibility, store.created.Visibility)
	}
	if len(out.SelectedUserIDs) != 2 || len(store.created.SelectedUserIDs) != 2 {
		t.Fatalf("draft allow-list lost: out=%v stored=%v", out.SelectedUserIDs, store.created.SelectedUserIDs)
	}
}

func TestCreateDraftRejectsSelectedVisibilityWithoutAllowList(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := VisibilitySelected
	if _, err := service.CreateDraft(context.Background(), "host-id", CreateInput{
		Title: "Coffee later", Activity: "coffee", PlaceText: "Central Cafe", Capacity: 4, Visibility: &visibility,
	}, "00000000-0000-0000-0000-000000000008"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for SELECTED with no allow-list, got %v", err)
	}
}

func TestCreateDraftRejectsSelectedVisibilityWithInvalidUserID(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := VisibilitySelected
	if _, err := service.CreateDraft(context.Background(), "host-id", CreateInput{
		Title: "Coffee later", Activity: "coffee", PlaceText: "Central Cafe", Capacity: 4,
		Visibility: &visibility, SelectedUserIDs: []string{"not-a-uuid"},
	}, "00000000-0000-0000-0000-000000000009"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for a malformed selected user id, got %v", err)
	}
}

func TestCreateDraftIgnoresSelectedUserIDsForOtherVisibilities(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	out, err := service.CreateDraft(context.Background(), "host-id", CreateInput{
		Title: "Coffee later", Activity: "coffee", PlaceText: "Central Cafe", Capacity: 4,
		SelectedUserIDs: []string{"00000000-0000-0000-0000-0000000000aa"},
	}, "00000000-0000-0000-0000-00000000000a")
	if err != nil {
		t.Fatal(err)
	}
	if out.Visibility != VisibilityPublic {
		t.Fatalf("expected default PUBLIC visibility, got %s", out.Visibility)
	}
	if len(out.SelectedUserIDs) != 0 {
		t.Fatalf("expected SelectedUserIDs to be ignored for PUBLIC visibility, got %v", out.SelectedUserIDs)
	}
}

func TestCreateDraftRejectsUnknownVisibility(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	bogus := Visibility("SELECTED")
	if _, err := service.CreateDraft(context.Background(), "host-id", CreateInput{
		Title: "Coffee later", Activity: "coffee", PlaceText: "Central Cafe", Capacity: 4, Visibility: &bogus,
	}, "00000000-0000-0000-0000-000000000005"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for an unimplemented visibility mode, got %v", err)
	}
}

func TestPublishDraftRequiresExpectedVersionAndDelegatesAtomicTransition(t *testing.T) {
	store := &hostingStoreStub{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC) }

	out, err := service.PublishDraft(
		context.Background(),
		"host-id",
		"00000000-0000-0000-0000-000000000010",
		7,
		"00000000-0000-0000-0000-000000000002",
	)
	if err != nil {
		t.Fatal(err)
	}
	if store.publishedID != out.ID || store.publishedVer != 7 || out.State != StateFilling || out.Version != 8 {
		t.Fatalf("unexpected publish transition: %#v store=%#v", out, store)
	}
}
