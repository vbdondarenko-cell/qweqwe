-- Forward-only Supabase boundary hardening for LinkUp v1.0.
-- Android never talks to Supabase Data API directly; the Go API is the sole
-- application-facing data authority. Supabase-managed PostGIS objects in
-- public are owned/granted by supabase_admin, so object-level REVOKE from the
-- project postgres role cannot reliably remove those owner-issued grants.
-- Remove public-schema USAGE from client roles instead. Without schema USAGE,
-- anon/authenticated cannot resolve or access any object in public even when a
-- managed extension retains an object ACL entry for those roles.

REVOKE USAGE ON SCHEMA public FROM PUBLIC;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE USAGE ON SCHEMA public FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE USAGE ON SCHEMA public FROM authenticated;
    END IF;

    -- Preserve the roles LinkUp and Supabase administration actually use.
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT USAGE ON SCHEMA public TO linkup_api;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'service_role') THEN
        GRANT USAGE ON SCHEMA public TO service_role;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'postgres') THEN
        GRANT USAGE ON SCHEMA public TO postgres;
    END IF;
END
$$;
