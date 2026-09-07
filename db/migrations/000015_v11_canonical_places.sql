-- LinkUp v1.1 Map foundation.
-- Map coordinates belong to canonical public places, never to a user's home or
-- raw device trace. Slots opt in by referencing a canonical place identity.

CREATE TABLE canonical_places (
    id uuid PRIMARY KEY,
    source text NOT NULL,
    source_place_id text NOT NULL,
    name text NOT NULL,
    category text,
    locality text,
    country_code text,
    latitude_e6 integer NOT NULL,
    longitude_e6 integer NOT NULL,
    precision_m integer NOT NULL DEFAULT 100,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, source_place_id),
    CHECK (char_length(source) BETWEEN 1 AND 32),
    CHECK (char_length(source_place_id) BETWEEN 1 AND 160),
    CHECK (char_length(name) BETWEEN 1 AND 160),
    CHECK (category IS NULL OR char_length(category) <= 64),
    CHECK (locality IS NULL OR char_length(locality) <= 120),
    CHECK (country_code IS NULL OR country_code ~ '^[A-Z]{2}$'),
    CHECK (latitude_e6 BETWEEN -90000000 AND 90000000),
    CHECK (longitude_e6 BETWEEN -180000000 AND 180000000),
    CHECK (precision_m BETWEEN 1 AND 10000)
);

CREATE INDEX canonical_places_viewport_idx
    ON canonical_places (latitude_e6, longitude_e6, id)
    WHERE active;
CREATE INDEX canonical_places_name_idx
    ON canonical_places (lower(name), id)
    WHERE active;

ALTER TABLE slots
    ADD COLUMN canonical_place_id uuid REFERENCES canonical_places(id) ON DELETE RESTRICT;

CREATE INDEX slots_public_map_place_idx
    ON slots (canonical_place_id, start_at, id)
    WHERE canonical_place_id IS NOT NULL
      AND visibility='PUBLIC'
      AND state IN ('PUBLISHED','FILLING','FULL');

-- If PostGIS is installed in public (as on the current Supabase project), add
-- a generated point plus GiST index. Local/test PostgreSQL without PostGIS still
-- uses the deterministic microdegree viewport columns above.
DO $$
BEGIN
    IF to_regtype('public.geometry') IS NOT NULL
       AND to_regprocedure('public.st_makepoint(double precision,double precision)') IS NOT NULL
       AND to_regprocedure('public.st_setsrid(public.geometry,integer)') IS NOT NULL THEN
        EXECUTE $sql$
            ALTER TABLE public.canonical_places
            ADD COLUMN map_point public.geometry(Point,4326)
            GENERATED ALWAYS AS (
                public.st_setsrid(
                    public.st_makepoint(longitude_e6::double precision / 1000000.0,
                                        latitude_e6::double precision / 1000000.0),
                    4326
                )
            ) STORED
        $sql$;
        EXECUTE 'CREATE INDEX canonical_places_map_point_gist ON public.canonical_places USING gist (map_point) WHERE active';
    END IF;
END
$$;

REVOKE ALL ON TABLE canonical_places FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
        REVOKE ALL ON TABLE canonical_places FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
        REVOKE ALL ON TABLE canonical_places FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='linkup_api') THEN
        GRANT SELECT ON TABLE canonical_places TO linkup_api;
    END IF;
END
$$;
