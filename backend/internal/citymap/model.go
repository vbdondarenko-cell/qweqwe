package citymap

import (
	"context"
	"errors"
	"time"
)

const MaxClusters = 200

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
	if v.From.IsZero() || v.To.IsZero() || !v.From.Before(v.To) {
		return false
	}
	return v.To.Sub(v.From) <= 7*24*time.Hour
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

type Store interface {
	Viewport(ctx context.Context, viewerID string, query Viewport) ([]Cluster, error)
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
