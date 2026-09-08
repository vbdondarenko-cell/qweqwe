-- LinkUp v1.1 City Context foundation.
-- Raw device coordinates are transient resolver input only. Persistent state keeps
-- a locality identity plus coarse observation quality, never a user GPS point.

CREATE TABLE localities (
    id uuid PRIMARY KEY,
    source text NOT NULL,
    source_locality_id text NOT NULL,
    name text NOT NULL,
    country_code text NOT NULL,
    timezone_name text NOT NULL,
    centroid_latitude_e6 integer NOT NULL,
    centroid_longitude_e6 integer NOT NULL,
    boundary_wkt text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, source_locality_id),
    CHECK (char_length(source) BETWEEN 1 AND 32),
    CHECK (char_length(source_locality_id) BETWEEN 1 AND 160),
    CHECK (char_length(name) BETWEEN 1 AND 160),
    CHECK (country_code ~ '^[A-Z]{2}$'),
    CHECK (char_length(timezone_name) BETWEEN 1 AND 80),
    CHECK (centroid_latitude_e6 BETWEEN -90000000 AND 90000000),
    CHECK (centroid_longitude_e6 BETWEEN -180000000 AND 180000000),
    CHECK (char_length(boundary_wkt) BETWEEN 10 AND 4000000)
);

CREATE INDEX localities_active_name_idx
    ON localities (lower(name), country_code, id)
    WHERE active;
CREATE INDEX localities_active_centroid_idx
    ON localities (centroid_latitude_e6, centroid_longitude_e6, id)
    WHERE active;

-- Production Supabase has PostGIS. Keep the base migration valid on disposable
-- PostgreSQL without PostGIS so the normal migration/test chain remains portable.
DO $$
BEGIN
    IF to_regtype('public.geometry') IS NOT NULL
       AND to_regprocedure('public.st_geomfromtext(text,integer)') IS NOT NULL
       AND to_regprocedure('public.st_multi(public.geometry)') IS NOT NULL THEN
        EXECUTE $sql$
            ALTER TABLE public.localities
            ADD COLUMN boundary public.geometry(MultiPolygon,4326)
            GENERATED ALWAYS AS (
                public.st_multi(public.st_geomfromtext(boundary_wkt, 4326))
            ) STORED
        $sql$;
        EXECUTE 'CREATE INDEX localities_boundary_gist ON public.localities USING gist (boundary) WHERE active';
    END IF;
END
$$;

CREATE TABLE city_context_locks (
    user_id uuid PRIMARY KEY REFERENCES app_users(id) ON DELETE CASCADE,
    locality_id uuid NOT NULL REFERENCES localities(id) ON DELETE RESTRICT,
    permission_class text NOT NULL,
    accuracy_m integer NOT NULL,
    observed_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    candidate_locality_id uuid REFERENCES localities(id) ON DELETE RESTRICT,
    candidate_count integer NOT NULL DEFAULT 0,
    candidate_observed_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (permission_class IN ('APPROXIMATE','PRECISE')),
    CHECK (accuracy_m BETWEEN 1 AND 10000),
    CHECK (expires_at > observed_at),
    CHECK (candidate_count BETWEEN 0 AND 10),
    CHECK (
        (candidate_locality_id IS NULL AND candidate_count = 0 AND candidate_observed_at IS NULL)
        OR
        (candidate_locality_id IS NOT NULL AND candidate_count > 0 AND candidate_observed_at IS NOT NULL)
    )
);

CREATE INDEX city_context_locks_expiry_idx ON city_context_locks (expires_at, user_id);
CREATE INDEX city_context_locks_locality_idx ON city_context_locks (locality_id, expires_at, user_id);

ALTER TABLE canonical_places
    ADD COLUMN locality_id uuid REFERENCES localities(id) ON DELETE RESTRICT;
CREATE INDEX canonical_places_locality_idx
    ON canonical_places (locality_id, id)
    WHERE active AND locality_id IS NOT NULL;

REVOKE ALL ON TABLE localities, city_context_locks FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
        REVOKE ALL ON TABLE localities, city_context_locks FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
        REVOKE ALL ON TABLE localities, city_context_locks FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='linkup_api') THEN
        GRANT SELECT ON TABLE localities TO linkup_api;
        GRANT SELECT, INSERT, UPDATE ON TABLE city_context_locks TO linkup_api;
    END IF;
END
$$;
