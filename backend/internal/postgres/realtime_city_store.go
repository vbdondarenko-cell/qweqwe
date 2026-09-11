package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
)

func (s *RealtimeViewerStore) PullCity(ctx context.Context, viewerID string, after int64, limit int) (realtime.CityBatch, error) {
	if s == nil || s.pool == nil || viewerID == "" || after < 0 || limit < 1 || limit > realtime.MaxViewerBatch {
		return realtime.CityBatch{}, realtime.ErrInvalidInput
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return realtime.CityBatch{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var localityID string
	err = tx.QueryRow(ctx, `
		SELECT c.locality_id::text
		FROM city_context_locks c
		JOIN localities l ON l.id=c.locality_id
		WHERE c.user_id=$1::uuid AND c.expires_at>now() AND l.active`, viewerID).Scan(&localityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return realtime.CityBatch{}, citycontext.ErrNotFound
	}
	if err != nil {
		return realtime.CityBatch{}, err
	}

	scanLimit := limit * 4
	if scanLimit < 100 {
		scanLimit = 100
	}
	if scanLimit > realtime.MaxBatch {
		scanLimit = realtime.MaxBatch
	}
	var scannedCursor int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(max(sequence), $1)
		FROM (
			SELECT sequence
			FROM domain_outbox_events
			WHERE sequence > $1
			ORDER BY sequence ASC
			LIMIT $2
		) scanned`, after, scanLimit).Scan(&scannedCursor); err != nil {
		return realtime.CityBatch{}, err
	}
	if scannedCursor == after {
		if err := tx.Commit(ctx); err != nil {
			return realtime.CityBatch{}, err
		}
		return realtime.CityBatch{Cursor: after, Invalidations: []realtime.CityInvalidation{}}, nil
	}

	rows, err := tx.Query(ctx, cityRealtimeSQL, viewerID, localityID, after, scannedCursor, limit+1)
	if err != nil {
		return realtime.CityBatch{}, err
	}
	defer rows.Close()
	items := make([]realtime.CityInvalidation, 0, limit+1)
	for rows.Next() {
		var item realtime.CityInvalidation
		if err := rows.Scan(&item.Sequence, &item.ID, &item.OccurredAt); err != nil {
			return realtime.CityBatch{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return realtime.CityBatch{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return realtime.CityBatch{}, err
	}
	return finalizeCityBatch(scannedCursor, items, limit), nil
}

func finalizeCityBatch(scannedCursor int64, items []realtime.CityInvalidation, limit int) realtime.CityBatch {
	if len(items) > limit {
		return realtime.CityBatch{Cursor: items[limit-1].Sequence, Invalidations: items[:limit]}
	}
	return realtime.CityBatch{Cursor: scannedCursor, Invalidations: items}
}

const cityRealtimeSQL = `
	SELECT e.sequence,e.event_id,e.occurred_at
	FROM domain_outbox_events e
	LEFT JOIN slots s ON s.id=e.slot_id
	WHERE e.sequence > $3 AND e.sequence <= $4
	  AND (
		(
			e.event_type='city.slot_changed'
			AND s.id IS NOT NULL
			AND (
				NULLIF(e.payload->>'localityId','')::uuid=$2::uuid
				OR NULLIF(e.payload->>'previousLocalityId','')::uuid=$2::uuid
			)
			AND (
				(e.payload->>'visibility'='PUBLIC' AND e.payload->>'state' IN ('PUBLISHED','FILLING','FULL'))
				OR (e.payload->>'previousVisibility'='PUBLIC' AND e.payload->>'previousState' IN ('PUBLISHED','FILLING','FULL'))
				OR (
					(
						(e.payload->>'visibility'='LINKS' AND e.payload->>'state' IN ('PUBLISHED','FILLING','FULL'))
						OR (e.payload->>'previousVisibility'='LINKS' AND e.payload->>'previousState' IN ('PUBLISHED','FILLING','FULL'))
					)
					AND EXISTS (
						SELECT 1 FROM friendships f
						WHERE f.user_lo_id=LEAST(s.host_id,$1::uuid) AND f.user_hi_id=GREATEST(s.host_id,$1::uuid)
					)
				)
			)
			AND NOT EXISTS (
				SELECT 1 FROM user_blocks b
				WHERE (b.blocker_id=$1::uuid AND b.blocked_id=s.host_id)
				   OR (b.blocker_id=s.host_id AND b.blocked_id=$1::uuid)
			)
		)
		OR (
			e.event_type IN ('user.block_created','user.block_removed')
			AND (e.subject_user_id=$1::uuid OR e.aggregate_id=$1::uuid)
		)
	  )
	ORDER BY e.sequence ASC
	LIMIT $5`
