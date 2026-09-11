package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
)

// emitSystemChatMessageTx records a server-originated chat notice (README
// §6.7 Chat V2 "system messages for important Slot lifecycle events") in the
// same transaction as the lifecycle mutation that triggered it, so the
// notice can never be observed without the state change it describes (or
// vice versa). It intentionally bypasses the chat domain package: system
// messages are not user input, carry no idempotency key, and are emitted
// exactly once per successful (non-replay) lifecycle transition by
// construction, since a replayed mutation returns the cached result without
// re-running its write path.
//
// created_at deliberately uses clock_timestamp(), not the mutation's own
// passed-in `now`: a single mutation (e.g. a WAITLIST leave that triggers a
// promotion) can emit more than one SYSTEM row in the same transaction, and
// every other timestamp in this codebase intentionally reuses one `now` for
// the whole transaction. now() (and any bound `now` parameter) is frozen to
// transaction start and would give such rows an identical created_at,
// leaving their relative chat order to an incidental id tiebreak instead of
// the order the events actually happened in. clock_timestamp() advances on
// every call, even inside one transaction, so sequential SYSTEM rows sort
// correctly without changing how any other table's timestamps are derived.
//
// The existing slot_messages_outbox_after_insert trigger fires on this
// INSERT exactly as it does for a user message, so the notice is realtime-
// visible through the existing v1.1 outbox/viewer-feed path for free. The
// existing terminal purge trigger also covers it: it deletes every
// slot_messages row for the Slot, not filtered by kind.
func emitSystemChatMessageTx(ctx context.Context, tx pgx.Tx, slotID, systemEventType string, subjectUserID *string) error {
	id, err := identifier.NewUUID()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO slot_messages (id,slot_id,kind,system_event_type,subject_user_id,created_at)
		VALUES ($1,$2,'SYSTEM',$3,$4,clock_timestamp())`,
		id, slotID, systemEventType, subjectUserID)
	return err
}
