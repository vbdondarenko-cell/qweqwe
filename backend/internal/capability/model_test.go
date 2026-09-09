package capability

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	entries []Entry
	err     error
}

func (f fakeStore) List(context.Context) ([]Entry, error) { return f.entries, f.err }

func TestCapabilitySnapshotFailsClosed(t *testing.T) {
	svc, err := NewService(fakeStore{err: errors.New("db unavailable")})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := svc.Snapshot(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected store error")
	}
	if snapshot.Revision != 0 {
		t.Fatalf("revision=%d", snapshot.Revision)
	}
	for key, enabled := range snapshot.Capabilities {
		if enabled {
			t.Fatalf("%s must fail closed", key)
		}
	}
}

func TestCapabilityScopeAndEffectiveTime(t *testing.T) {
	now := time.Date(2026, 9, 9, 8, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	svc, err := NewService(fakeStore{entries: []Entry{
		{Key: Realtime, Enabled: true, Revision: 4, ScopeType: "ALL"},
		{Key: Map, Enabled: true, Revision: 5, ScopeType: "USER_ALLOWLIST", ScopeUserIDs: []string{"user-2"}},
		{Key: Notifications, Enabled: true, Revision: 6, ScopeType: "ALL", EffectiveAt: &future},
		{Key: Key("unknown_future"), Enabled: true, Revision: 7, ScopeType: "ALL"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return now }
	snapshot, err := svc.Snapshot(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Revision != 7 {
		t.Fatalf("revision=%d", snapshot.Revision)
	}
	if !snapshot.Capabilities[string(Realtime)] {
		t.Fatal("realtime should be enabled")
	}
	if snapshot.Capabilities[string(Map)] {
		t.Fatal("map must be disabled outside allowlist")
	}
	if snapshot.Capabilities[string(Notifications)] {
		t.Fatal("future capability must be disabled")
	}
	if _, exists := snapshot.Capabilities["unknown_future"]; exists {
		t.Fatal("unknown capability must not be exposed")
	}
}

func TestUnknownKeyNeverEnables(t *testing.T) {
	svc, _ := NewService(fakeStore{})
	if svc.Enabled(context.Background(), "user-1", Key("made_up")) {
		t.Fatal("unknown capability enabled")
	}
}
