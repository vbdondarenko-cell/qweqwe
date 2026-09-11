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
		if err := attachSelectedUserIDsTx(ctx, tx, &out); err != nil {
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
	if len(candidate.SelectedUserIDs) > 0 {
		rows := make([][]any, len(candidate.SelectedUserIDs))
		for i, userID := range candidate.SelectedUserIDs {
			rows[i] = []any{candidate.ID, userID}
		}
		if _, err := tx.CopyFrom(ctx,
			pgx.Identifier{"slot_selected_viewers"},
			[]string{"slot_id", "user_id"},
			pgx.CopyFromRows(rows),
		); err != nil {
			return slot.Slot{}, err
		}
	}
	out, err := getV11SlotInternalTx(ctx, tx, candidate.ID, actorID)
	if err != nil {
		return slot.Slot{}, err
	}
	if err := attachSelectedUserIDsTx(ctx, tx, &out); err != nil {
		return slot.Slot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return slot.Slot{}, err
	}
	return out, nil
}

// rowQuerier is the common subset of *pgxpool.Pool and pgx.Tx this file
// needs, so the same attach helper works whether or not a transaction is
// already open.
type rowQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// attachSelectedUserIDsTx populates out.SelectedUserIDs from the persisted
// slot_selected_viewers table (never from a possibly-stale in-memory
// candidate) whenever out is a VisibilitySelected Slot; a no-op for every
// other visibility. Callers decide who is allowed to see the populated
// list (see attachSelectedUserIDsForHost for the read-path, host-only
// gate); this function itself applies no viewer restriction.
func attachSelectedUserIDsTx(ctx context.Context, q rowQuerier, out *slot.Slot) error {
	if out.Visibility != slot.VisibilitySelected {
		return nil
	}
	rows, err := q.Query(ctx, `SELECT user_id::text FROM slot_selected_viewers WHERE slot_id=$1 ORDER BY user_id`, out.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	out.SelectedUserIDs = ids
	return nil
}

// attachSelectedUserIDsForHost is the read-path (Get/ListPulse/ListMine)
// counterpart of attachSelectedUserIDsTx: it only populates
// out.SelectedUserIDs when the requesting actor is the Slot's own host.
// The allow-list is the host's private curation of who they picked; a
// selected viewer (or anyone else who can otherwise see/reach the Slot)
// still only ever gets the create-time echo documented on
// Slot.SelectedUserIDs, never the full roster of who else was picked, on a
// later read. This closes README §4.3's "not re-populated on every read"
// gap (worklog §59/§60) without introducing a new information leak.
func attachSelectedUserIDsForHost(ctx context.Context, q rowQuerier, out *slot.Slot, actorID string) error {
	if out.Organizer.ID != actorID {
		return nil
	}
	return attachSelectedUserIDsTx(ctx, q, out)
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
	// All canonical access modes now have server-side admission semantics.
	// WAITLIST publication is additionally capability-gated at the HTTP boundary.
	if accessMode != string(slot.AccessApproval) && accessMode != string(slot.AccessInstant) && accessMode != string(slot.AccessWaitlist) {
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
