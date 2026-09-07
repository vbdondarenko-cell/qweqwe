package postgres

import (
	"context"

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
		videos       int
		qualified    int
	)
	const query = `
SELECT
    (SELECT max(pg.ends_at) FROM premium_grants pg WHERE pg.user_id = $1),
    COALESCE((SELECT rp.videos_watched_count FROM monetization_rewarded_progress rp WHERE rp.user_id = $1), 0),
    (SELECT rp.last_video_watched_at FROM monetization_rewarded_progress rp WHERE rp.user_id = $1),
    (SELECT rp.last_free_premium_claimed_at FROM monetization_rewarded_progress rp WHERE rp.user_id = $1),
    (SELECT count(*)::int FROM monetization_referrals mr WHERE mr.inviter_id = $1 AND mr.qualified_at IS NOT NULL)`
	if err := s.pool.QueryRow(ctx, query, userID).Scan(&premiumUntil, &videos, &lastVideo, &lastClaim, &qualified); err != nil {
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
	return status, nil
}
