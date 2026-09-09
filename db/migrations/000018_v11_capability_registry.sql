-- LinkUp v1.1 capability rollout registry.
-- All future capabilities are disabled by default. Mobile clients can only read
-- evaluated state through the Go API; PostgreSQL remains server-authoritative.

CREATE SEQUENCE capability_registry_revision_seq AS bigint START WITH 1 INCREMENT BY 1 NO CYCLE;

CREATE TABLE capability_registry (
    capability_key text PRIMARY KEY,
    enabled boolean NOT NULL DEFAULT false,
    revision bigint NOT NULL DEFAULT nextval('capability_registry_revision_seq'),
    scope_type text NOT NULL DEFAULT 'ALL',
    scope_user_ids uuid[] NOT NULL DEFAULT '{}'::uuid[],
    effective_at timestamptz,
    reason text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (capability_key ~ '^[a-z][a-z0-9_]{1,63}$'),
    CHECK (revision > 0),
    CHECK (scope_type IN ('ALL','USER_ALLOWLIST')),
    CHECK (
        (scope_type = 'ALL' AND cardinality(scope_user_ids) = 0)
        OR
        (scope_type = 'USER_ALLOWLIST' AND cardinality(scope_user_ids) > 0)
    ),
    CHECK (char_length(reason) <= 500)
);

CREATE FUNCTION linkup_capability_registry_bump_revision() RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
BEGIN
    NEW.revision := nextval('public.capability_registry_revision_seq');
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER capability_registry_revision_before_update
BEFORE UPDATE ON capability_registry
FOR EACH ROW EXECUTE FUNCTION linkup_capability_registry_bump_revision();

INSERT INTO capability_registry (capability_key, enabled, reason) VALUES
    ('realtime', false, 'v1.1 rollout gate'),
    ('city_context', false, 'v1.1 rollout gate'),
    ('map', false, 'v1.1 rollout gate'),
    ('waitlist', false, 'v1.1 rollout gate'),
    ('chat_v2', false, 'v1.1 rollout gate'),
    ('notifications', false, 'v1.1 rollout gate'),
    ('bump', false, 'v1.1 rollout gate'),
    ('city_bpm', false, 'v1.1 rollout gate'),
    ('swarms', false, 'v1.1 rollout gate'),
    ('fly_now', false, 'v1.1 rollout gate'),
    ('fly_travel', false, 'v1.1 rollout gate'),
    ('fly_motion', false, 'v1.1 rollout gate');

CREATE INDEX capability_registry_enabled_revision_idx
    ON capability_registry (enabled, revision, capability_key);

REVOKE ALL ON TABLE capability_registry FROM PUBLIC;
REVOKE ALL ON SEQUENCE capability_registry_revision_seq FROM PUBLIC;
REVOKE ALL ON FUNCTION linkup_capability_registry_bump_revision() FROM PUBLIC;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
        REVOKE ALL ON TABLE capability_registry FROM anon;
        REVOKE ALL ON SEQUENCE capability_registry_revision_seq FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
        REVOKE ALL ON TABLE capability_registry FROM authenticated;
        REVOKE ALL ON SEQUENCE capability_registry_revision_seq FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='linkup_api') THEN
        GRANT SELECT ON TABLE capability_registry TO linkup_api;
    END IF;
END
$$;
