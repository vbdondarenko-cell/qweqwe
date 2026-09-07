BEGIN;

-- v1.0 host request queue is ordered oldest-first and bounded to 100 rows.
-- The PK (slot_id,user_id) does not support that ordering efficiently.
CREATE INDEX IF NOT EXISTS slot_requests_slot_created_user_idx
    ON slot_requests (slot_id, created_at ASC, user_id ASC);

-- Chat reads are bounded recent-thread scans ordered newest-first internally.
CREATE INDEX IF NOT EXISTS slot_messages_slot_created_id_idx
    ON slot_messages (slot_id, created_at DESC, id DESC);

-- Active session lookup is by token hash; the UNIQUE constraint already covers it.
-- Keep this migration deliberately narrow and forward-only.

COMMIT;
