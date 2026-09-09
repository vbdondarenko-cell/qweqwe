package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
)

type CapabilityStore struct{ pool *pgxpool.Pool }

func NewCapabilityStore(pool *pgxpool.Pool) (*CapabilityStore, error) {
	if pool == nil {
		return nil, errors.New("invalid capability store dependency")
	}
	return &CapabilityStore{pool: pool}, nil
}

func (s *CapabilityStore) List(ctx context.Context) ([]capability.Entry, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT capability_key, enabled, revision, scope_type,
               COALESCE(scope_user_ids::text[], ARRAY[]::text[]), effective_at
        FROM capability_registry
        ORDER BY capability_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]capability.Entry, 0, len(capability.KnownKeys()))
	for rows.Next() {
		var entry capability.Entry
		var key string
		if err := rows.Scan(
			&key, &entry.Enabled, &entry.Revision, &entry.ScopeType,
			&entry.ScopeUserIDs, &entry.EffectiveAt,
		); err != nil {
			return nil, err
		}
		entry.Key = capability.Key(key)
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
