package capability

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

type Key string

const (
	Realtime      Key = "realtime"
	CityContext   Key = "city_context"
	Map           Key = "map"
	Waitlist      Key = "waitlist"
	ChatV2        Key = "chat_v2"
	Notifications Key = "notifications"
	Bump          Key = "bump"
	CityBPM       Key = "city_bpm"
	Swarms        Key = "swarms"
	FlyNow        Key = "fly_now"
	FlyTravel     Key = "fly_travel"
	FlyMotion     Key = "fly_motion"
)

var knownKeys = []Key{
	Realtime, CityContext, Map, Waitlist, ChatV2, Notifications,
	Bump, CityBPM, Swarms, FlyNow, FlyTravel, FlyMotion,
}

var ErrUnavailable = errors.New("capability registry unavailable")

type Entry struct {
	Key          Key
	Enabled      bool
	Revision     int64
	ScopeType    string
	ScopeUserIDs []string
	EffectiveAt  *time.Time
}

type Store interface {
	List(context.Context) ([]Entry, error)
}

type Snapshot struct {
	Revision     int64           `json:"revision"`
	Capabilities map[string]bool `json:"capabilities"`
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, ErrUnavailable
	}
	return &Service{store: store, now: time.Now}, nil
}

func KnownKeys() []Key {
	out := append([]Key(nil), knownKeys...)
	return out
}

func DisabledSnapshot() Snapshot {
	out := Snapshot{Capabilities: make(map[string]bool, len(knownKeys))}
	for _, key := range knownKeys {
		out.Capabilities[string(key)] = false
	}
	return out
}

func (s *Service) Snapshot(ctx context.Context, userID string) (Snapshot, error) {
	if s == nil || s.store == nil || strings.TrimSpace(userID) == "" {
		return DisabledSnapshot(), ErrUnavailable
	}
	entries, err := s.store.List(ctx)
	if err != nil {
		return DisabledSnapshot(), err
	}
	out := DisabledSnapshot()
	now := s.now().UTC()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Revision < entries[j].Revision })
	for _, entry := range entries {
		if entry.Revision > out.Revision {
			out.Revision = entry.Revision
		}
		if !isKnown(entry.Key) {
			continue
		}
		out.Capabilities[string(entry.Key)] = entryAllows(entry, userID, now)
	}
	return out, nil
}

func (s *Service) Enabled(ctx context.Context, userID string, key Key) bool {
	if !isKnown(key) {
		return false
	}
	snapshot, err := s.Snapshot(ctx, userID)
	return err == nil && snapshot.Capabilities[string(key)]
}

func entryAllows(entry Entry, userID string, now time.Time) bool {
	if !entry.Enabled || entry.Revision <= 0 {
		return false
	}
	if entry.EffectiveAt != nil && entry.EffectiveAt.UTC().After(now) {
		return false
	}
	switch entry.ScopeType {
	case "ALL":
		return len(entry.ScopeUserIDs) == 0
	case "USER_ALLOWLIST":
		if len(entry.ScopeUserIDs) == 0 {
			return false
		}
		for _, allowed := range entry.ScopeUserIDs {
			if strings.EqualFold(strings.TrimSpace(allowed), strings.TrimSpace(userID)) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func isKnown(key Key) bool {
	for _, candidate := range knownKeys {
		if candidate == key {
			return true
		}
	}
	return false
}
