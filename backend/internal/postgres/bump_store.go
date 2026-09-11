package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/bump"
)

type BumpStore struct {
	pool *pgxpool.Pool
}

func NewBumpStore(pool *pgxpool.Pool) (*BumpStore, error) {
	if pool == nil {
		return nil, errors.New("invalid bump store dependency")
	}
	return &BumpStore{pool: pool}, nil
}

// isEligibleBumpParticipantTx reports whether userID is currently the host
// or an accepted member of slotID, AND the Slot has actually reached ACTIVE
// or COMPLETED — the "this meeting really happened, and you were really in
// it" boundary README §6.9 calls anti-farm. It returns bump.ErrNotFound if
// the Slot itself does not exist; a Slot that exists but is in the wrong
// state or has a caller who is not host/accepted returns (false, nil), which
// callers map to bump.ErrNotEligible.
func isEligibleBumpParticipantTx(ctx context.Context, tx pgx.Tx, slotID, userID string) (bool, error) {
	var hostID, state string
	err := tx.QueryRow(ctx, `SELECT host_id,state FROM slots WHERE id=$1 FOR SHARE`, slotID).Scan(&hostID, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, bump.ErrNotFound
	}
	if err != nil {
		return false, err
	}
	if state != "ACTIVE" && state != "COMPLETED" {
		return false, nil
	}
	if hostID == userID {
		return true, nil
	}
	var member bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM slot_memberships WHERE slot_id=$1 AND user_id=$2)`, slotID, userID).Scan(&member); err != nil {
		return false, err
	}
	return member, nil
}

func (s *BumpStore) IssueChallenge(ctx context.Context, actorID, slotID string, ttl time.Duration) (bump.Challenge, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return bump.Challenge{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	eligible, err := isEligibleBumpParticipantTx(ctx, tx, slotID, actorID)
	if err != nil {
		return bump.Challenge{}, err
	}
	if !eligible {
		return bump.Challenge{}, bump.ErrNotEligible
	}

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	var nonce string
	if err := tx.QueryRow(ctx, `
		INSERT INTO bump_challenges (slot_id,user_id,issued_at,expires_at)
		VALUES ($1,$2,$3,$4)
		RETURNING nonce`, slotID, actorID, now, expiresAt).Scan(&nonce); err != nil {
		return bump.Challenge{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return bump.Challenge{}, err
	}
	return bump.Challenge{Nonce: nonce, SlotID: slotID, ExpiresAt: expiresAt}, nil
}

func (s *BumpStore) Confirm(ctx context.Context, actorID, slotID, counterpartID, nonce string) (bump.ConfirmResult, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return bump.ConfirmResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Consume the nonce first: single-use, must belong to this actor+slot,
	// must be unexpired and not already consumed. FOR UPDATE prevents two
	// concurrent requests from both successfully consuming the same nonce.
	var challengeSlotID, challengeUserID string
	var expiresAt time.Time
	var consumedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT slot_id,user_id,expires_at,consumed_at FROM bump_challenges WHERE nonce=$1::uuid FOR UPDATE`, nonce).
		Scan(&challengeSlotID, &challengeUserID, &expiresAt, &consumedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return bump.ConfirmResult{}, bump.ErrInvalidChallenge
	}
	if err != nil {
		return bump.ConfirmResult{}, err
	}
	now := time.Now().UTC()
	if consumedAt != nil || now.After(expiresAt) || challengeSlotID != slotID || challengeUserID != actorID {
		return bump.ConfirmResult{}, bump.ErrInvalidChallenge
	}
	if _, err := tx.Exec(ctx, `UPDATE bump_challenges SET consumed_at=$2 WHERE nonce=$1::uuid`, nonce, now); err != nil {
		return bump.ConfirmResult{}, err
	}

	// Both sides must be a real host/accepted participant of THIS Slot —
	// the anti-farm boundary: you cannot BUMP someone who was never
	// actually in the Slot with you.
	actorEligible, err := isEligibleBumpParticipantTx(ctx, tx, slotID, actorID)
	if err != nil {
		return bump.ConfirmResult{}, err
	}
	if !actorEligible {
		return bump.ConfirmResult{}, bump.ErrNotEligible
	}
	counterpartEligible, err := isEligibleBumpParticipantTx(ctx, tx, slotID, counterpartID)
	if err != nil {
		return bump.ConfirmResult{}, err
	}
	if !counterpartEligible {
		return bump.ConfirmResult{}, bump.ErrNotEligible
	}

	var blocked bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_blocks
			WHERE (blocker_id=$1 AND blocked_id=$2) OR (blocker_id=$2 AND blocked_id=$1)
		)`, actorID, counterpartID).Scan(&blocked); err != nil {
		return bump.ConfirmResult{}, err
	}
	if blocked {
		return bump.ConfirmResult{}, bump.ErrForbidden
	}

	// Record this directed claim. ON CONFLICT DO NOTHING: a submitter
	// cannot claim the same counterpart twice for the same Slot (README
	// §6.9 "one-event/one-contribution") — a repeat attempt (a fresh nonce
	// after the pair is already mutual) is a harmless no-op, not an error.
	if _, err := tx.Exec(ctx, `
		INSERT INTO bump_submissions (slot_id,submitter_id,counterpart_id,challenge_nonce)
		VALUES ($1,$2,$3,$4::uuid)
		ON CONFLICT (slot_id,submitter_id,counterpart_id) DO NOTHING`,
		slotID, actorID, counterpartID, nonce); err != nil {
		return bump.ConfirmResult{}, err
	}

	var reverseExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM bump_submissions WHERE slot_id=$1 AND submitter_id=$2 AND counterpart_id=$3)`,
		slotID, counterpartID, actorID).Scan(&reverseExists); err != nil {
		return bump.ConfirmResult{}, err
	}

	result := bump.ConfirmResult{}
	if reverseExists {
		userLo, userHi := actorID, counterpartID
		if userHi < userLo {
			userLo, userHi = userHi, userLo
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO bump_confirmations (slot_id,user_lo_id,user_hi_id,confirmed_at)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (slot_id,user_lo_id,user_hi_id) DO NOTHING`,
			slotID, userLo, userHi, now)
		if err != nil {
			return bump.ConfirmResult{}, err
		}
		// RowsAffected()==0 means a confirmation for this pair/Slot already
		// existed (e.g. a concurrent request finished first) — reliability
		// was already credited by whichever call actually inserted the row,
		// so this call must not double-credit it.
		if tag.RowsAffected() == 1 {
			result.Verified = true
			result.ConfirmedAt = now
			for _, pair := range [][2]string{{actorID, counterpartID}, {counterpartID, actorID}} {
				if _, err := tx.Exec(ctx, `
					INSERT INTO reliability_events (user_id,event_type,slot_id,counterpart_id,created_at)
					VALUES ($1,'BUMP_VERIFIED',$2,$3,$4)`, pair[0], slotID, pair[1], now); err != nil {
					return bump.ConfirmResult{}, err
				}
				if err := creditUserReliabilityTx(ctx, tx, pair[0], now); err != nil {
					return bump.ConfirmResult{}, err
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return bump.ConfirmResult{}, err
	}
	return result, nil
}

// creditUserReliabilityTx increments userID's verified BUMP count by one and
// recomputes their band from the new total, creating the summary row on
// first credit.
func creditUserReliabilityTx(ctx context.Context, tx pgx.Tx, userID string, now time.Time) error {
	var count int
	if err := tx.QueryRow(ctx, `
		INSERT INTO user_reliability (user_id,verified_bump_count,band,updated_at)
		VALUES ($1,1,'NEW',$2)
		ON CONFLICT (user_id) DO UPDATE SET
			verified_bump_count = user_reliability.verified_bump_count + 1,
			updated_at = $2
		RETURNING verified_bump_count`, userID, now).Scan(&count); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE user_reliability SET band=$2 WHERE user_id=$1`,
		userID, string(bump.ComputeBand(count))); err != nil {
		return err
	}
	return nil
}

func (s *BumpStore) Reliability(ctx context.Context, userID string) (bump.Reliability, error) {
	var count int
	var band string
	err := s.pool.QueryRow(ctx, `SELECT verified_bump_count,band FROM user_reliability WHERE user_id=$1`, userID).Scan(&count, &band)
	if errors.Is(err, pgx.ErrNoRows) {
		return bump.Reliability{UserID: userID, VerifiedBumpCount: 0, Band: bump.BandNew}, nil
	}
	if err != nil {
		return bump.Reliability{}, err
	}
	return bump.Reliability{UserID: userID, VerifiedBumpCount: count, Band: bump.Band(band)}, nil
}

func (s *BumpStore) Vault(ctx context.Context, userID string, limit int) ([]bump.VaultEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT b.slot_id,s.title,
		       CASE WHEN b.user_lo_id=$1 THEN b.user_hi_id ELSE b.user_lo_id END,
		       b.confirmed_at
		FROM bump_confirmations b
		JOIN slots s ON s.id=b.slot_id
		WHERE b.user_lo_id=$1 OR b.user_hi_id=$1
		ORDER BY b.confirmed_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]bump.VaultEntry, 0)
	for rows.Next() {
		var item bump.VaultEntry
		if err := rows.Scan(&item.SlotID, &item.SlotTitle, &item.CounterpartID, &item.ConfirmedAt); err != nil {
			return nil, err
		}
		item.ConfirmedAt = item.ConfirmedAt.UTC()
		items = append(items, item)
	}
	return items, rows.Err()
}
