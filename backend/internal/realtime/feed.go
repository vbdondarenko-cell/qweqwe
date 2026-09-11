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
	// CurrentCursor reports the outbox's current maximum sequence, with no
	// per-viewer filtering: the cursor value itself is an opaque position
	// marker, never event content, so it carries nothing to authorize. It
	// exists for reconnect/first-run bootstrap (README §6.2's "obtain
	// authoritative snapshot/cursor" step): a client with no prior
	// position already gets its actual current state from Pulse/Get/
	// ListMine, which are authoritative on their own — it never needs the
	// realtime channel's historical deltas replayed to reconstruct state
	// that direct reads already give it. Without this, such a client's
	// only path to "caught up" was pulling forward from sequence 0,
	// correctly bounded and self-advancing (PullViewer's own doc comment)
	// but still wasted round trips through the entire outbox history it
	// will never actually use. Fast-forwarding straight to the current
	// value up front closes that real gap.
	CurrentCursor(ctx context.Context) (int64, error)
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

// CurrentCursor exposes ViewerFeedStore.CurrentCursor through the same
// validated Service boundary every other read here goes through, even
// though the underlying value carries no viewer-specific content — kept
// consistent with Pull's own auth-adjacent shape rather than bypassing the
// service layer for a "simpler" direct store call.
func (s *FeedService) CurrentCursor(ctx context.Context, viewerID string) (int64, error) {
	if strings.TrimSpace(viewerID) == "" {
		return 0, ErrInvalidInput
	}
	return s.store.CurrentCursor(ctx)
}
