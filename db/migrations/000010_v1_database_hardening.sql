-- Forward-only database hardening for the active v1.0 release.
-- Keep canonical query indexes non-duplicated, cover current foreign keys,
-- and close Supabase/PostGIS objects that are not application-facing APIs.

DROP INDEX IF EXISTS public.slot_messages_slot_recent_idx;

CREATE INDEX IF NOT EXISTS slot_messages_author_id_idx
    ON public.slot_messages (author_id);

CREATE INDEX IF NOT EXISTS monetization_referrals_referral_code_idx
    ON public.monetization_referrals (referral_code);

CREATE INDEX IF NOT EXISTS monetization_referrals_qualifying_receipt_id_idx
    ON public.monetization_referrals (qualifying_receipt_id);

CREATE INDEX IF NOT EXISTS monetization_referral_milestone_awards_triggering_invitee_idx
    ON public.monetization_referral_milestone_awards (triggering_invitee_id);

DO $$
BEGIN
    IF to_regclass('public.spatial_ref_sys') IS NOT NULL THEN
        REVOKE ALL ON TABLE public.spatial_ref_sys FROM PUBLIC;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
            REVOKE ALL ON TABLE public.spatial_ref_sys FROM anon;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
            REVOKE ALL ON TABLE public.spatial_ref_sys FROM authenticated;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
            GRANT SELECT ON TABLE public.spatial_ref_sys TO linkup_api;
        END IF;
    END IF;
END
$$;

DO $$
BEGIN
    IF to_regprocedure('public.st_estimatedextent(text,text)') IS NOT NULL THEN
        REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text) FROM PUBLIC;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text) FROM anon;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text) FROM authenticated;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
            GRANT EXECUTE ON FUNCTION public.st_estimatedextent(text,text) TO linkup_api;
        END IF;
    END IF;

    IF to_regprocedure('public.st_estimatedextent(text,text,text)') IS NOT NULL THEN
        REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text) FROM PUBLIC;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text) FROM anon;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text) FROM authenticated;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
            GRANT EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text) TO linkup_api;
        END IF;
    END IF;

    IF to_regprocedure('public.st_estimatedextent(text,text,text,boolean)') IS NOT NULL THEN
        REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text,boolean) FROM PUBLIC;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text,boolean) FROM anon;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
            REVOKE EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text,boolean) FROM authenticated;
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
            GRANT EXECUTE ON FUNCTION public.st_estimatedextent(text,text,text,boolean) TO linkup_api;
        END IF;
    END IF;
END
$$;
