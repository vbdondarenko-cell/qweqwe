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
)

var ErrInvalidViewport = errors.New("invalid map viewport")

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
	if v.WestE6 < -180000000 || v.WestE6 > 180000000 ||
		v.EastE6 < -180000000 || v.EastE6 > 180000000 ||
		v.SouthE6 < -90000000 || v.SouthE6 > 90000000 ||
		v.NorthE6 < -90000000 || v.NorthE6 > 90000000 ||
		v.SouthE6 >= v.NorthE6 || v.WestE6 == v.EastE6 {
		return false
	}
	if v.Zoom < 1 || v.Zoom > 20 || v.Limit < 1 || v.Limit > MaxClusters {
		return false
	}
	return validTimeWindow(v.From, v.To)
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
	Key        string  `json:"key"`
	LatitudeE6 int     `json:"latitudeE6"`
	LongitudeE6 int    `json:"longitudeE6"`
	PlaceCount int     `json:"placeCount"`
	SlotCount  int     `json:"slotCount"`
	PlaceID    *string `json:"placeId,omitempty"`
	PlaceName  *string `json:"placeName,omitempty"`
}

type PlaceSlotsQuery struct {
	PlaceID string
	From    time.Time
	To      time.Time
	Limit   int
}

func (q PlaceSlotsQuery) Valid() bool {
	return validUUID(q.PlaceID) && q.Limit >= 1 && q.Limit <= MaxPlaceSlots && validTimeWindow(q.From, q.To)
}

type Store interface {
	Viewport(ctx context.Context, viewerID string, query Viewport) ([]Cluster, error)
	PlaceSlots(ctx context.Context, viewerID string, query PlaceSlotsQuery) ([]slot.Slot, error)
}

type Service struct{ store Store }

func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, ErrInvalidViewport
	}
	return &Service{store: store}, nil
}

func (s *Service) Viewport(ctx context.Context, viewerID string, query Viewport) ([]Cluster, error) {
	if viewerID == "" || !query.Valid() {
		return nil, ErrInvalidViewport
	}
	return s.store.Viewport(ctx, viewerID, query)
}

func (s *Service) PlaceSlots(ctx context.Context, viewerID string, query PlaceSlotsQuery) ([]slot.Slot, error) {
	if viewerID == "" || !query.Valid() {
		return nil, ErrInvalidViewport
	}
	return s.store.PlaceSlots(ctx, viewerID, query)
}

func validTimeWindow(from, to time.Time) bool {
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return false
	}
	return to.Sub(from) <= 7*24*time.Hour
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
