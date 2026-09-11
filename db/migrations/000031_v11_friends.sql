-- LinkUp v1.1 Friends/Links domain (README §4.3 "Friends/Links" visibility
-- mode prerequisite; §5.4 "My LINKs dashboard"; §6.8 FRIEND_REQUEST/
-- FRIEND_ACCEPTED notification types; §6.13 "Links Graph" foundation).
--
-- A friend request is a directed proposal that resolves into a symmetric
-- friendship once accepted. friendships canonicalizes the unordered pair
-- to a stable (user_lo_id, user_hi_id) ordering, the same technique
-- bump_confirmations (000028) already uses for a pairwise relationship, so
-- either side reading "are we friends" sees the same single row.
--
-- Every mutation (internal/friend.Store / internal/postgres/friend_store.go)
-- is designed to be naturally idempotent through these constraints — a
-- unique partial index on the PENDING state, and conditional UPDATEs keyed
-- on that state — rather than the generic mutation_idempotency table the
-- Slot domain uses; a repeat Request/Accept/Reject/Cancel/Remove call has
-- an obviously safe no-op or already-current-state answer, mirroring how
-- internal/blocklist's Block/Unblock already work in this codebase.

CREATE TABLE friend_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    target_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'PENDING',
    created_at timestamptz NOT NULL DEFAULT now(),
    responded_at timestamptz,
    CHECK (requester_id <> target_id),
    CHECK (status IN ('PENDING','ACCEPTED','REJECTED','CANCELLED'))
);

-- At most one PENDING request per ordered (requester,target) pair. The
-- reverse direction is deliberately NOT constrained here — A requesting B
-- while B already has a PENDING request to A is a real, valid state
-- (resolved as an immediate mutual match by application code, not rejected
-- by this index).
CREATE UNIQUE INDEX friend_requests_pending_pair_idx
    ON friend_requests (requester_id, target_id)
    WHERE status = 'PENDING';

CREATE INDEX friend_requests_target_pending_idx
    ON friend_requests (target_id, created_at DESC) WHERE status = 'PENDING';
CREATE INDEX friend_requests_requester_pending_idx
    ON friend_requests (requester_id, created_at DESC) WHERE status = 'PENDING';

CREATE TABLE friendships (
    user_lo_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    user_hi_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_lo_id, user_hi_id),
    CHECK (user_lo_id < user_hi_id)
);

CREATE INDEX friendships_hi_created_idx ON friendships (user_hi_id, created_at DESC);

INSERT INTO capability_registry (capability_key, enabled, reason) VALUES
    ('friends', false, 'v1.1 rollout gate');

REVOKE ALL ON TABLE friend_requests, friendships FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE friend_requests, friendships FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE friend_requests, friendships FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT SELECT, INSERT, UPDATE ON TABLE friend_requests TO linkup_api;
        GRANT SELECT, INSERT, DELETE ON TABLE friendships TO linkup_api;
        -- linkup_enqueue_outbox is already GRANTed to linkup_api as of
        -- 000029 (for ReminderScanner) — friend_store.go reuses that same
        -- grant to emit friend.requested/friend.accepted events in the
        -- same transaction, needing no further grant here.
    END IF;
END
$$;
