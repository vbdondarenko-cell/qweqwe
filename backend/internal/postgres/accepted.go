package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func (s *SlotStore) ListAccepted(ctx context.Context, actorID, slotID string) ([]slot.Organizer, error) {
	var data []byte
	err := s.pool.QueryRow(ctx, acceptedRosterSQL, actorID, slotID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) { return nil, slot.ErrNotFound }
	if err != nil { return nil, err }
	var items []slot.Organizer
	if err := json.Unmarshal(data, &items); err != nil { return nil, err }
	return items, nil
}

// Authorization, lifecycle, membership and blocks share one statement snapshot.
// An inaccessible Slot returns the same result as a missing Slot.
const acceptedRosterSQL = `
SELECT COALESCE(jsonb_agg(jsonb_build_object(
	'id',u.id,'username',u.username,'displayName',u.display_name,'avatarUrl',u.avatar_url
) ORDER BY u.id) FILTER (WHERE u.id IS NOT NULL), '[]'::jsonb)
FROM slots s
LEFT JOIN slot_memberships m ON m.slot_id=s.id AND m.user_id<>s.host_id
LEFT JOIN app_users u ON u.id=m.user_id
	AND NOT EXISTS (SELECT 1 FROM user_blocks b
		WHERE (b.blocker_id=$1 AND b.blocked_id=u.id)
		   OR (b.blocker_id=u.id AND b.blocked_id=$1))
WHERE s.id=$2 AND s.host_id=$1
	AND s.state IN ('PUBLISHED','FILLING','FULL','ACTIVE')
GROUP BY s.id`
