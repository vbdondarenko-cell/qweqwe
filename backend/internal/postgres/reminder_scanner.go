package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReminderScanner implements the time-triggered half of README §6.8's
// EVENT_REMINDER type (see migration 000030's doc comment for why this is
// a separate scanner rather than a DB trigger): it finds Slots whose
// start_at has just come within leadTime of "now" and have not yet had a
// reminder queued, and emits one genuine domain_outbox_events row per
// recipient (host + every accepted member) via the same
// linkup_enqueue_outbox function every trigger-driven event already uses,
// so NotificationProjector needs no special-casing to consume them beyond
// recognizing the new event type.
type ReminderScanner struct {
	pool     *pgxpool.Pool
	leadTime time.Duration
}

func NewReminderScanner(pool *pgxpool.Pool, leadTime time.Duration) (*ReminderScanner, error) {
	if pool == nil || leadTime <= 0 {
		return nil, errors.New("invalid reminder scanner dependencies")
	}
	return &ReminderScanner{pool: pool, leadTime: leadTime}, nil
}

// ScanAndEmit finds up to limit Slots newly due for a reminder and emits
// their events. It returns the number of Slots claimed (not the number of
// per-recipient events emitted, which can be larger). A Slot whose
// start_at already passed before this ever ran (e.g. the service was down)
// is intentionally never reminded late: `start_at > now()` excludes it —
// this is a "starting soon" notice, not a historical record, so silently
// not sending a late one is correct, not a missed case.
func (s *ReminderScanner) ScanAndEmit(ctx context.Context, limit int) (int, error) {
	if limit < 1 {
		return 0, errors.New("invalid reminder scan limit")
	}
	now := time.Now().UTC()
	rows, err := s.pool.Query(ctx, `
		SELECT id::text FROM slots
		WHERE start_at IS NOT NULL
		  AND start_at > $1
		  AND start_at <= $2
		  AND state IN ('PUBLISHED','FILLING','FULL')
		  AND NOT EXISTS (SELECT 1 FROM slot_reminder_emissions WHERE slot_id=slots.id)
		ORDER BY start_at
		LIMIT $3`, now, now.Add(s.leadTime), limit)
	if err != nil {
		return 0, err
	}
	var slotIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		slotIDs = append(slotIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	claimed := 0
	for _, slotID := range slotIDs {
		ok, err := s.emitForSlot(ctx, slotID)
		if err != nil {
			return claimed, err
		}
		if ok {
			claimed++
		}
	}
	return claimed, nil
}

// emitForSlot claims slotID's reminder (idempotent: returns false, nil if
// another scan already claimed it — normal under a multi-replica
// deployment racing the same eligible Slot, not an error) and, only if it
// won the claim, emits one slot.starting_soon event per current recipient
// (host plus every accepted member) inside the same transaction as the
// claim, so a crash between claiming and emitting cannot happen: either
// both commit or neither does, and a retried scan would find the Slot
// still eligible (the claim row rolled back with everything else).
func (s *ReminderScanner) emitForSlot(ctx context.Context, slotID string) (bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `INSERT INTO slot_reminder_emissions (slot_id) VALUES ($1::uuid) ON CONFLICT DO NOTHING`, slotID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}

	var hostID string
	if err := tx.QueryRow(ctx, `SELECT host_id::text FROM slots WHERE id=$1::uuid`, slotID).Scan(&hostID); err != nil {
		return false, err
	}
	// seen de-duplicates recipients: a host is never expected to also hold
	// a slot_memberships row for their own Slot (a host cannot
	// request/join their own Slot — see the Approval/WAITLIST/INSTANT
	// invariants), but emitting two separate outbox events for the same
	// recipient would each get its own event_id and bypass
	// notification_deliveries' one-row-per-source_event_id dedupe boundary
	// entirely, so this is enforced defensively rather than assumed.
	seen := map[string]bool{hostID: true}
	recipients := []string{hostID}
	memberRows, err := tx.Query(ctx, `SELECT user_id::text FROM slot_memberships WHERE slot_id=$1::uuid`, slotID)
	if err != nil {
		return false, err
	}
	for memberRows.Next() {
		var userID string
		if err := memberRows.Scan(&userID); err != nil {
			memberRows.Close()
			return false, err
		}
		if !seen[userID] {
			seen[userID] = true
			recipients = append(recipients, userID)
		}
	}
	if err := memberRows.Err(); err != nil {
		memberRows.Close()
		return false, err
	}
	memberRows.Close()

	for _, recipientID := range recipients {
		if _, err := tx.Exec(ctx, `
			SELECT linkup_enqueue_outbox('slot.starting_soon','slot',$1::uuid,$2::uuid,$1::uuid,'{}'::jsonb)`,
			slotID, recipientID); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
