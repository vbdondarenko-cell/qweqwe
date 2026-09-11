package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
)

type BlockStore struct {
	pool *pgxpool.Pool
	// waitlistRequestTTL matches V11SlotStore.waitlistRequestTTL so a block
	// that frees a WAITLIST seat promotes using the same expiry rule as
	// every other promotion path. 0 disables expiry (v1.0-only wiring).
	waitlistRequestTTL time.Duration
}

func NewBlockStore(pool *pgxpool.Pool, waitlistRequestTTL time.Duration) *BlockStore {
	return &BlockStore{pool: pool, waitlistRequestTTL: waitlistRequestTTL}
}

func (s *BlockStore) Block(ctx context.Context, blockerID, blockedID string, now time.Time) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := blockPairTx(ctx, tx, blockerID, blockedID, now, s.waitlistRequestTTL); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func blockPairTx(ctx context.Context, tx pgx.Tx, blockerID, blockedID string, now time.Time, waitlistRequestTTL time.Duration) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app_users WHERE id=$1)`, blockedID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return blocklist.ErrInvalidTarget
	}

	// Social mutations lock the Slot row before checking user_blocks. Block must
	// use the same ordering: lock every mutable Slot hosted by either side first,
	// then publish the block and clean existing relationships. If a mutation won
	// the Slot lock first, cleanup runs after it commits; if Block wins first, the
	// mutation sees the committed block after waiting for the Slot lock.
	if err := lockHostedSlotsForBlock(ctx, tx, blockerID, blockedID); err != nil {
		return err
	}

	// Remember exactly which (Slot, user) memberships this block is about to
	// remove, so a MEMBER_LEFT system notice can be emitted per membership
	// after the bulk removal below. The bulk statement is a single
	// set-based DELETE/UPDATE across every Slot the two users share, not a
	// per-row loop, so this is the only point where those pairs are known.
	// The notice reads exactly like a voluntary leave ("X left") — it does
	// not say a block caused it — so it adds no information beyond what
	// every other participant already learns as soon as the blocked user's
	// name silently disappears from the roster (README §8.1: blocked users
	// are already filtered from every active chat/social/realtime layer).
	memberRows, err := tx.Query(ctx, `
		SELECT m.slot_id::text,m.user_id::text
		FROM slot_memberships m
		JOIN slots s ON s.id=m.slot_id
		WHERE (s.host_id=$1 AND m.user_id=$2) OR (s.host_id=$2 AND m.user_id=$1)
		ORDER BY m.slot_id::text,m.user_id::text`, blockerID, blockedID)
	if err != nil {
		return err
	}
	type removedMembership struct{ slotID, userID string }
	var removedMemberships []removedMembership
	for memberRows.Next() {
		var m removedMembership
		if err := memberRows.Scan(&m.slotID, &m.userID); err != nil {
			memberRows.Close()
			return err
		}
		removedMemberships = append(removedMemberships, m)
	}
	if err := memberRows.Err(); err != nil {
		memberRows.Close()
		return err
	}
	memberRows.Close()

	// Remember WAITLIST Slots where this block removes an accepted member. The
	// Slot rows are already locked above, so promotion can safely happen in the
	// same transaction after relationship cleanup without creating a seat gap.
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT s.id::text
		FROM slots s
		JOIN slot_memberships m ON m.slot_id=s.id
		WHERE s.access_mode='WAITLIST'
		  AND s.state IN ('PUBLISHED','FILLING','FULL')
		  AND ((s.host_id=$1 AND m.user_id=$2) OR (s.host_id=$2 AND m.user_id=$1))
		ORDER BY s.id::text`, blockerID, blockedID)
	if err != nil {
		return err
	}
	var waitlistFreed []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		waitlistFreed = append(waitlistFreed, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	if _, err := tx.Exec(ctx, `
		INSERT INTO user_blocks (blocker_id,blocked_id,created_at)
		VALUES ($1,$2,$3)
		ON CONFLICT (blocker_id,blocked_id) DO NOTHING`, blockerID, blockedID, now); err != nil {
		return err
	}

	// Pending and accepted relationships are revoked in one statement. Every
	// affected Slot advances version exactly once, including pending-only cleanup.
	// Membership removals also correct accepted_count and reopen FULL to FILLING.
	if _, err := tx.Exec(ctx, `
		WITH removed_requests AS (
			DELETE FROM slot_requests r
			USING slots s
			WHERE r.slot_id=s.id
			  AND ((s.host_id=$1 AND r.user_id=$2) OR (s.host_id=$2 AND r.user_id=$1))
			RETURNING r.slot_id
		), removed_members AS (
			DELETE FROM slot_memberships m
			USING slots s
			WHERE m.slot_id=s.id
			  AND ((s.host_id=$1 AND m.user_id=$2) OR (s.host_id=$2 AND m.user_id=$1))
			RETURNING m.slot_id
		), affected AS (
			SELECT slot_id,0::int AS removed_members FROM removed_requests
			UNION ALL
			SELECT slot_id,1::int AS removed_members FROM removed_members
		), counts AS (
			SELECT slot_id,sum(removed_members)::int AS removed_members
			FROM affected
			GROUP BY slot_id
		)
		UPDATE slots s
		SET accepted_count=GREATEST(0,s.accepted_count-counts.removed_members),
			state=CASE WHEN counts.removed_members>0 AND s.state='FULL' THEN 'FILLING' ELSE s.state END,
			version=s.version+1,
			updated_at=$3
		FROM counts
		WHERE s.id=counts.slot_id`, blockerID, blockedID, now); err != nil {
		return err
	}

	for _, m := range removedMemberships {
		if err := emitSystemChatMessageTx(ctx, tx, m.slotID, string(chat.SystemEventMemberLeft), &m.userID); err != nil {
			return err
		}
	}

	for _, slotID := range waitlistFreed {
		var hostID, state string
		var acceptedCount, capacity int
		if err := tx.QueryRow(ctx, `SELECT host_id,state,accepted_count,capacity FROM slots WHERE id=$1`, slotID).
			Scan(&hostID, &state, &acceptedCount, &capacity); err != nil {
			return err
		}
		for acceptedCount < capacity {
			next, promotedID, err := promoteOldestWaitlistTx(ctx, tx, slotID, hostID, acceptedCount, capacity, now, waitlistRequestTTL)
			if err != nil {
				return err
			}
			acceptedCount = next
			if promotedID == "" {
				break
			}
		}
		state = "FILLING"
		if acceptedCount >= capacity {
			state = "FULL"
		}
		if _, err := tx.Exec(ctx, `UPDATE slots SET accepted_count=$2,state=$3 WHERE id=$1`, slotID, acceptedCount, state); err != nil {
			return err
		}
	}
	return nil
}

func lockHostedSlotsForBlock(ctx context.Context, tx pgx.Tx, a, b string) error {
	rows, err := tx.Query(ctx, `
		SELECT id
		FROM slots
		WHERE host_id IN ($1,$2)
		  AND state NOT IN ('COMPLETED','CANCELLED','EXPIRED','MODERATED')
		ORDER BY id
		FOR UPDATE`, a, b)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *BlockStore) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM user_blocks WHERE blocker_id=$1 AND blocked_id=$2`, blockerID, blockedID)
	return err
}

func (s *BlockStore) List(ctx context.Context, blockerID string, limit int) ([]blocklist.UserSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id,u.username,u.display_name,u.avatar_url
		FROM user_blocks b
		JOIN app_users u ON u.id=b.blocked_id
		WHERE b.blocker_id=$1
		ORDER BY b.created_at DESC,u.id
		LIMIT $2`, blockerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]blocklist.UserSummary, 0)
	for rows.Next() {
		var item blocklist.UserSummary
		if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.AvatarURL); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
