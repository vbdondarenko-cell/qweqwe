package httpserver

import (
	"errors"
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/appupdate"
)

// getAppUpdateLatest and downloadAppUpdate are both deliberately
// unauthenticated: an update check must work for a client whose session
// has already expired (or one that never had a valid one, e.g. right
// after a fresh sideload) -- this mirrors GET /v1/guardian-links/{token}'s
// same reasoning for a different reason (there, the holder need not be a
// LinkUp user at all; here, an expired session must not become a reason
// the app can never learn a fix exists).
func (s *Server) getAppUpdateLatest(w http.ResponseWriter, r *http.Request) {
	if !s.deps.AppUpdate.Configured() {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_configured", "app update is not configured")
		return
	}
	manifest, err := appupdate.ReadManifest(s.deps.AppUpdate.ManifestPath)
	if err != nil {
		s.writeAppUpdateError(w, r, err)
		return
	}
	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		scheme = "http"
	}
	writeJSON(w, http.StatusOK, appupdate.LatestInfo{
		Manifest: manifest,
		APKURL:   scheme + "://" + r.Host + "/v1/app-update/download",
	})
}

func (s *Server) downloadAppUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.deps.AppUpdate.Configured() {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_configured", "app update is not configured")
		return
	}
	// Confirm the manifest itself is still readable before streaming a
	// (possibly stale/orphaned) APK file -- the same fail-closed check
	// getAppUpdateLatest applies, so a broken publish never serves a
	// mismatched binary silently.
	if _, err := appupdate.ReadManifest(s.deps.AppUpdate.ManifestPath); err != nil {
		s.writeAppUpdateError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	w.Header().Set("Content-Disposition", `attachment; filename="LinkUp-update.apk"`)
	http.ServeFile(w, r, s.deps.AppUpdate.APKPath)
}

func (s *Server) writeAppUpdateError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, appupdate.ErrNotConfigured):
		writeProblem(w, r, http.StatusServiceUnavailable, "not_configured", "app update is not configured")
	case errors.Is(err, appupdate.ErrUnavailable):
		writeProblem(w, r, http.StatusServiceUnavailable, "update_unavailable", "app update information is temporarily unavailable")
	default:
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
	}
}
