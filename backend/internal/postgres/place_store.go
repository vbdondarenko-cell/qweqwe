package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/places"
)

type PlaceStore struct{ pool *pgxpool.Pool }

func NewPlaceStore(pool *pgxpool.Pool) *PlaceStore {
	return &PlaceStore{pool: pool}
}

func (s *PlaceStore) Search(ctx context.Context, query places.SearchQuery) ([]places.Place, error) {
	if s == nil || s.pool == nil {
		return nil, places.ErrInvalidSearch
	}
	if err := query.Normalize(); err != nil {
		return nil, err
	}

	pattern := "%" + escapeLike(query.Text) + "%"
	prefix := escapeLike(query.Text) + "%"
	rows, err := s.pool.Query(ctx, `
		SELECT id::text,name,category,locality,locality_id::text,country_code,latitude_e6,longitude_e6,precision_m
		FROM canonical_places
		WHERE active
		  AND name ILIKE $1 ESCAPE '\'
		  AND (
			($3<>'' AND locality_id=NULLIF($3,'')::uuid)
			OR ($3='' AND ($4='' OR lower(COALESCE(locality,''))=lower($4)))
		  )
		ORDER BY
		  CASE WHEN name ILIKE $2 ESCAPE '\' THEN 0 ELSE 1 END,
		  lower(name),id
		LIMIT $5`, pattern, prefix, query.LocalityID, query.Locality, query.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]places.Place, 0, query.Limit)
	for rows.Next() {
		var item places.Place
		var category, locality, localityID, country pgtype.Text
		if err := rows.Scan(
			&item.ID, &item.Name, &category, &locality, &localityID, &country,
			&item.LatitudeE6, &item.LongitudeE6, &item.PrecisionM,
		); err != nil {
			return nil, err
		}
		item.Category = textPtr(category)
		item.Locality = textPtr(locality)
		item.LocalityID = textPtr(localityID)
		item.CountryCode = textPtr(country)
		items = append(items, item)
	}
	return items, rows.Err()
}

func escapeLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	out := value.String
	return &out
}
