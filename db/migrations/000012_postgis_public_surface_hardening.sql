-- Forward-only repair for PostGIS privileges that drifted after 000010.
-- LinkUp clients must reach application data only through the Go API role.
-- Keep PostGIS metadata/functions unavailable to Supabase anon/authenticated roles.

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
