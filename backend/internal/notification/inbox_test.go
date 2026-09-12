package notification

import (
	"context"
	"errors"
	"testing"
)

type fakeInboxStore struct {
	items  []Delivery
	unread int
	marked bool
	err    error
}

func (f *fakeInboxStore) List(_ context.Context, _ string, limit int, _ int64) ([]Delivery, error) {
	if f.err != nil {
		return nil, f.err
	}
	if limit < len(f.items) {
		return f.items[:limit], nil
	}
	return f.items, nil
}

func (f *fakeInboxStore) UnreadCount(_ context.Context, _ string) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.unread, nil
}

func (f *fakeInboxStore) MarkAllRead(_ context.Context, _ string) error {
	if f.err != nil {
		return f.err
	}
	f.marked = true
	return nil
}

func TestNewInboxServiceRejectsNilStore(t *testing.T) {
	if _, err := NewInboxService(nil); !errors.Is(err, ErrInvalidInboxRequest) {
		t.Fatalf("want ErrInvalidInboxRequest, got %v", err)
	}
}

func TestInboxServiceListRejectsBlankUserID(t *testing.T) {
	svc, err := NewInboxService(&fakeInboxStore{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.List(context.Background(), "  ", 10, 0); !errors.Is(err, ErrInvalidInboxRequest) {
		t.Fatalf("want ErrInvalidInboxRequest, got %v", err)
	}
}

func TestInboxServiceListClampsLimit(t *testing.T) {
	store := &fakeInboxStore{items: make([]Delivery, InboxListLimitMax+5)}
	svc, err := NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := svc.List(context.Background(), "user-1", InboxListLimitMax+50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Items) != InboxListLimitMax {
		t.Fatalf("want %d items (clamped), got %d", InboxListLimitMax, len(snapshot.Items))
	}
}

func TestInboxServiceListDefaultsZeroLimit(t *testing.T) {
	store := &fakeInboxStore{items: make([]Delivery, InboxListLimitDefault+10)}
	svc, err := NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := svc.List(context.Background(), "user-1", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Items) != InboxListLimitDefault {
		t.Fatalf("want %d items (default), got %d", InboxListLimitDefault, len(snapshot.Items))
	}
}

func TestInboxServiceListCarriesUnreadCountAndCursor(t *testing.T) {
	store := &fakeInboxStore{
		items:  []Delivery{{ID: "a", CreatedAt: 200}, {ID: "b", CreatedAt: 100}},
		unread: 7,
	}
	svc, err := NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := svc.List(context.Background(), "user-1", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.UnreadCount != 7 {
		t.Fatalf("unreadCount = %d, want 7", snapshot.UnreadCount)
	}
	// A full page (len(items) == limit) implies there may be more --
	// NextCursor should be set to the oldest item's timestamp.
	if snapshot.NextCursor != 100 {
		t.Fatalf("nextCursor = %d, want 100", snapshot.NextCursor)
	}
}

func TestInboxServiceListOmitsCursorOnPartialPage(t *testing.T) {
	store := &fakeInboxStore{items: []Delivery{{ID: "a", CreatedAt: 200}}}
	svc, err := NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := svc.List(context.Background(), "user-1", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.NextCursor != 0 {
		t.Fatalf("nextCursor = %d, want 0 (no more pages)", snapshot.NextCursor)
	}
}

func TestInboxServiceMarkAllReadRejectsBlankUserID(t *testing.T) {
	svc, err := NewInboxService(&fakeInboxStore{})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkAllRead(context.Background(), ""); !errors.Is(err, ErrInvalidInboxRequest) {
		t.Fatalf("want ErrInvalidInboxRequest, got %v", err)
	}
}

func TestInboxServiceMarkAllReadDelegatesToStore(t *testing.T) {
	store := &fakeInboxStore{}
	svc, err := NewInboxService(store)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkAllRead(context.Background(), "user-1"); err != nil {
		t.Fatal(err)
	}
	if !store.marked {
		t.Fatal("expected MarkAllRead to reach the store")
	}
}
