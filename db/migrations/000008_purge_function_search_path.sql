-- Forward-only v1.0 security hardening.
-- Pin the trigger function search_path so object resolution cannot be influenced
-- by caller/session search_path settings. The function remains non-public API.

ALTER FUNCTION public.linkup_purge_terminal_slot_messages()
    SET search_path = pg_catalog, public;
