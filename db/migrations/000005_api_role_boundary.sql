-- Forward-only v1.0 security migration.
-- The Go API is the only application-facing auth/domain/data authority.
-- Supabase Data API client roles must not be able to bypass it by querying
-- LinkUp canonical tables or executing LinkUp database functions directly.
--
-- This migration intentionally does NOT enable FORCE RLS: the Go runtime DB
-- role/ownership model is deployment configuration and must be verified before
-- any RLS policy rollout. Explicit privilege denial is safe for anon/authenticated
-- while preserving direct owner/application-role access.

REVOKE ALL ON TABLE
    app_users,
    user_sessions,
    password_reset_tokens,
    user_blocks,
    slots,
    mutation_idempotency,
    slot_requests,
    slot_memberships,
    slot_messages
FROM PUBLIC;

REVOKE ALL ON FUNCTION linkup_purge_terminal_slot_messages() FROM PUBLIC;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE
            app_users,
            user_sessions,
            password_reset_tokens,
            user_blocks,
            slots,
            mutation_idempotency,
            slot_requests,
            slot_memberships,
            slot_messages
        FROM anon;
        REVOKE ALL ON FUNCTION linkup_purge_terminal_slot_messages() FROM anon;
    END IF;

    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE
            app_users,
            user_sessions,
            password_reset_tokens,
            user_blocks,
            slots,
            mutation_idempotency,
            slot_requests,
            slot_memberships,
            slot_messages
        FROM authenticated;
        REVOKE ALL ON FUNCTION linkup_purge_terminal_slot_messages() FROM authenticated;
    END IF;
END
$$;

-- Guard future objects created by the same migration owner. These defaults are
-- additive defense; every future migration must still explicitly review its
-- exposure boundary.
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON FUNCTIONS FROM PUBLIC;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        EXECUTE 'ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM anon';
        EXECUTE 'ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON FUNCTIONS FROM anon';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        EXECUTE 'ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM authenticated';
        EXECUTE 'ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON FUNCTIONS FROM authenticated';
    END IF;
END
$$;
