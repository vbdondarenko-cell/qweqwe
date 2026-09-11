package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/notification"
)

type NotificationPreferencesStore struct {
	pool *pgxpool.Pool
}

func NewNotificationPreferencesStore(pool *pgxpool.Pool) (*NotificationPreferencesStore, error) {
	if pool == nil {
		return nil, errors.New("invalid notification preferences store dependency")
	}
	return &NotificationPreferencesStore{pool: pool}, nil
}

func (s *NotificationPreferencesStore) Get(ctx context.Context, userID string) (notification.StoredPreferences, error) {
	var p notification.StoredPreferences
	err := s.pool.QueryRow(ctx, `
		SELECT social_enabled,event_enabled,recommendation_enabled,promo_enabled,
		       quiet_hours_start_minute,quiet_hours_end_minute,timezone_name
		FROM notification_preferences WHERE user_id=$1`, userID).
		Scan(&p.SocialEnabled, &p.EventEnabled, &p.RecommendationEnabled, &p.PromoEnabled,
			&p.QuietHoursStartMinute, &p.QuietHoursEndMinute, &p.TimezoneName)
	if errors.Is(err, pgx.ErrNoRows) {
		return notification.DefaultStoredPreferences(), nil
	}
	if err != nil {
		return notification.StoredPreferences{}, err
	}
	return p, nil
}

func (s *NotificationPreferencesStore) Update(ctx context.Context, userID string, prefs notification.StoredPreferences) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notification_preferences (
			user_id,social_enabled,event_enabled,recommendation_enabled,promo_enabled,
			quiet_hours_start_minute,quiet_hours_end_minute,timezone_name,updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,now())
		ON CONFLICT (user_id) DO UPDATE SET
			social_enabled=EXCLUDED.social_enabled,
			event_enabled=EXCLUDED.event_enabled,
			recommendation_enabled=EXCLUDED.recommendation_enabled,
			promo_enabled=EXCLUDED.promo_enabled,
			quiet_hours_start_minute=EXCLUDED.quiet_hours_start_minute,
			quiet_hours_end_minute=EXCLUDED.quiet_hours_end_minute,
			timezone_name=EXCLUDED.timezone_name,
			updated_at=now()`,
		userID, prefs.SocialEnabled, prefs.EventEnabled, prefs.RecommendationEnabled, prefs.PromoEnabled,
		prefs.QuietHoursStartMinute, prefs.QuietHoursEndMinute, prefs.TimezoneName)
	return err
}
