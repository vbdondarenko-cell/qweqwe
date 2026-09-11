package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
)

type RealtimeOutboxStore struct {
	pool *pgxpool.Pool
}

func NewRealtimeOutboxStore(pool *pgxpool.Pool) (*RealtimeOutboxStore, error) {
	if pool == nil {
		return nil, realtime.ErrInvalidInput
	}
	return &RealtimeOutboxStore{pool: pool}, nil
}

func (s *RealtimeOutboxStore) ListAfter(ctx context.Context, after int64, limit int) ([]realtime.Event, error) {
	if after < 0 || limit < 1 || limit > realtime.MaxBatch {
		return nil, realtime.ErrInvalidInput
	}
	rows, err := s.pool.Query(ctx, `
		SELECT sequence,event_id,event_type,aggregate_type,aggregate_id,
		       subject_user_id,slot_id,payload,occurred_at
		FROM domain_outbox_events
		WHERE sequence > $1
		ORDER BY sequence ASC
		LIMIT $2`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]realtime.Event, 0, limit)
	for rows.Next() {
		var event realtime.Event
		if err := rows.Scan(
			&event.Sequence, &event.ID, &event.Type, &event.AggregateType, &event.AggregateID,
			&event.SubjectUserID, &event.SlotID, &event.Payload, &event.OccurredAt,
		); err != nil {
			return nil, err
		}
		if !event.Valid() {
			return nil, errors.New("invalid canonical outbox event")
		}
		items = append(items, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *RealtimeOutboxStore) Cursor(ctx context.Context, connector string) (int64, error) {
	connector = strings.TrimSpace(connector)
	if !validConnectorName(connector) {
		return 0, realtime.ErrInvalidInput
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO connector_cursors (connector,last_sequence)
		VALUES ($1,0)
		ON CONFLICT (connector) DO NOTHING`, connector); err != nil {
		return 0, err
	}
	var sequence int64
	if err := s.pool.QueryRow(ctx, `SELECT last_sequence FROM connector_cursors WHERE connector=$1`, connector).Scan(&sequence); err != nil {
		return 0, err
	}
	return sequence, nil
}

// Checkpoint records the successful/explicitly-skipped outcome and advances the
// durable cursor in one DB transaction. External side effects must happen before
// this call and use Event.ID as their dedupe/idempotency key.
func (s *RealtimeOutboxStore) Checkpoint(ctx context.Context, connector string, event realtime.Event, outcome realtime.DeliveryOutcome) error {
	connector = strings.TrimSpace(connector)
	if !validConnectorName(connector) || !event.Valid() || !outcome.Valid() {
		return realtime.ErrInvalidInput
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO connector_cursors (connector,last_sequence)
		VALUES ($1,0)
		ON CONFLICT (connector) DO NOTHING`, connector); err != nil {
		return err
	}

	var current int64
	if err := tx.QueryRow(ctx, `
		SELECT last_sequence FROM connector_cursors WHERE connector=$1 FOR UPDATE`, connector).Scan(&current); err != nil {
		return err
	}
	if event.Sequence < current {
		return realtime.ErrCursorOutOfOrder
	}
	if event.Sequence == current {
		if err := receiptMatches(ctx, tx, connector, event.ID, outcome); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	// event.Sequence > current here. Advancing straight to it (rather than
	// requiring event.Sequence == current+1) is deliberate: `sequence` is a
	// Postgres IDENTITY column, which is not transactional — a rolled-back
	// transaction anywhere in the system that happened to touch an
	// outbox-triggering table permanently burns the sequence value it
	// claimed, leaving a numeric gap with no row ever occupying it. Such a
	// gap is not a missed event (nothing was ever there to miss) and must
	// not permanently wedge a connector's cursor. What must still be
	// rejected is skipping over a sequence number that IS a real,
	// unprocessed row: that would silently drop an actual event, which is
	// exactly what this connector-cursor mechanism exists to prevent.
	var realGapExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM domain_outbox_events WHERE sequence>$1 AND sequence<$2)`,
		current, event.Sequence).Scan(&realGapExists); err != nil {
		return err
	}
	if realGapExists {
		return realtime.ErrCursorOutOfOrder
	}

	var canonicalID string
	if err := tx.QueryRow(ctx, `SELECT event_id FROM domain_outbox_events WHERE sequence=$1`, event.Sequence).Scan(&canonicalID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return realtime.ErrInvalidInput
		}
		return err
	}
	if canonicalID != event.ID {
		return realtime.ErrInvalidInput
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO connector_delivery_receipts (connector,event_id,outcome)
		VALUES ($1,$2,$3)
		ON CONFLICT (connector,event_id) DO NOTHING`, connector, event.ID, string(outcome))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if err := receiptMatches(ctx, tx, connector, event.ID, outcome); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE connector_cursors
		SET last_sequence=$2,updated_at=now()
		WHERE connector=$1`, connector, event.Sequence); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func receiptMatches(ctx context.Context, tx pgx.Tx, connector, eventID string, outcome realtime.DeliveryOutcome) error {
	var stored string
	err := tx.QueryRow(ctx, `
		SELECT outcome FROM connector_delivery_receipts
		WHERE connector=$1 AND event_id=$2`, connector, eventID).Scan(&stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return realtime.ErrCursorOutOfOrder
	}
	if err != nil {
		return err
	}
	if stored != string(outcome) {
		return realtime.ErrReceiptConflict
	}
	return nil
}

func validConnectorName(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for i, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (i > 0 && (r == '.' || r == '_' || r == '-')) {
			continue
		}
		return false
	}
	return true
}

// EqualPayload is intentionally strict: connector code can use it before a
// side effect when payload identity matters without normalizing JSON objects.
func EqualPayload(a, b json.RawMessage) bool {
	return bytes.Equal(a, b)
}
