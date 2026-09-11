package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/friend"
)

type FriendStore struct {
	pool *pgxpool.Pool
}

func NewFriendStore(pool *pgxpool.Pool) (*FriendStore, error) {
	if pool == nil {
		return nil, errors.New("invalid friend store dependency")
	}
	return &FriendStore{pool: pool}, nil
}

// canonicalPair returns (lo, hi) with lo always the lexicographically
// smaller UUID string, mirroring bump_store.go's identical pattern for
// bump_confirmations — the same equivalence between Go string comparison
// and PostgreSQL's uuid btree ordering that pattern already relies on.
func canonicalPair(a, b string) (string, string) {
	if b < a {
		return b, a
	}
	return a, b
}

func (s *FriendStore) Request(ctx context.Context, actorID, targetID string, now time.Time) (friend.RequestResult, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return friend.RequestResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var targetExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app_users WHERE id=$1)`, targetID).Scan(&targetExists); err != nil {
		return friend.RequestResult{}, err
	}
	if !targetExists {
		return friend.RequestResult{}, friend.ErrInvalidInput
	}

	blocked, err := blockedPairTx(ctx, tx, actorID, targetID)
	if err != nil {
		return friend.RequestResult{}, err
	}
	if blocked {
		return friend.RequestResult{}, friend.ErrForbidden
	}

	already, err := areFriendsTx(ctx, tx, actorID, targetID)
	if err != nil {
		return friend.RequestResult{}, err
	}
	if already {
		return friend.RequestResult{}, friend.ErrAlreadyFriends
	}

	// A reverse PENDING request (targetID already asked actorID) resolves
	// this call as an immediate mutual accept rather than creating a
	// second, parallel request — see friend.Outcome's doc comment.
	var reverseID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM friend_requests
		WHERE requester_id=$1 AND target_id=$2 AND status='PENDING'
		FOR UPDATE`, targetID, actorID).Scan(&reverseID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return friend.RequestResult{}, err
	}
	if err == nil {
		if err := acceptRequestRowTx(ctx, tx, reverseID, targetID, actorID, now); err != nil {
			return friend.RequestResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return friend.RequestResult{}, err
		}
		return friend.RequestResult{Outcome: friend.OutcomeAccepted}, nil
	}

	var forwardExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM friend_requests WHERE requester_id=$1 AND target_id=$2 AND status='PENDING')`,
		actorID, targetID).Scan(&forwardExists); err != nil {
		return friend.RequestResult{}, err
	}
	if forwardExists {
		if err := tx.Commit(ctx); err != nil {
			return friend.RequestResult{}, err
		}
		return friend.RequestResult{Outcome: friend.OutcomeAlreadyPending}, nil
	}

	var requestID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO friend_requests (requester_id,target_id,status,created_at)
		VALUES ($1,$2,'PENDING',$3)
		RETURNING id`, actorID, targetID, now).Scan(&requestID); err != nil {
		return friend.RequestResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		SELECT linkup_enqueue_outbox('friend.requested','friend_request',$1::uuid,$2::uuid,NULL,'{}'::jsonb)`,
		requestID, targetID); err != nil {
		return friend.RequestResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return friend.RequestResult{}, err
	}
	return friend.RequestResult{Outcome: friend.OutcomeRequested}, nil
}

// acceptRequestRowTx marks a known PENDING request row ACCEPTED, inserts
// the resulting friendship (ON CONFLICT DO NOTHING: the friendship may
// already exist from a concurrent/replayed call), and emits the
// friend.accepted outbox event for the original requester (subjectUserID
// = requesterID, never the acceptor — README's FRIEND_ACCEPTED tells the
// person who asked that their request was accepted, not the person who
// just clicked accept and already knows).
func acceptRequestRowTx(ctx context.Context, tx pgx.Tx, requestID, requesterID, targetID string, now time.Time) error {
	if _, err := tx.Exec(ctx, `UPDATE friend_requests SET status='ACCEPTED',responded_at=$2 WHERE id=$1`, requestID, now); err != nil {
		return err
	}
	lo, hi := canonicalPair(requesterID, targetID)
	if _, err := tx.Exec(ctx, `
		INSERT INTO friendships (user_lo_id,user_hi_id,created_at)
		VALUES ($1,$2,$3)
		ON CONFLICT (user_lo_id,user_hi_id) DO NOTHING`, lo, hi, now); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		SELECT linkup_enqueue_outbox('friend.accepted','friend_request',$1::uuid,$2::uuid,NULL,'{}'::jsonb)`,
		requestID, requesterID)
	return err
}

func (s *FriendStore) Accept(ctx context.Context, actorID, requesterID string, now time.Time) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var requestID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM friend_requests
		WHERE requester_id=$1 AND target_id=$2 AND status='PENDING'
		FOR UPDATE`, requesterID, actorID).Scan(&requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Idempotent replay: if the two are already friends (this exact
		// Accept already succeeded once), report success again instead of
		// ErrNotFound — a lost response must not turn a real success into
		// a client-visible error on retry.
		already, areErr := areFriendsTx(ctx, tx, actorID, requesterID)
		if areErr != nil {
			return areErr
		}
		if already {
			return tx.Commit(ctx)
		}
		return friend.ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := acceptRequestRowTx(ctx, tx, requestID, requesterID, actorID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *FriendStore) Reject(ctx context.Context, actorID, requesterID string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE friend_requests SET status='REJECTED',responded_at=now()
		WHERE requester_id=$1 AND target_id=$2 AND status='PENDING'`, requesterID, actorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var alreadyRejected bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM friend_requests WHERE requester_id=$1 AND target_id=$2 AND status='REJECTED')`,
			requesterID, actorID).Scan(&alreadyRejected); err != nil {
			return err
		}
		if !alreadyRejected {
			return friend.ErrNotFound
		}
	}
	return tx.Commit(ctx)
}

func (s *FriendStore) Cancel(ctx context.Context, actorID, targetID string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE friend_requests SET status='CANCELLED',responded_at=now()
		WHERE requester_id=$1 AND target_id=$2 AND status='PENDING'`, actorID, targetID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var alreadyCancelled bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM friend_requests WHERE requester_id=$1 AND target_id=$2 AND status='CANCELLED')`,
			actorID, targetID).Scan(&alreadyCancelled); err != nil {
			return err
		}
		if !alreadyCancelled {
			return friend.ErrNotFound
		}
	}
	return tx.Commit(ctx)
}

func (s *FriendStore) Remove(ctx context.Context, actorID, friendID string) error {
	lo, hi := canonicalPair(actorID, friendID)
	_, err := s.pool.Exec(ctx, `DELETE FROM friendships WHERE user_lo_id=$1 AND user_hi_id=$2`, lo, hi)
	return err
}

const friendUserSummaryColumns = `u.id,u.username,u.display_name,u.avatar_url`

func (s *FriendStore) ListFriends(ctx context.Context, actorID string, limit int) ([]friend.Friend, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+friendUserSummaryColumns+`,f.created_at
		FROM friendships f
		JOIN app_users u ON u.id = CASE WHEN f.user_lo_id=$1 THEN f.user_hi_id ELSE f.user_lo_id END
		WHERE f.user_lo_id=$1 OR f.user_hi_id=$1
		ORDER BY f.created_at DESC
		LIMIT $2`, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]friend.Friend, 0)
	for rows.Next() {
		var item friend.Friend
		if err := rows.Scan(&item.User.ID, &item.User.Username, &item.User.DisplayName, &item.User.AvatarURL, &item.Since); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *FriendStore) ListIncoming(ctx context.Context, actorID string, limit int) ([]friend.PendingRequest, error) {
	return s.listPending(ctx, `
		SELECT r.id,`+friendUserSummaryColumns+`,r.created_at
		FROM friend_requests r
		JOIN app_users u ON u.id = r.requester_id
		WHERE r.target_id=$1 AND r.status='PENDING'
		ORDER BY r.created_at ASC
		LIMIT $2`, actorID, limit)
}

func (s *FriendStore) ListOutgoing(ctx context.Context, actorID string, limit int) ([]friend.PendingRequest, error) {
	return s.listPending(ctx, `
		SELECT r.id,`+friendUserSummaryColumns+`,r.created_at
		FROM friend_requests r
		JOIN app_users u ON u.id = r.target_id
		WHERE r.requester_id=$1 AND r.status='PENDING'
		ORDER BY r.created_at ASC
		LIMIT $2`, actorID, limit)
}

func (s *FriendStore) listPending(ctx context.Context, query, actorID string, limit int) ([]friend.PendingRequest, error) {
	rows, err := s.pool.Query(ctx, query, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]friend.PendingRequest, 0)
	for rows.Next() {
		var item friend.PendingRequest
		if err := rows.Scan(&item.RequestID, &item.User.ID, &item.User.Username, &item.User.DisplayName, &item.User.AvatarURL, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *FriendStore) AreFriends(ctx context.Context, a, b string) (bool, error) {
	return areFriendsTx(ctx, s.pool, a, b)
}

// queryRower is the minimal subset both pgx.Tx and *pgxpool.Pool satisfy,
// letting areFriendsTx run identically inside a transaction (Request,
// Accept's idempotent-replay check) or standalone (AreFriends).
type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func areFriendsTx(ctx context.Context, q queryRower, a, b string) (bool, error) {
	lo, hi := canonicalPair(a, b)
	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM friendships WHERE user_lo_id=$1 AND user_hi_id=$2)`, lo, hi).Scan(&exists)
	return exists, err
}
