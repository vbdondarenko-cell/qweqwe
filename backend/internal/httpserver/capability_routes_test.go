package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
)

type capabilityRouteStore struct {
	entries []capability.Entry
	err     error
}

func (f capabilityRouteStore) List(context.Context) ([]capability.Entry, error) {
	return f.entries, f.err
}

func requestWithCapabilityAuth() *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(request.Context(), authKey, authContext{User: account.User{ID: "11111111-1111-1111-1111-111111111111"}})
	return request.WithContext(ctx)
}

func TestCapabilityGateRejectsDisabledEvenWhenHandlerWouldRun(t *testing.T) {
	svc, err := capability.NewService(capabilityRouteStore{entries: []capability.Entry{
		{Key: capability.Realtime, Enabled: false, Revision: 1, ScopeType: "ALL"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Capabilities: svc}}
	called := false
	handler := server.requireCapability(capability.Realtime, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, requestWithCapabilityAuth())
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if called {
		t.Fatal("disabled capability reached protected handler")
	}
	var body problem
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "capability_disabled" {
		t.Fatalf("code=%q", body.Code)
	}
}

func TestCapabilityGateFailsClosedOnRegistryError(t *testing.T) {
	svc, err := capability.NewService(capabilityRouteStore{err: errors.New("db unavailable")})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Capabilities: svc}}
	called := false
	handler := server.requireCapability(capability.Map, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, requestWithCapabilityAuth())
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if called {
		t.Fatal("registry error must never open protected handler")
	}
}

func TestCapabilityGateAllowsServerEnabledScope(t *testing.T) {
	userID := "11111111-1111-1111-1111-111111111111"
	svc, err := capability.NewService(capabilityRouteStore{entries: []capability.Entry{
		{Key: capability.Map, Enabled: true, Revision: 3, ScopeType: "USER_ALLOWLIST", ScopeUserIDs: []string{userID}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Capabilities: svc}}
	handler := server.requireCapability(capability.Map, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, requestWithCapabilityAuth())
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCapabilitiesEndpointReturnsDisabledSnapshotWhenRegistryUnavailable(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	server.capabilities(recorder, requestWithCapabilityAuth())
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d", recorder.Code)
	}
	var snapshot capability.Snapshot
	if err := json.NewDecoder(recorder.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Revision != 0 {
		t.Fatalf("revision=%d", snapshot.Revision)
	}
	if len(snapshot.Capabilities) != len(capability.KnownKeys()) {
		t.Fatalf("capabilities=%d", len(snapshot.Capabilities))
	}
	for key, enabled := range snapshot.Capabilities {
		if enabled {
			t.Fatalf("%s unexpectedly enabled", key)
		}
	}
}
