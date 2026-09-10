package httpserver

import (
	"net/http"
	"strconv"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citymap"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type mapViewportResponse struct {
	Items []citymap.Cluster `json:"items"`
}

type mapPlaceSlotsResponse struct {
	Items []slot.Slot `json:"items"`
}

func (s *Server) mapViewport(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Map == nil || s.deps.CityContext == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "map dependencies are unavailable")
		return
	}
	city, err := s.deps.CityContext.Current(r.Context(), auth.User.ID)
	if err != nil {
		s.writeCityContextError(w, r, err)
		return
	}

	west, err1 := strconv.Atoi(r.URL.Query().Get("westE6"))
	south, err2 := strconv.Atoi(r.URL.Query().Get("southE6"))
	east, err3 := strconv.Atoi(r.URL.Query().Get("eastE6"))
	north, err4 := strconv.Atoi(r.URL.Query().Get("northE6"))
	zoom, err5 := strconv.Atoi(r.URL.Query().Get("zoom"))
	from, err6 := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, err7 := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	limit := 100
	var err8 error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err8 = strconv.Atoi(raw)
	}
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil || err7 != nil || err8 != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_viewport", "invalid map viewport")
		return
	}

	items, err := s.deps.Map.Viewport(r.Context(), auth.User.ID, city.Locality.ID, citymap.Viewport{
		WestE6: west, SouthE6: south, EastE6: east, NorthE6: north,
		Zoom: zoom, From: from, To: to, Limit: limit,
	})
	if err != nil {
		if err == citymap.ErrInvalidViewport {
			writeProblem(w, r, http.StatusBadRequest, "invalid_viewport", "invalid map viewport")
		} else {
			writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, mapViewportResponse{Items: items})
}

func (s *Server) mapPlaceSlots(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Map == nil || s.deps.CityContext == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "map dependencies are unavailable")
		return
	}
	city, err := s.deps.CityContext.Current(r.Context(), auth.User.ID)
	if err != nil {
		s.writeCityContextError(w, r, err)
		return
	}

	from, err1 := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, err2 := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	limit := 50
	var err3 error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err3 = strconv.Atoi(raw)
	}
	if err1 != nil || err2 != nil || err3 != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_map_place", "invalid map place query")
		return
	}

	items, err := s.deps.Map.PlaceSlots(r.Context(), auth.User.ID, city.Locality.ID, citymap.PlaceSlotsQuery{
		PlaceID: r.PathValue("placeID"), From: from, To: to, Limit: limit,
	})
	if err != nil {
		if err == citymap.ErrInvalidViewport {
			writeProblem(w, r, http.StatusBadRequest, "invalid_map_place", "invalid map place query")
		} else {
			writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, mapPlaceSlotsResponse{Items: items})
}
