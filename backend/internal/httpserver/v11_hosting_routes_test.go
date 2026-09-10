package httpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type hostingHTTPStore struct {
	slot.Store
	created slot.Slot
}

func (s *hostingHTTPStore) CreateDraft(_ context.Context, _ string, candidate slot.Slot, _ string, _ []byte) (slot.Slot, error) {
	s.created = candidate
	return candidate, nil
}

func (s *hostingHTTPStore) PublishDraft(_ context.Context, _ string, slotID string, expectedVersion int64, _ string, _ []byte, now time.Time) (slot.Slot, error) {
	return slot.Slot{
		ID: slotID, State: slot.StateFilling, Version: expectedVersion + 1,
		UpdatedAt: now, ViewerState: slot.ViewerHost,
	}, nil
}

func TestCreateDraftSlotHTTPReturnsDraft(t *testing.T) {
	store := &hostingHTTPStore{}
	service, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Slots: service}}
	body := []byte(`{"title":"Coffee","activity":"coffee","placeText":"Center","capacity":4,"accessMode":"WAITLIST"}`)
	req := authenticatedRequest(http.MethodPost, "/v1/slots/drafts", body)
	req.Header.Set("Idempotency-Key", "00000000-0000-0000-0000-000000000001")
	rr := httptest.NewRecorder()

	server.createDraftSlot(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if store.created.State != slot.StateDraft {
		t.Fatalf("create draft routed to discoverable state: %s", store.created.State)
	}
	if store.created.AccessMode != slot.AccessWaitlist {
		t.Fatalf("draft access mode lost at HTTP boundary: %s", store.created.AccessMode)
	}
}

func TestPublishDraftSlotHTTPRequiresExpectedVersion(t *testing.T) {
	store := &hostingHTTPStore{}
	service, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Slots: service}}
	req := authenticatedRequest(
		http.MethodPost,
		"/v1/slots/00000000-0000-0000-0000-000000000010/publish",
		[]byte(`{"expectedVersion":7}`),
	)
	req.SetPathValue("slotID", "00000000-0000-0000-0000-000000000010")
	req.Header.Set("Idempotency-Key", "00000000-0000-0000-0000-000000000002")
	rr := httptest.NewRecorder()

	server.publishDraftSlot(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"state":"FILLING"`)) || !bytes.Contains(rr.Body.Bytes(), []byte(`"version":8`)) {
		t.Fatalf("unexpected publish response: %s", rr.Body.String())
	}
}
