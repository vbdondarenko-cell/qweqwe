package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/notification"
)

// NotificationInboxStore reads back the same notification_deliveries rows
// internal/postgres.NotificationProjector already writes -- see that type's
// own doc comment and migration 000027's: this is the in-app/domain truth,
// independent of push_outcome. read_at (migration 000040) is the only
// column this store adds meaning to that the projector doesn't itself
// write.
type NotificationInboxStore struct {
	pool *pgxpool.Pool
}

func NewNotificationInboxStore(pool *pgxpool.Pool) (*NotificationInboxStore, error) {
	if pool == nil {
		return nil, errors.New("invalid notification inbox store dependency")
	}
	return &NotificationInboxStore{pool: pool}, nil
}

func (s *NotificationInboxStore) List(ctx context.Context, userID string, limit int, beforeEpochMillis int64) ([]notification.Delivery, error) {
	var rows pgx.Rows
	var err error
	if beforeEpochMillis > 0 {
		before := time.UnixMilli(beforeEpochMillis)
		rows, err = s.pool.Query(ctx, `
			SELECT id,notification_type,title,body,deep_link,slot_id,created_at,read_at
			FROM notification_deliveries
			WHERE user_id=$1 AND created_at < $2
			ORDER BY created_at DESC, id DESC
			LIMIT $3`, userID, before, limit)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id,notification_type,title,body,deep_link,slot_id,created_at,read_at
			FROM notification_deliveries
			WHERE user_id=$1
			ORDER BY created_at DESC, id DESC
			LIMIT $2`, userID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]notification.Delivery, 0, limit)
	for rows.Next() {
		var (
			id, typ, title, body, deepLink string
			slotID                         *string
			createdAt                      time.Time
			readAt                         *time.Time
		)
		if err := rows.Scan(&id, &typ, &title, &body, &deepLink, &slotID, &createdAt, &readAt); err != nil {
			return nil, err
		}
		item := notification.Delivery{
			ID:        id,
			Type:      notification.Type(typ),
			Title:     title,
			Body:      body,
			DeepLink:  deepLink,
			CreatedAt: createdAt.UnixMilli(),
			Read:      readAt != nil,
		}
		if slotID != nil {
			item.SlotID = *slotID
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *NotificationInboxStore) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM notification_deliveries WHERE user_id=$1 AND read_at IS NULL`, userID).
		Scan(&count)
	return count, err
}

func (s *NotificationInboxStore) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notification_deliveries SET read_at=now() WHERE user_id=$1 AND read_at IS NULL`, userID)
	return err
}
