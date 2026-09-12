package appupdate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadManifestFailsClosedWhenUnconfigured(t *testing.T) {
	if _, err := ReadManifest(""); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
	if _, err := ReadManifest("   "); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured for blank path, got %v", err)
	}
}

func TestReadManifestFailsClosedWhenMissingOrMalformed(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist.json")
	if _, err := ReadManifest(missing); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable for missing file, got %v", err)
	}

	malformed := filepath.Join(dir, "malformed.json")
	if err := os.WriteFile(malformed, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManifest(malformed); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable for malformed JSON, got %v", err)
	}

	incomplete := filepath.Join(dir, "incomplete.json")
	if err := os.WriteFile(incomplete, []byte(`{"versionCode":0,"versionName":"","sha256":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManifest(incomplete); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable for zero-value manifest, got %v", err)
	}
}

func TestReadManifestParsesValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update-manifest.json")
	body := `{"versionCode":2,"versionName":"1.0.0-debug","sha256":"abc123","releaseNotes":"bug fixes","mandatory":true}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.VersionCode != 2 || m.VersionName != "1.0.0-debug" || m.SHA256 != "abc123" || m.ReleaseNotes != "bug fixes" || !m.Mandatory {
		t.Fatalf("unexpected manifest: %#v", m)
	}
}

func TestConfigConfigured(t *testing.T) {
	if (Config{}).Configured() {
		t.Fatal("empty config must not be configured")
	}
	if (Config{ManifestPath: "x"}).Configured() {
		t.Fatal("manifest path alone must not be configured")
	}
	if (Config{APKPath: "y"}).Configured() {
		t.Fatal("apk path alone must not be configured")
	}
	if !(Config{ManifestPath: "x", APKPath: "y"}).Configured() {
		t.Fatal("both paths set must be configured")
	}
}
