package httpserver

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/places"
)

type placeSearchResponse struct {
	Items []places.Place `json:"items"`
}

func (s *Server) searchPlaces(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r); !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Places == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "place service is unavailable")
		return
	}

	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeProblem(w, r, http.StatusBadRequest, "invalid_place_search", "invalid place search")
			return
		}
		limit = parsed
	}
	items, err := s.deps.Places.Search(r.Context(), places.SearchQuery{
		Text:       r.URL.Query().Get("q"),
		Locality:   r.URL.Query().Get("locality"),
		LocalityID: r.URL.Query().Get("localityId"),
		Limit:      limit,
	})
	if err != nil {
		if errors.Is(err, places.ErrInvalidSearch) {
			writeProblem(w, r, http.StatusBadRequest, "invalid_place_search", "invalid place search")
		} else {
			writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, placeSearchResponse{Items: items})
}
