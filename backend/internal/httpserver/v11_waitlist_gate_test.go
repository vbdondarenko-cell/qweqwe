package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type waitlistHTTPStore struct {
	slot.Store
	current      slot.Slot
	requestCalls int
}

func (s *waitlistHTTPStore) Get(_ context.Context, _, slotID string) (slot.Slot, error) {
	if s.current.ID != slotID {
		return slot.Slot{}, slot.ErrNotFound
	}
	return s.current, nil
}

func (s *waitlistHTTPStore) Request(_ context.Context, _, slotID, _ string, _ []byte, _ time.Time) (slot.Slot, error) {
	if s.current.ID != slotID {
		return slot.Slot{}, slot.ErrNotFound
	}
	s.requestCalls++
	out := s.current
	out.ViewerState = slot.ViewerPending
	out.Version++
	return out, nil
}

func TestWaitlistRequestFailsClosedBeforeMutation(t *testing.T) {
	store := &waitlistHTTPStore{current: slot.Slot{
		ID: "00000000-0000-0000-0000-000000000021", AccessMode: slot.AccessWaitlist,
		State: slot.StateFull, Organizer: slot.Organizer{ID: "00000000-0000-0000-0000-000000000001"},
	}}
	service, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Slots: service}}
	req := authenticatedRequest(http.MethodPost, "/v1/slots/00000000-0000-0000-0000-000000000021/request", nil)
	req.SetPathValue("slotID", store.current.ID)
	req.Header.Set("Idempotency-Key", "00000000-0000-0000-0000-000000000021")
	rr := httptest.NewRecorder()

	server.requestSlot(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if store.requestCalls != 0 {
		t.Fatalf("disabled waitlist reached mutation: calls=%d", store.requestCalls)
	}
}

func TestWaitlistRequestRunsWhenCapabilityEnabled(t *testing.T) {
	store := &waitlistHTTPStore{current: slot.Slot{
		ID: "00000000-0000-0000-0000-000000000022", AccessMode: slot.AccessWaitlist,
		State: slot.StateFull, Organizer: slot.Organizer{ID: "00000000-0000-0000-0000-000000000001"},
	}}
	service, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	caps, err := capability.NewService(capabilityRouteStore{entries: []capability.Entry{{
		Key: capability.Waitlist, Enabled: true, Revision: 1, ScopeType: "ALL",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Slots: service, Capabilities: caps}}
	req := authenticatedRequest(http.MethodPost, "/v1/slots/00000000-0000-0000-0000-000000000022/request", nil)
	req.SetPathValue("slotID", store.current.ID)
	req.Header.Set("Idempotency-Key", "00000000-0000-0000-0000-000000000022")
	rr := httptest.NewRecorder()

	server.requestSlot(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if store.requestCalls != 1 {
		t.Fatalf("enabled waitlist mutation calls=%d", store.requestCalls)
	}
}
