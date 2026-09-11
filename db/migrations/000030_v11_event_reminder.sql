-- LinkUp v1.1 EVENT_REMINDER (README §6.8 canonical notification type).
--
-- Unlike every other notification type shipped so far (§45-§48), a
-- reminder is not triggered by a domain mutation: it is triggered by wall
-- clock time crossing a threshold relative to slots.start_at. There is
-- nothing for a DB trigger to fire on. notification_deliveries.
-- source_event_id is a NOT NULL UNIQUE FK into domain_outbox_events
-- (migration 000027), so rather than relaxing that constraint (which every
-- other notification type still relies on for its dedupe boundary), a
-- time-based scanner (internal/postgres.ReminderScanner) emits a genuine
-- synthetic domain_outbox_events row per recipient via the existing
-- linkup_enqueue_outbox function (migration 000014, now callable by
-- linkup_api per migration 000029) — the same canonical path every trigger
-- already uses. The existing dedupe/connector-cursor/notification pipeline
-- then needs no further changes at all.
--
-- slot_reminder_emissions is that scanner's own idempotency boundary:
-- "has a reminder already been queued for this Slot" is a fact about the
-- Slot, checked and claimed once via this table's primary key, independent
-- of and prior to emitting any per-recipient outbox event.
CREATE TABLE slot_reminder_emissions (
    slot_id uuid PRIMARY KEY REFERENCES slots(id) ON DELETE CASCADE,
    emitted_at timestamptz NOT NULL DEFAULT now()
);

REVOKE ALL ON TABLE slot_reminder_emissions FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE slot_reminder_emissions FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE slot_reminder_emissions FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT SELECT, INSERT ON TABLE slot_reminder_emissions TO linkup_api;
    END IF;
END
$$;
