DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
          AND tablename <> 'spatial_ref_sys'
    LOOP
        EXECUTE format('DROP TABLE IF EXISTS public.%I CASCADE;', r.tablename);
    END LOOP;
END $$;
