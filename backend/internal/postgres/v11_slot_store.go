package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// V11SlotStore extends the stable v1.0 SlotStore with canonical-place support
// without weakening the existing social-state/authorization implementation.
// The embedded store remains the authority for relationship/lifecycle commands;
// v1.1 overrides the reads plus create/edit so canonical_place_id participates
// in the same transaction and optimistic-version contract as the Slot aggregate.
type V11SlotStore struct {
	*SlotStore
	// waitlistRequestTTL bounds how long a WAITLIST queue position stays
	// eligible for promotion. It is a v1.1-only concept: v1.0 APPROVAL
	// requests are untouched and keep their existing behavior.
	waitlistRequestTTL time.Duration
}

func NewV11SlotStore(base *SlotStore, waitlistRequestTTL time.Duration) (*V11SlotStore, error) {
	if base == nil || base.pool == nil || waitlistRequestTTL <= 0 {
		return nil, errors.New("invalid v1.1 slot store dependency")
	}
	return &V11SlotStore{SlotStore: base, waitlistRequestTTL: waitlistRequestTTL}, nil
}

func (s *V11SlotStore) Create(ctx context.Context, actorID string, candidate slot.Slot, key string, requestHash []byte) (slot.Slot, error) {
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
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::uuid,$9,$10,0,$11,$12,$13,1,$14,$14)`,
		candidate.ID, actorID, candidate.Title, candidate.Activity, candidate.Details,
		candidate.PlaceText, candidate.ZoneText, nullableString(candidate.CanonicalPlaceID), candidate.StartAt,
		candidate.Capacity, string(candidate.State), string(candidate.AccessMode), string(candidate.Visibility), candidate.CreatedAt,
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

func (s *V11SlotStore) Get(ctx context.Context, actorID, slotID string) (slot.Slot, error) {
	return scanV11Slot(s.pool.QueryRow(ctx, getV11SlotSQL, slotID, actorID, s.waitlistExpiryCutoff()))
}

func (s *V11SlotStore) ListPulse(ctx context.Context, actorID string, limit int) ([]slot.Slot, error) {
	rows, err := s.pool.Query(ctx, listV11PulseSQL, actorID, limit, s.waitlistExpiryCutoff())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]slot.Slot, 0)
	for rows.Next() {
		item, err := scanV11Slot(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// waitlistExpiryCutoff bounds how far back a WAITLIST slot_requests row can
// still read as the viewer's PENDING relationship. A row older than this is
// display-only stale: mutations (promoteOldestWaitlistTx, waitlistRequest)
// already treat it as expired and lazily purge it, but nothing forced a read
// to agree until now, so a viewer could see themselves as PENDING on a queue
// position that will never promote and that they were already free to
// request again. This only changes what CASE...WHEN computes as the
// viewer's relationship; it never removes a Slot's visibility grant, so a
// viewer who can already see the Slot keeps seeing it, now correctly as NONE
// instead of a stale PENDING. Every other access mode (host/accepted/PUBLIC
// visibility) and the WHERE-clause access grant itself are unaffected.
func (s *V11SlotStore) waitlistExpiryCutoff() time.Time {
	return time.Now().UTC().Add(-s.waitlistRequestTTL)
}

func (s *V11SlotStore) ListMine(ctx context.Context, actorID, view string, limit int) ([]slot.Slot, error) {
	rows, err := s.pool.Query(ctx, listV11MySlotsSQL, actorID, view, limit, s.waitlistExpiryCutoff())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]slot.Slot, 0)
	for rows.Next() {
		item, err := scanV11Slot(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListPending overrides the v1.0 SlotStore.ListPending to add WAITLIST
// queue awareness (README §6.6 "complete roster/request/waitlist states"):
// for a WAITLIST-mode Slot, each pending request also reports its 1-based
// FIFO queue position and whether it has passed the same expiry TTL every
// other WAITLIST-aware read already applies. APPROVAL-mode Slots are
// unaffected — QueuePosition stays nil and Expired stays false, matching
// v1.0's original contract exactly, since APPROVAL has no queue or expiry
// concept for a pending request.
func (s *V11SlotStore) ListPending(ctx context.Context, actorID, slotID string) ([]slot.PendingRequest, error) {
	var hostID, accessMode string
	if err := s.pool.QueryRow(ctx, `SELECT host_id,access_mode FROM slots WHERE id=$1`, slotID).Scan(&hostID, &accessMode); errors.Is(err, pgx.ErrNoRows) {
		return nil, slot.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if hostID != actorID {
		return nil, slot.ErrForbidden
	}

	rows, err := s.pool.Query(ctx, `
		SELECT u.id,u.username,u.display_name,u.avatar_url,r.created_at,
		       ROW_NUMBER() OVER (ORDER BY r.created_at ASC,u.id ASC)
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

	isWaitlist := accessMode == "WAITLIST"
	cutoff := s.waitlistExpiryCutoff()
	out := make([]slot.PendingRequest, 0, slot.MaxPendingRequests)
	for rows.Next() {
		var item slot.PendingRequest
		var position int64
		if err := rows.Scan(&item.User.ID, &item.User.Username, &item.User.DisplayName, &item.User.AvatarURL, &item.RequestedAt, &position); err != nil {
			return nil, err
		}
		if isWaitlist {
			p := int(position)
			item.QueuePosition = &p
			item.Expired = !item.RequestedAt.After(cutoff)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *V11SlotStore) Edit(ctx context.Context, actorID, slotID string, patch slot.EditInput, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
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
		out, err := getV11SlotInternalTx(ctx, tx, resourceID, actorID)
		if err != nil {
			return slot.Slot{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return slot.Slot{}, err
		}
		return out, nil
	}

	var hostID, state, currentAccessMode string
	var version int64
	var acceptedCount, currentCapacity int
	err = tx.QueryRow(ctx, `SELECT host_id,state,version,accepted_count,capacity,access_mode FROM slots WHERE id=$1 FOR UPDATE`, slotID).
		Scan(&hostID, &state, &version, &acceptedCount, &currentCapacity, &currentAccessMode)
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
	if patch.AccessMode != nil && state != "DRAFT" {
		return slot.Slot{}, slot.ErrInvalidState
	}
	if patch.Visibility != nil && state != "DRAFT" {
		return slot.Slot{}, slot.ErrInvalidState
	}
	if err := ensureCanonicalPlaceActiveTx(ctx, tx, patch.CanonicalPlaceID); err != nil {
		return slot.Slot{}, err
	}

	newCapacity := currentCapacity
	if patch.Capacity != nil {
		newCapacity = *patch.Capacity
	}
	if newCapacity < acceptedCount {
		return slot.Slot{}, slot.ErrInvalidInput
	}

	titleSet, title := patch.Title != nil, ""
	if patch.Title != nil {
		title = *patch.Title
	}
	detailsSet, details := patch.Details != nil, ""
	if patch.Details != nil {
		details = *patch.Details
	}
	placeSet, place := patch.PlaceText != nil, ""
	if patch.PlaceText != nil {
		place = *patch.PlaceText
	}
	zoneSet, zone := patch.ZoneText != nil, ""
	if patch.ZoneText != nil {
		zone = *patch.ZoneText
	}
	canonicalSet := patch.CanonicalPlaceID != nil || patch.ClearCanonicalPlaceID
	startSet := patch.StartAt != nil || patch.ClearStartAt
	capacitySet := patch.Capacity != nil
	accessModeSet := patch.AccessMode != nil
	newAccessMode := currentAccessMode
	if patch.AccessMode != nil {
		newAccessMode = string(*patch.AccessMode)
	}
	visibilitySet := patch.Visibility != nil
	newVisibility := ""
	if patch.Visibility != nil {
		newVisibility = string(*patch.Visibility)
	}
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
			canonical_place_id=CASE WHEN $10 THEN $11::uuid ELSE canonical_place_id END,
			start_at=CASE WHEN $12 THEN $13 ELSE start_at END,
			capacity=CASE WHEN $14 THEN $15 ELSE capacity END,
			access_mode=CASE WHEN $16 THEN $17 ELSE access_mode END,
			visibility=CASE WHEN $20 THEN $21 ELSE visibility END,
			state=$18,version=version+1,updated_at=$19
		WHERE id=$1`,
		slotID, titleSet, title, detailsSet, details, placeSet, place, zoneSet, zone,
		canonicalSet, nullableString(patch.CanonicalPlaceID), startSet, patch.StartAt,
		capacitySet, newCapacity, accessModeSet, newAccessMode, newState, now,
		visibilitySet, newVisibility,
	)
	if err != nil {
		return slot.Slot{}, err
	}
	if currentAccessMode == string(slot.AccessWaitlist) && state != "DRAFT" && state != "ACTIVE" && newCapacity > acceptedCount {
		beforePromotion := acceptedCount
		for acceptedCount < newCapacity {
			nextCount, promotedID, promoteErr := promoteOldestWaitlistTx(ctx, tx, slotID, hostID, acceptedCount, newCapacity, now, s.waitlistRequestTTL)
			if promoteErr != nil {
				return slot.Slot{}, promoteErr
			}
			acceptedCount = nextCount
			if promotedID == "" {
				break
			}
		}
		if acceptedCount != beforePromotion {
			promotedState := "FILLING"
			if acceptedCount >= newCapacity {
				promotedState = "FULL"
			}
			if _, err := tx.Exec(ctx, `UPDATE slots SET accepted_count=$2,state=$3 WHERE id=$1`, slotID, acceptedCount, promotedState); err != nil {
				return slot.Slot{}, err
			}
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

func (s *V11SlotStore) Cancel(ctx context.Context, actorID, slotID string, expectedVersion int64, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	if _, err := s.SlotStore.Cancel(ctx, actorID, slotID, expectedVersion, key, requestHash, now); err != nil {
		return slot.Slot{}, err
	}
	return s.getInternal(ctx, actorID, slotID)
}

func (s *V11SlotStore) Approve(ctx context.Context, actorID, slotID, requesterID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	if _, err := s.SlotStore.Approve(ctx, actorID, slotID, requesterID, key, requestHash, now); err != nil {
		return slot.Slot{}, err
	}
	return s.getInternal(ctx, actorID, slotID)
}

func (s *V11SlotStore) Reject(ctx context.Context, actorID, slotID, requesterID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	if _, err := s.SlotStore.Reject(ctx, actorID, slotID, requesterID, key, requestHash, now); err != nil {
		return slot.Slot{}, err
	}
	return s.getInternal(ctx, actorID, slotID)
}

func (s *V11SlotStore) Start(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	if _, err := s.SlotStore.Start(ctx, actorID, slotID, key, requestHash, now); err != nil {
		return slot.Slot{}, err
	}
	return s.getInternal(ctx, actorID, slotID)
}

func (s *V11SlotStore) Complete(ctx context.Context, actorID, slotID, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	if _, err := s.SlotStore.Complete(ctx, actorID, slotID, key, requestHash, now); err != nil {
		return slot.Slot{}, err
	}
	return s.getInternal(ctx, actorID, slotID)
}

func (s *V11SlotStore) RemoveMember(ctx context.Context, actorID, slotID, memberID string, expectedVersion int64, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
	mode, err := s.slotAccessMode(ctx, slotID)
	if err != nil {
		return slot.Slot{}, err
	}
	if mode == string(slot.AccessWaitlist) {
		return s.waitlistRemoveMember(ctx, actorID, slotID, memberID, expectedVersion, key, requestHash, now)
	}
	if _, err := s.SlotStore.RemoveMember(ctx, actorID, slotID, memberID, expectedVersion, key, requestHash, now); err != nil {
		return slot.Slot{}, err
	}
	return s.getInternal(ctx, actorID, slotID)
}

func (s *V11SlotStore) getInternal(ctx context.Context, actorID, slotID string) (slot.Slot, error) {
	return scanV11Slot(s.pool.QueryRow(ctx, getV11SlotInternalSQL, slotID, actorID))
}

func ensureCanonicalPlaceActiveTx(ctx context.Context, tx pgx.Tx, placeID *string) error {
	if placeID == nil {
		return nil
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM canonical_places WHERE id=$1::uuid AND active)`, *placeID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return slot.ErrInvalidInput
	}
	return nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

const v11SlotColumns = `
	s.id,u.id,u.username,u.display_name,u.avatar_url,
	s.title,s.activity,s.details,s.place_text,s.zone_text,s.canonical_place_id::text,s.start_at,
	s.capacity,s.accepted_count,s.state,s.access_mode,s.visibility,
	s.version,s.created_at,s.updated_at`

const getV11SlotSQL = `SELECT ` + v11SlotColumns + `,
	CASE
		WHEN s.host_id=$2 THEN 'HOST'
		WHEN EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$2) THEN 'ACCEPTED'
		WHEN EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$2
			AND (s.access_mode<>'WAITLIST' OR r.created_at>$3)) THEN 'PENDING'
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
	OR (
		s.visibility='LINKS'
		AND s.state IN ('PUBLISHED','FILLING','FULL')
		AND EXISTS(
			SELECT 1 FROM friendships f
			WHERE f.user_lo_id=LEAST(s.host_id,$2) AND f.user_hi_id=GREATEST(s.host_id,$2)
		)
		AND NOT EXISTS(
			SELECT 1 FROM user_blocks b
			WHERE (b.blocker_id=$2 AND b.blocked_id=s.host_id)
			   OR (b.blocker_id=s.host_id AND b.blocked_id=$2)
		)
	)
)`

// listV11PulseSQL's visibility gate is the OR of two branches — s.visibility
// IN ('PUBLIC','LINKS') would be simpler text but cannot express "and, for
// LINKS only, require a friendship" without a CASE, so this stays two
// explicit branches like getV11SlotSQL above it.
const listV11PulseSQL = `SELECT ` + v11SlotColumns + `,
	CASE
		WHEN s.host_id=$1 THEN 'HOST'
		WHEN EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$1) THEN 'ACCEPTED'
		WHEN EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$1
			AND (s.access_mode<>'WAITLIST' OR r.created_at>$3)) THEN 'PENDING'
		ELSE 'NONE'
	END
FROM slots s
JOIN app_users u ON u.id=s.host_id
WHERE (
	s.visibility='PUBLIC'
	OR (
		s.visibility='LINKS'
		AND EXISTS(
			SELECT 1 FROM friendships f
			WHERE f.user_lo_id=LEAST(s.host_id,$1) AND f.user_hi_id=GREATEST(s.host_id,$1)
		)
	)
  )
  AND s.state IN ('PUBLISHED','FILLING','FULL')
  AND NOT EXISTS(
	SELECT 1 FROM user_blocks b
	WHERE (b.blocker_id=$1 AND b.blocked_id=s.host_id)
	   OR (b.blocker_id=s.host_id AND b.blocked_id=$1)
  )
ORDER BY s.created_at DESC,s.id
LIMIT $2`

const getV11SlotInternalSQL = `SELECT ` + v11SlotColumns + `,
	CASE
		WHEN s.host_id=$2 THEN 'HOST'
		WHEN EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$2) THEN 'ACCEPTED'
		WHEN EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$2) THEN 'PENDING'
		ELSE 'NONE'
	END
FROM slots s
JOIN app_users u ON u.id=s.host_id
WHERE s.id=$1`

const listV11MySlotsSQL = `SELECT ` + v11SlotColumns + `,
	CASE WHEN s.host_id=$1 THEN 'HOST'
	     WHEN EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$1) THEN 'ACCEPTED'
	     ELSE 'PENDING' END
FROM slots s JOIN app_users u ON u.id=s.host_id
WHERE s.state IN ('PUBLISHED','FILLING','FULL','ACTIVE')
  AND (
    ($2='HOSTING' AND s.host_id=$1)
    OR ($2='JOINED' AND s.host_id<>$1 AND EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$1))
    OR ($2='REQUESTED' AND s.host_id<>$1 AND s.state<>'ACTIVE'
        AND EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$1
              AND (s.access_mode<>'WAITLIST' OR r.created_at>$4)))
  )
  AND NOT EXISTS(SELECT 1 FROM user_blocks b
    WHERE (b.blocker_id=$1 AND b.blocked_id=s.host_id)
       OR (b.blocker_id=s.host_id AND b.blocked_id=$1))
ORDER BY s.updated_at DESC,s.id
LIMIT $3`

func scanV11Slot(row scanner) (slot.Slot, error) {
	var out slot.Slot
	var state, access, visibility, viewer string
	var canonical pgtype.Text
	err := row.Scan(
		&out.ID, &out.Organizer.ID, &out.Organizer.Username, &out.Organizer.DisplayName, &out.Organizer.AvatarURL,
		&out.Title, &out.Activity, &out.Details, &out.PlaceText, &out.ZoneText, &canonical, &out.StartAt,
		&out.Capacity, &out.AcceptedCount, &state, &access, &visibility,
		&out.Version, &out.CreatedAt, &out.UpdatedAt, &viewer,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return slot.Slot{}, slot.ErrNotFound
	}
	if err != nil {
		return slot.Slot{}, err
	}
	if canonical.Valid {
		value := canonical.String
		out.CanonicalPlaceID = &value
	}
	out.State = slot.State(state)
	out.AccessMode = slot.AccessMode(access)
	out.Visibility = slot.Visibility(visibility)
	out.ViewerState = slot.ViewerState(viewer)
	return out, nil
}

func getV11SlotInternalTx(ctx context.Context, tx pgx.Tx, slotID, actorID string) (slot.Slot, error) {
	return scanV11Slot(tx.QueryRow(ctx, getV11SlotInternalSQL, slotID, actorID))
}
