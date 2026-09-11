package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeViewerFeedStore struct {
	batch     ViewerBatch
	err       error
	cursor    int64
	cursorErr error
	seen      struct {
		viewer string
		after  int64
		limit  int
	}
}

func (f *fakeViewerFeedStore) PullViewer(_ context.Context, viewerID string, after int64, limit int) (ViewerBatch, error) {
	f.seen.viewer = viewerID
	f.seen.after = after
	f.seen.limit = limit
	return f.batch, f.err
}

func (f *fakeViewerFeedStore) CurrentCursor(context.Context) (int64, error) {
	return f.cursor, f.cursorErr
}

func TestFeedServicePullDefaultsAndValidatesBatch(t *testing.T) {
	store := &fakeViewerFeedStore{batch: ViewerBatch{
		Cursor: 8,
		Events: []Event{testFeedEvent(7, "e7"), testFeedEvent(8, "e8")},
	}}
	service, err := NewFeedService(store)
	if err != nil {
		t.Fatal(err)
	}

	batch, err := service.Pull(context.Background(), " viewer ", 6, 0)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Cursor != 8 || len(batch.Events) != 2 {
		t.Fatalf("unexpected batch: %+v", batch)
	}
	if store.seen.viewer != "viewer" || store.seen.after != 6 || store.seen.limit != 100 {
		t.Fatalf("unexpected store call: %+v", store.seen)
	}
}

func TestFeedServiceRejectsInvalidInput(t *testing.T) {
	service, err := NewFeedService(&fakeViewerFeedStore{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		viewer string
		after  int64
		limit  int
	}{
		{"", 0, 1},
		{"viewer", -1, 1},
		{"viewer", 0, -1},
		{"viewer", 0, MaxViewerBatch + 1},
	} {
		if _, err := service.Pull(context.Background(), tc.viewer, tc.after, tc.limit); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("viewer=%q after=%d limit=%d err=%v", tc.viewer, tc.after, tc.limit, err)
		}
	}
}

func TestFeedServiceRejectsOutOfOrderOrDuplicateBatch(t *testing.T) {
	cases := []ViewerBatch{
		{Cursor: 4, Events: []Event{testFeedEvent(5, "e5")}},
		{Cursor: 6, Events: []Event{testFeedEvent(6, "same"), testFeedEvent(6, "other")}},
		{Cursor: 7, Events: []Event{testFeedEvent(6, "dup"), testFeedEvent(7, "dup")}},
	}
	for index, batch := range cases {
		service, _ := NewFeedService(&fakeViewerFeedStore{batch: batch})
		if _, err := service.Pull(context.Background(), "viewer", 5, 10); !errors.Is(err, ErrCursorOutOfOrder) {
			t.Fatalf("case %d err=%v", index, err)
		}
	}
}

func TestFeedServicePropagatesStoreFailure(t *testing.T) {
	want := errors.New("db down")
	service, _ := NewFeedService(&fakeViewerFeedStore{err: want})
	if _, err := service.Pull(context.Background(), "viewer", 0, 10); !errors.Is(err, want) {
		t.Fatalf("err=%v", err)
	}
}

func TestFeedServiceCurrentCursor(t *testing.T) {
	store := &fakeViewerFeedStore{cursor: 42}
	service, err := NewFeedService(store)
	if err != nil {
		t.Fatal(err)
	}
	got, err := service.CurrentCursor(context.Background(), " viewer ")
	if err != nil {
		t.Fatal(err)
	}
	if got != 42 {
		t.Fatalf("expected cursor 42, got %d", got)
	}
}

func TestFeedServiceCurrentCursorRejectsEmptyViewer(t *testing.T) {
	service, _ := NewFeedService(&fakeViewerFeedStore{})
	if _, err := service.CurrentCursor(context.Background(), "  "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestFeedServiceCurrentCursorPropagatesStoreFailure(t *testing.T) {
	want := errors.New("db down")
	service, _ := NewFeedService(&fakeViewerFeedStore{cursorErr: want})
	if _, err := service.CurrentCursor(context.Background(), "viewer"); !errors.Is(err, want) {
		t.Fatalf("err=%v", err)
	}
}

func testFeedEvent(sequence int64, id string) Event {
	return Event{
		Sequence:      sequence,
		ID:            id,
		Type:          "slot.updated",
		AggregateType: "slot",
		AggregateID:   "00000000-0000-0000-0000-000000000001",
		Payload:       json.RawMessage(`{"version":2}`),
		OccurredAt:    time.Unix(sequence, 0).UTC(),
	}
}
