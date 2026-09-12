-- Two real gaps closed together (README §6.8):
--
-- 1. notification_deliveries.source_event_id was UNIQUE on its own, which
--    meant at most ONE recipient could ever be notified per outbox event --
--    fine for every event type wired so far (each already had exactly one
--    recipient: the approver's requester, the reminder's own participant,
--    etc.), but wrong for a chat message, which can have many simultaneous
--    recipients (every other accepted participant/host of the Slot).
--    ReminderScanner's own existing per-recipient-event-emission pattern
--    (see its own comment in internal/postgres/reminder_scanner.go) already
--    works fine under the new, looser (source_event_id, user_id) pair --
--    every event it emits already has a single, distinct user_id, so this
--    change only ever grants additional flexibility, never removes any.
--
-- 2. push_outcome's CHECK constraint gains SUPPRESSED_GROUPED (README
--    §6.8's grouping/collapse rule -- see internal/notification.Decide and
--    Type.Groupable).

ALTER TABLE notification_deliveries DROP CONSTRAINT notification_deliveries_source_event_id_key;
ALTER TABLE notification_deliveries ADD CONSTRAINT notification_deliveries_source_event_user_key UNIQUE (source_event_id, user_id);

ALTER TABLE notification_deliveries DROP CONSTRAINT notification_deliveries_push_outcome_check;
ALTER TABLE notification_deliveries ADD CONSTRAINT notification_deliveries_push_outcome_check CHECK (push_outcome IN (
    'PENDING','SENT','EXPIRED','SUPPRESSED_PREFERENCE','SUPPRESSED_QUIET_HOURS',
    'SUPPRESSED_FREQUENCY_CAP','SUPPRESSED_GROUPED','SKIPPED_NO_DEVICE','SEND_FAILED'
));
