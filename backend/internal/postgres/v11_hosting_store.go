package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func (s *V11SlotStore) CreateDraft(
	ctx context.Context,
	actorID string,
	candidate slot.Slot,
	key string,
	requestHash []byte,
) (slot.Slot, error) {
	if candidate.State != slot.StateDraft || candidate.AcceptedCount != 0 {
		return slot.Slot{}, slot.ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	replay, resourceID, err := s.claimIdempotency(
		ctx, tx, actorID, key, "slot.create_draft", requestHash, candidate.ID, candidate.CreatedAt,
	)
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
	if err := ensureCanonicalPlaceActiveTx(ctx, tx, candidate.CanonicalPlaceID); err != nil {
		return slot.Slot{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO slots (
			id,host_id,title,activity,details,place_text,zone_text,canonical_place_id,start_at,
			capacity,accepted_count,state,access_mode,visibility,version,created_at,updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::uuid,$9,$10,0,'DRAFT',$11,$12,1,$13,$13)`,
		candidate.ID, actorID, candidate.Title, candidate.Activity, candidate.Details,
		candidate.PlaceText, candidate.ZoneText, nullableString(candidate.CanonicalPlaceID), candidate.StartAt,
		candidate.Capacity, string(candidate.AccessMode), string(candidate.Visibility), candidate.CreatedAt,
	)
	if err != nil {
		return slot.Slot{}, err
	}
	out, err := getV11SlotInternalTx(ctx, tx, candidate.ID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}

func (s *V11SlotStore) PublishDraft(
	ctx context.Context,
	actorID, slotID string,
	expectedVersion int64,
	key string,
	requestHash []byte,
	now time.Time,
) (slot.Slot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return slot.Slot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	replay, resourceID, err := s.claimIdempotency(
		ctx, tx, actorID, key, "slot.publish_draft", requestHash, slotID, now,
	)
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
	var version int64
	var acceptedCount, capacity int
	var startAt pgtype.Timestamptz
	err = tx.QueryRow(ctx, `
		SELECT host_id,state,version,accepted_count,capacity,access_mode,start_at
		FROM slots WHERE id=$1 FOR UPDATE`, slotID).Scan(
		&hostID, &state, &version, &acceptedCount, &capacity, &accessMode, &startAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	if hostID != actorID {
		return slot.Slot{}, slot.ErrForbidden
	}
	if state != "DRAFT" {
		return slot.Slot{}, slot.ErrInvalidState
	}
	if version != expectedVersion {
		return slot.Slot{}, slot.ErrConflict
	}
	if acceptedCount != 0 || capacity < 2 {
		return slot.Slot{}, slot.ErrInvalidState
	}
	// Instant and Waitlist require their own concurrency semantics before they
	// can be published. Approval is the only active publish authority for now.
	if accessMode != string(slot.AccessApproval) {
		return slot.Slot{}, slot.ErrInvalidState
	}
	if startAt.Valid && startAt.Time.Before(now.Add(-30*time.Second)) {
		return slot.Slot{}, slot.ErrInvalidInput
	}

	if _, err := tx.Exec(ctx, `
		UPDATE slots
		SET state='FILLING',version=version+1,updated_at=$2
		WHERE id=$1`, slotID, now.UTC()); err != nil {
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
