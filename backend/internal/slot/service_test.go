package slot

import (
	"context"
	"errors"
	"testing"
	"time"
)

type memoryStore struct {
	created     Slot
	createKey   string
	createHash  []byte
	lastEdit    EditInput
	lastEditKey string
	cancelled   bool
}

func (m *memoryStore) Create(_ context.Context, _ string, candidate Slot, key string, requestHash []byte) (Slot, error) {
	m.created = candidate
	m.createKey = key
	m.createHash = append([]byte(nil), requestHash...)
	candidate.Organizer = Organizer{ID: candidate.Organizer.ID, Username: "host", DisplayName: "Host"}
	return candidate, nil
}

func (m *memoryStore) Get(_ context.Context, _, slotID string) (Slot, error) {
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	return m.created, nil
}

func (m *memoryStore) ListPulse(_ context.Context, _ string, _ int) ([]Slot, error) {
	if m.created.ID == "" || m.cancelled {
		return []Slot{}, nil
	}
	return []Slot{m.created}, nil
}

func (m *memoryStore) Edit(_ context.Context, _, slotID string, patch EditInput, key string, _ []byte, now time.Time) (Slot, error) {
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	m.lastEdit = patch
	m.lastEditKey = key
	if patch.ExpectedVersion != m.created.Version {
		return Slot{}, ErrConflict
	}
	if patch.Title != nil {
		m.created.Title = *patch.Title
	}
	if patch.Capacity != nil {
		m.created.Capacity = *patch.Capacity
	}
	m.created.Version++
	m.created.UpdatedAt = now
	return m.created, nil
}

func (m *memoryStore) Cancel(_ context.Context, _, slotID string, expectedVersion int64, _ string, _ []byte, now time.Time) (Slot, error) {
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	if expectedVersion != m.created.Version {
		return Slot{}, ErrConflict
	}
	m.cancelled = true
	m.created.State = StateCancelled
	m.created.Version++
	m.created.UpdatedAt = now
	return m.created, nil
}

func TestCreateUsesFoundationPublicApprovalContract(t *testing.T) {
	store := &memoryStore{}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.Create(context.Background(), "host-id", CreateInput{
		Title:     "  Morning Coffee  ",
		Activity:  " COFFEE ",
		PlaceText: "  Podil  ",
		Capacity:  6,
	}, "create-slot-000001")
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != "Morning Coffee" || out.Activity != "coffee" || out.PlaceText != "Podil" {
		t.Fatalf("normalization failed: %#v", out)
	}
	if out.State != StateFilling || out.AccessMode != AccessApproval || out.Visibility != VisibilityPublic || out.Version != 1 {
		t.Fatalf("foundation contract mismatch: %#v", out)
	}
	if store.createKey != "create-slot-000001" || len(store.createHash) != 32 {
		t.Fatalf("idempotency data not forwarded: key=%q hashLen=%d", store.createKey, len(store.createHash))
	}
}

func TestCreateRejectsMissingIdempotencyKey(t *testing.T) {
	svc, _ := NewService(&memoryStore{})
	_, err := svc.Create(context.Background(), "host-id", CreateInput{
		Title:     "Coffee",
		Activity:  "coffee",
		PlaceText: "Podil",
		Capacity:  4,
	}, "")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestEditNormalizesAndRequiresExpectedVersion(t *testing.T) {
	store := &memoryStore{created: Slot{ID: "slot-id", Version: 3, Capacity: 6}}
	svc, _ := NewService(store)
	title := "  Updated title  "
	capacity := 8
	out, err := svc.Edit(context.Background(), "host-id", "slot-id", EditInput{
		ExpectedVersion: 3,
		Title:           &title,
		Capacity:        &capacity,
	}, "edit-slot-0000001")
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != "Updated title" || out.Capacity != 8 || out.Version != 4 {
		t.Fatalf("unexpected edit result: %#v", out)
	}
	if store.lastEditKey != "edit-slot-0000001" || store.lastEdit.Title == nil || *store.lastEdit.Title != "Updated title" {
		t.Fatalf("edit not normalized/forwarded: %#v", store.lastEdit)
	}

	_, err = svc.Edit(context.Background(), "host-id", "slot-id", EditInput{Title: &title}, "edit-slot-0000002")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected version validation, got %v", err)
	}
}

func TestCancelRequiresCurrentVersion(t *testing.T) {
	store := &memoryStore{created: Slot{ID: "slot-id", Version: 2, State: StateFilling}}
	svc, _ := NewService(store)
	if _, err := svc.Cancel(context.Background(), "host-id", "slot-id", 1, "cancel-slot-00001"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	out, err := svc.Cancel(context.Background(), "host-id", "slot-id", 2, "cancel-slot-00002")
	if err != nil {
		t.Fatal(err)
	}
	if out.State != StateCancelled || out.Version != 3 {
		t.Fatalf("unexpected cancel result: %#v", out)
	}
}
