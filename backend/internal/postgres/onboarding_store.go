package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/onboarding"
)

type OnboardingStore struct{ pool *pgxpool.Pool }

func NewOnboardingStore(pool *pgxpool.Pool) *OnboardingStore { return &OnboardingStore{pool: pool} }

func (s *OnboardingStore) Create(ctx context.Context, p onboarding.PendingRegistration) error {
	preferences, err := json.Marshal(p.Preferences)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO registration_onboarding
		(id,email,username,display_name,password_hash,language,device_label,birth_date,city_id,city_name,preferences,teen_mode,verification_token_hash,created_at,expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,NULLIF($9,''),$10,$11,$12,$13,$14,$15)`,
		p.ID, p.Email, p.Username, p.DisplayName, p.PasswordHash, p.Language, p.DeviceLabel,
		p.BirthDate, p.CityID, p.CityName, preferences, p.TeenMode, p.TokenHash, p.CreatedAt, p.ExpiresAt,
	)
	return mapOnboardingError(err)
}

func (s *OnboardingStore) BindTelegram(ctx context.Context, tokenHash []byte, telegramUserID int64, now time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE registration_onboarding
		SET telegram_user_id=$2
		WHERE verification_token_hash=$1 AND completed_at IS NULL AND expires_at>$3 AND phone_verified_at IS NULL`,
		tokenHash, telegramUserID, now,
	)
	if err != nil {
		return mapOnboardingError(err)
	}
	if tag.RowsAffected() == 0 {
		return onboarding.ErrNotFound
	}
	return nil
}

func (s *OnboardingStore) VerifyTelegramContact(ctx context.Context, telegramUserID int64, phoneE164 string, now time.Time) (string, error) {
	var language string
	err := s.pool.QueryRow(ctx, `
		UPDATE registration_onboarding
		SET phone_e164=$2, phone_verified_at=$3
		WHERE telegram_user_id=$1 AND completed_at IS NULL AND expires_at>$3 AND phone_verified_at IS NULL
		RETURNING language`, telegramUserID, phoneE164, now,
	).Scan(&language)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", onboarding.ErrNotFound
	}
	if err != nil {
		return "", mapOnboardingError(err)
	}
	return language, nil
}

func (s *OnboardingStore) Status(ctx context.Context, tokenHash []byte, now time.Time) (onboarding.Status, error) {
	var out onboarding.Status
	err := s.pool.QueryRow(ctx, `
		SELECT teen_mode, (phone_verified_at IS NOT NULL), expires_at, completed_at
		FROM registration_onboarding WHERE verification_token_hash=$1 LIMIT 1`, tokenHash,
	).Scan(&out.TeenMode, &out.PhoneVerified, &out.ExpiresAt, &out.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, onboarding.ErrNotFound
	}
	if err != nil {
		return out, err
	}
	if !out.ExpiresAt.After(now) && out.CompletedAt == nil {
		return out, onboarding.ErrExpired
	}
	return out, nil
}

func (s *OnboardingStore) Finalize(ctx context.Context, tokenHash []byte, userID string, sess account.Session, now time.Time) (account.User, error) {
	var out account.User
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var email, username, displayName, passwordHash, language, deviceLabel, cityID, cityName, phoneE164 string
	var birthDate, expiresAt time.Time
	var preferences []byte
	var teenMode bool
	var telegramUserID *int64
	var phoneVerifiedAt *time.Time
	var completedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT email,username,display_name,password_hash,language,COALESCE(device_label,''),birth_date,
		       COALESCE(city_id,''),city_name,preferences,teen_mode,COALESCE(phone_e164,''),telegram_user_id,
		       phone_verified_at,expires_at,completed_at
		FROM registration_onboarding WHERE verification_token_hash=$1 FOR UPDATE`, tokenHash,
	).Scan(&email, &username, &displayName, &passwordHash, &language, &deviceLabel, &birthDate,
		&cityID, &cityName, &preferences, &teenMode, &phoneE164, &telegramUserID, &phoneVerifiedAt, &expiresAt, &completedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, onboarding.ErrNotFound
	}
	if err != nil {
		return out, err
	}
	if completedAt != nil {
		return out, onboarding.ErrConflict
	}
	if !expiresAt.After(now) {
		return out, onboarding.ErrExpired
	}
	if phoneVerifiedAt == nil || phoneE164 == "" || telegramUserID == nil || *telegramUserID <= 0 {
		return out, onboarding.ErrNotVerified
	}

	out = account.User{
		ID: userID, Email: email, Username: username, DisplayName: displayName,
		ProfileVisibility: "PUBLIC", Language: language, CreatedAt: now, UpdatedAt: now,
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO app_users (id,email,username,display_name,password_hash,avatar_url,profile_visibility,language,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,NULL,'PUBLIC',$6,$7,$7)`,
		userID, email, username, displayName, passwordHash, language, now,
	)
	if err != nil {
		return account.User{}, mapOnboardingError(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_sessions (id,user_id,token_hash,created_at,expires_at,device_label)
		VALUES ($1,$2,$3,$4,$5,NULLIF($6,''))`,
		sess.ID, userID, sess.TokenHash, sess.CreatedAt, sess.ExpiresAt, deviceLabel,
	)
	if err != nil {
		return account.User{}, mapOnboardingError(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_onboarding_profiles
		(user_id,birth_date,city_id,city_name,preferences,teen_mode,phone_e164,telegram_user_id,phone_verified_at,onboarding_completed_at)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10)`,
		userID, birthDate, cityID, cityName, preferences, teenMode, phoneE164, *telegramUserID, *phoneVerifiedAt, now,
	)
	if err != nil {
		return account.User{}, mapOnboardingError(err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE registration_onboarding SET completed_at=$2
		WHERE verification_token_hash=$1 AND completed_at IS NULL`, tokenHash, now,
	)
	if err != nil {
		return account.User{}, err
	}
	if tag.RowsAffected() != 1 {
		return account.User{}, onboarding.ErrConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return account.User{}, mapOnboardingError(err)
	}
	return out, nil
}

func mapOnboardingError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" || pgErr.Code == "23503" || pgErr.Code == "23514" || pgErr.Code == "40001" {
			return onboarding.ErrConflict
		}
	}
	return err
}
