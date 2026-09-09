-- LinkUp v1.1 City Context / canonical place privilege hardening.
-- Runtime resolves/reads locality data but does not administer provider data.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='linkup_api') THEN
        REVOKE ALL ON TABLE localities FROM linkup_api;
        REVOKE ALL ON TABLE canonical_places FROM linkup_api;
        REVOKE ALL ON TABLE city_context_locks FROM linkup_api;

        GRANT SELECT ON TABLE localities TO linkup_api;
        GRANT SELECT ON TABLE canonical_places TO linkup_api;
        GRANT SELECT, INSERT, UPDATE ON TABLE city_context_locks TO linkup_api;
    END IF;
END
$$;
