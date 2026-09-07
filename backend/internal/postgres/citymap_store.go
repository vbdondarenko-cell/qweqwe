package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citymap"
)

type CityMapStore struct{ pool *pgxpool.Pool }

func NewCityMapStore(pool *pgxpool.Pool) (*CityMapStore, error) {
	if pool == nil {
		return nil, errors.New("invalid city map store dependency")
	}
	return &CityMapStore{pool: pool}, nil
}

func (s *CityMapStore) Viewport(ctx context.Context, viewerID string, query citymap.Viewport) ([]citymap.Cluster, error) {
	if viewerID == "" || !query.Valid() {
		return nil, citymap.ErrInvalidViewport
	}
	rows, err := s.pool.Query(ctx, `
		WITH visible_places AS (
			SELECT p.id,p.name,p.latitude_e6,p.longitude_e6,count(*)::int AS slot_count
			FROM canonical_places p
			JOIN slots s ON s.canonical_place_id=p.id
			WHERE p.active
			  AND s.visibility='PUBLIC'
			  AND s.state IN ('PUBLISHED','FILLING','FULL')
			  AND s.start_at IS NOT NULL
			  AND s.start_at >= $7 AND s.start_at < $8
			  AND p.latitude_e6 BETWEEN $2 AND $4
			  AND (
				($1 < $3 AND p.longitude_e6 BETWEEN $1 AND $3)
				OR ($1 > $3 AND (p.longitude_e6 >= $1 OR p.longitude_e6 <= $3))
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM user_blocks b
				WHERE (b.blocker_id=$6 AND b.blocked_id=s.host_id)
				   OR (b.blocker_id=s.host_id AND b.blocked_id=$6)
			  )
			GROUP BY p.id,p.name,p.latitude_e6,p.longitude_e6
		), bucketed AS (
			SELECT
				floor(latitude_e6::numeric / $5)::bigint AS bucket_lat,
				floor(longitude_e6::numeric / $5)::bigint AS bucket_lng,
				id,name,latitude_e6,longitude_e6,slot_count
			FROM visible_places
		), clusters AS (
			SELECT
				bucket_lat,bucket_lng,
				round(avg(latitude_e6))::int AS latitude_e6,
				round(avg(longitude_e6))::int AS longitude_e6,
				count(*)::int AS place_count,
				sum(slot_count)::int AS slot_count,
				CASE WHEN count(*)=1 THEN min(id::text) END AS place_id,
				CASE WHEN count(*)=1 THEN min(name) END AS place_name
			FROM bucketed
			GROUP BY bucket_lat,bucket_lng
		)
		SELECT bucket_lat::text || ':' || bucket_lng::text,
		       latitude_e6,longitude_e6,place_count,slot_count,place_id,place_name
		FROM clusters
		ORDER BY slot_count DESC,bucket_lat,bucket_lng
		LIMIT $9`,
		query.WestE6, query.SouthE6, query.EastE6, query.NorthE6, query.BucketE6(),
		viewerID, query.From, query.To, query.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]citymap.Cluster, 0, query.Limit)
	for rows.Next() {
		var item citymap.Cluster
		if err := rows.Scan(
			&item.Key, &item.LatitudeE6, &item.LongitudeE6,
			&item.PlaceCount, &item.SlotCount, &item.PlaceID, &item.PlaceName,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
