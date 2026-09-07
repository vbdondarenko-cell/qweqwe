package httpserver

import "net/http"

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
    auth, _ := authFrom(r)
    if s.deps.Slots == nil {
        writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
        return
    }
    var in struct { ExpectedVersion int64 `json:"expectedVersion"` }
    if err := decodeJSON(w, r, &in); err != nil {
        writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
        return
    }
    out, err := s.deps.Slots.RemoveMember(r.Context(), auth.User.ID, r.PathValue("slotID"), r.PathValue("userID"), in.ExpectedVersion, r.Header.Get("Idempotency-Key"))
    if err != nil { s.writeSlotError(w, r, err); return }
    writeJSON(w, http.StatusOK, out)
}
