package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
)

type fakeCityContextStore struct {
	applied citycontext.Observation
	result  citycontext.Context
	err     error
}

func (f *fakeCityContextStore) Apply(_ context.Context, _ string, observation citycontext.Observation, _ citycontext.Policy, _ time.Time) (citycontext.Context, error) {
	f.applied = observation
	return f.result, f.err
}

func (f *fakeCityContextStore) Current(_ context.Context, _ string, _ time.Time) (citycontext.Context, error) {
	return f.result, f.err
}

func TestResolveCityContextReturnsOnlyPrivacySafeLock(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeCityContextStore{result: citycontext.Context{
		Locality: citycontext.Locality{
			ID: "00000000-0000-0000-0000-000000000001", Name: "Kyiv", CountryCode: "UA", Timezone: "Europe/Kyiv",
		},
		PermissionClass: citycontext.PermissionPrecise,
		AccuracyM: 25,
		ObservedAt: now,
		ExpiresAt: now.Add(6 * time.Hour),
	}}
	service, err := citycontext.NewService(store, citycontext.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{CityContext: service}}
	body, _ := json.Marshal(resolveCityContextRequest{
		LatitudeE6: 50_450_100, LongitudeE6: 30_523_400, AccuracyM: 25,
		PermissionClass: string(citycontext.PermissionPrecise), CapturedAt: now.Format(time.RFC3339Nano),
	})
	req := authenticatedRequest(http.MethodPost, "/v1/city-context/resolve", body)
	rr := httptest.NewRecorder()

	server.resolveCityContext(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if store.applied.LatitudeE6 != 50_450_100 || store.applied.LongitudeE6 != 30_523_400 {
		t.Fatal("resolver did not receive observation")
	}
	response := rr.Body.String()
	if strings.Contains(response, "latitude") || strings.Contains(response, "longitude") {
		t.Fatalf("city context response must not expose observation coordinates: %s", response)
	}
	if !strings.Contains(response, "\"name\":\"Kyiv\"") || !strings.Contains(response, "\"timezone\":\"Europe/Kyiv\"") {
		t.Fatalf("missing locality metadata: %s", response)
	}
}

func TestResolveCityContextRejectsMockedObservationBeforeStore(t *testing.T) {
	store := &fakeCityContextStore{}
	service, err := citycontext.NewService(store, citycontext.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{CityContext: service}}
	now := time.Now().UTC()
	body, _ := json.Marshal(resolveCityContextRequest{
		LatitudeE6: 50_450_100, LongitudeE6: 30_523_400, AccuracyM: 25,
		PermissionClass: string(citycontext.PermissionPrecise), CapturedAt: now.Format(time.RFC3339Nano), Mocked: true,
	})
	req := authenticatedRequest(http.MethodPost, "/v1/city-context/resolve", body)
	rr := httptest.NewRecorder()

	server.resolveCityContext(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if store.applied.CapturedAt.IsZero() == false {
		t.Fatal("invalid mocked observation must not reach persistence store")
	}
}

func TestGetCityContextMapsMissingFreshLockTo404(t *testing.T) {
	store := &fakeCityContextStore{err: citycontext.ErrNotFound}
	service, err := citycontext.NewService(store, citycontext.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{CityContext: service}}
	req := authenticatedRequest(http.MethodGet, "/v1/city-context", nil)
	rr := httptest.NewRecorder()

	server.getCityContext(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var problemBody problem
	if err := json.Unmarshal(rr.Body.Bytes(), &problemBody); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(citycontext.ErrNotFound, citycontext.ErrNotFound) || problemBody.Code != "city_context_unavailable" {
		t.Fatalf("unexpected problem: %+v", problemBody)
	}
}

func authenticatedRequest(method, target string, body []byte) *http.Request {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	ctx := context.WithValue(req.Context(), authKey, authContext{User: account.User{ID: "00000000-0000-0000-0000-000000000099"}})
	return req.WithContext(ctx)
}
