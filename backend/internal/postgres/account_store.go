package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
)

type AccountStore struct{ pool *pgxpool.Pool }

func NewAccountStore(pool *pgxpool.Pool) *AccountStore { return &AccountStore{pool: pool} }

func (s *AccountStore) Register(ctx context.Context, u account.User, passwordHash string, sess account.Session) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil { return err }
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO app_users (id,email,username,display_name,password_hash,avatar_url,profile_visibility,language,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, u.ID,u.Email,u.Username,u.DisplayName,passwordHash,u.AvatarURL,u.ProfileVisibility,u.Language,u.CreatedAt,u.UpdatedAt)
	if err != nil { return mapError(err) }
	_, err = tx.Exec(ctx, `INSERT INTO user_sessions (id,user_id,token_hash,created_at,expires_at,device_label) VALUES ($1,$2,$3,$4,$5,NULLIF($6,''))`, sess.ID,sess.UserID,sess.TokenHash,sess.CreatedAt,sess.ExpiresAt,sess.DeviceLabel)
	if err != nil { return mapError(err) }
	return tx.Commit(ctx)
}

func (s *AccountStore) FindByLogin(ctx context.Context, identifier string) (account.UserWithPassword, error) {
	var out account.UserWithPassword
	err := s.pool.QueryRow(ctx, `SELECT id,email,username,display_name,password_hash,avatar_url,profile_visibility,language,created_at,updated_at FROM app_users WHERE lower(email)=lower($1) OR lower(username)=lower($1) LIMIT 1`, identifier).Scan(&out.ID,&out.Email,&out.Username,&out.DisplayName,&out.PasswordHash,&out.AvatarURL,&out.ProfileVisibility,&out.Language,&out.CreatedAt,&out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) { return out, account.ErrNotFound }
	return out, err
}

func (s *AccountStore) CreateSession(ctx context.Context, sess account.Session) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO user_sessions (id,user_id,token_hash,created_at,expires_at,device_label) VALUES ($1,$2,$3,$4,$5,NULLIF($6,''))`, sess.ID,sess.UserID,sess.TokenHash,sess.CreatedAt,sess.ExpiresAt,sess.DeviceLabel)
	return mapError(err)
}

func (s *AccountStore) Authenticate(ctx context.Context, tokenHash []byte, now time.Time) (account.User, string, error) {
	var u account.User
	var sessionID string
	err := s.pool.QueryRow(ctx, `SELECT u.id,u.email,u.username,u.display_name,u.avatar_url,u.profile_visibility,u.language,u.created_at,u.updated_at,s.id FROM user_sessions s JOIN app_users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>$2 LIMIT 1`, tokenHash, now).Scan(&u.ID,&u.Email,&u.Username,&u.DisplayName,&u.AvatarURL,&u.ProfileVisibility,&u.Language,&u.CreatedAt,&u.UpdatedAt,&sessionID)
	if errors.Is(err, pgx.ErrNoRows) { return u,"",account.ErrUnauthorized }
	return u, sessionID, err
}

func (s *AccountStore) RevokeSession(ctx context.Context, tokenHash []byte, now time.Time) error {
	tag, err := s.pool.Exec(ctx, `UPDATE user_sessions SET revoked_at=$2 WHERE token_hash=$1 AND revoked_at IS NULL`, tokenHash, now)
	if err != nil { return err }
	if tag.RowsAffected()==0 { return account.ErrNotFound }
	return nil
}

func (s *AccountStore) UpdateProfile(ctx context.Context, userID string, p account.ProfilePatch, now time.Time) (account.User, error) {
	var out account.User
	displaySet := p.DisplayName != nil; display := ""; if p.DisplayName != nil { display=*p.DisplayName }
	avatarSet := p.AvatarURL != nil; avatar := ""; if p.AvatarURL != nil { avatar=*p.AvatarURL }
	visibilitySet := p.ProfileVisibility != nil; visibility := ""; if p.ProfileVisibility != nil { visibility=*p.ProfileVisibility }
	languageSet := p.Language != nil; language := ""; if p.Language != nil { language=*p.Language }
	err := s.pool.QueryRow(ctx, `UPDATE app_users SET display_name=CASE WHEN $2 THEN $3 ELSE display_name END, avatar_url=CASE WHEN $4 THEN NULLIF($5,'') ELSE avatar_url END, profile_visibility=CASE WHEN $6 THEN $7 ELSE profile_visibility END, language=CASE WHEN $8 THEN $9 ELSE language END, updated_at=$10 WHERE id=$1 RETURNING id,email,username,display_name,avatar_url,profile_visibility,language,created_at,updated_at`, userID,displaySet,display,avatarSet,avatar,visibilitySet,visibility,languageSet,language,now).Scan(&out.ID,&out.Email,&out.Username,&out.DisplayName,&out.AvatarURL,&out.ProfileVisibility,&out.Language,&out.CreatedAt,&out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) { return out, account.ErrNotFound }
	return out, err
}

func mapError(err error) error {
	if err == nil { return nil }
	var pgErr *pgconn.PgError
	if errors.As(err,&pgErr) && pgErr.Code=="23505" { return account.ErrConflict }
	return err
}
