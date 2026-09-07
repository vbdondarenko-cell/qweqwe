package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func authorizeIdempotencyReplayTx(ctx context.Context, tx pgx.Tx, actorID, resourceID, operation string) error {
	if resourceID == "" {
		return slot.ErrForbidden
	}

	var allowed bool
	var err error
	switch operation {
	case "slot.create", "slot.edit", "slot.cancel", "slot.approve", "slot.reject", "slot.start", "slot.complete", "slot.remove_member":
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM slots
				WHERE id=$1 AND host_id=$2
			)`, resourceID, actorID).Scan(&allowed)

	case "slot.request":
		// A lost REQUEST response may be replayed after the request became accepted,
		// but never after rejection/withdrawal/block removed the relationship.
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM slots s
				WHERE s.id=$1
				  AND s.host_id<>$2
				  AND NOT EXISTS (
					SELECT 1 FROM user_blocks b
					WHERE (b.blocker_id=$2 AND b.blocked_id=s.host_id)
					   OR (b.blocker_id=s.host_id AND b.blocked_id=$2)
				  )
				  AND (
					EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$2)
					OR EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$2)
				  )
			)`, resourceID, actorID).Scan(&allowed)

	case "slot.leave":
		// Successful LEAVE intentionally removes the actor's relationship. A replay
		// may therefore return current state only when the resource is still readable
		// through the normal v1.0 access model (or the relationship still exists).
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM slots s
				WHERE s.id=$1
				  AND (
					EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$2)
					OR EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$2)
					OR (
						s.visibility='PUBLIC'
						AND s.state IN ('PUBLISHED','FILLING','FULL')
						AND NOT EXISTS (
							SELECT 1 FROM user_blocks b
							WHERE (b.blocker_id=$2 AND b.blocked_id=s.host_id)
							   OR (b.blocker_id=s.host_id AND b.blocked_id=$2)
						)
					)
				  )
			)`, resourceID, actorID).Scan(&allowed)

	default:
		return slot.ErrForbidden
	}
	if err != nil {
		return err
	}
	if !allowed {
		return slot.ErrForbidden
	}
	return nil
}
