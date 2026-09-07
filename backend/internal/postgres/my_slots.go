package postgres

import (
	"context"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func (s *SlotStore) ListMine(ctx context.Context, actorID, view string, limit int) ([]slot.Slot, error) {
	rows, err := s.pool.Query(ctx, listMySlotsSQL, actorID, view, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	items := make([]slot.Slot, 0)
	for rows.Next() {
		item, err := scanSlot(rows)
		if err != nil { return nil, err }
		items = append(items, item)
	}
	return items, rows.Err()
}

const listMySlotsSQL = `SELECT ` + slotColumns + `,
	CASE WHEN s.host_id=$1 THEN 'HOST'
	     WHEN EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$1) THEN 'ACCEPTED'
	     ELSE 'PENDING' END
FROM slots s JOIN app_users u ON u.id=s.host_id
WHERE s.state IN ('PUBLISHED','FILLING','FULL','ACTIVE')
  AND (
    ($2='HOSTING' AND s.host_id=$1)
    OR ($2='JOINED' AND s.host_id<>$1 AND EXISTS(SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$1))
    OR ($2='REQUESTED' AND s.host_id<>$1 AND s.state<>'ACTIVE'
        AND EXISTS(SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$1))
  )
  AND NOT EXISTS(SELECT 1 FROM user_blocks b
    WHERE (b.blocker_id=$1 AND b.blocked_id=s.host_id)
       OR (b.blocker_id=s.host_id AND b.blocked_id=$1))
ORDER BY s.updated_at DESC,s.id
LIMIT $3`
