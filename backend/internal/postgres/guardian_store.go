package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/guardian"
)

type GuardianStore struct {
	pool *pgxpool.Pool
}

func NewGuardianStore(pool *pgxpool.Pool) *GuardianStore { return &GuardianStore{pool: pool} }

func (s *GuardianStore) CreateLink(ctx context.Context, id, slotID, actorID string, mode guardian.Mode, tokenHash []byte, now, expiresAt time.Time) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var hostID string
	if err := tx.QueryRow(ctx, `SELECT host_id FROM slots WHERE id=$1 FOR SHARE`, slotID).Scan(&hostID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return guardian.ErrForbidden
		}
		return err
	}
	if hostID != actorID {
		var member bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM slot_memberships WHERE slot_id=$1 AND user_id=$2)`, slotID, actorID).Scan(&member); err != nil {
			return err
		}
		if !member {
			return guardian.ErrForbidden
		}
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO guardian_links (id, slot_id, created_by, mode, token_hash, created_at, expires_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)`, id, slotID, actorID, string(mode), tokenHash, now, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *GuardianStore) Revoke(ctx context.Context, linkID, actorID string, now time.Time) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE guardian_links SET revoked_at=$3
WHERE id=$1 AND created_by=$2 AND revoked_at IS NULL`, linkID, actorID, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return guardian.ErrForbidden
	}
	return nil
}

func (s *GuardianStore) Access(ctx context.Context, tokenHash []byte, now time.Time) (guardian.Status, error) {
	var displayName, slotState string
	err := s.pool.QueryRow(ctx, `
SELECT u.display_name, s.state
FROM guardian_links g
JOIN app_users u ON u.id=g.created_by
JOIN slots s ON s.id=g.slot_id
WHERE g.token_hash=$1 AND g.revoked_at IS NULL AND g.expires_at>$2`, tokenHash, now).Scan(&displayName, &slotState)
	if errors.Is(err, pgx.ErrNoRows) {
		return guardian.Status{}, guardian.ErrLinkUnavailable
	}
	if err != nil {
		return guardian.Status{}, err
	}
	status := guardian.SlotStatusEnded
	switch slotState {
	case "PUBLISHED", "FILLING", "FULL", "ACTIVE":
		status = guardian.SlotStatusActive
	}
	return guardian.Status{SharerDisplayName: displayName, SlotStatus: status}, nil
}
