-- LinkUp v1.1 SELECTED visibility mode (README §4.3 "Selected people" —
-- the third of the six additional visibility modes, after PRIVATE (§52)
-- and LINKS (§55-§57)).
--
-- A SELECTED Slot is discoverable only to the specific individual users its
-- host chose at creation time — an explicit per-viewer allow-list, distinct
-- from LINKS's "anyone who is a real, mutual friend" rule. slots.visibility
-- has allowed 'SELECTED' since its original CHECK constraint (migration
-- 000002); this migration adds the allow-list table itself.
--
-- Same "discoverability gate, not access-control gate" contract PRIVATE and
-- LINKS already established: Request()/Join()/Approve() never reference
-- this table at all, so a stranger not on the list but handed the Slot ID
-- directly can still Request() it.
CREATE TABLE slot_selected_viewers (
    slot_id uuid NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    PRIMARY KEY (slot_id, user_id)
);

CREATE INDEX slot_selected_viewers_user_idx ON slot_selected_viewers (user_id);

REVOKE ALL ON TABLE slot_selected_viewers FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE slot_selected_viewers FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE slot_selected_viewers FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        -- INSERT/SELECT only in this block: the allow-list is set once at
        -- creation (CreateDraft) and read by discovery queries. No UPDATE
        -- or DELETE grant yet — editing an existing SELECTED Slot's
        -- allow-list is deferred, matching EditInput's current scope (see
        -- normalizeEdit's explicit rejection of SELECTED as an edit target).
        GRANT SELECT, INSERT ON TABLE slot_selected_viewers TO linkup_api;
    END IF;
END
$$;
