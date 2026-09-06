package ratelimit

import (
	"errors"
	"sync"
	"time"
)

type Config struct {
	Limit      int
	Window     time.Duration
	IdleTTL    time.Duration
	MaxEntries int
}

type entry struct {
	count       int
	windowStart time.Time
	lastSeen    time.Time
}

type Limiter struct {
	mu      sync.Mutex
	cfg     Config
	entries map[string]entry
	now     func() time.Time
}

func New(cfg Config) (*Limiter, error) {
	if cfg.Limit <= 0 || cfg.Window <= 0 || cfg.IdleTTL < cfg.Window || cfg.MaxEntries <= 0 {
		return nil, errors.New("invalid rate limiter configuration")
	}
	return &Limiter{cfg: cfg, entries: make(map[string]entry), now: time.Now}, nil
}

// Allow applies a fixed-window limit per caller-supplied key. Storage is
// bounded; if the map is saturated after pruning idle entries, new keys fail
// closed instead of allowing unbounded memory growth.
func (l *Limiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	if key == "" {
		return false, l.cfg.Window
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().UTC()
	if current, ok := l.entries[key]; ok {
		if now.Sub(current.windowStart) >= l.cfg.Window {
			current = entry{count: 1, windowStart: now, lastSeen: now}
			l.entries[key] = current
			return true, 0
		}
		current.lastSeen = now
		if current.count >= l.cfg.Limit {
			l.entries[key] = current
			retry := l.cfg.Window - now.Sub(current.windowStart)
			if retry < time.Second {
				retry = time.Second
			}
			return false, retry
		}
		current.count++
		l.entries[key] = current
		return true, 0
	}

	if len(l.entries) >= l.cfg.MaxEntries {
		l.pruneLocked(now)
		if len(l.entries) >= l.cfg.MaxEntries {
			return false, l.cfg.Window
		}
	}

	l.entries[key] = entry{count: 1, windowStart: now, lastSeen: now}
	return true, 0
}

func (l *Limiter) pruneLocked(now time.Time) {
	for key, current := range l.entries {
		if now.Sub(current.lastSeen) >= l.cfg.IdleTTL {
			delete(l.entries, key)
		}
	}
}

func (l *Limiter) Size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}
