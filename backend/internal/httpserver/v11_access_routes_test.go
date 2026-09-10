package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type accessHTTPStore struct {
	slot.Store
	joinedActor string
	joinedSlot  string
}

func (s *accessHTTPStore) Join(_ context.Context, actorID, slotID, _ string, _ []byte, _ time.Time) (slot.Slot, error) {
	s.joinedActor = actorID
	s.joinedSlot = slotID
	return slot.Slot{ID: slotID, AccessMode: slot.AccessInstant, ViewerState: slot.ViewerAccepted, State: slot.StateFilling, Version: 2}, nil
}

func TestJoinSlotHTTPUsesDedicatedInstantTransition(t *testing.T) {
	store := &accessHTTPStore{}
	service, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Slots: service}}
	req := authenticatedRequest(http.MethodPost, "/v1/slots/00000000-0000-0000-0000-000000000010/join", nil)
	req.SetPathValue("slotID", "00000000-0000-0000-0000-000000000010")
	req.Header.Set("Idempotency-Key", "00000000-0000-0000-0000-000000000002")
	rr := httptest.NewRecorder()

	server.joinSlot(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if store.joinedSlot != "00000000-0000-0000-0000-000000000010" || store.joinedActor == "" {
		t.Fatalf("join did not reach access authority: %#v", store)
	}
}
