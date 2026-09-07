package postgres

import (
    "context"
    "errors"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func (s *SlotStore) RemoveMember(ctx context.Context, actorID, slotID, memberID string, expectedVersion int64, key string, requestHash []byte, now time.Time) (slot.Slot, error) {
    tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
    if err != nil { return slot.Slot{}, err }
    defer func() { _ = tx.Rollback(ctx) }()
    replay, _, err := s.claimIdempotency(ctx, tx, actorID, key, "slot.remove_member", requestHash, slotID, now)
    if err != nil { return slot.Slot{}, err }

    if err := removeMemberTx(ctx, tx, actorID, slotID, memberID, expectedVersion, replay, now); err != nil { return slot.Slot{}, err }
    out, err := getSlotInternalTx(ctx, tx, slotID, actorID)
    if err != nil { return slot.Slot{}, err }
    if err := tx.Commit(ctx); err != nil { return slot.Slot{}, err }
    return out, nil
}

func removeMemberTx(ctx context.Context, tx pgx.Tx, actorID, slotID, memberID string, expectedVersion int64, replay bool, now time.Time) error {
    var hostID, state string
    var count int
    var version int64
    err := tx.QueryRow(ctx, `SELECT host_id,state,accepted_count,version FROM slots WHERE id=$1 FOR UPDATE`, slotID).Scan(&hostID, &state, &count, &version)
    if errors.Is(err, pgx.ErrNoRows) { return slot.ErrNotFound }
    if err != nil { return err }
    // Replays recheck the current host before returning any Slot data.
    if hostID != actorID || memberID == hostID { return slot.ErrForbidden }
    blocked, err := blockedPairTx(ctx, tx, actorID, memberID)
    if err != nil { return err }
    if blocked { return slot.ErrForbidden }
    if !replay {
        if state != "PUBLISHED" && state != "FILLING" && state != "FULL" && state != "ACTIVE" { return slot.ErrInvalidState }
        if version != expectedVersion { return slot.ErrConflict }
        tag, err := tx.Exec(ctx, `DELETE FROM slot_memberships WHERE slot_id=$1 AND user_id=$2`, slotID, memberID)
        if err != nil { return err }
        if tag.RowsAffected() == 0 { return slot.ErrNotFound }
        if count <= 0 { return errors.New("slot accepted_count invariant violated") }
        if state == "FULL" { state = "FILLING" }
        if _, err := tx.Exec(ctx, `UPDATE slots SET accepted_count=accepted_count-1,state=$2,version=version+1,updated_at=$3 WHERE id=$1`, slotID, state, now); err != nil { return err }
    }
    return nil
}
