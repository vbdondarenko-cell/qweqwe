-- Fixes a real, live privilege gap introduced in migration 000032
-- (v11_selected_visibility.sql): that migration granted linkup_api only
-- SELECT/INSERT on slot_selected_viewers, reasoning (correctly, at the
-- time) that the allow-list was write-once at creation and never edited.
-- A later block (see IMPLEMENTATION_STATUS.md worklog) added the ability
-- to replace an existing SELECTED Slot's allow-list while it is still
-- DRAFT, which does `DELETE FROM slot_selected_viewers WHERE slot_id=$1`
-- before re-inserting the new list -- but no migration ever granted
-- linkup_api DELETE on this table. Every integration test for that
-- feature passed anyway because this repository's test harness connects
-- as the postgres superuser, not as linkup_api, so the missing grant was
-- never actually exercised until now. Found by auditing existing grants
-- before starting new geometry-visibility work, not by a production
-- failure -- but it is a real gap that would have surfaced the first time
-- this code path ran against a connection actually using the linkup_api
-- role (e.g. production Supabase).

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT DELETE ON TABLE slot_selected_viewers TO linkup_api;
    END IF;
END
$$;
