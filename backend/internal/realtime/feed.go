package realtime

import (
	"context"
	"strings"
)

const MaxViewerBatch = 200

type ViewerBatch struct {
	Cursor int64   `json:"cursor"`
	Events []Event `json:"events"`
}

func (b ViewerBatch) ValidAfter(after int64) bool {
	if after < 0 || b.Cursor < after || len(b.Events) > MaxViewerBatch {
		return false
	}
	previous := after
	seen := make(map[string]struct{}, len(b.Events))
	for _, event := range b.Events {
		if !event.Valid() || event.Sequence <= previous || event.Sequence > b.Cursor {
			return false
		}
		if _, exists := seen[event.ID]; exists {
			return false
		}
		seen[event.ID] = struct{}{}
		previous = event.Sequence
	}
	return true
}

type ViewerFeedStore interface {
	PullViewer(ctx context.Context, viewerID string, after int64, limit int) (ViewerBatch, error)
}

type FeedService struct {
	store ViewerFeedStore
}

func NewFeedService(store ViewerFeedStore) (*FeedService, error) {
	if store == nil {
		return nil, ErrInvalidInput
	}
	return &FeedService{store: store}, nil
}

func (s *FeedService) Pull(ctx context.Context, viewerID string, after int64, limit int) (ViewerBatch, error) {
	viewerID = strings.TrimSpace(viewerID)
	if viewerID == "" || after < 0 {
		return ViewerBatch{}, ErrInvalidInput
	}
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > MaxViewerBatch {
		return ViewerBatch{}, ErrInvalidInput
	}
	batch, err := s.store.PullViewer(ctx, viewerID, after, limit)
	if err != nil {
		return ViewerBatch{}, err
	}
	if !batch.ValidAfter(after) {
		return ViewerBatch{}, ErrCursorOutOfOrder
	}
	return batch, nil
}
