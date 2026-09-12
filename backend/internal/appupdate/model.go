// Package appupdate implements over-the-air update checks for the
// sideloaded Android build: the app asks "is there a newer version?" and,
// if so, downloads the new APK straight from this same server rather than
// the user needing to reinstall by hand. This is deliberately NOT a Play
// Store replacement — there is no delta/streaming install, no staged
// rollout, no automatic silent install (Android requires explicit user
// consent to install an APK from outside a store either way) — it is the
// minimum real mechanism for "the app can tell you a new build exists and
// hand it to you," matching what the user asked for.
//
// The manifest this package reads is written by the release pipeline
// (ops/build_v1.sh's aapt-derived version info + ops/deploy_v1.sh's
// publish step), never by this package — Go only ever reads a small JSON
// file and serves a static APK file already sitting on disk; it never
// inspects the APK's own bytes to derive version info itself.
package appupdate

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

var (
	// ErrNotConfigured is returned when the manifest/APK path isn't set at
	// all — the update endpoints fail closed rather than serving nothing
	// useful or a stale artifact from an unknown location.
	ErrNotConfigured = errors.New("app update is not configured")
	ErrUnavailable   = errors.New("app update manifest is unavailable")
)

// Manifest is the on-disk JSON file the release pipeline writes next to
// the APK it describes. SHA256 is lowercase hex, matching every other
// checksum representation already used across this codebase's release
// tooling (ops/build_v1.sh's own SHA256SUMS.txt).
type Manifest struct {
	VersionCode  int    `json:"versionCode"`
	VersionName  string `json:"versionName"`
	SHA256       string `json:"sha256"`
	ReleaseNotes string `json:"releaseNotes"`
	// Mandatory marks a build old clients should treat as a forced update
	// (e.g. a break in wire compatibility). Nothing in this package
	// enforces that itself -- it is client-interpreted, same as every
	// other advisory field here.
	Mandatory bool `json:"mandatory"`
}

// LatestInfo is what GET /v1/app-update/latest returns: the Manifest's own
// fields plus a same-origin download URL the client can fetch immediately
// (no separate lookup step, no credentials needed -- an update check must
// work for a client that might not even have a valid session yet).
type LatestInfo struct {
	Manifest
	APKURL string `json:"apkUrl"`
}

// Config points at the two files the release pipeline publishes. Reading
// the manifest is cheap enough (a few hundred bytes) to do on every
// request rather than caching it in memory -- this endpoint is called at
// most once per app launch, not a hot path, and a fresh read means a
// deploy takes effect immediately with no server restart, matching how
// this codebase's capability_registry flags already behave.
type Config struct {
	ManifestPath string
	APKPath      string
}

// Configured reports whether both paths were actually set — the fail-
// closed boundary every caller must check before trusting this Config.
func (c Config) Configured() bool {
	return strings.TrimSpace(c.ManifestPath) != "" && strings.TrimSpace(c.APKPath) != ""
}

// ReadManifest loads and parses the manifest file. Returns ErrNotConfigured
// if no path was ever set, ErrUnavailable if the file is missing/unreadable/
// malformed -- deliberately not distinguished further to callers, since
// "the update check is temporarily broken" is the only actionable fact
// either way (this is display-only advisory data, never something a
// client's own correctness depends on).
func ReadManifest(path string) (Manifest, error) {
	if strings.TrimSpace(path) == "" {
		return Manifest{}, ErrNotConfigured
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, ErrUnavailable
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, ErrUnavailable
	}
	if m.VersionCode <= 0 || strings.TrimSpace(m.VersionName) == "" || strings.TrimSpace(m.SHA256) == "" {
		return Manifest{}, ErrUnavailable
	}
	return m, nil
}
