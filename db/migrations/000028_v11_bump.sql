-- LinkUp v1.1 BUMP proof baseline (README §6.9).
--
-- A BUMP is mutual, server-verified proof that two accepted participants of
-- the same Slot were both physically present together. It is expressed as
-- two independent one-directional claims (each side names the other) that
-- must BOTH exist before either user's reliability changes at all -- a
-- single client tap can never move reliability on its own (README §6.9:
-- "Client tap alone can never increase trust/reliability"). See
-- internal/bump and internal/postgres/bump_store.go for the confirmation
-- logic that enforces this.
--
-- Anti-replay is a server-issued, single-use challenge nonce (bump_challenges):
-- a real device-proof signature (README "Android Keystore-backed device
-- proof") is future work once Android is back in active scope, but the
-- nonce already stops a captured/replayed HTTP request from resubmitting.
--
-- Anti-farm / "one-event/one-contribution" is the (slot_id, submitter_id,
-- counterpart_id) primary key on bump_submissions: a submitter cannot claim
-- the same counterpart twice for the same Slot even with a fresh nonce, and
-- both sides must be an actual host/accepted participant of that Slot
-- (enforced in application code, not by a table constraint, since it needs
-- a live join against slots/slot_memberships).

CREATE TABLE bump_challenges (
    nonce uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id uuid NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    issued_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    CHECK (expires_at > issued_at)
);

CREATE INDEX bump_challenges_slot_user_idx ON bump_challenges (slot_id, user_id);

CREATE TABLE bump_submissions (
    slot_id uuid NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    submitter_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    counterpart_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    challenge_nonce uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (slot_id, submitter_id, counterpart_id),
    CHECK (submitter_id <> counterpart_id)
);

-- Verified (mutual) confirmation: exactly one row per Slot/pair, created
-- only once both directions of bump_submissions exist. user_lo_id/user_hi_id
-- normalize the unordered pair (user_lo_id is always the lexicographically
-- smaller uuid) so whichever side completes the pair second cannot create a
-- second row.
CREATE TABLE bump_confirmations (
    slot_id uuid NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    user_lo_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    user_hi_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    confirmed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (slot_id, user_lo_id, user_hi_id),
    CHECK (user_lo_id < user_hi_id)
);

CREATE INDEX bump_confirmations_user_lo_idx ON bump_confirmations (user_lo_id, confirmed_at DESC);
CREATE INDEX bump_confirmations_user_hi_idx ON bump_confirmations (user_hi_id, confirmed_at DESC);

-- Immutable per-user audit trail of reliability-affecting events. Only
-- BUMP_VERIFIED exists in this foundation block; future event types
-- (moderation penalties, forgiveness credits per README §6.26) extend the
-- CHECK constraint in a later migration rather than reusing this one.
CREATE TABLE reliability_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    event_type text NOT NULL,
    slot_id uuid REFERENCES slots(id) ON DELETE SET NULL,
    counterpart_id uuid REFERENCES app_users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (event_type IN ('BUMP_VERIFIED'))
);

CREATE INDEX reliability_events_user_created_idx ON reliability_events (user_id, created_at DESC);

-- Aggregated, cheap-to-read summary derived from reliability_events.
-- verified_bump_count is the private exact figure (README "private ...
-- reliability bands": a user sees their own precise count); band is the
-- coarse public figure other surfaces may show. Both live in one row kept
-- current by the same transaction that inserts a reliability_events row.
CREATE TABLE user_reliability (
    user_id uuid PRIMARY KEY REFERENCES app_users(id) ON DELETE CASCADE,
    verified_bump_count integer NOT NULL DEFAULT 0,
    band text NOT NULL DEFAULT 'NEW',
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (verified_bump_count >= 0),
    CHECK (band IN ('NEW','BUILDING','RELIABLE','TRUSTED'))
);

REVOKE ALL ON TABLE bump_challenges, bump_submissions, bump_confirmations, reliability_events, user_reliability FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE bump_challenges, bump_submissions, bump_confirmations, reliability_events, user_reliability FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE bump_challenges, bump_submissions, bump_confirmations, reliability_events, user_reliability FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT SELECT, INSERT, UPDATE ON TABLE bump_challenges TO linkup_api;
        GRANT SELECT, INSERT ON TABLE bump_submissions TO linkup_api;
        GRANT SELECT, INSERT ON TABLE bump_confirmations TO linkup_api;
        GRANT SELECT, INSERT ON TABLE reliability_events TO linkup_api;
        GRANT SELECT, INSERT, UPDATE ON TABLE user_reliability TO linkup_api;
    END IF;
END
$$;
