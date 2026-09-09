-- LinkUp v1.1 realtime outbox privilege hardening.
-- The HTTP API consumes viewer events read-only. Connector checkpoint writes
-- belong to a dedicated future worker role, never to the API runtime role.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='linkup_api') THEN
        REVOKE ALL ON TABLE domain_outbox_events FROM linkup_api;
        REVOKE ALL ON TABLE connector_cursors FROM linkup_api;
        REVOKE ALL ON TABLE connector_delivery_receipts FROM linkup_api;
        REVOKE ALL ON SEQUENCE domain_outbox_events_sequence_seq FROM linkup_api;
        REVOKE ALL ON FUNCTION linkup_enqueue_outbox(text,text,uuid,uuid,uuid,jsonb) FROM linkup_api;
        REVOKE ALL ON FUNCTION linkup_emit_canonical_outbox() FROM linkup_api;

        GRANT SELECT ON TABLE domain_outbox_events TO linkup_api;
    END IF;
END
$$;
