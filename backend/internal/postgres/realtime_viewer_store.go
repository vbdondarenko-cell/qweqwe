package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
)

type RealtimeViewerStore struct {
	pool *pgxpool.Pool
}

func NewRealtimeViewerStore(pool *pgxpool.Pool) (*RealtimeViewerStore, error) {
	if pool == nil {
		return nil, realtime.ErrInvalidInput
	}
	return &RealtimeViewerStore{pool: pool}, nil
}

// PullViewer treats canonical outbox events as invalidation deltas, not as
// client authority. The cursor advances across invisible events so a client
// cannot use the feed as a side channel by repeatedly probing the same range.
// When more than limit visible events occur inside the scanned range, however,
// the cursor stops at the last returned visible event so no authorized delta is
// skipped by pagination.
func (s *RealtimeViewerStore) PullViewer(ctx context.Context, viewerID string, after int64, limit int) (realtime.ViewerBatch, error) {
	if viewerID == "" || after < 0 || limit < 1 || limit > realtime.MaxViewerBatch {
		return realtime.ViewerBatch{}, realtime.ErrInvalidInput
	}

	scanLimit := limit * 4
	if scanLimit < 100 {
		scanLimit = 100
	}
	if scanLimit > realtime.MaxBatch {
		scanLimit = realtime.MaxBatch
	}

	var scannedCursor int64
	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(max(sequence), $1)
		FROM (
			SELECT sequence
			FROM domain_outbox_events
			WHERE sequence > $1
			ORDER BY sequence ASC
			LIMIT $2
		) scanned`, after, scanLimit).Scan(&scannedCursor); err != nil {
		return realtime.ViewerBatch{}, err
	}

	if scannedCursor == after {
		return realtime.ViewerBatch{Cursor: after, Events: []realtime.Event{}}, nil
	}

	rows, err := s.pool.Query(ctx, viewerRealtimeSQL, viewerID, after, scannedCursor, limit+1)
	if err != nil {
		return realtime.ViewerBatch{}, err
	}
	defer rows.Close()

	items := make([]realtime.Event, 0, limit+1)
	for rows.Next() {
		var event realtime.Event
		if err := rows.Scan(
			&event.Sequence, &event.ID, &event.Type, &event.AggregateType, &event.AggregateID,
			&event.SubjectUserID, &event.SlotID, &event.Payload, &event.OccurredAt,
		); err != nil {
			return realtime.ViewerBatch{}, err
		}
		if !event.Valid() {
			return realtime.ViewerBatch{}, errors.New("invalid viewer realtime event")
		}
		items = append(items, event)
	}
	if err := rows.Err(); err != nil {
		return realtime.ViewerBatch{}, err
	}
	return finalizeViewerBatch(scannedCursor, items, limit), nil
}

func finalizeViewerBatch(scannedCursor int64, events []realtime.Event, limit int) realtime.ViewerBatch {
	if len(events) > limit {
		return realtime.ViewerBatch{
			Cursor: events[limit-1].Sequence,
			Events: events[:limit],
		}
	}
	return realtime.ViewerBatch{Cursor: scannedCursor, Events: events}
}

const viewerRealtimeSQL = `
	SELECT e.sequence,e.event_id,e.event_type,e.aggregate_type,e.aggregate_id,
	       e.subject_user_id,e.slot_id,e.payload,e.occurred_at
	FROM domain_outbox_events e
	LEFT JOIN slots s ON s.id=e.slot_id
	WHERE e.sequence > $2 AND e.sequence <= $3
	  AND (
		(
			e.event_type IN ('slot.created','slot.updated','slot.state_changed')
			AND s.id IS NOT NULL
			AND (
				s.host_id=$1::uuid
				OR EXISTS (SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$1::uuid)
				OR EXISTS (SELECT 1 FROM slot_requests r WHERE r.slot_id=s.id AND r.user_id=$1::uuid)
				OR (
					s.visibility='PUBLIC'
					AND s.state IN ('PUBLISHED','FILLING','FULL')
					AND NOT EXISTS (
						SELECT 1 FROM user_blocks b
						WHERE (b.blocker_id=$1::uuid AND b.blocked_id=s.host_id)
						   OR (b.blocker_id=s.host_id AND b.blocked_id=$1::uuid)
					)
				)
			)
		)
		OR (
			e.event_type IN (
				'slot.request_created','slot.request_removed',
				'slot.membership_added','slot.membership_removed'
			)
			AND s.id IS NOT NULL
			AND (e.subject_user_id=$1::uuid OR s.host_id=$1::uuid)
		)
		OR (
			e.event_type='slot.chat_message_created'
			AND s.id IS NOT NULL
			AND (
				s.host_id=$1::uuid
				OR EXISTS (SELECT 1 FROM slot_memberships m WHERE m.slot_id=s.id AND m.user_id=$1::uuid)
			)
		)
		OR (
			e.event_type IN ('user.block_created','user.block_removed')
			AND (e.subject_user_id=$1::uuid OR e.aggregate_id=$1::uuid)
		)
		OR (
			e.event_type='user.profile_changed'
			AND e.subject_user_id=$1::uuid
		)
	  )
	ORDER BY e.sequence ASC
	LIMIT $4`