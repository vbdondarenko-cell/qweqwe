-- README §6.10's Recommendation/ranking pipeline "Layer 2 -- deterministic
-- ranking from explicit user signals" needs an explicit signal to rank
-- against. This adds the first one: a user-editable list of free-text
-- interest tags, set via PATCH /v1/me (account.ProfilePatch.Interests) and
-- consumed by internal/postgres's listV11PulseRelevanceSQL (GET
-- /v1/pulse?sort=relevance) to rank a Slot whose activity matches one of the
-- viewer's interests ahead of one that doesn't.
--
-- Per-element validation (lowercase, trim, length, dedupe) lives in Go
-- (account.normalizeInterests), mirroring how slot.normalizeSelectedUserIDs
-- validates its own array-typed input rather than pushing that into a CHECK
-- subquery. Only the outer array-size ceiling is enforced here, as a second
-- line of defense against a client bypassing the Go validation layer.
--
-- No new linkup_api grant is needed: app_users predates the v1.0 database
-- hardening migration (000005) and linkup_api already owns it, so a new
-- column is automatically covered by that existing ownership.

ALTER TABLE app_users
    ADD COLUMN interests text[] NOT NULL DEFAULT '{}',
    ADD CONSTRAINT app_users_interests_size_check CHECK (cardinality(interests) <= 20);
