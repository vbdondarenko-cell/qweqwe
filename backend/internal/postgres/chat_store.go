package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
)

type ChatStore struct {
	pool *pgxpool.Pool
}

func NewChatStore(pool *pgxpool.Pool) *ChatStore { return &ChatStore{pool: pool} }

func (s *ChatStore) Send(ctx context.Context, actorID, slotID, messageID, idempotencyKey, text string) (chat.Message, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return chat.Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Authorization is checked before insert/replay lookup. A previously valid
	// idempotency key therefore cannot bypass LEAVE/block/terminal revocation.
	if err := authorizeChatTx(ctx, tx, actorID, slotID); err != nil {
		return chat.Message{}, err
	}

	resolvedID := messageID
	resolvedText := text
	var createdAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO slot_messages (id,slot_id,author_id,body,idempotency_key)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (slot_id,author_id,idempotency_key) DO NOTHING
		RETURNING id,body,created_at`, messageID, slotID, actorID, text, idempotencyKey).
		Scan(&resolvedID, &resolvedText, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT id,body,created_at
			FROM slot_messages
			WHERE slot_id=$1 AND author_id=$2 AND idempotency_key=$3`, slotID, actorID, idempotencyKey).
			Scan(&resolvedID, &resolvedText, &createdAt)
	}
	if err != nil {
		return chat.Message{}, err
	}
	if resolvedText != text {
		return chat.Message{}, chat.ErrIdempotencyConflict
	}

	var author chat.Author
	if err := tx.QueryRow(ctx, `SELECT id,username,display_name,avatar_url FROM app_users WHERE id=$1`, actorID).
		Scan(&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL); err != nil {
		return chat.Message{}, err
	}
	out := chat.Message{
		ID:        resolvedID,
		SlotID:    slotID,
		Kind:      chat.KindUser,
		Author:    &author,
		Text:      resolvedText,
		CreatedAt: createdAt.UTC(),
	}
	if err := tx.Commit(ctx); err != nil {
		return chat.Message{}, err
	}
	return out, nil
}

func (s *ChatStore) ListRecent(ctx context.Context, actorID, slotID string, limit int) ([]chat.Message, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := authorizeChatTx(ctx, tx, actorID, slotID); err != nil {
		return nil, err
	}

	// System messages (author_id IS NULL) are never matched by the block
	// filter below: a NULL author_id can never equal a blocked/blocker id,
	// so NOT EXISTS(...) is unconditionally true for them.
	rows, err := tx.Query(ctx, `
		SELECT q.id,q.slot_id,q.kind,
		       u.id,u.username,u.display_name,u.avatar_url,
		       q.body,q.system_event_type,
		       su.id,su.username,su.display_name,su.avatar_url,
		       q.created_at
		FROM (
			SELECT m.id,m.slot_id,m.kind,m.author_id,m.body,m.system_event_type,m.subject_user_id,m.created_at
			FROM slot_messages m
			WHERE m.slot_id=$1
			  AND NOT EXISTS (
				SELECT 1 FROM user_blocks b
				WHERE (b.blocker_id=$2 AND b.blocked_id=m.author_id)
				   OR (b.blocker_id=m.author_id AND b.blocked_id=$2)
			  )
			ORDER BY m.created_at DESC,m.id DESC
			LIMIT $3
		) q
		LEFT JOIN app_users u ON u.id=q.author_id
		LEFT JOIN app_users su ON su.id=q.subject_user_id
		ORDER BY q.created_at ASC,q.id ASC`, slotID, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]chat.Message, 0, limit)
	for rows.Next() {
		var item chat.Message
		var kind string
		var authorID, authorUsername, authorDisplayName, authorAvatar *string
		var body, systemEventType *string
		var subjectID, subjectUsername, subjectDisplayName, subjectAvatar *string
		if err := rows.Scan(
			&item.ID,
			&item.SlotID,
			&kind,
			&authorID, &authorUsername, &authorDisplayName, &authorAvatar,
			&body, &systemEventType,
			&subjectID, &subjectUsername, &subjectDisplayName, &subjectAvatar,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Kind = chat.MessageKind(kind)
		if authorID != nil {
			item.Author = &chat.Author{ID: *authorID, Username: *authorUsername, DisplayName: *authorDisplayName, AvatarURL: authorAvatar}
		}
		if body != nil {
			item.Text = *body
		}
		if systemEventType != nil {
			eventType := chat.SystemEventType(*systemEventType)
			item.SystemEventType = &eventType
		}
		if subjectID != nil {
			item.Subject = &chat.Author{ID: *subjectID, Username: *subjectUsername, DisplayName: *subjectDisplayName, AvatarURL: subjectAvatar}
		}
		item.CreatedAt = item.CreatedAt.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return items, nil
}

// SplitBillRoster authorizes actorID with the exact same boundary as
// Send/ListRecent, then returns the host plus every currently-accepted
// member (block-excluded from the actor's own view, same as ListRecent) so
// chat.Service.SplitBill can validate a requested participant list against
// real Slot membership.
func (s *ChatStore) SplitBillRoster(ctx context.Context, actorID, slotID string) ([]chat.Author, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := authorizeChatTx(ctx, tx, actorID, slotID); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT u.id,u.username,u.display_name,u.avatar_url
		FROM app_users u
		WHERE u.id IN (
			SELECT host_id FROM slots WHERE id=$1
			UNION
			SELECT user_id FROM slot_memberships WHERE slot_id=$1
		)
		AND NOT EXISTS (
			SELECT 1 FROM user_blocks b
			WHERE (b.blocker_id=$2 AND b.blocked_id=u.id)
			   OR (b.blocker_id=u.id AND b.blocked_id=$2)
		)
		ORDER BY u.id`, slotID, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]chat.Author, 0)
	for rows.Next() {
		var item chat.Author
		if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.AvatarURL); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return items, nil
}

func authorizeChatTx(ctx context.Context, tx pgx.Tx, actorID, slotID string) error {
	var hostID, state string
	if err := tx.QueryRow(ctx, `SELECT host_id,state FROM slots WHERE id=$1 FOR SHARE`, slotID).Scan(&hostID, &state); errors.Is(err, pgx.ErrNoRows) {
		return chat.ErrNotFound
	} else if err != nil {
		return err
	}
	if state != "FILLING" && state != "FULL" && state != "ACTIVE" {
		return chat.ErrClosed
	}
	if hostID == actorID {
		return nil
	}

	var member, blocked bool
	if err := tx.QueryRow(ctx, `
		SELECT
			EXISTS(SELECT 1 FROM slot_memberships WHERE slot_id=$1 AND user_id=$2),
			EXISTS(
				SELECT 1 FROM user_blocks
				WHERE (blocker_id=$2 AND blocked_id=$3)
				   OR (blocker_id=$3 AND blocked_id=$2)
			)`, slotID, actorID, hostID).Scan(&member, &blocked); err != nil {
		return err
	}
	if !member || blocked {
		return chat.ErrForbidden
	}
	return nil
}
