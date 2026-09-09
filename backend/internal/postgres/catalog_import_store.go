package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/catalog"
)

type CatalogImportResult struct {
	LocalitiesUpserted int
	PlacesUpserted     int
}

type CatalogImportStore struct{ pool *pgxpool.Pool }

func NewCatalogImportStore(pool *pgxpool.Pool) (*CatalogImportStore, error) {
	if pool == nil {
		return nil, catalog.ErrInvalidCatalog
	}
	return &CatalogImportStore{pool: pool}, nil
}

func (s *CatalogImportStore) Import(ctx context.Context, doc catalog.Document) (CatalogImportResult, error) {
	if err := doc.NormalizeAndValidate(); err != nil {
		return CatalogImportResult{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return CatalogImportResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result := CatalogImportResult{}
	for _, locality := range doc.Localities {
		if err := upsertLocality(ctx, tx, locality); err != nil {
			return CatalogImportResult{}, err
		}
		result.LocalitiesUpserted++
	}
	for _, place := range doc.Places {
		if err := upsertPlace(ctx, tx, place); err != nil {
			return CatalogImportResult{}, err
		}
		result.PlacesUpserted++
	}
	if err := tx.Commit(ctx); err != nil {
		return CatalogImportResult{}, err
	}
	return result, nil
}

func upsertLocality(ctx context.Context, tx pgx.Tx, r catalog.LocalityRecord) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO localities (
			id,source,source_locality_id,name,country_code,timezone_name,
			centroid_latitude_e6,centroid_longitude_e6,boundary_wkt,active
		) VALUES (
			gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,$9
		)
		ON CONFLICT (source,source_locality_id) DO UPDATE SET
			name=EXCLUDED.name,
			country_code=EXCLUDED.country_code,
			timezone_name=EXCLUDED.timezone_name,
			centroid_latitude_e6=EXCLUDED.centroid_latitude_e6,
			centroid_longitude_e6=EXCLUDED.centroid_longitude_e6,
			boundary_wkt=EXCLUDED.boundary_wkt,
			active=EXCLUDED.active,
			updated_at=now()`,
		r.Source, r.SourceLocalityID, r.Name, r.CountryCode, r.Timezone,
		r.CentroidLatitudeE6, r.CentroidLongitudeE6, r.BoundaryWKT, *r.Active,
	)
	return err
}

func upsertPlace(ctx context.Context, tx pgx.Tx, r catalog.PlaceRecord) error {
	var localityID any
	var localityName any
	if r.LocalitySource != "" {
		var id, name string
		var active bool
		err := tx.QueryRow(ctx, `
			SELECT id::text,name,active
			FROM localities
			WHERE source=$1 AND source_locality_id=$2`, r.LocalitySource, r.LocalitySourceID,
		).Scan(&id, &name, &active)
		if errors.Is(err, pgx.ErrNoRows) {
			return catalog.ErrInvalidCatalog
		}
		if err != nil {
			return err
		}
		if *r.Active && !active {
			return catalog.ErrInvalidCatalog
		}
		localityID = id
		localityName = name
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO canonical_places (
			id,source,source_place_id,name,category,locality,country_code,
			latitude_e6,longitude_e6,precision_m,active,locality_id
		) VALUES (
			gen_random_uuid(),$1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11::uuid
		)
		ON CONFLICT (source,source_place_id) DO UPDATE SET
			name=EXCLUDED.name,
			category=EXCLUDED.category,
			locality=EXCLUDED.locality,
			country_code=EXCLUDED.country_code,
			latitude_e6=EXCLUDED.latitude_e6,
			longitude_e6=EXCLUDED.longitude_e6,
			precision_m=EXCLUDED.precision_m,
			active=EXCLUDED.active,
			locality_id=EXCLUDED.locality_id,
			updated_at=now()`,
		r.Source, r.SourcePlaceID, r.Name, r.Category, localityName, r.CountryCode,
		r.LatitudeE6, r.LongitudeE6, r.PrecisionM, *r.Active, localityID,
	)
	return err
}
