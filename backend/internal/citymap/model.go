package citymap

import (
	"context"
	"errors"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

const (
	MaxClusters   = 200
	MaxPlaceSlots = 100

	// LiveWindowMax is the free map's existing time-window bound (README
	// §6.4) — unchanged, just named so HistoricalWindowMax below can refer
	// to it.
	LiveWindowMax = 7 * 24 * time.Hour
	// HistoricalWindowMax is README §6.23's LinkUp+ "Pulse Time-Machine"
	// extension: up to ~180 days of aggregate history, never available
	// through the free Viewport path (Service.HistoricalViewport is the
	// only caller that accepts a window this wide, and only once its own
	// premiumActive check passes).
	HistoricalWindowMax = 180 * 24 * time.Hour
	// HistoricalMaxZoom is a deliberately coarse ceiling applied whenever a
	// historical query reaches further back than LiveWindowMax before now:
	// §6.23 requires "no individual route reconstruction," and a
	// fine-grained query repeated across many narrow historical time
	// slices could otherwise approximate one. This threshold (BucketE6's
	// own zoom<=13 tier is already ~11km-wide) is a conservative, stated
	// policy choice — not derived from any formula in the doc.
	HistoricalMaxZoom = 12
)

var (
	ErrInvalidViewport = errors.New("invalid map viewport")
	// ErrPremiumRequired is returned by HistoricalViewport when the caller
	// is not an active LinkUp+ subscriber — Pulse Time-Machine is a
	// LinkUp+-gated capability (README §6.23), never available for free.
	ErrPremiumRequired = errors.New("this map query requires an active LinkUp+ subscription")
)

type Viewport struct {
	WestE6  int
	SouthE6 int
	EastE6  int
	NorthE6 int
	Zoom    int
	From    time.Time
	To      time.Time
	Limit   int
}

func (v Viewport) Valid() bool {
	if !v.validSpatialAndZoom() {
		return false
	}
	return validTimeWindow(v.From, v.To, LiveWindowMax)
}

// ValidHistorical is README §6.23's Pulse Time-Machine variant of Valid:
// the same spatial/zoom/limit bounds, but a time window (To-From span) up
// to HistoricalWindowMax instead of LiveWindowMax — and, whenever that
// span itself is wider than LiveWindowMax, a zoom no finer than
// HistoricalMaxZoom (see that constant's own doc comment for why). This is
// deliberately span-based, not "how long ago From is": Valid() already
// lets the free map query any short (<=LiveWindowMax-wide) window no
// matter how far in the past it starts, so the only new risk this method
// introduces is a WIDE historical window at a fine-enough zoom to
// approximate route reconstruction across it.
func (v Viewport) ValidHistorical() bool {
	if !v.validSpatialAndZoom() {
		return false
	}
	if !validTimeWindow(v.From, v.To, HistoricalWindowMax) {
		return false
	}
	if v.To.Sub(v.From) > LiveWindowMax && v.Zoom > HistoricalMaxZoom {
		return false
	}
	return true
}

func (v Viewport) validSpatialAndZoom() bool {
	if v.WestE6 < -180000000 || v.WestE6 > 180000000 ||
		v.EastE6 < -180000000 || v.EastE6 > 180000000 ||
		v.SouthE6 < -90000000 || v.SouthE6 > 90000000 ||
		v.NorthE6 < -90000000 || v.NorthE6 > 90000000 ||
		v.SouthE6 >= v.NorthE6 || v.WestE6 == v.EastE6 {
		return false
	}
	return v.Zoom >= 1 && v.Zoom <= 20 && v.Limit >= 1 && v.Limit <= MaxClusters
}

func (v Viewport) BucketE6() int {
	switch {
	case v.Zoom <= 7:
		return 2_000_000
	case v.Zoom <= 10:
		return 500_000
	case v.Zoom <= 13:
		return 100_000
	case v.Zoom <= 15:
		return 25_000
	default:
		return 5_000
	}
}

type Cluster struct {
	Key         string  `json:"key"`
	LatitudeE6  int     `json:"latitudeE6"`
	LongitudeE6 int     `json:"longitudeE6"`
	PlaceCount  int     `json:"placeCount"`
	SlotCount   int     `json:"slotCount"`
	PlaceID     *string `json:"placeId,omitempty"`
	PlaceName   *string `json:"placeName,omitempty"`
}

type PlaceSlotsQuery struct {
	PlaceID string
	From    time.Time
	To      time.Time
	Limit   int
}

func (q PlaceSlotsQuery) Valid() bool {
	return validUUID(q.PlaceID) && q.Limit >= 1 && q.Limit <= MaxPlaceSlots && validTimeWindow(q.From, q.To, LiveWindowMax)
}

type Store interface {
	Viewport(ctx context.Context, viewerID, localityID string, query Viewport) ([]Cluster, error)
	PlaceSlots(ctx context.Context, viewerID, localityID string, query PlaceSlotsQuery) ([]slot.Slot, error)
}

type Service struct{ store Store }

func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, ErrInvalidViewport
	}
	return &Service{store: store}, nil
}

func (s *Service) Viewport(ctx context.Context, viewerID, localityID string, query Viewport) ([]Cluster, error) {
	if viewerID == "" || !validUUID(localityID) || !query.Valid() {
		return nil, ErrInvalidViewport
	}
	return s.store.Viewport(ctx, viewerID, localityID, query)
}

// HistoricalViewport is README §6.23's "Pulse Time-Machine": the exact
// same aggregate cluster query as Viewport (no new SQL, no new store
// method — the underlying data was already privacy-safe aggregate
// counts), but gated on premiumActive and accepting a much wider time
// window at a correspondingly coarser zoom ceiling. The caller (this
// package's own httpserver handler) is responsible for actually checking
// entitlement (via monetization.Service) and passing the result in as
// premiumActive — this method never looks up billing state itself, same
// separation of concerns as the rest of this codebase's capability gates.
func (s *Service) HistoricalViewport(ctx context.Context, viewerID, localityID string, query Viewport, premiumActive bool) ([]Cluster, error) {
	if viewerID == "" || !validUUID(localityID) {
		return nil, ErrInvalidViewport
	}
	if !premiumActive {
		return nil, ErrPremiumRequired
	}
	if !query.ValidHistorical() {
		return nil, ErrInvalidViewport
	}
	return s.store.Viewport(ctx, viewerID, localityID, query)
}

func (s *Service) PlaceSlots(ctx context.Context, viewerID, localityID string, query PlaceSlotsQuery) ([]slot.Slot, error) {
	if viewerID == "" || !validUUID(localityID) || !query.Valid() {
		return nil, ErrInvalidViewport
	}
	return s.store.PlaceSlots(ctx, viewerID, localityID, query)
}

func validTimeWindow(from, to time.Time, max time.Duration) bool {
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return false
	}
	return to.Sub(from) <= max
}

func validUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	for i := 0; i < len(value); i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		c := value[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
