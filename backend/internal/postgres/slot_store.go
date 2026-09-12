package postgres

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type SlotStore struct {
	pool           *pgxpool.Pool
	idempotencyTTL time.Duration
}

func NewSlotStore(pool *pgxpool.Pool, idempotencyTTL time.Duration) (*SlotStore, error) {
	if pool == nil || idempotencyTTL <= 0 {
		return nil, errors.New("invalid slot store dependencies")
	}
	return &SlotStore{pool: pool, idempotencyTTL: idempotencyTTL}, nil
}

func (s *SlotStore) Create(ctx context.Context, actorID string, candidate slot.Slot, key string, requestHash []byte) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	replay, resourceID, err := s.claimIdempotency(ctx, tx, actorID, key, "slot.create", requestHash, candidate.ID, candidate.CreatedAt)
	if err != nil {
		return slot.Slot{}, err
	}
	if replay {
		out, err := getSlotInternalTx(ctx, tx, resourceID, actorID)
		if err != nil {
			return slot.Slot{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return slot.Slot{}, err
		}
		return out, nil
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO slots (
			id,host_id,title,activity,details,place_text,zone_text,start_at,
			capacity,accepted_count,state,access_mode,visibility,version,created_at,updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,0,$10,$11,$12,1,$13,$13)`,
		candidate.ID, actorID, candidate.Title, candidate.Activity, candidate.Details,
		candidate.PlaceText, candidate.ZoneText, candidate.StartAt, candidate.Capacity,
		string(candidate.State), string(candidate.AccessMode), string(candidate.Visibility), candidate.CreatedAt,
	)
	if err != nil {
		return slot.Slot{}, err
	}
	out, err := getSlotInternalTx(ctx, tx, candidate.ID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}

func (s *SlotStore) Get(ctx context.Context, actorID, slotID string) (slot.Slot, error) {
	return scanSlot(s.pool.QueryRow(ctx, getSlotSQL, slotID, actorID))
}

// sort is accepted (to satisfy slot.Store) but ignored: this v1.0 base
// store only ever surfaces PUBLIC Slots, and README §6.10's ranking
// pipeline is v1.1 scope — V11SlotStore.ListPulse is where sort actually
// does something; this legacy path always stays pure recency, matching
// how it already has no visibility-mode concept beyond PUBLIC either.
func (s *SlotStore) ListPulse(ctx context.Context, actorID string, limit int, sort slot.PulseSort) ([]slot.Slot, error) {
	rows, err := s.pool.Query(ctx, listPulseSQL, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]slot.Slot, 0)
	for rows.Next() {
		item, err := scanSlot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SlotStore) Edit(ctx context.Context, actorID, slotID string, patch slot.EditInput, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	replay, resourceID, err := s.claimIdempotency(ctx, tx, actorID, key, "slot.edit", requestHash, slotID, now)
	if err != nil {
		return slot.Slot{}, err
	}
	if replay {
		out, err := getSlotInternalTx(ctx, tx, resourceID, actorID)
		if err != nil {
			return slot.Slot{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return slot.Slot{}, err
		}
		return out, nil
	}

	var hostID, state string
	var version int64
	var acceptedCount, currentCapacity int
	err = tx.QueryRow(ctx, `SELECT host_id,state,version,accepted_count,capacity FROM slots WHERE id=$1 FOR UPDATE`, slotID).
		Scan(&hostID, &state, &version, &acceptedCount, &currentCapacity)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	if hostID != actorID {
		return slot.Slot{}, slot.ErrForbidden
	}
	if state != "DRAFT" && state != "PUBLISHED" && state != "FILLING" && state != "FULL" {
		return slot.Slot{}, slot.ErrInvalidState
	}
	if version != patch.ExpectedVersion {
		return slot.Slot{}, slot.ErrConflict
	}

	newCapacity := currentCapacity
	if patch.Capacity != nil {
		newCapacity = *patch.Capacity
	}
	if newCapacity < acceptedCount {
		return slot.Slot{}, slot.ErrInvalidInput
	}

	titleSet := patch.Title != nil
	title := ""
	if patch.Title != nil {
		title = *patch.Title
	}
	detailsSet := patch.Details != nil
	details := ""
	if patch.Details != nil {
		details = *patch.Details
	}
	placeSet := patch.PlaceText != nil
	place := ""
	if patch.PlaceText != nil {
		place = *patch.PlaceText
	}
	zoneSet := patch.ZoneText != nil
	zone := ""
	if patch.ZoneText != nil {
		zone = *patch.ZoneText
	}
	startSet := patch.StartAt != nil || patch.ClearStartAt
	capacitySet := patch.Capacity != nil
	newState := state
	if state == "FILLING" || state == "FULL" {
		if acceptedCount >= newCapacity {
			newState = "FULL"
		} else {
			newState = "FILLING"
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE slots SET
			title=CASE WHEN $2 THEN $3 ELSE title END,
			details=CASE WHEN $4 THEN NULLIF($5,'') ELSE details END,
			place_text=CASE WHEN $6 THEN $7 ELSE place_text END,
			zone_text=CASE WHEN $8 THEN NULLIF($9,'') ELSE zone_text END,
			start_at=CASE WHEN $10 THEN $11 ELSE start_at END,
			capacity=CASE WHEN $12 THEN $13 ELSE capacity END,
			state=$14,version=version+1,updated_at=$15
		WHERE id=$1`,
		slotID, titleSet, title, detailsSet, details, placeSet, place, zoneSet, zone,
		startSet, patch.StartAt, capacitySet, newCapacity, newState, now,
	)
	if err != nil {
		return slot.Slot{}, err
	}
	out, err := getSlotInternalTx(ctx, tx, slotID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}

func (s *SlotStore) Cancel(ctx context.Context, actorID, slotID string, expectedVersion int64, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	replay, resourceID, err := s.claimIdempotency(ctx, tx, actorID, key, "slot.cancel", requestHash, slotID, now)
	if err != nil {
		return slot.Slot{}, err
	}
	if replay {
		out, err := getSlotInternalTx(ctx, tx, resourceID, actorID)
		if err != nil {
			return slot.Slot{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return slot.Slot{}, err
		}
		return out, nil
	}

	var hostID, state string
	var version int64
	err = tx.QueryRow(ctx, `SELECT host_id,state,version FROM slots WHERE id=$1 FOR UPDATE`, slotID).Scan(&hostID, &state, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	if hostID != actorID {
		return slot.Slot{}, slot.ErrForbidden
	}
	if isTerminal(state) {
		return slot.Slot{}, slot.ErrInvalidState
	}
	if version != expectedVersion {
		return slot.Slot{}, slot.ErrConflict
	}

	if _, err := tx.Exec(ctx, `DELETE FROM slot_requests WHERE slot_id=$1`, slotID); err != nil {
		return slot.Slot{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE slots SET state='CANCELLED',cancelled_at=$2,updated_at=$2,version=version+1 WHERE id=$1`, slotID, now); err != nil {
		return slot.Slot{}, err
	}
	out, err := getSlotInternalTx(ctx, tx, slotID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}

func (s *SlotStore) Request(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
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
		out, err := getSlotInternalTx(ctx, tx, resourceID, actorID)
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
	if accessMode != "APPROVAL" {
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
		return slot.Slot{}, slot.ErrDuplicateRequest
	}

	// The Slot row lock serializes all REQUEST inserts for this Slot, so this
	// count is an atomic queue bound rather than a racy preflight check.
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
	if _, err := tx.Exec(ctx, `UPDATE slots SET version=version+1,updated_at=$2 WHERE id=$1`, slotID, now); err != nil {
		return slot.Slot{}, err
	}
	out, err := getSlotInternalTx(ctx, tx, slotID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}

func (s *SlotStore) Leave(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
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
		out, err := getSlotInternalTx(ctx, tx, resourceID, actorID)
		if err != nil {
			return slot.Slot{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return slot.Slot{}, err
		}
		return out, nil
	}

	var hostID, state string
	var acceptedCount int
	err = tx.QueryRow(ctx, `SELECT host_id,state,accepted_count FROM slots WHERE id=$1 FOR UPDATE`, slotID).Scan(&hostID, &state, &acceptedCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	if hostID == actorID {
		return slot.Slot{}, slot.ErrForbidden
	}
	if isTerminal(state) {
		return slot.Slot{}, slot.ErrInvalidState
	}

	tag, err := tx.Exec(ctx, `DELETE FROM slot_requests WHERE slot_id=$1 AND user_id=$2`, slotID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if tag.RowsAffected() > 0 {
		if _, err := tx.Exec(ctx, `UPDATE slots SET version=version+1,updated_at=$2 WHERE id=$1`, slotID, now); err != nil {
			return slot.Slot{}, err
		}
		out, err := getSlotInternalTx(ctx, tx, slotID, actorID)
		if err != nil {
			return slot.Slot{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return slot.Slot{}, err
		}
		return out, nil
	}

	tag, err = tx.Exec(ctx, `DELETE FROM slot_memberships WHERE slot_id=$1 AND user_id=$2`, slotID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if tag.RowsAffected() == 0 {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err := emitSystemChatMessageTx(ctx, tx, slotID, string(chat.SystemEventMemberLeft), &actorID); err != nil {
		return slot.Slot{}, err
	}
	if acceptedCount <= 0 {
		return slot.Slot{}, errors.New("slot accepted_count invariant violated")
	}
	newState := state
	if state == "FULL" {
		newState = "FILLING"
	}
	if _, err := tx.Exec(ctx, `UPDATE slots SET accepted_count=accepted_count-1,state=$2,version=version+1,updated_at=$3 WHERE id=$1`, slotID, newState, now); err != nil {
		return slot.Slot{}, err
	}
	out, err := getSlotInternalTx(ctx, tx, slotID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}

func (s *SlotStore) ListPending(ctx context.Context, actorID, slotID string) ([]slot.PendingRequest, error) {
	var hostID string
	if err := s.pool.QueryRow(ctx, `SELECT host_id FROM slots WHERE id=$1`, slotID).Scan(&hostID); errors.Is(err, pgx.ErrNoRows) {
		return nil, slot.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if hostID != actorID {
		return nil, slot.ErrForbidden
	}

	rows, err := s.pool.Query(ctx, `
		SELECT u.id,u.username,u.display_name,u.avatar_url,r.created_at
		FROM slot_requests r
		JOIN app_users u ON u.id=r.user_id
		WHERE r.slot_id=$1
		  AND NOT EXISTS (
			SELECT 1 FROM user_blocks b
			WHERE (b.blocker_id=$2 AND b.blocked_id=r.user_id)
			   OR (b.blocker_id=r.user_id AND b.blocked_id=$2)
		  )
		ORDER BY r.created_at ASC,u.id
		LIMIT $3`, slotID, actorID, slot.MaxPendingRequests)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]slot.PendingRequest, 0, slot.MaxPendingRequests)
	for rows.Next() {
		var item slot.PendingRequest
		if err := rows.Scan(&item.User.ID, &item.User.Username, &item.User.DisplayName, &item.User.AvatarURL, &item.RequestedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SlotStore) Approve(ctx context.Context, actorID, slotID, requesterID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	replay, resourceID, err := s.claimIdempotency(ctx, tx, actorID, key, "slot.approve", requestHash, slotID, now)
	if err != nil {
		return slot.Slot{}, err
	}
	if replay {
		out, err := getSlotInternalTx(ctx, tx, resourceID, actorID)
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
	if hostID != actorID {
		return slot.Slot{}, slot.ErrForbidden
	}
	if accessMode != "APPROVAL" || (state != "FILLING" && state != "FULL") {
		return slot.Slot{}, slot.ErrInvalidState
	}
	if state == "FULL" || acceptedCount >= capacity {
		return slot.Slot{}, slot.ErrCapacityFull
	}
	blocked, err := blockedPairTx(ctx, tx, actorID, requesterID)
	if err != nil {
		return slot.Slot{}, err
	}
	if blocked {
		return slot.Slot{}, slot.ErrForbidden
	}

	var requestExists, memberExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM slot_requests WHERE slot_id=$1 AND user_id=$2), EXISTS(SELECT 1 FROM slot_memberships WHERE slot_id=$1 AND user_id=$2)`, slotID, requesterID).
		Scan(&requestExists, &memberExists); err != nil {
		return slot.Slot{}, err
	}
	if memberExists {
		return slot.Slot{}, slot.ErrAlreadyMember
	}
	if !requestExists {
		return slot.Slot{}, slot.ErrRequestNotFound
	}

	if _, err := tx.Exec(ctx, `INSERT INTO slot_memberships (slot_id,user_id,accepted_at) VALUES ($1,$2,$3)`, slotID, requesterID, now); err != nil {
		return slot.Slot{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM slot_requests WHERE slot_id=$1 AND user_id=$2`, slotID, requesterID); err != nil {
		return slot.Slot{}, err
	}
	if err := emitSystemChatMessageTx(ctx, tx, slotID, string(chat.SystemEventMemberJoined), &requesterID); err != nil {
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
	out, err := getSlotInternalTx(ctx, tx, slotID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}

func (s *SlotStore) Reject(ctx context.Context, actorID, slotID, requesterID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	replay, resourceID, err := s.claimIdempotency(ctx, tx, actorID, key, "slot.reject", requestHash, slotID, now)
	if err != nil {
		return slot.Slot{}, err
	}
	if replay {
		out, err := getSlotInternalTx(ctx, tx, resourceID, actorID)
		if err != nil {
			return slot.Slot{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return slot.Slot{}, err
		}
		return out, nil
	}

	var hostID, state string
	err = tx.QueryRow(ctx, `SELECT host_id,state FROM slots WHERE id=$1 FOR UPDATE`, slotID).Scan(&hostID, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	if hostID != actorID {
		return slot.Slot{}, slot.ErrForbidden
	}
	if isTerminal(state) || state == "ACTIVE" {
		return slot.Slot{}, slot.ErrInvalidState
	}
	tag, err := tx.Exec(ctx, `DELETE FROM slot_requests WHERE slot_id=$1 AND user_id=$2`, slotID, requesterID)
	if err != nil {
		return slot.Slot{}, err
	}
	if tag.RowsAffected() == 0 {
		return slot.Slot{}, slot.ErrRequestNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE slots SET version=version+1,updated_at=$2 WHERE id=$1`, slotID, now); err != nil {
		return slot.Slot{}, err
	}
	out, err := getSlotInternalTx(ctx, tx, slotID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}

func (s *SlotStore) Start(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	return s.hostLifecycle(ctx, actorID, slotID, key, requestHash, now, true)
}

func (s *SlotStore) Complete(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	return s.hostLifecycle(ctx, actorID, slotID, key, requestHash, now, false)
}

func (s *SlotStore) hostLifecycle(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time, start bool) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	operation := "slot.complete"
	if start {
		operation = "slot.start"
	}
	replay, resourceID, err := s.claimIdempotency(ctx, tx, actorID, key, operation, requestHash, slotID, now)
	if err != nil {
		return slot.Slot{}, err
	}
	if replay {
		out, err := getSlotInternalTx(ctx, tx, resourceID, actorID)
		if err != nil {
			return slot.Slot{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return slot.Slot{}, err
		}
		return out, nil
	}

	var hostID, state string
	var acceptedCount int
	err = tx.QueryRow(ctx, `SELECT host_id,state,accepted_count FROM slots WHERE id=$1 FOR UPDATE`, slotID).Scan(&hostID, &state, &acceptedCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	if hostID != actorID {
		return slot.Slot{}, slot.ErrForbidden
	}
	if start {
		if (state != "FILLING" && state != "FULL") || acceptedCount < 1 {
			return slot.Slot{}, slot.ErrInvalidState
		}
		if _, err := tx.Exec(ctx, `DELETE FROM slot_requests WHERE slot_id=$1`, slotID); err != nil {
			return slot.Slot{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE slots SET state='ACTIVE',started_at=$2,version=version+1,updated_at=$2 WHERE id=$1`, slotID, now); err != nil {
			return slot.Slot{}, err
		}
		if err := emitSystemChatMessageTx(ctx, tx, slotID, string(chat.SystemEventSlotStarted), &hostID); err != nil {
			return slot.Slot{}, err
		}
	} else {
		if state != "ACTIVE" {
			return slot.Slot{}, slot.ErrInvalidState
		}
		if _, err := tx.Exec(ctx, `UPDATE slots SET state='COMPLETED',completed_at=$2,version=version+1,updated_at=$2 WHERE id=$1`, slotID, now); err != nil {
			return slot.Slot{}, err
		}
	}
	out, err := getSlotInternalTx(ctx, tx, slotID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}

func (s *SlotStore) claimIdempotency(ctx context.Context, tx pgx.Tx, actorID, key, operation string, requestHash []byte, resourceID string, now time.Time) (bool, string, error) {
	if _, err := tx.Exec(ctx, `DELETE FROM mutation_idempotency WHERE actor_id=$1 AND idempotency_key=$2 AND expires_at<=$3`, actorID, key, now); err != nil {
		return false, "", err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO mutation_idempotency (actor_id,idempotency_key,operation,request_hash,resource_id,created_at,expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (actor_id,idempotency_key) DO NOTHING`,
		actorID, key, operation, requestHash, resourceID, now, now.Add(s.idempotencyTTL),
	)
	if err != nil {
		return false, "", err
	}
	if tag.RowsAffected() == 1 {
		return false, resourceID, nil
	}

	var existingOperation, existingResource string
	var existingHash []byte
	err = tx.QueryRow(ctx, `SELECT operation,request_hash,resource_id FROM mutation_idempotency WHERE actor_id=$1 AND idempotency_key=$2`, actorID, key).
		Scan(&existingOperation, &existingHash, &existingResource)
	if err != nil {
		return false, "", err
	}
	if existingOperation != operation || !bytes.Equal(existingHash, requestHash) {
		return false, "", slot.ErrIdempotencyConflict
	}
	if err := authorizeIdempotencyReplayTx(ctx, tx, actorID, existingResource, existingOperation); err != nil {
		return false, "", err
	}
	return true, existingResource, nil
}

func blockedPairTx(ctx context.Context, tx pgx.Tx, a, b string) (bool, error) {
	var blocked bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_blocks
			WHERE (blocker_id=$1 AND blocked_id=$2)
			   OR (blocker_id=$2 AND blocked_id=$1)
		)`, a, b).Scan(&blocked)
	return blocked, err
}

func isTerminal(state string) bool {
	return state == "COMPLETED" || state == "CANCELLED" || state == "EXPIRED" || state == "MODERATED"
}

const slotColumns = `
	s.id,u.id,u.username,u.display_name,u.avatar_url,
	s.title,s.activity,s.details,s.place_text,s.zone_text,s.start_at,
	s.capacity,s.accepted_count,s.state,s.access_mode,s.visibility,
	s.version,s.created_at,s.updated_at`

const getSlotSQL = `SELECT ` + slotColumns + `,
	CASE
		WHEN s.host_id=$2 THEN 'HOST'
		WHEN EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$2) THEN 'ACCEPTED'
		WHEN EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$2) THEN 'PENDING'
		ELSE 'NONE'
	END
FROM slots s
JOIN app_users u ON u.id=s.host_id
WHERE s.id=$1 AND (
	s.host_id=$2
	OR EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$2)
	OR EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$2)
	OR (
		s.visibility='PUBLIC'
		AND s.state IN ('PUBLISHED','FILLING','FULL')
		AND NOT EXISTS(
			SELECT 1 FROM user_blocks b
			WHERE (b.blocker_id=$2 AND b.blocked_id=s.host_id)
			   OR (b.blocker_id=s.host_id AND b.blocked_id=$2)
		)
	)
)`

const listPulseSQL = `SELECT ` + slotColumns + `,
	CASE
		WHEN s.host_id=$1 THEN 'HOST'
		WHEN EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$1) THEN 'ACCEPTED'
		WHEN EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$1) THEN 'PENDING'
		ELSE 'NONE'
	END
FROM slots s
JOIN app_users u ON u.id=s.host_id
WHERE s.visibility='PUBLIC'
  AND s.state IN ('PUBLISHED','FILLING','FULL')
  AND NOT EXISTS(
	SELECT 1 FROM user_blocks b
	WHERE (b.blocker_id=$1 AND b.blocked_id=s.host_id)
	   OR (b.blocker_id=s.host_id AND b.blocked_id=$1)
  )
ORDER BY s.created_at DESC,s.id
LIMIT $2`

const getSlotInternalSQL = `SELECT ` + slotColumns + `,
	CASE
		WHEN s.host_id=$2 THEN 'HOST'
		WHEN EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$2) THEN 'ACCEPTED'
		WHEN EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$2) THEN 'PENDING'
		ELSE 'NONE'
	END
FROM slots s
JOIN app_users u ON u.id=s.host_id
WHERE s.id=$1`

type scanner interface {
	Scan(dest ...any) error
}

func scanSlot(row scanner) (slot.Slot, error) {
	var out slot.Slot
	var state, access, visibility, viewer string
	err := row.Scan(
		&out.ID, &out.Organizer.ID, &out.Organizer.Username, &out.Organizer.DisplayName, &out.Organizer.AvatarURL,
		&out.Title, &out.Activity, &out.Details, &out.PlaceText, &out.ZoneText, &out.StartAt,
		&out.Capacity, &out.AcceptedCount, &state, &access, &visibility,
		&out.Version, &out.CreatedAt, &out.UpdatedAt, &viewer,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	out.State = slot.State(state)
	out.AccessMode = slot.AccessMode(access)
	out.Visibility = slot.Visibility(visibility)
	out.ViewerState = slot.ViewerState(viewer)
	return out, nil
}

func getSlotInternalTx(ctx context.Context, tx pgx.Tx, slotID, actorID string) (slot.Slot, error) {
	return scanSlot(tx.QueryRow(ctx, getSlotInternalSQL, slotID, actorID))
}
