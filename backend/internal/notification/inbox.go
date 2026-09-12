package notification

import (
	"context"
	"errors"
	"strings"
)

var ErrInvalidInboxRequest = errors.New("invalid notification inbox request")

// InboxListLimitDefault/Max bound a single page the same way this codebase's
// other cursor-paginated lists do (e.g. realtime pull batches): a caller
// that asks for nothing gets a sane default, one that asks for too much is
// clamped rather than rejected.
const (
	InboxListLimitDefault = 30
	InboxListLimitMax     = 100
)

// Delivery is one row of a user's real notification history -- the exact
// same notification_deliveries row the projector already wrote (README
// §6.8 / migration 000027's own words: "the in-app/domain truth,
// independent of whether push delivery is configured or succeeds"). Every
// field here already existed before this package did; only Read is new.
type Delivery struct {
	ID        string `json:"id"`
	Type      Type   `json:"type"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	DeepLink  string `json:"deepLink"`
	SlotID    string `json:"slotId,omitempty"`
	CreatedAt int64  `json:"createdAtEpochMillis"`
	Read      bool   `json:"read"`
}

// InboxSnapshot is what GET /v1/me/notifications returns: the page itself
// plus the unread count, which the client needs for the bell-icon badge
// without a second round trip.
type InboxSnapshot struct {
	Items       []Delivery `json:"items"`
	UnreadCount int        `json:"unreadCount"`
	NextCursor  int64      `json:"nextCursorEpochMillis,omitempty"`
}

// InboxStore is implemented by the postgres package.
type InboxStore interface {
	// List returns up to limit deliveries for userID, newest first. When
	// beforeEpochMillis is non-zero, only deliveries strictly older than
	// that instant are returned (keyset pagination on created_at, matching
	// this codebase's other cursor-based lists rather than an offset that
	// drifts under concurrent writes).
	List(ctx context.Context, userID string, limit int, beforeEpochMillis int64) ([]Delivery, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkAllRead(ctx context.Context, userID string) error
}

type InboxService struct {
	store InboxStore
}

func NewInboxService(store InboxStore) (*InboxService, error) {
	if store == nil {
		return nil, ErrInvalidInboxRequest
	}
	return &InboxService{store: store}, nil
}

// List clamps limit into [1, InboxListLimitMax] rather than rejecting an
// out-of-range value -- a caller asking for 0 or a huge page almost
// certainly wants "the default" or "as much as we'll give you", not an
// error, matching this codebase's other list endpoints (e.g. realtime
// pull's own batch-size clamping).
func (s *InboxService) List(ctx context.Context, userID string, limit int, beforeEpochMillis int64) (InboxSnapshot, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return InboxSnapshot{}, ErrInvalidInboxRequest
	}
	if limit <= 0 {
		limit = InboxListLimitDefault
	}
	if limit > InboxListLimitMax {
		limit = InboxListLimitMax
	}
	items, err := s.store.List(ctx, userID, limit, beforeEpochMillis)
	if err != nil {
		return InboxSnapshot{}, err
	}
	unread, err := s.store.UnreadCount(ctx, userID)
	if err != nil {
		return InboxSnapshot{}, err
	}
	snapshot := InboxSnapshot{Items: items, UnreadCount: unread}
	if len(items) == limit {
		snapshot.NextCursor = items[len(items)-1].CreatedAt
	}
	return snapshot, nil
}

func (s *InboxService) MarkAllRead(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrInvalidInboxRequest
	}
	return s.store.MarkAllRead(ctx, userID)
}
