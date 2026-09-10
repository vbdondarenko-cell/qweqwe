package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// Request preserves the v1.0 APPROVAL authority and extends only WAITLIST.
func (s *V11SlotStore) Request(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	mode, err := s.slotAccessMode(ctx, slotID)
	if err != nil {
		return slot.Slot{}, err
	}
	if mode != string(slot.AccessWaitlist) {
		if _, err := s.SlotStore.Request(ctx, actorID, slotID, key, requestHash, now); err != nil {
			return slot.Slot{}, err
		}
		return s.getInternal(ctx, actorID, slotID)
	}
	return s.waitlistRequest(ctx, actorID, slotID, key, requestHash, now)
}

func (s *V11SlotStore) waitlistRequest(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	replay, resourceID, err := s.claimIdempotency(ctx, tx, actorID, key, "slot.request", requestHash, slotID, now)
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
	if accessMode != string(slot.AccessWaitlist) || (state != "PUBLISHED" && state != "FILLING" && state != "FULL") {
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
		return slot.Slot{}, slot.ErrDuplicateRequest
	}

	if acceptedCount < capacity {
		if _, err := tx.Exec(ctx, `INSERT INTO slot_memberships (slot_id,user_id,accepted_at) VALUES ($1,$2,$3)`, slotID, actorID, now); err != nil {
			return slot.Slot{}, err
		}
		acceptedCount++
		state = "FILLING"
		if acceptedCount >= capacity {
			state = "FULL"
		}
	} else {
		var pendingCount int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM slot_requests WHERE slot_id=$1`, slotID).Scan(&pendingCount); err != nil {
			return slot.Slot{}, err
		}
		if pendingCount >= slot.MaxPendingRequests {
			return slot.Slot{}, slot.ErrRequestLimit
		}
		if _, err := tx.Exec(ctx, `INSERT INTO slot_requests (slot_id,user_id,created_at) VALUES ($1,$2,$3)`, slotID, actorID, now); err != nil {
			return slot.Slot{}, err
		}
		state = "FULL"
	}

	if _, err := tx.Exec(ctx, `UPDATE slots SET accepted_count=$2,state=$3,version=version+1,updated_at=$4 WHERE id=$1`, slotID, acceptedCount, state, now); err != nil {
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

func (s *V11SlotStore) Leave(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	mode, err := s.slotAccessMode(ctx, slotID)
	if err != nil {
		return slot.Slot{}, err
	}
	if mode != string(slot.AccessWaitlist) {
		if _, err := s.SlotStore.Leave(ctx, actorID, slotID, key, requestHash, now); err != nil {
			return slot.Slot{}, err
		}
		return s.getInternal(ctx, actorID, slotID)
	}
	return s.waitlistLeave(ctx, actorID, slotID, key, requestHash, now)
}

func (s *V11SlotStore) waitlistLeave(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	replay, resourceID, err := s.claimIdempotency(ctx, tx, actorID, key, "slot.leave", requestHash, slotID, now)
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
	var acceptedCount, capacity int
	err = tx.QueryRow(ctx, `SELECT host_id,state,access_mode,accepted_count,capacity FROM slots WHERE id=$1 FOR UPDATE`, slotID).
		Scan(&hostID, &state, &accessMode, &acceptedCount, &capacity)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	if hostID == actorID {
		return slot.Slot{}, slot.ErrForbidden
	}
	if accessMode != string(slot.AccessWaitlist) || isTerminal(state) {
		return slot.Slot{}, slot.ErrInvalidState
	}

	if tag, err := tx.Exec(ctx, `DELETE FROM slot_requests WHERE slot_id=$1 AND user_id=$2`, slotID, actorID); err != nil {
		return slot.Slot{}, err
	} else if tag.RowsAffected() > 0 {
		if _, err := tx.Exec(ctx, `UPDATE slots SET version=version+1,updated_at=$2 WHERE id=$1`, slotID, now); err != nil {
			return slot.Slot{}, err
		}
	} else {
		tag, err := tx.Exec(ctx, `DELETE FROM slot_memberships WHERE slot_id=$1 AND user_id=$2`, slotID, actorID)
		if err != nil {
			return slot.Slot{}, err
		}
		if tag.RowsAffected() == 0 {
			return slot.Slot{}, slot.ErrNotFound
		}
		if acceptedCount <= 0 {
			return slot.Slot{}, errors.New("slot accepted_count invariant violated")
		}
		acceptedCount--
		if state != "ACTIVE" {
			acceptedCount, _, err = promoteOldestWaitlistTx(ctx, tx, slotID, hostID, acceptedCount, capacity, now)
			if err != nil {
				return slot.Slot{}, err
			}
			state = "FILLING"
			if acceptedCount >= capacity {
				state = "FULL"
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE slots SET accepted_count=$2,state=$3,version=version+1,updated_at=$4 WHERE id=$1`, slotID, acceptedCount, state, now); err != nil {
			return slot.Slot{}, err
		}
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

func (s *V11SlotStore) waitlistRemoveMember(ctx context.Context, actorID, slotID, memberID string, expectedVersion int64, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	replay, resourceID, err := s.claimIdempotency(ctx, tx, actorID, key, "slot.remove_member", requestHash, slotID, now)
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
	var acceptedCount, capacity int
	var version int64
	err = tx.QueryRow(ctx, `SELECT host_id,state,access_mode,accepted_count,capacity,version FROM slots WHERE id=$1 FOR UPDATE`, slotID).
		Scan(&hostID, &state, &accessMode, &acceptedCount, &capacity, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	if hostID != actorID || memberID == hostID {
		return slot.Slot{}, slot.ErrForbidden
	}
	if accessMode != string(slot.AccessWaitlist) || (state != "PUBLISHED" && state != "FILLING" && state != "FULL" && state != "ACTIVE") {
		return slot.Slot{}, slot.ErrInvalidState
	}
	if version != expectedVersion {
		return slot.Slot{}, slot.ErrConflict
	}
	blocked, err := blockedPairTx(ctx, tx, actorID, memberID)
	if err != nil {
		return slot.Slot{}, err
	}
	if blocked {
		return slot.Slot{}, slot.ErrForbidden
	}
	tag, err := tx.Exec(ctx, `DELETE FROM slot_memberships WHERE slot_id=$1 AND user_id=$2`, slotID, memberID)
	if err != nil {
		return slot.Slot{}, err
	}
	if tag.RowsAffected() == 0 {
		return slot.Slot{}, slot.ErrNotFound
	}
	if acceptedCount <= 0 {
		return slot.Slot{}, errors.New("slot accepted_count invariant violated")
	}
	acceptedCount--
	if state != "ACTIVE" {
		acceptedCount, _, err = promoteOldestWaitlistTx(ctx, tx, slotID, hostID, acceptedCount, capacity, now)
		if err != nil {
			return slot.Slot{}, err
		}
		state = "FILLING"
		if acceptedCount >= capacity {
			state = "FULL"
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE slots SET accepted_count=$2,state=$3,version=version+1,updated_at=$4 WHERE id=$1`, slotID, acceptedCount, state, now); err != nil {
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

func (s *V11SlotStore) slotAccessMode(ctx context.Context, slotID string) (string, error) {
	var mode string
	if err := s.pool.QueryRow(ctx, `SELECT access_mode FROM slots WHERE id=$1`, slotID).Scan(&mode); errors.Is(err, pgx.ErrNoRows) {
		return "", slot.ErrNotFound
	} else if err != nil {
		return "", err
	}
	return mode, nil
}

func promoteOldestWaitlistTx(ctx context.Context, tx pgx.Tx, slotID, hostID string, acceptedCount, capacity int, now time.Time) (int, string, error) {
	if acceptedCount >= capacity {
		return acceptedCount, "", nil
	}
	rows, err := tx.Query(ctx, `SELECT user_id::text FROM slot_requests WHERE slot_id=$1 ORDER BY created_at ASC,user_id ASC FOR UPDATE`, slotID)
	if err != nil {
		return acceptedCount, "", err
	}
	var candidates []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return acceptedCount, "", err
		}
		candidates = append(candidates, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return acceptedCount, "", err
	}
	rows.Close()

	for _, candidateID := range candidates {
		blocked, err := blockedPairTx(ctx, tx, candidateID, hostID)
		if err != nil {
			return acceptedCount, "", err
		}
		if blocked {
			if _, err := tx.Exec(ctx, `DELETE FROM slot_requests WHERE slot_id=$1 AND user_id=$2`, slotID, candidateID); err != nil {
				return acceptedCount, "", err
			}
			continue
		}
		var memberExists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM slot_memberships WHERE slot_id=$1 AND user_id=$2)`, slotID, candidateID).Scan(&memberExists); err != nil {
			return acceptedCount, "", err
		}
		if memberExists {
			if _, err := tx.Exec(ctx, `DELETE FROM slot_requests WHERE slot_id=$1 AND user_id=$2`, slotID, candidateID); err != nil {
				return acceptedCount, "", err
			}
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO slot_memberships (slot_id,user_id,accepted_at) VALUES ($1,$2,$3)`, slotID, candidateID, now); err != nil {
			return acceptedCount, "", err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM slot_requests WHERE slot_id=$1 AND user_id=$2`, slotID, candidateID); err != nil {
			return acceptedCount, "", err
		}
		return acceptedCount + 1, candidateID, nil
	}
	return acceptedCount, "", nil
}
