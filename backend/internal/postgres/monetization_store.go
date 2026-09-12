package postgres

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/monetization"
)

type MonetizationStore struct {
	pool *pgxpool.Pool
}

func NewMonetizationStore(pool *pgxpool.Pool) *MonetizationStore {
	return &MonetizationStore{pool: pool}
}

func (s *MonetizationStore) Status(ctx context.Context, userID string) (monetization.StoreStatus, error) {
	var (
		premiumUntil pgtype.Timestamptz
		lastVideo    pgtype.Timestamptz
		lastClaim    pgtype.Timestamptz
		boundCode    pgtype.Text
		deadline     pgtype.Timestamptz
		videos       int
		qualified    int
	)
	const query = `
SELECT
    (SELECT max(pg.ends_at) FROM premium_grants pg WHERE pg.user_id = $1),
    COALESCE((SELECT rp.videos_watched_count FROM monetization_rewarded_progress rp WHERE rp.user_id = $1), 0),
    (SELECT rp.last_video_watched_at FROM monetization_rewarded_progress rp WHERE rp.user_id = $1),
    (SELECT rp.last_free_premium_claimed_at FROM monetization_rewarded_progress rp WHERE rp.user_id = $1),
    (SELECT count(*)::int FROM monetization_referrals mr WHERE mr.inviter_id = $1 AND mr.qualified_at IS NOT NULL),
    (SELECT mr.referral_code FROM monetization_referrals mr WHERE mr.invitee_id = $1),
    (SELECT mr.qualifying_deadline FROM monetization_referrals mr WHERE mr.invitee_id = $1)`
	if err := s.pool.QueryRow(ctx, query, userID).Scan(
		&premiumUntil,
		&videos,
		&lastVideo,
		&lastClaim,
		&qualified,
		&boundCode,
		&deadline,
	); err != nil {
		return monetization.StoreStatus{}, err
	}
	status := monetization.StoreStatus{VideosWatchedCount: videos, QualifiedReferrals: qualified}
	if premiumUntil.Valid {
		t := premiumUntil.Time
		status.PremiumUntil = &t
	}
	if lastVideo.Valid {
		t := lastVideo.Time
		status.LastVideoWatchedAt = &t
	}
	if lastClaim.Valid {
		t := lastClaim.Time
		status.LastFreePremiumClaimedAt = &t
	}
	if boundCode.Valid {
		value := boundCode.String
		status.BoundReferralCode = &value
	}
	if deadline.Valid {
		t := deadline.Time
		status.ReferralQualifyingDeadline = &t
	}
	return status, nil
}

func (s *MonetizationStore) EnsureReferralCode(ctx context.Context, userID, code string) (string, error) {
	if _, err := s.pool.Exec(ctx, `
INSERT INTO monetization_referral_codes (user_id, code)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`, userID, code); err != nil {
		return "", err
	}
	var stored string
	if err := s.pool.QueryRow(ctx, `
SELECT code
FROM monetization_referral_codes
WHERE user_id = $1`, userID).Scan(&stored); err != nil {
		return "", err
	}
	return stored, nil
}

func (s *MonetizationStore) BindReferral(ctx context.Context, inviteeID, code string, deadlineDays int) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var createdAt, dbNow time.Time
	if err := tx.QueryRow(ctx, `
SELECT created_at, now()
FROM app_users
WHERE id = $1
FOR UPDATE`, inviteeID).Scan(&createdAt, &dbNow); err != nil {
		return err
	}

	var existing string
	err = tx.QueryRow(ctx, `
SELECT referral_code
FROM monetization_referrals
WHERE invitee_id = $1`, inviteeID).Scan(&existing)
	if err == nil {
		if existing == code {
			return nil
		}
		return monetization.ErrReferralAlreadyBound
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	var inviterID string
	if err := tx.QueryRow(ctx, `
SELECT user_id::text
FROM monetization_referral_codes
WHERE code = $1`, code).Scan(&inviterID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return monetization.ErrInvalidReferralCode
		}
		return err
	}
	if inviterID == inviteeID {
		return monetization.ErrSelfReferral
	}

	deadline := createdAt.Add(time.Duration(deadlineDays) * 24 * time.Hour)
	if !dbNow.Before(deadline) {
		return monetization.ErrReferralExpired
	}

	command, err := tx.Exec(ctx, `
INSERT INTO monetization_referrals (
    invitee_id,
    inviter_id,
    referral_code,
    accepted_at,
    qualifying_deadline
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (invitee_id) DO NOTHING`, inviteeID, inviterID, code, dbNow, deadline)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return monetization.ErrReferralAlreadyBound
	}
	return tx.Commit(ctx)
}

// RecordRewardedView implements monetization.Store's exact contract: see
// that interface's doc comment for the quest/idempotency/cooldown rules
// enforced here.
func (s *MonetizationStore) RecordRewardedView(ctx context.Context, userID, provider string, receiptHash []byte, now time.Time, minInterval, questWindow, grantDuration, cooldown time.Duration) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	receiptID, err := identifier.NewUUID()
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
INSERT INTO monetization_rewarded_receipts (id, user_id, provider, receipt_hash, watched_at)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (provider, receipt_hash) DO NOTHING`, receiptID, userID, provider, receiptHash, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Already-processed verified view (duplicate callback/replay) --
		// docs/LINKUP_PLUS_MONETIZATION.md §6's own required protection.
		// Nothing else to do; the quest must not be advanced twice for one
		// real ad watch.
		return tx.Commit(ctx)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO monetization_rewarded_progress (user_id) VALUES ($1)
ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return err
	}

	var count int
	var questStarted, lastVideo, lastClaim pgtype.Timestamptz
	if err := tx.QueryRow(ctx, `
SELECT videos_watched_count, quest_started_at, last_video_watched_at, last_free_premium_claimed_at
FROM monetization_rewarded_progress
WHERE user_id = $1
FOR UPDATE`, userID).Scan(&count, &questStarted, &lastVideo, &lastClaim); err != nil {
		return err
	}

	if lastClaim.Valid && now.Before(lastClaim.Time.Add(cooldown)) {
		return monetization.ErrRewardedCooldownActive
	}

	// A quest "continues" only if there is prior progress AND its own
	// 24h window (from the FIRST view, not the most recent one) hasn't
	// elapsed. Otherwise this view starts a brand new quest -- it still
	// counts, it just resets the window rather than being discarded.
	continuingQuest := count > 0 && questStarted.Valid && now.Before(questStarted.Time.Add(questWindow))
	if continuingQuest && (!lastVideo.Valid || now.Before(lastVideo.Time.Add(minInterval))) {
		return monetization.ErrRewardedTooSoon
	}

	newCount := 1
	newQuestStarted := now
	if continuingQuest {
		newCount = count + 1
		newQuestStarted = questStarted.Time
	}

	if newCount >= monetization.RewardedVideosRequired {
		grantID, err := identifier.NewUUID()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO premium_grants (id, user_id, source, source_key, starts_at, ends_at)
VALUES ($1,$2,'REWARDED',$3,$4,$5)`, grantID, userID, grantID, now, now.Add(grantDuration)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
UPDATE monetization_rewarded_progress
SET videos_watched_count = 0, quest_started_at = NULL, last_video_watched_at = $2,
    last_free_premium_claimed_at = $2, updated_at = $2
WHERE user_id = $1`, userID, now); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}

	if _, err := tx.Exec(ctx, `
UPDATE monetization_rewarded_progress
SET videos_watched_count = $2, quest_started_at = $3, last_video_watched_at = $4, updated_at = $4
WHERE user_id = $1`, userID, newCount, newQuestStarted, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RecordPurchase implements monetization.Store's exact contract: see that
// interface's doc comment for the entitlement/idempotency/referral rules
// enforced here.
func (s *MonetizationStore) RecordPurchase(ctx context.Context, userID, provider, productID string, purchaseHash []byte, periodStart, periodEnd time.Time, state string, now time.Time, milestones []monetization.ReferralMilestone) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	receiptID, err := identifier.NewUUID()
	if err != nil {
		return err
	}
	var storedReceiptID string
	if err := tx.QueryRow(ctx, `
INSERT INTO monetization_subscription_receipts (id, user_id, provider, product_id, purchase_token_hash, period_start, period_end, state, verified_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (provider, purchase_token_hash) DO UPDATE SET
    product_id = EXCLUDED.product_id,
    period_start = EXCLUDED.period_start,
    period_end = EXCLUDED.period_end,
    state = EXCLUDED.state,
    verified_at = EXCLUDED.verified_at
RETURNING id`, receiptID, userID, provider, productID, purchaseHash, periodStart, periodEnd, state, now,
	).Scan(&storedReceiptID); err != nil {
		return err
	}

	// GRACE still honors entitlement (payment retry in progress); every
	// other non-ACTIVE state does not -- see the migration adding these
	// states for why that split matches standard store semantics.
	sourceKey := hex.EncodeToString(purchaseHash)
	if state == "ACTIVE" || state == "GRACE" {
		grantID, err := identifier.NewUUID()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO premium_grants (id, user_id, source, source_key, starts_at, ends_at)
VALUES ($1,$2,'PAID',$3,$4,$5)
ON CONFLICT (source, source_key) DO UPDATE SET
    starts_at = LEAST(premium_grants.starts_at, EXCLUDED.starts_at),
    ends_at = EXCLUDED.ends_at
WHERE premium_grants.ends_at <> EXCLUDED.ends_at`, grantID, userID, sourceKey, periodStart, periodEnd); err != nil {
			return err
		}
	} else {
		// EXPIRED/BILLING_RETRY/REVOKED/REFUNDED retract this purchase's
		// own grant. The audit trail stays in monetization_subscription_receipts
		// (just updated above, never deleted) -- premium_grants only ever
		// reflects currently-effective entitlement.
		if _, err := tx.Exec(ctx, `DELETE FROM premium_grants WHERE source='PAID' AND source_key=$1`, sourceKey); err != nil {
			return err
		}
	}

	if state == "ACTIVE" {
		if err := qualifyReferralAndAwardMilestones(ctx, tx, userID, storedReceiptID, now, milestones); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// qualifyReferralAndAwardMilestones implements docs/LINKUP_PLUS_MONETIZATION.md
// §7: a referral qualifies only on the invitee's own server-verified paid
// purchase, before its deadline, and only once. Newly-crossed inviter
// milestones are awarded exactly once each (the milestone table's own
// (inviter_id, milestone) primary key is the idempotency guard against a
// concurrent qualification for a different invitee of the same inviter
// racing this one -- whichever transaction's INSERT commits first is
// credited as the triggering invitee for that milestone; both still
// qualify individually, so correctness never depends on which one wins).
func qualifyReferralAndAwardMilestones(ctx context.Context, tx pgx.Tx, inviteeID, receiptID string, now time.Time, milestones []monetization.ReferralMilestone) error {
	var inviterID string
	err := tx.QueryRow(ctx, `
UPDATE monetization_referrals
SET qualified_at = $2, qualifying_receipt_id = $3
WHERE invitee_id = $1 AND qualified_at IS NULL AND qualifying_deadline > $2
RETURNING inviter_id`, inviteeID, now, receiptID).Scan(&inviterID)
	if errors.Is(err, pgx.ErrNoRows) {
		// No pending, unexpired, unqualified referral for this invitee --
		// nothing to qualify (including the common case of no referral
		// bound at all, or one already qualified by an earlier purchase).
		return nil
	}
	if err != nil {
		return err
	}

	var qualifiedCount int
	if err := tx.QueryRow(ctx, `
SELECT count(*) FROM monetization_referrals WHERE inviter_id=$1 AND qualified_at IS NOT NULL`, inviterID).Scan(&qualifiedCount); err != nil {
		return err
	}

	for _, m := range milestones {
		if qualifiedCount < m.QualifiedReferrals {
			continue
		}
		tag, err := tx.Exec(ctx, `
INSERT INTO monetization_referral_milestone_awards (inviter_id, milestone, triggering_invitee_id, inviter_reward_days, invitee_reward_days)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (inviter_id, milestone) DO NOTHING`, inviterID, m.QualifiedReferrals, inviteeID, m.InviterRewardDays, m.InviteeRewardDays)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			continue // already awarded by an earlier qualification
		}
		inviterGrantID, err := identifier.NewUUID()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO premium_grants (id, user_id, source, source_key, starts_at, ends_at)
VALUES ($1,$2,'REFERRAL',$3,$4,$5)`, inviterGrantID, inviterID,
			fmt.Sprintf("referral-inviter:%s:%d", inviterID, m.QualifiedReferrals),
			now, now.Add(time.Duration(m.InviterRewardDays)*24*time.Hour)); err != nil {
			return err
		}
		inviteeGrantID, err := identifier.NewUUID()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO premium_grants (id, user_id, source, source_key, starts_at, ends_at)
VALUES ($1,$2,'REFERRAL',$3,$4,$5)`, inviteeGrantID, inviteeID,
			fmt.Sprintf("referral-invitee:%s:%d", inviteeID, m.QualifiedReferrals),
			now, now.Add(time.Duration(m.InviteeRewardDays)*24*time.Hour)); err != nil {
			return err
		}
	}
	return nil
}
