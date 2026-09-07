-- Forward-only v1.0 chat send idempotency.
-- Every new chat send carries a client-generated Idempotency-Key. Existing rows
-- are backfilled from their server message id so the column can be NOT NULL.

ALTER TABLE slot_messages
    ADD COLUMN idempotency_key text;

UPDATE slot_messages
SET idempotency_key = id::text
WHERE idempotency_key IS NULL;

ALTER TABLE slot_messages
    ALTER COLUMN idempotency_key SET NOT NULL;

ALTER TABLE slot_messages
    ADD CONSTRAINT slot_messages_idempotency_key_len
    CHECK (char_length(idempotency_key) BETWEEN 16 AND 128);

CREATE UNIQUE INDEX slot_messages_author_idempotency_uq
    ON slot_messages (slot_id, author_id, idempotency_key);
