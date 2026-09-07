package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/push"
)

type PushStore struct {
	pool *pgxpool.Pool
}

func NewPushStore(pool *pgxpool.Pool) *PushStore { return &PushStore{pool: pool} }

func (s *PushStore) UpsertAndroid(ctx context.Context, d push.EncryptedDevice) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil { return err }
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
DELETE FROM push_devices
WHERE token_hash = $1
  AND NOT (platform = 'ANDROID' AND installation_id = $2)`, d.TokenHash, d.InstallationID); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
INSERT INTO push_devices (
    id, user_id, session_id, platform, installation_id,
    token_hash, token_ciphertext, token_nonce, key_id, app_version,
    created_at, updated_at, last_seen_at, revoked_at
)
VALUES ($1,$2,$3,'ANDROID',$4,$5,$6,$7,$8,NULLIF($9,''),now(),now(),now(),NULL)
ON CONFLICT (platform, installation_id) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    session_id = EXCLUDED.session_id,
    token_hash = EXCLUDED.token_hash,
    token_ciphertext = EXCLUDED.token_ciphertext,
    token_nonce = EXCLUDED.token_nonce,
    key_id = EXCLUDED.key_id,
    app_version = EXCLUDED.app_version,
    updated_at = now(),
    last_seen_at = now(),
    revoked_at = NULL`,
		d.ID, d.UserID, d.SessionID, d.InstallationID,
		d.TokenHash, d.Ciphertext, d.Nonce, d.KeyID, d.AppVersion,
	)
	if err != nil { return err }
	return tx.Commit(ctx)
}

func (s *PushStore) RevokeInstallation(ctx context.Context, userID, installationID string) error {
	_, err := s.pool.Exec(ctx, `
UPDATE push_devices
SET revoked_at = now(), updated_at = now()
WHERE user_id = $1 AND platform = 'ANDROID' AND installation_id = $2 AND revoked_at IS NULL`, userID, installationID)
	return err
}

func (s *PushStore) RevokeSession(ctx context.Context, sessionID string) error {
	_, err := s.pool.Exec(ctx, `
UPDATE push_devices
SET revoked_at = now(), updated_at = now()
WHERE session_id = $1 AND revoked_at IS NULL`, sessionID)
	return err
}

func (s *PushStore) ActiveTokens(ctx context.Context, userID string) ([]push.StoredToken, error) {
	rows, err := s.pool.Query(ctx, `
SELECT pd.installation_id::text, pd.token_ciphertext, pd.token_nonce, pd.key_id
FROM push_devices pd
JOIN user_sessions us ON us.id = pd.session_id AND us.user_id = pd.user_id
WHERE pd.user_id = $1
  AND pd.revoked_at IS NULL
  AND us.revoked_at IS NULL
  AND us.expires_at > now()
ORDER BY pd.updated_at DESC
LIMIT 20`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	items := make([]push.StoredToken, 0)
	for rows.Next() {
		var item push.StoredToken
		if err := rows.Scan(&item.InstallationID, &item.Ciphertext, &item.Nonce, &item.KeyID); err != nil { return nil, err }
		items = append(items, item)
	}
	return items, rows.Err()
}
