-- Storage for README §4.3's last two visibility modes: LASSO (an
-- arbitrary host-drawn polygon) and TRAVEL_CORRIDOR (a host-drawn route
-- plus a buffer radius). Both are configured once at creation, mirroring
-- SELECTED's original (pre-000033-fix) write-once allow-list contract --
-- editing an existing LASSO/TRAVEL_CORRIDOR Slot's shape is out of this
-- block's scope, same as SELECTED's allow-list was out of scope for its
-- own first block.
--
-- Plain nullable columns directly on slots, not a side table: each Slot
-- has at most one shape of one kind, unlike SELECTED's per-user rows.
-- Geometry validity/containment is evaluated by casting the stored WKT
-- text through PostGIS (public.st_geomfromtext) directly in the
-- visibility-gate SQL at query time, the same functions
-- internal/postgres/city_context_store.go's resolveLocalityTx already
-- requires -- there is no meaningful non-PostGIS fallback for arbitrary
-- polygon/corridor containment, so unlike localities.boundary this block
-- does not attempt one; production Supabase has PostGIS (canonical
-- infrastructure, PROJECT_RULES.md) and every disposable test database
-- this session runs against has it installed for the same reason.  A
-- GIST-indexed generated geometry column (mirroring localities.boundary/
-- canonical_places.map_point) would speed up containment lookups but is
-- a real optimization left for later, stated here rather than silently
-- assumed: this block is a correctness-first foundation, not yet
-- performance-tuned for scale.

ALTER TABLE slots
    ADD COLUMN lasso_polygon_wkt text,
    ADD COLUMN corridor_line_wkt text,
    ADD COLUMN corridor_radius_m integer;

ALTER TABLE slots
    ADD CONSTRAINT slots_lasso_polygon_wkt_length
        CHECK (lasso_polygon_wkt IS NULL OR char_length(lasso_polygon_wkt) BETWEEN 10 AND 200000),
    ADD CONSTRAINT slots_corridor_line_wkt_length
        CHECK (corridor_line_wkt IS NULL OR char_length(corridor_line_wkt) BETWEEN 10 AND 200000),
    ADD CONSTRAINT slots_corridor_radius_m_range
        CHECK (corridor_radius_m IS NULL OR corridor_radius_m BETWEEN 1 AND 50000),
    -- A TRAVEL_CORRIDOR Slot must have both the line and the radius, or
    -- neither -- never one without the other, which would silently mean
    -- "a corridor with an undefined width" or "a radius around nothing."
    ADD CONSTRAINT slots_corridor_line_radius_paired
        CHECK ((corridor_line_wkt IS NULL) = (corridor_radius_m IS NULL));
