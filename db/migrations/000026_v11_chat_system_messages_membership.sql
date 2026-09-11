-- LinkUp v1.1 Coordination Chat V2 continuation of migration 000025.
-- README §6.7 "system messages for important Slot lifecycle events":
-- extends the SYSTEM message allow-list with the two membership-change
-- events (a member joining and a member leaving, by any access mode or
-- removal path) alongside the existing SLOT_STARTED. No other shape of
-- slot_messages changes; the 000025 CHECK already covers these rows.

ALTER TABLE slot_messages DROP CONSTRAINT slot_messages_system_event_type_check;

ALTER TABLE slot_messages
    ADD CONSTRAINT slot_messages_system_event_type_check CHECK (
        system_event_type IS NULL
        OR system_event_type IN ('SLOT_STARTED','MEMBER_JOINED','MEMBER_LEFT')
    );
