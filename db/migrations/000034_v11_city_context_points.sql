-- Privacy-safe coarse point storage, the user-explicitly-approved
-- prerequisite for README §4.3's remaining two visibility modes
-- (LASSO/TRAVEL_CORRIDOR). city_context_locks (migration 000017)
-- deliberately stores only a resolved locality id, never a raw
-- coordinate -- correct for every visibility mode built so far
-- (PUBLIC/PRIVATE/LINKS/SELECTED/CITY), none of which need more than "is
-- this viewer in the same named place." LASSO/TRAVEL_CORRIDOR are
-- different: they test whether a viewer is currently inside an arbitrary
-- host-drawn shape, which a locality id cannot express -- some form of
-- the viewer's own approximate position is unavoidable.
--
-- The user was asked directly how to close this gap (a real
-- privacy/product decision, not something to invent unilaterally) and
-- chose: a new, separate, privacy-safe table storing only a COARSENED
-- point (rounded to a ~111m grid -- see roundToPointGrid in
-- internal/postgres/city_context_store.go -- never the raw observation),
-- under the exact same freshness/expiry discipline city_context_locks
-- already uses: written by the same Resolve() call, sharing its
-- expires_at, cleared the same way. It is never itself public-facing or
-- queryable by anything other than the visibility-gate SQL this Slot
-- domain needs; no new API surface reads it directly.

CREATE TABLE city_context_points (
    user_id uuid PRIMARY KEY REFERENCES app_users(id) ON DELETE CASCADE,
    latitude_e6 integer NOT NULL,
    longitude_e6 integer NOT NULL,
    permission_class text NOT NULL,
    accuracy_m integer NOT NULL,
    observed_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (latitude_e6 BETWEEN -90000000 AND 90000000),
    CHECK (longitude_e6 BETWEEN -180000000 AND 180000000),
    CHECK (permission_class IN ('APPROXIMATE','PRECISE')),
    CHECK (accuracy_m > 0),
    CHECK (expires_at > observed_at)
);

CREATE INDEX city_context_points_expiry_idx ON city_context_points (expires_at, user_id);

REVOKE ALL ON TABLE city_context_points FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE city_context_points FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE city_context_points FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        -- UPDATE is required: like city_context_locks, this is an
        -- upsert-on-every-resolve row per user, not an append-only log.
        GRANT SELECT, INSERT, UPDATE ON TABLE city_context_points TO linkup_api;
    END IF;
END
$$;
