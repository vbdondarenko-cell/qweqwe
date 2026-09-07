-- Forward-only v1.0 deployment hardening.
-- The runtime role is environment/deployment configuration, so this migration
-- is intentionally conditional and remains a no-op where the role is absent.
-- The Go API must never depend on BYPASSRLS or schema-creation privileges.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        ALTER ROLE linkup_api NOBYPASSRLS NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION;
        REVOKE CREATE ON SCHEMA public FROM linkup_api;
    END IF;
END
$$;
