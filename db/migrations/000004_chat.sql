CREATE TABLE slot_messages (
    id uuid PRIMARY KEY,
    slot_id uuid NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    author_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    body text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (char_length(btrim(body)) BETWEEN 1 AND 2000)
);

CREATE INDEX slot_messages_slot_recent_idx
    ON slot_messages (slot_id, created_at DESC, id DESC);

CREATE OR REPLACE FUNCTION linkup_purge_terminal_slot_messages()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.state IN ('COMPLETED', 'CANCELLED', 'EXPIRED', 'MODERATED')
       AND OLD.state IS DISTINCT FROM NEW.state THEN
        DELETE FROM slot_messages WHERE slot_id = NEW.id;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER slots_terminal_message_purge
AFTER UPDATE OF state ON slots
FOR EACH ROW
WHEN (
    OLD.state IS DISTINCT FROM NEW.state
    AND NEW.state IN ('COMPLETED', 'CANCELLED', 'EXPIRED', 'MODERATED')
)
EXECUTE FUNCTION linkup_purge_terminal_slot_messages();
