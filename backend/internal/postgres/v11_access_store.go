package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// Join serializes all INSTANT admissions on the Slot row so capacity cannot be oversubscribed.
func (s *V11SlotStore) Join(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	replay, resourceID, err := s.claimIdempotency(ctx, tx, actorID, key, "slot.join", requestHash, slotID, now)
	if err != nil {
		return slot.Slot{}, err
	}
	if replay {
		out, err := getV11SlotInternalTx(ctx, tx, resourceID, actorID)
		if err != nil {
			return slot.Slot{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return slot.Slot{}, err
		}
		return out, nil
	}

	var hostID, state, accessMode string
	var capacity, acceptedCount int
	err = tx.QueryRow(ctx, `SELECT host_id,state,access_mode,capacity,accepted_count FROM slots WHERE id=$1 FOR UPDATE`, slotID).
		Scan(&hostID, &state, &accessMode, &capacity, &acceptedCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	if hostID == actorID {
		return slot.Slot{}, slot.ErrForbidden
	}
	if accessMode != string(slot.AccessInstant) {
		return slot.Slot{}, slot.ErrInvalidState
	}
	if state == "FULL" || acceptedCount >= capacity {
		return slot.Slot{}, slot.ErrCapacityFull
	}
	if state != "FILLING" && state != "PUBLISHED" {
		return slot.Slot{}, slot.ErrInvalidState
	}
	blocked, err := blockedPairTx(ctx, tx, actorID, hostID)
	if err != nil {
		return slot.Slot{}, err
	}
	if blocked {
		return slot.Slot{}, slot.ErrForbidden
	}

	var memberExists, requestExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM slot_memberships WHERE slot_id=$1 AND user_id=$2), EXISTS(SELECT 1 FROM slot_requests WHERE slot_id=$1 AND user_id=$2)`, slotID, actorID).
		Scan(&memberExists, &requestExists); err != nil {
		return slot.Slot{}, err
	}
	if memberExists {
		return slot.Slot{}, slot.ErrAlreadyMember
	}
	if requestExists {
		return slot.Slot{}, slot.ErrInvalidState
	}

	if _, err := tx.Exec(ctx, `INSERT INTO slot_memberships (slot_id,user_id,accepted_at) VALUES ($1,$2,$3)`, slotID, actorID, now); err != nil {
		return slot.Slot{}, err
	}
	if err := emitSystemChatMessageTx(ctx, tx, slotID, string(chat.SystemEventMemberJoined), &actorID); err != nil {
		return slot.Slot{}, err
	}
	newCount := acceptedCount + 1
	newState := "FILLING"
	if newCount >= capacity {
		newState = "FULL"
	}
	if _, err := tx.Exec(ctx, `UPDATE slots SET accepted_count=$2,state=$3,version=version+1,updated_at=$4 WHERE id=$1`, slotID, newCount, newState, now); err != nil {
		return slot.Slot{}, err
	}
	out, err := getV11SlotInternalTx(ctx, tx, slotID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}
