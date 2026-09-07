package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
