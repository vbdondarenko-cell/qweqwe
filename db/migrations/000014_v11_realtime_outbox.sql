-- LinkUp v1.1 / Realtime City Network foundation.
-- Canonical social writes remain authoritative in PostgreSQL. Durable outbox
-- events are emitted in the SAME transaction by database triggers so a worker
-- can reconnect/catch up without treating transient push delivery as state.

CREATE TABLE domain_outbox_events (
    sequence bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id uuid NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    event_type text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    subject_user_id uuid REFERENCES app_users(id) ON DELETE SET NULL,
    slot_id uuid REFERENCES slots(id) ON DELETE SET NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    CHECK (char_length(event_type) BETWEEN 3 AND 96),
    CHECK (char_length(aggregate_type) BETWEEN 1 AND 48),
    CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX domain_outbox_events_slot_sequence_idx
    ON domain_outbox_events (slot_id, sequence) WHERE slot_id IS NOT NULL;
CREATE INDEX domain_outbox_events_subject_sequence_idx
    ON domain_outbox_events (subject_user_id, sequence) WHERE subject_user_id IS NOT NULL;
CREATE INDEX domain_outbox_events_type_sequence_idx
    ON domain_outbox_events (event_type, sequence);

CREATE TABLE connector_cursors (
    connector text PRIMARY KEY,
    last_sequence bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (connector ~ '^[a-z0-9][a-z0-9._-]{0,63}$'),
    CHECK (last_sequence >= 0)
);

CREATE TABLE connector_delivery_receipts (
    connector text NOT NULL REFERENCES connector_cursors(connector) ON DELETE CASCADE,
    event_id uuid NOT NULL REFERENCES domain_outbox_events(event_id) ON DELETE CASCADE,
    outcome text NOT NULL CHECK (outcome IN ('DELIVERED','SKIPPED')),
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (connector, event_id)
);

CREATE INDEX connector_delivery_receipts_processed_idx
    ON connector_delivery_receipts (processed_at, connector);

CREATE FUNCTION linkup_enqueue_outbox(
    p_event_type text,
    p_aggregate_type text,
    p_aggregate_id uuid,
    p_subject_user_id uuid,
    p_slot_id uuid,
    p_payload jsonb
) RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
BEGIN
    INSERT INTO public.domain_outbox_events (
        event_type, aggregate_type, aggregate_id, subject_user_id, slot_id, payload
    ) VALUES (
        p_event_type, p_aggregate_type, p_aggregate_id, p_subject_user_id, p_slot_id,
        COALESCE(p_payload, '{}'::jsonb)
    );
END;
$$;

CREATE FUNCTION linkup_emit_canonical_outbox() RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
DECLARE
    v_type text;
BEGIN
    IF TG_TABLE_NAME = 'slots' THEN
        IF TG_OP = 'INSERT' THEN
            PERFORM public.linkup_enqueue_outbox(
                'slot.created', 'slot', NEW.id, NEW.host_id, NEW.id,
                jsonb_build_object('state', NEW.state, 'version', NEW.version)
            );
        ELSIF TG_OP = 'UPDATE' THEN
            IF OLD.state IS DISTINCT FROM NEW.state THEN
                v_type := 'slot.state_changed';
            ELSE
                v_type := 'slot.updated';
            END IF;
            PERFORM public.linkup_enqueue_outbox(
                v_type, 'slot', NEW.id, NEW.host_id, NEW.id,
                jsonb_build_object(
                    'state', NEW.state,
                    'previousState', OLD.state,
                    'version', NEW.version
                )
            );
        END IF;
        RETURN NEW;
    END IF;

    IF TG_TABLE_NAME = 'slot_requests' THEN
        IF TG_OP = 'INSERT' THEN
            PERFORM public.linkup_enqueue_outbox(
                'slot.request_created', 'slot', NEW.slot_id, NEW.user_id, NEW.slot_id,
                jsonb_build_object('userId', NEW.user_id)
            );
            RETURN NEW;
        END IF;
        PERFORM public.linkup_enqueue_outbox(
            'slot.request_removed', 'slot', OLD.slot_id, OLD.user_id, OLD.slot_id,
            jsonb_build_object('userId', OLD.user_id)
        );
        RETURN OLD;
    END IF;

    IF TG_TABLE_NAME = 'slot_memberships' THEN
        IF TG_OP = 'INSERT' THEN
            PERFORM public.linkup_enqueue_outbox(
                'slot.membership_added', 'slot', NEW.slot_id, NEW.user_id, NEW.slot_id,
                jsonb_build_object('userId', NEW.user_id)
            );
            RETURN NEW;
        END IF;
        PERFORM public.linkup_enqueue_outbox(
            'slot.membership_removed', 'slot', OLD.slot_id, OLD.user_id, OLD.slot_id,
            jsonb_build_object('userId', OLD.user_id)
        );
        RETURN OLD;
    END IF;

    IF TG_TABLE_NAME = 'slot_messages' AND TG_OP = 'INSERT' THEN
        PERFORM public.linkup_enqueue_outbox(
            'slot.chat_message_created', 'message', NEW.id, NEW.author_id, NEW.slot_id,
            jsonb_build_object('messageId', NEW.id)
        );
        RETURN NEW;
    END IF;

    IF TG_TABLE_NAME = 'user_blocks' THEN
        IF TG_OP = 'INSERT' THEN
            PERFORM public.linkup_enqueue_outbox(
                'user.block_created', 'user', NEW.blocked_id, NEW.blocker_id, NULL,
                jsonb_build_object('blockedUserId', NEW.blocked_id)
            );
            RETURN NEW;
        END IF;
        PERFORM public.linkup_enqueue_outbox(
            'user.block_removed', 'user', OLD.blocked_id, OLD.blocker_id, NULL,
            jsonb_build_object('blockedUserId', OLD.blocked_id)
        );
        RETURN OLD;
    END IF;

    IF TG_TABLE_NAME = 'app_users' AND TG_OP = 'UPDATE' THEN
        PERFORM public.linkup_enqueue_outbox(
            'user.profile_changed', 'user', NEW.id, NEW.id, NULL, '{}'::jsonb
        );
        RETURN NEW;
    END IF;

    RAISE EXCEPTION 'unsupported outbox trigger source %.%', TG_TABLE_SCHEMA, TG_TABLE_NAME;
END;
$$;

CREATE TRIGGER slots_outbox_after_write
AFTER INSERT OR UPDATE ON slots
FOR EACH ROW EXECUTE FUNCTION linkup_emit_canonical_outbox();

CREATE TRIGGER slot_requests_outbox_after_write
AFTER INSERT OR DELETE ON slot_requests
FOR EACH ROW EXECUTE FUNCTION linkup_emit_canonical_outbox();

CREATE TRIGGER slot_memberships_outbox_after_write
AFTER INSERT OR DELETE ON slot_memberships
FOR EACH ROW EXECUTE FUNCTION linkup_emit_canonical_outbox();

CREATE TRIGGER slot_messages_outbox_after_insert
AFTER INSERT ON slot_messages
FOR EACH ROW EXECUTE FUNCTION linkup_emit_canonical_outbox();

CREATE TRIGGER user_blocks_outbox_after_write
AFTER INSERT OR DELETE ON user_blocks
FOR EACH ROW EXECUTE FUNCTION linkup_emit_canonical_outbox();

CREATE TRIGGER app_users_profile_outbox_after_update
AFTER UPDATE OF display_name, avatar_url, profile_visibility, language ON app_users
FOR EACH ROW
WHEN (
    OLD.display_name IS DISTINCT FROM NEW.display_name OR
    OLD.avatar_url IS DISTINCT FROM NEW.avatar_url OR
    OLD.profile_visibility IS DISTINCT FROM NEW.profile_visibility OR
    OLD.language IS DISTINCT FROM NEW.language
)
EXECUTE FUNCTION linkup_emit_canonical_outbox();

-- No Supabase client role may use the outbox as a side-channel. The Go API and
-- connector workers are the only application authorities allowed to consume it.
REVOKE ALL ON TABLE domain_outbox_events, connector_cursors, connector_delivery_receipts FROM PUBLIC;
REVOKE ALL ON FUNCTION linkup_enqueue_outbox(text,text,uuid,uuid,uuid,jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION linkup_emit_canonical_outbox() FROM PUBLIC;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE domain_outbox_events, connector_cursors, connector_delivery_receipts FROM anon;
        REVOKE ALL ON FUNCTION linkup_enqueue_outbox(text,text,uuid,uuid,uuid,jsonb) FROM anon;
        REVOKE ALL ON FUNCTION linkup_emit_canonical_outbox() FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE domain_outbox_events, connector_cursors, connector_delivery_receipts FROM authenticated;
        REVOKE ALL ON FUNCTION linkup_enqueue_outbox(text,text,uuid,uuid,uuid,jsonb) FROM authenticated;
        REVOKE ALL ON FUNCTION linkup_emit_canonical_outbox() FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT SELECT ON TABLE domain_outbox_events TO linkup_api;
        GRANT SELECT, INSERT, UPDATE ON TABLE connector_cursors TO linkup_api;
        GRANT SELECT, INSERT ON TABLE connector_delivery_receipts TO linkup_api;
    END IF;
END
$$;
