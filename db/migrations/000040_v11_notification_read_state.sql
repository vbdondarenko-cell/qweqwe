-- In-app notification inbox (README §6.8's own "the notification record
-- itself (the in-app/domain truth, independent of whether push delivery is
-- configured or succeeds)" -- 000027's comment already staked this claim,
-- but nothing ever read notification_deliveries back for the user; only
-- the push-delivery pipeline wrote to it). This migration adds exactly the
-- one thing missing to serve a real in-app notifications list matching the
-- frozen design reference's NotificationsPanel: whether the user has
-- actually seen a delivery in-app, independent of push_outcome (which is
-- about the push channel, not in-app visibility).

ALTER TABLE notification_deliveries ADD COLUMN read_at timestamptz;

-- Serves both "how many unread" and "list my unread first" without a full
-- table scan; the existing notification_deliveries_user_created_idx already
-- covers the general (user_id, created_at DESC) listing case.
CREATE INDEX notification_deliveries_user_unread_idx
    ON notification_deliveries (user_id, created_at DESC)
    WHERE read_at IS NULL;
