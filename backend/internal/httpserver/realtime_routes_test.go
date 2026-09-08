package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
)

type fakeRealtimeFeed struct {
	batch realtime.ViewerBatch
	err   error
	seen  struct {
		viewer string
		after  int64
		limit  int
	}
}

func (f *fakeRealtimeFeed) Pull(_ context.Context, viewerID string, after int64, limit int) (realtime.ViewerBatch, error) {
	f.seen.viewer = viewerID
	f.seen.after = after
	f.seen.limit = limit
	return f.batch, f.err
}

func TestRealtimeEventsRequiresAuthContext(t *testing.T) {
	s := &Server{deps: Dependencies{Realtime: &fakeRealtimeFeed{}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/realtime/events", nil)
	s.realtimeEvents(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRealtimeEventsFailsClosedWithoutService(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()
	s.realtimeEvents(rec, authenticatedRealtimeRequest("/v1/realtime/events", "viewer-1"))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRealtimeEventsValidatesCursorAndLimit(t *testing.T) {
	s := &Server{deps: Dependencies{Realtime: &fakeRealtimeFeed{}}}
	for _, path := range []string{
		"/v1/realtime/events?after=-1",
		"/v1/realtime/events?after=abc",
		"/v1/realtime/events?limit=0",
		"/v1/realtime/events?limit=201",
		"/v1/realtime/events?limit=abc",
	} {
		rec := httptest.NewRecorder()
		s.realtimeEvents(rec, authenticatedRealtimeRequest(path, "viewer-1"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestRealtimeEventsReturnsViewerBatch(t *testing.T) {
	feed := &fakeRealtimeFeed{batch: realtime.ViewerBatch{
		Cursor: 42,
		Events: []realtime.Event{{
			Sequence: 42,
			ID: "event-42",
			Type: "slot.updated",
			AggregateType: "slot",
			AggregateID: "00000000-0000-0000-0000-000000000001",
			Payload: json.RawMessage(`{"version":2}`),
			OccurredAt: time.Unix(42, 0).UTC(),
		}},
	}}
	s := &Server{deps: Dependencies{Realtime: feed}}
	rec := httptest.NewRecorder()
	s.realtimeEvents(rec, authenticatedRealtimeRequest("/v1/realtime/events?after=41&limit=25", "viewer-1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got realtime.ViewerBatch
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Cursor != 42 || len(got.Events) != 1 || got.Events[0].ID != "event-42" {
		t.Fatalf("unexpected response: %+v", got)
	}
	if feed.seen.viewer != "viewer-1" || feed.seen.after != 41 || feed.seen.limit != 25 {
		t.Fatalf("unexpected feed call: %+v", feed.seen)
	}
}

func TestRealtimeEventsMapsFeedErrors(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{realtime.ErrInvalidInput, http.StatusBadRequest},
		{realtime.ErrCursorOutOfOrder, http.StatusBadRequest},
		{errors.New("db unavailable"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		s := &Server{deps: Dependencies{Realtime: &fakeRealtimeFeed{err: tc.err}}}
		rec := httptest.NewRecorder()
		s.realtimeEvents(rec, authenticatedRealtimeRequest("/v1/realtime/events", "viewer-1"))
		if rec.Code != tc.want {
			t.Fatalf("err=%v status=%d want=%d body=%s", tc.err, rec.Code, tc.want, rec.Body.String())
		}
	}
}

func authenticatedRealtimeRequest(path, userID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	ctx := context.WithValue(req.Context(), authKey, authContext{User: account.User{ID: userID}})
	return req.WithContext(ctx)
}
