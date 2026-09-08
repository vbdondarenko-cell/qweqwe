-- LinkUp v1.1 hardening after production advisor review.
-- Keep the realtime receipt FK covered and reassert that Supabase Data API roles
-- cannot invoke PostGIS SECURITY DEFINER extent helpers directly.

CREATE INDEX connector_delivery_receipts_event_id_idx
    ON connector_delivery_receipts (event_id);

DO $$
BEGIN
    IF to_regprocedure('public.st_estimatedextent(text,text)') IS NOT NULL THEN
        REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text) FROM PUBLIC;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text) FROM anon;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text) FROM authenticated;
        END IF;
    END IF;

    IF to_regprocedure('public.st_estimatedextent(text,text,text)') IS NOT NULL THEN
        REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text) FROM PUBLIC;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text) FROM anon;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text) FROM authenticated;
        END IF;
    END IF;

    IF to_regprocedure('public.st_estimatedextent(text,text,text,boolean)') IS NOT NULL THEN
        REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text,boolean) FROM PUBLIC;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text,boolean) FROM anon;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text,boolean) FROM authenticated;
        END IF;
    END IF;
END
$$;
