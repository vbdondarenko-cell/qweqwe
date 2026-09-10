package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
)

type fakeCityRealtimeFeed struct {
	batch realtime.CityBatch
	err   error
	seen  struct {
		viewer string
		after  int64
		limit  int
	}
}

func (f *fakeCityRealtimeFeed) Pull(_ context.Context, viewerID string, after int64, limit int) (realtime.CityBatch, error) {
	f.seen.viewer = viewerID
	f.seen.after = after
	f.seen.limit = limit
	return f.batch, f.err
}

func TestRealtimeCityRequiresAuthContext(t *testing.T) {
	s := &Server{deps: Dependencies{CityRealtime: &fakeCityRealtimeFeed{}}}
	rec := httptest.NewRecorder()
	s.realtimeCity(rec, httptest.NewRequest(http.MethodGet, "/v1/realtime/city", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRealtimeCityFailsClosedWithoutService(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()
	s.realtimeCity(rec, authenticatedRealtimeRequest("/v1/realtime/city", "viewer-1"))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRealtimeCityValidatesCursorAndLimit(t *testing.T) {
	s := &Server{deps: Dependencies{CityRealtime: &fakeCityRealtimeFeed{}}}
	for _, path := range []string{
		"/v1/realtime/city?after=-1",
		"/v1/realtime/city?after=abc",
		"/v1/realtime/city?limit=0",
		"/v1/realtime/city?limit=201",
		"/v1/realtime/city?limit=abc",
	} {
		rec := httptest.NewRecorder()
		s.realtimeCity(rec, authenticatedRealtimeRequest(path, "viewer-1"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestRealtimeCityMapsContextAndFeedErrors(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{citycontext.ErrNotFound, http.StatusNotFound},
		{realtime.ErrInvalidInput, http.StatusBadRequest},
		{realtime.ErrCursorOutOfOrder, http.StatusBadRequest},
		{errors.New("db unavailable"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		s := &Server{deps: Dependencies{CityRealtime: &fakeCityRealtimeFeed{err: tc.err}}}
		rec := httptest.NewRecorder()
		s.realtimeCity(rec, authenticatedRealtimeRequest("/v1/realtime/city", "viewer-1"))
		if rec.Code != tc.want {
			t.Fatalf("err=%v status=%d want=%d body=%s", tc.err, rec.Code, tc.want, rec.Body.String())
		}
	}
}

func TestRealtimeCityReturnsPrivacySafeBatch(t *testing.T) {
	feed := &fakeCityRealtimeFeed{batch: realtime.CityBatch{
		Cursor: 42,
		Invalidations: []realtime.CityInvalidation{{
			Sequence:   42,
			ID:         "event-42",
			OccurredAt: time.Unix(42, 0).UTC(),
		}},
	}}
	s := &Server{deps: Dependencies{CityRealtime: feed}}
	rec := httptest.NewRecorder()
	s.realtimeCity(rec, authenticatedRealtimeRequest("/v1/realtime/city?after=41&limit=25", "viewer-1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	raw := rec.Body.String()
	var got realtime.CityBatch
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Cursor != 42 || len(got.Invalidations) != 1 || got.Invalidations[0].ID != "event-42" {
		t.Fatalf("unexpected response: %+v", got)
	}
	if feed.seen.viewer != "viewer-1" || feed.seen.after != 41 || feed.seen.limit != 25 {
		t.Fatalf("unexpected feed call: %+v", feed.seen)
	}
	if raw == "" || containsAny(raw, "slotId", "latitude", "longitude", "localityId") {
		t.Fatalf("city realtime response leaked canonical/user detail: %s", raw)
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if len(needle) > 0 && len(value) >= len(needle) {
			for i := 0; i+len(needle) <= len(value); i++ {
				if value[i:i+len(needle)] == needle {
					return true
				}
			}
		}
	}
	return false
}
