-- Harden v1.1 capability registry privileges after environments with broad
-- default grants may have granted the runtime API role write/sequence/function access.

REVOKE ALL ON TABLE capability_registry FROM PUBLIC;
REVOKE ALL ON SEQUENCE capability_registry_revision_seq FROM PUBLIC;
REVOKE ALL ON FUNCTION linkup_capability_registry_bump_revision() FROM PUBLIC;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
        REVOKE ALL ON TABLE capability_registry FROM anon;
        REVOKE ALL ON SEQUENCE capability_registry_revision_seq FROM anon;
        REVOKE ALL ON FUNCTION linkup_capability_registry_bump_revision() FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
        REVOKE ALL ON TABLE capability_registry FROM authenticated;
        REVOKE ALL ON SEQUENCE capability_registry_revision_seq FROM authenticated;
        REVOKE ALL ON FUNCTION linkup_capability_registry_bump_revision() FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='linkup_api') THEN
        REVOKE ALL ON TABLE capability_registry FROM linkup_api;
        GRANT SELECT ON TABLE capability_registry TO linkup_api;
        REVOKE ALL ON SEQUENCE capability_registry_revision_seq FROM linkup_api;
        REVOKE ALL ON FUNCTION linkup_capability_registry_bump_revision() FROM linkup_api;
    END IF;
END
$$;
