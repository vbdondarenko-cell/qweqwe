-- LinkUp v1.1 privacy-safe city activity invalidation channel.
-- Emits locality-level invalidations only; no raw user/device coordinates are stored.

CREATE FUNCTION linkup_emit_city_slot_outbox() RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
DECLARE
    v_old_locality uuid;
    v_new_locality uuid;
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.canonical_place_id IS NOT NULL THEN
        SELECT locality_id INTO v_old_locality
        FROM public.canonical_places
        WHERE id = OLD.canonical_place_id;
    END IF;

    IF NEW.canonical_place_id IS NOT NULL THEN
        SELECT locality_id INTO v_new_locality
        FROM public.canonical_places
        WHERE id = NEW.canonical_place_id;
    END IF;

    IF v_old_locality IS NULL AND v_new_locality IS NULL THEN
        RETURN NEW;
    END IF;

    PERFORM public.linkup_enqueue_outbox(
        'city.slot_changed',
        'slot',
        NEW.id,
        NEW.host_id,
        NEW.id,
        jsonb_build_object(
            'localityId', v_new_locality,
            'previousLocalityId', v_old_locality,
            'state', NEW.state,
            'previousState', CASE WHEN TG_OP='UPDATE' THEN OLD.state ELSE NULL END,
            'visibility', NEW.visibility,
            'previousVisibility', CASE WHEN TG_OP='UPDATE' THEN OLD.visibility ELSE NULL END,
            'version', NEW.version
        )
    );
    RETURN NEW;
END;
$$;

CREATE TRIGGER slots_city_outbox_after_write
AFTER INSERT OR UPDATE ON slots
FOR EACH ROW EXECUTE FUNCTION linkup_emit_city_slot_outbox();

REVOKE ALL ON FUNCTION linkup_emit_city_slot_outbox() FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
        REVOKE ALL ON FUNCTION linkup_emit_city_slot_outbox() FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
        REVOKE ALL ON FUNCTION linkup_emit_city_slot_outbox() FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='linkup_api') THEN
        REVOKE ALL ON FUNCTION linkup_emit_city_slot_outbox() FROM linkup_api;
    END IF;
END
$$;
