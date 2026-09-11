-- Fixes a real production-blocking regression discovered while verifying
-- this session's work: migration 000014 granted linkup_api exactly what a
-- connector consumer needs on connector_cursors/connector_delivery_receipts
-- (SELECT/INSERT/UPDATE and SELECT/INSERT respectively), but migration
-- 000021's hardening pass revoked ALL of linkup_api's privileges on both
-- tables and never restored them, on the stated assumption that "connector
-- checkpoint writes belong to a dedicated future worker role, never to the
-- API runtime role". That dedicated worker role/process was never actually
-- created anywhere in this repository: cmd/api is still the only server
-- binary, and it is the same in-process ticker (NotificationProjector,
-- wired in cmd/api/main.go since migration 000027 / IMPLEMENTATION_STATUS
-- §45) that calls RealtimeOutboxStore.Cursor/Checkpoint under the linkup_api
-- role in real deployment.
--
-- Confirmed empirically, not just by reading the SQL: connecting as a local
-- linkup_api role after applying migrations through 000028 and attempting
-- `INSERT INTO connector_cursors ...` fails with
-- "permission denied for table connector_cursors". Every test run in this
-- repository's history connected as the Postgres superuser, which bypasses
-- role grants entirely, so this was never caught by any executed test
-- despite the Notifications feature (§45-§48) being verified green.
--
-- Until a genuinely separate worker process/role exists, linkup_api must
-- hold these grants for the shipped in-process connector to function at
-- all. This migration restores exactly what 000014 originally granted, plus
-- EXECUTE on linkup_enqueue_outbox: the new EVENT_REMINDER time-based
-- scanner (migration 000030) is the first in-process caller that needs to
-- enqueue a genuinely new outbox event rather than only read/checkpoint
-- existing ones, so it needs this too.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT SELECT, INSERT, UPDATE ON TABLE connector_cursors TO linkup_api;
        GRANT SELECT, INSERT ON TABLE connector_delivery_receipts TO linkup_api;
        GRANT EXECUTE ON FUNCTION linkup_enqueue_outbox(text,text,uuid,uuid,uuid,jsonb) TO linkup_api;
    END IF;
END
$$;
