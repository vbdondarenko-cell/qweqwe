package httpserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/appupdate"
)

func TestGetAppUpdateLatestFailsClosedWhenNotConfigured(t *testing.T) {
	server := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/app-update/latest", nil)
	server.getAppUpdateLatest(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDownloadAppUpdateFailsClosedWhenNotConfigured(t *testing.T) {
	server := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/app-update/download", nil)
	server.downloadAppUpdate(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetAppUpdateLatestReturnsManifestAndDerivedURL(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "update-manifest.json")
	apkPath := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(manifestPath, []byte(`{"versionCode":3,"versionName":"1.0.0-debug","sha256":"deadbeef","releaseNotes":"notes","mandatory":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(apkPath, []byte("fake-apk-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{AppUpdate: appupdate.Config{ManifestPath: manifestPath, APKPath: apkPath}}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/app-update/latest", nil)
	req.Host = "linkupapp-ua.duckdns.org"
	// Production sits behind a TLS-terminating reverse proxy, which forwards
	// the original scheme this way -- r.TLS is never set on the Go process's
	// own (plain HTTP) side of that proxy.
	req.Header.Set("X-Forwarded-Proto", "https")
	server.getAppUpdateLatest(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"versionCode":3`) || !strings.Contains(body, `"apkUrl":"https://linkupapp-ua.duckdns.org/v1/app-update/download"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestDownloadAppUpdateServesFileWithCorrectContentType(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "update-manifest.json")
	apkPath := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(manifestPath, []byte(`{"versionCode":3,"versionName":"1.0.0-debug","sha256":"deadbeef"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	want := []byte("fake-apk-bytes")
	if err := os.WriteFile(apkPath, want, 0o600); err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{AppUpdate: appupdate.Config{ManifestPath: manifestPath, APKPath: apkPath}}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/app-update/download", nil)
	server.downloadAppUpdate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/vnd.android.package-archive" {
		t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != string(want) {
		t.Fatalf("body=%q want=%q", rec.Body.String(), want)
	}
}

func TestDownloadAppUpdateFailsClosedWhenManifestBroken(t *testing.T) {
	dir := t.TempDir()
	apkPath := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(apkPath, []byte("fake"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{AppUpdate: appupdate.Config{
		ManifestPath: filepath.Join(dir, "missing-manifest.json"),
		APKPath:      apkPath,
	}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/app-update/download", nil)
	server.downloadAppUpdate(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
