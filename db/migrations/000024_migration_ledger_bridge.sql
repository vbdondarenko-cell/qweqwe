-- LinkUp canonical migration-ledger bridge.
-- Production migrations were historically applied through Supabase management
-- before the Go checksum ledger existed there. Backfill only immutable migrations
-- whose production application has been verified; checksum mismatches fail closed.

CREATE TABLE IF NOT EXISTS linkup_schema_migrations (
    name text PRIMARY KEY,
    checksum bytea NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now()
);

DO $$
DECLARE
    expected record;
    existing bytea;
BEGIN
    FOR expected IN
        SELECT * FROM (VALUES
            ('000001_accounts.sql', decode('52937dbe9915e9dbef8fe4812a8f43e806c9d9f26052438cfc0bfa9f59582ff0','hex')),
            ('000002_slots.sql', decode('dcdae136c6342ca114e131cbfc63d1aad2cc44c6bef9aa349878d90c797a70da','hex')),
            ('000003_approval.sql', decode('47f29c29af5a9117a079b1927153195f3e075ca5d43a87ccc663de916b666041','hex')),
            ('000004_chat.sql', decode('6ac040904a66202b8b15d687ee6ce6d5df3af76d1e7979f93b1f72de3dded49c','hex')),
            ('000005_api_role_boundary.sql', decode('0580549c9a5a5f2898eb89711a3a5aa044c650231ddde5d55990e44e201c22e7','hex')),
            ('000006_chat_idempotency.sql', decode('dcc73341de24b7c68a22ecac86270b9089dfdbd04abf5ba7fc66d42b2c848292','hex')),
            ('000007_v1_query_indexes.sql', decode('4c4109c614af784b3acb3f8fa89002278ace0ed2d10cc5e3865a05329fb92dc4','hex')),
            ('000008_purge_function_search_path.sql', decode('456c886e1dad80c14bf00e71690b63a1a3f3b721a4080d7b50678395498bfa3a','hex')),
            ('000009_linkup_plus_monetization.sql', decode('1230895938d8e28fd4149580de35c550c204c4e0ad60a6c3f07ed6de330447fc','hex')),
            ('000010_v1_database_hardening.sql', decode('6b4b7c8cf6e56264035ecf6f5a62f96bc0ef35a8a7279bb4ee598cd97292c8b4','hex')),
            ('000011_android_push_devices.sql', decode('0212a6d12382b1f8791e149ee0f3e54f3b21ebd2ddb2f3f3a7bdf7980d51c4dd','hex')),
            ('000012_postgis_public_surface_hardening.sql', decode('5f6d2532aa175f9f72cd9b401da7115a0752a3a2468ac385533fab95e24657b7','hex')),
            ('000013_disable_public_data_api_roles.sql', decode('6542fd763d1bb389ad228b6718b37df490a5e414076af0fda9e72e757fcd1842','hex')),
            ('000014_v11_realtime_outbox.sql', decode('2da7610d1a9d875f83934481c8d9de2a80350c93a8abff965c3af041d953f6a4','hex')),
            ('000015_v11_canonical_places.sql', decode('7cda9babc3673a816166ebc35329dc339365dd282215aff9d9f702b05dd41086','hex')),
            ('000016_v11_realtime_hardening.sql', decode('782738e22895217c6aa27bc3d3a1244795539b987977e9684b21821c5e11f721','hex')),
            ('000017_v11_city_context.sql', decode('f9d94fa8e50721e5ab8539e2c93e4a1b21f41e26dc13d134888c75f4dee9c88f','hex')),
            ('000018_v11_capability_registry.sql', decode('279b049b6eab5d8cf43d00b32f0ccac3e294efc5a2e354a2c565d9cac4253af6','hex')),
            ('000019_v10_onboarding_telegram.sql', decode('62aeb8d83fcf42888e493031b7b785862993a7e3c70d2040a80a82e640fff374','hex')),
            ('000020_v11_capability_registry_hardening.sql', decode('de37ced118f2ef9635b5782f814dbf2ece5ce6021fe154b312544c31f595a4e1','hex')),
            ('000021_v11_realtime_outbox_hardening.sql', decode('1e7872742a074fcf0e505b123af25bc8f7c878deec14dd7ebbdb5bc344a0b460','hex')),
            ('000022_v11_city_context_privilege_hardening.sql', decode('9d250bd3f78b1235d12e0724d2fea42f41f1c79fa6b206bd62c1c3da3992cfa2','hex')),
            ('000023_v11_city_realtime_channel.sql', decode('00614addf3529c3baf933cb2c412d4d05ac4b5e85848c4b71aee4fdcac31461f','hex'))
        ) AS v(name, checksum)
    LOOP
        SELECT m.checksum INTO existing
        FROM linkup_schema_migrations m
        WHERE m.name = expected.name;

        IF FOUND THEN
            IF existing IS DISTINCT FROM expected.checksum THEN
                RAISE EXCEPTION 'migration ledger checksum mismatch for %', expected.name;
            END IF;
        ELSE
            INSERT INTO linkup_schema_migrations (name, checksum)
            VALUES (expected.name, expected.checksum);
        END IF;
    END LOOP;
END
$$;

REVOKE ALL ON TABLE linkup_schema_migrations FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
        REVOKE ALL ON TABLE linkup_schema_migrations FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
        REVOKE ALL ON TABLE linkup_schema_migrations FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='linkup_api') THEN
        REVOKE ALL ON TABLE linkup_schema_migrations FROM linkup_api;
    END IF;
END
$$;
