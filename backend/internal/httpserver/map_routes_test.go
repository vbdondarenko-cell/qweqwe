package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citymap"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type recordingMapStore struct {
	viewportCalls        int
	placeSlotsCalls      int
	viewportLocalityID   string
	placeSlotsLocalityID string
}

func (s *recordingMapStore) Viewport(_ context.Context, _, localityID string, _ citymap.Viewport) ([]citymap.Cluster, error) {
	s.viewportCalls++
	s.viewportLocalityID = localityID
	return []citymap.Cluster{}, nil
}

func (s *recordingMapStore) PlaceSlots(_ context.Context, _, localityID string, _ citymap.PlaceSlotsQuery) ([]slot.Slot, error) {
	s.placeSlotsCalls++
	s.placeSlotsLocalityID = localityID
	return []slot.Slot{}, nil
}

func TestMapViewportRequiresFreshServerCityContext(t *testing.T) {
	cityService, err := citycontext.NewService(&fakeCityContextStore{err: citycontext.ErrNotFound}, citycontext.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	mapStore := &recordingMapStore{}
	mapService, err := citymap.NewService(mapStore)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{CityContext: cityService, Map: mapService}}
	req := authenticatedRequest(http.MethodGet, validMapTarget(), nil)
	rr := httptest.NewRecorder()

	server.mapViewport(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if mapStore.viewportCalls != 0 {
		t.Fatalf("map store called without fresh City Context: %d", mapStore.viewportCalls)
	}
}

func TestMapViewportUsesServerDerivedLocality(t *testing.T) {
	localityID := "00000000-0000-0000-0000-000000000777"
	cityService, err := citycontext.NewService(&fakeCityContextStore{result: citycontext.Context{
		Locality:        citycontext.Locality{ID: localityID, Name: "Kyiv", CountryCode: "UA", Timezone: "Europe/Kyiv"},
		PermissionClass: citycontext.PermissionApproximate,
		AccuracyM:       1000,
		ObservedAt:      time.Now().UTC(),
		ExpiresAt:       time.Now().UTC().Add(time.Hour),
	}}, citycontext.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	mapStore := &recordingMapStore{}
	mapService, err := citymap.NewService(mapStore)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{CityContext: cityService, Map: mapService}}
	req := authenticatedRequest(http.MethodGet, validMapTarget(), nil)
	rr := httptest.NewRecorder()

	server.mapViewport(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if mapStore.viewportCalls != 1 || mapStore.viewportLocalityID != localityID {
		t.Fatalf("server locality was not enforced: calls=%d locality=%q", mapStore.viewportCalls, mapStore.viewportLocalityID)
	}
}

func TestMapPlaceSlotsUsesServerDerivedLocality(t *testing.T) {
	localityID := "00000000-0000-0000-0000-000000000888"
	cityService, err := citycontext.NewService(&fakeCityContextStore{result: citycontext.Context{
		Locality:        citycontext.Locality{ID: localityID, Name: "Cherkasy", CountryCode: "UA", Timezone: "Europe/Kyiv"},
		PermissionClass: citycontext.PermissionApproximate,
		AccuracyM:       1000,
		ObservedAt:      time.Now().UTC(),
		ExpiresAt:       time.Now().UTC().Add(time.Hour),
	}}, citycontext.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	mapStore := &recordingMapStore{}
	mapService, err := citymap.NewService(mapStore)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{CityContext: cityService, Map: mapService}}
	req := authenticatedRequest(http.MethodGet, "/v1/map/places/00000000-0000-0000-0000-000000000123/slots?from=2026-09-10T10:00:00Z&to=2026-09-10T11:00:00Z&limit=20", nil)
	req.SetPathValue("placeID", "00000000-0000-0000-0000-000000000123")
	rr := httptest.NewRecorder()

	server.mapPlaceSlots(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if mapStore.placeSlotsCalls != 1 || mapStore.placeSlotsLocalityID != localityID {
		t.Fatalf("server locality was not enforced for place detail: calls=%d locality=%q", mapStore.placeSlotsCalls, mapStore.placeSlotsLocalityID)
	}
}

func validMapTarget() string {
	return "/v1/map?westE6=30000000&southE6=50000000&eastE6=31000000&northE6=51000000&zoom=13&from=2026-09-10T10:00:00Z&to=2026-09-10T11:00:00Z&limit=20"
}
