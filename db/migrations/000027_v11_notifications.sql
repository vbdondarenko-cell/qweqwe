-- LinkUp v1.1 Notifications foundation (README §6.8).
-- Canonical delivery flow: domain transaction -> transactional outbox (already
-- exists: db/migrations/000014_v11_realtime_outbox.sql) -> notification
-- projector/worker -> dedupe -> TTL -> quiet-hours -> frequency-cap ->
-- FCM/APNs adapter (already exists: internal/push.Service.NotifyUser).
-- This migration adds the two tables in between: durable per-user
-- preferences, and the notification record itself (the in-app/domain truth,
-- independent of whether push delivery is configured or succeeds).

CREATE TABLE notification_preferences (
    user_id uuid PRIMARY KEY REFERENCES app_users(id) ON DELETE CASCADE,
    social_enabled boolean NOT NULL DEFAULT true,
    event_enabled boolean NOT NULL DEFAULT true,
    recommendation_enabled boolean NOT NULL DEFAULT true,
    promo_enabled boolean NOT NULL DEFAULT true,
    -- Default quiet hours 23:00-08:00 local time (README §6.8), expressed as
    -- minute-of-day so the DST-sensitive part (resolving "local time" for a
    -- given instant) stays in application code against a real IANA zone,
    -- not encoded as a fixed UTC offset here.
    quiet_hours_start_minute integer NOT NULL DEFAULT 1380,
    quiet_hours_end_minute integer NOT NULL DEFAULT 480,
    timezone_name text NOT NULL DEFAULT 'UTC',
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (quiet_hours_start_minute BETWEEN 0 AND 1439),
    CHECK (quiet_hours_end_minute BETWEEN 0 AND 1439),
    CHECK (char_length(timezone_name) BETWEEN 1 AND 80)
);

CREATE TABLE notification_deliveries (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    notification_type text NOT NULL,
    -- Dedupe is per logical notification (README §6.8): the same domain
    -- outbox event can never produce more than one notification_deliveries
    -- row, even if the projector reprocesses it after a crash/restart.
    source_event_id uuid NOT NULL UNIQUE REFERENCES domain_outbox_events(event_id),
    slot_id uuid REFERENCES slots(id) ON DELETE SET NULL,
    deep_link text NOT NULL,
    title text NOT NULL,
    body text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    -- PENDING is transient (should not normally be observed at rest; the
    -- projector decides and writes the final outcome in the same statement
    -- that inserts the row) but kept as an explicit state rather than NULL
    -- so a stuck row is a visible, queryable anomaly rather than silence.
    push_outcome text NOT NULL DEFAULT 'PENDING',
    delivered_at timestamptz,
    CHECK (char_length(notification_type) BETWEEN 1 AND 32),
    CHECK (char_length(deep_link) BETWEEN 1 AND 200),
    CHECK (char_length(title) BETWEEN 1 AND 120),
    CHECK (char_length(body) BETWEEN 1 AND 240),
    CHECK (push_outcome IN (
        'PENDING','SENT','EXPIRED','SUPPRESSED_PREFERENCE','SUPPRESSED_QUIET_HOURS',
        'SUPPRESSED_FREQUENCY_CAP','SKIPPED_NO_DEVICE','SEND_FAILED'
    ))
);

CREATE INDEX notification_deliveries_user_created_idx
    ON notification_deliveries (user_id, created_at DESC, id DESC);
-- Frequency-cap evaluation counts recent SENT deliveries for a user within a
-- capped category; this index serves that lookup without a full table scan.
CREATE INDEX notification_deliveries_user_type_created_idx
    ON notification_deliveries (user_id, notification_type, created_at DESC);

REVOKE ALL ON TABLE notification_preferences, notification_deliveries FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE notification_preferences, notification_deliveries FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE notification_preferences, notification_deliveries FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT SELECT, INSERT, UPDATE ON TABLE notification_preferences TO linkup_api;
        GRANT SELECT, INSERT, UPDATE ON TABLE notification_deliveries TO linkup_api;
    END IF;
END
$$;
