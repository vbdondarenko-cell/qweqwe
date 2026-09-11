-- LinkUp v1.1 Coordination Chat V2 foundation.
-- README §6.7 "system messages for important Slot lifecycle events". Adds a
-- server-originated SYSTEM message kind alongside the existing USER kind.
-- SYSTEM rows have no author (nobody "said" them) and instead carry a typed
-- event plus an optional subject user, so clients render a localized string
-- rather than trusting server-composed free text. Existing zero-trace rules
-- are unchanged: the terminal purge trigger already deletes every row for a
-- Slot (slot_messages, not filtered by kind) on COMPLETED/CANCELLED/EXPIRED/
-- MODERATED, and the existing chat_message_created outbox trigger already
-- fires on any INSERT into slot_messages, so SYSTEM rows are realtime-visible
-- for free through the existing v1.1 outbox/viewer-feed path.

ALTER TABLE slot_messages DROP CONSTRAINT slot_messages_body_check;

ALTER TABLE slot_messages
    ALTER COLUMN author_id DROP NOT NULL,
    ALTER COLUMN body DROP NOT NULL,
    ALTER COLUMN idempotency_key DROP NOT NULL,
    ADD COLUMN kind text NOT NULL DEFAULT 'USER',
    ADD COLUMN system_event_type text,
    ADD COLUMN subject_user_id uuid REFERENCES app_users(id) ON DELETE SET NULL;

ALTER TABLE slot_messages
    ADD CONSTRAINT slot_messages_kind_check CHECK (kind IN ('USER','SYSTEM')),
    ADD CONSTRAINT slot_messages_system_event_type_check CHECK (
        system_event_type IS NULL OR system_event_type IN ('SLOT_STARTED')
    ),
    ADD CONSTRAINT slot_messages_kind_shape_check CHECK (
        (kind = 'USER'
            AND author_id IS NOT NULL
            AND idempotency_key IS NOT NULL
            AND system_event_type IS NULL
            AND body IS NOT NULL
            AND char_length(btrim(body)) BETWEEN 1 AND 2000)
        OR
        (kind = 'SYSTEM'
            AND author_id IS NULL
            AND idempotency_key IS NULL
            AND system_event_type IS NOT NULL
            AND body IS NULL)
    );
