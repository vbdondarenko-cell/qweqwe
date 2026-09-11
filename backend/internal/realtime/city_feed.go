package realtime

import (
	"context"
	"strings"
	"time"
)

type CityInvalidation struct {
	Sequence   int64     `json:"sequence"`
	ID         string    `json:"eventId"`
	OccurredAt time.Time `json:"occurredAt"`
}

type CityBatch struct {
	Cursor        int64              `json:"cursor"`
	Invalidations []CityInvalidation `json:"invalidations"`
}

func (b CityBatch) ValidAfter(after int64) bool {
	if after < 0 || b.Cursor < after || len(b.Invalidations) > MaxViewerBatch {
		return false
	}
	previous := after
	seen := make(map[string]struct{}, len(b.Invalidations))
	for _, item := range b.Invalidations {
		if item.Sequence <= previous || item.Sequence > b.Cursor || item.ID == "" || item.OccurredAt.IsZero() {
			return false
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return false
		}
		seen[item.ID] = struct{}{}
		previous = item.Sequence
	}
	return true
}

type CityFeedStore interface {
	PullCity(ctx context.Context, viewerID string, after int64, limit int) (CityBatch, error)
	// CurrentCursor mirrors ViewerFeedStore.CurrentCursor (see its doc
	// comment): a bootstrap fast-forward point for a client with no prior
	// city-channel position, so a fresh/reconnecting client does not pull
	// forward through the entire city outbox history it will never use.
	CurrentCursor(ctx context.Context) (int64, error)
}

type CityFeedService struct{ store CityFeedStore }

func NewCityFeedService(store CityFeedStore) (*CityFeedService, error) {
	if store == nil {
		return nil, ErrInvalidInput
	}
	return &CityFeedService{store: store}, nil
}

func (s *CityFeedService) Pull(ctx context.Context, viewerID string, after int64, limit int) (CityBatch, error) {
	viewerID = strings.TrimSpace(viewerID)
	if viewerID == "" || after < 0 {
		return CityBatch{}, ErrInvalidInput
	}
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > MaxViewerBatch {
		return CityBatch{}, ErrInvalidInput
	}
	batch, err := s.store.PullCity(ctx, viewerID, after, limit)
	if err != nil {
		return CityBatch{}, err
	}
	if !batch.ValidAfter(after) {
		return CityBatch{}, ErrCursorOutOfOrder
	}
	return batch, nil
}

// CurrentCursor mirrors FeedService.CurrentCursor for the city channel.
func (s *CityFeedService) CurrentCursor(ctx context.Context, viewerID string) (int64, error) {
	if strings.TrimSpace(viewerID) == "" {
		return 0, ErrInvalidInput
	}
	return s.store.CurrentCursor(ctx)
}
