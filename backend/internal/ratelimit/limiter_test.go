package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBlocksAndResets(t *testing.T) {
	l, err := New(Config{Limit: 2, Window: time.Minute, IdleTTL: 2 * time.Minute, MaxEntries: 10})
	if err != nil { t.Fatal(err) }
	now := time.Unix(1_700_000_000, 0).UTC()
	l.now = func() time.Time { return now }

	if ok, _ := l.Allow("ip:/login"); !ok { t.Fatal("first request should pass") }
	if ok, _ := l.Allow("ip:/login"); !ok { t.Fatal("second request should pass") }
	if ok, retry := l.Allow("ip:/login"); ok || retry <= 0 { t.Fatalf("third request should be blocked, retry=%v", retry) }

	now = now.Add(time.Minute)
	if ok, _ := l.Allow("ip:/login"); !ok { t.Fatal("window should reset") }
}

func TestLimiterStorageIsBoundedAndPrunesIdle(t *testing.T) {
	l, err := New(Config{Limit: 1, Window: time.Minute, IdleTTL: time.Minute, MaxEntries: 2})
	if err != nil { t.Fatal(err) }
	now := time.Unix(1_700_000_000, 0).UTC()
	l.now = func() time.Time { return now }

	if ok, _ := l.Allow("a"); !ok { t.Fatal("a should pass") }
	if ok, _ := l.Allow("b"); !ok { t.Fatal("b should pass") }
	if ok, _ := l.Allow("c"); ok { t.Fatal("new key must fail closed while storage is saturated") }
	if got := l.Size(); got != 2 { t.Fatalf("size=%d want 2", got) }

	now = now.Add(time.Minute)
	if ok, _ := l.Allow("c"); !ok { t.Fatal("idle entries should be pruned") }
	if got := l.Size(); got != 1 { t.Fatalf("size=%d want 1", got) }
}
