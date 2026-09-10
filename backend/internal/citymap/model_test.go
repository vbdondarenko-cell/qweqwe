package citymap

import (
	"context"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func TestViewportValidationAndBuckets(t *testing.T) {
	from := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	base := Viewport{
		WestE6: 30_000_000, SouthE6: 49_000_000,
		EastE6: 31_000_000, NorthE6: 51_000_000,
		Zoom: 12, From: from, To: from.Add(24 * time.Hour), Limit: 100,
	}
	if !base.Valid() {
		t.Fatal("expected valid viewport")
	}
	if base.BucketE6() != 100_000 {
		t.Fatalf("zoom 12 bucket=%d", base.BucketE6())
	}

	antiMeridian := base
	antiMeridian.WestE6 = 170_000_000
	antiMeridian.EastE6 = -170_000_000
	if !antiMeridian.Valid() {
		t.Fatal("antimeridian viewport must be accepted")
	}

	for name, mutate := range map[string]func(*Viewport){
		"flat latitude": func(v *Viewport) { v.NorthE6 = v.SouthE6 },
		"zero width":    func(v *Viewport) { v.EastE6 = v.WestE6 },
		"too long":      func(v *Viewport) { v.To = v.From.Add(8 * 24 * time.Hour) },
		"bad zoom":      func(v *Viewport) { v.Zoom = 0 },
		"bad limit":     func(v *Viewport) { v.Limit = MaxClusters + 1 },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := base
			mutate(&candidate)
			if candidate.Valid() {
				t.Fatal("invalid viewport accepted")
			}
		})
	}
}

func TestServiceRequiresCanonicalLocality(t *testing.T) {
	store := &recordingStore{}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	viewport := Viewport{
		WestE6: 30_000_000, SouthE6: 49_000_000,
		EastE6: 31_000_000, NorthE6: 51_000_000,
		Zoom: 12, From: from, To: from.Add(time.Hour), Limit: 20,
	}
	viewerID := "00000000-0000-0000-0000-000000000001"
	localityID := "00000000-0000-0000-0000-000000000002"

	if _, err := service.Viewport(context.Background(), viewerID, "not-a-uuid", viewport); err != ErrInvalidViewport {
		t.Fatalf("invalid locality must fail closed: %v", err)
	}
	if store.viewportCalls != 0 {
		t.Fatalf("store called for invalid locality: %d", store.viewportCalls)
	}
	if _, err := service.Viewport(context.Background(), viewerID, localityID, viewport); err != nil {
		t.Fatal(err)
	}
	if store.viewportCalls != 1 || store.lastLocalityID != localityID {
		t.Fatalf("canonical locality not forwarded: calls=%d locality=%q", store.viewportCalls, store.lastLocalityID)
	}
}

type recordingStore struct {
	viewportCalls  int
	lastLocalityID string
}

func (s *recordingStore) Viewport(_ context.Context, _, localityID string, _ Viewport) ([]Cluster, error) {
	s.viewportCalls++
	s.lastLocalityID = localityID
	return nil, nil
}

func (s *recordingStore) PlaceSlots(_ context.Context, _, localityID string, _ PlaceSlotsQuery) ([]slot.Slot, error) {
	s.lastLocalityID = localityID
	return nil, nil
}
