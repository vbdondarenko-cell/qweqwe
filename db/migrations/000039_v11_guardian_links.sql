-- README §6.17: Ghost Guardian -- a temporary, opaque, revocable link that
-- lets a participant show a trusted person (who need not even have a
-- LinkUp account) whether they're still safely checked into a Slot,
-- without exposing the Slot itself, its location, or its roster.
--
-- Only the STATUS_ONLY sharing mode is implemented this block -- see
-- internal/guardian's own doc comment for why ETA_APPROXIMATE,
-- SAFETY_RADAR and LIVE_PRECISE (all three real README-named modes) are
-- deliberately rejected rather than faked: all three depend on a
-- live-location-submission pipeline this repo doesn't have and isn't
-- specified yet. Extending the mode CHECK constraint below is exactly how
-- a future block would turn one of them on for real.
--
-- token_hash, not the raw token, is stored -- same pattern as
-- user_sessions/password_reset_tokens. expires_at/revoked_at together
-- implement "indistinguishable/burned access after expiry/revoke" at the
-- query level: Go never needs to (and must not) tell a caller which of
-- "never existed" / "expired" / "revoked" applied.

CREATE TABLE guardian_links (
    id uuid PRIMARY KEY,
    slot_id uuid NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    created_by uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    mode text NOT NULL CHECK (mode IN ('STATUS_ONLY')),
    token_hash bytea NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CHECK (expires_at > created_at)
);

CREATE INDEX guardian_links_slot_idx ON guardian_links (slot_id);
CREATE INDEX guardian_links_creator_idx ON guardian_links (created_by, created_at DESC);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT SELECT, INSERT, UPDATE ON TABLE guardian_links TO linkup_api;
    END IF;
END
$$;
