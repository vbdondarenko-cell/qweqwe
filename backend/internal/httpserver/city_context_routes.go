package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
)

type resolveCityContextRequest struct {
	LatitudeE6      int    `json:"latitudeE6"`
	LongitudeE6     int    `json:"longitudeE6"`
	AccuracyM       int    `json:"accuracyM"`
	PermissionClass string `json:"permissionClass"`
	CapturedAt      string `json:"capturedAt"`
	Mocked           bool   `json:"mocked"`
}

func (s *Server) resolveCityContext(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.CityContext == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "city context service is unavailable")
		return
	}
	var in resolveCityContextRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid city context request")
		return
	}
	capturedAt, err := time.Parse(time.RFC3339Nano, in.CapturedAt)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_observation", "invalid location observation")
		return
	}
	out, err := s.deps.CityContext.Resolve(r.Context(), auth.User.ID, citycontext.Observation{
		LatitudeE6: in.LatitudeE6,
		LongitudeE6: in.LongitudeE6,
		AccuracyM: in.AccuracyM,
		PermissionClass: citycontext.PermissionClass(in.PermissionClass),
		CapturedAt: capturedAt.UTC(),
		Mocked: in.Mocked,
	})
	if err != nil {
		s.writeCityContextError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getCityContext(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.CityContext == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "city context service is unavailable")
		return
	}
	out, err := s.deps.CityContext.Current(r.Context(), auth.User.ID)
	if err != nil {
		s.writeCityContextError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) writeCityContextError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, citycontext.ErrInvalidObservation):
		writeProblem(w, r, http.StatusBadRequest, "invalid_observation", "location observation failed freshness or accuracy policy")
	case errors.Is(err, citycontext.ErrNoLocality):
		writeProblem(w, r, http.StatusUnprocessableEntity, "locality_unresolved", "no supported locality contains this observation")
	case errors.Is(err, citycontext.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "city_context_unavailable", "no fresh city context is available")
	case errors.Is(err, citycontext.ErrResolverUnavailable):
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "city context resolver is unavailable")
	default:
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
	}
}
