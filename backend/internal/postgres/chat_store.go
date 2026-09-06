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

func (s *ChatStore) Send(ctx context.Context, actorID, slotID, messageID, text string) (chat.Message, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil { return chat.Message{}, err }
	defer func() { _ = tx.Rollback(ctx) }()

	if err := authorizeChatTx(ctx, tx, actorID, slotID); err != nil { return chat.Message{}, err }

	var createdAt time.Time
	if err := tx.QueryRow(ctx, `INSERT INTO slot_messages (id,slot_id,author_id,body) VALUES ($1,$2,$3,$4) RETURNING created_at`, messageID, slotID, actorID, text).Scan(&createdAt); err != nil {
		return chat.Message{}, err
	}
	var author chat.Author
	if err := tx.QueryRow(ctx, `SELECT id,username,display_name,avatar_url FROM app_users WHERE id=$1`, actorID).
		Scan(&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL); err != nil {
		return chat.Message{}, err
	}
	out := chat.Message{ID: messageID, SlotID: slotID, Author: author, Text: text, CreatedAt: createdAt.UTC()}
	if err := tx.Commit(ctx); err != nil { return chat.Message{}, err }
	return out, nil
}

func (s *ChatStore) ListRecent(ctx context.Context, actorID, slotID string, limit int) ([]chat.Message, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil { return nil, err }
	defer func() { _ = tx.Rollback(ctx) }()

	if err := authorizeChatTx(ctx, tx, actorID, slotID); err != nil { return nil, err }

	rows, err := tx.Query(ctx, `
		SELECT q.id,q.slot_id,u.id,u.username,u.display_name,u.avatar_url,q.body,q.created_at
		FROM (
			SELECT m.id,m.slot_id,m.author_id,m.body,m.created_at
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
		JOIN app_users u ON u.id=q.author_id
		ORDER BY q.created_at ASC,q.id ASC`, slotID, actorID, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	items := make([]chat.Message, 0, limit)
	for rows.Next() {
		var item chat.Message
		if err := rows.Scan(&item.ID, &item.SlotID, &item.Author.ID, &item.Author.Username, &item.Author.DisplayName, &item.Author.AvatarURL, &item.Text, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.CreatedAt = item.CreatedAt.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil { return nil, err }
	if err := tx.Commit(ctx); err != nil { return nil, err }
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
	if !member || blocked { return chat.ErrForbidden }
	return nil
}
