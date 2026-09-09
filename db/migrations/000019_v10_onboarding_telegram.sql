-- Forward-only v1.0 onboarding persistence.
-- Phone ownership is verified only by the official Telegram bot flow.
-- Raw verification tokens are never stored: only SHA-256 token hashes are persisted.

CREATE TABLE IF NOT EXISTS registration_onboarding (
    id uuid PRIMARY KEY,
    email text NOT NULL,
    username text NOT NULL,
    display_name text NOT NULL,
    password_hash text NOT NULL,
    language text NOT NULL DEFAULT 'uk' CHECK (language IN ('uk', 'en')),
    device_label text,
    birth_date date NOT NULL,
    city_id text,
    city_name text NOT NULL,
    preferences jsonb NOT NULL,
    teen_mode boolean NOT NULL DEFAULT false,
    verification_token_hash bytea NOT NULL UNIQUE,
    telegram_user_id bigint,
    phone_e164 text,
    phone_verified_at timestamptz,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    completed_at timestamptz,
    CHECK (char_length(email) BETWEEN 3 AND 320),
    CHECK (char_length(username) BETWEEN 3 AND 32),
    CHECK (char_length(display_name) BETWEEN 1 AND 80),
    CHECK (char_length(city_name) BETWEEN 1 AND 160),
    CHECK (expires_at > created_at),
    CHECK ((phone_verified_at IS NULL AND phone_e164 IS NULL) OR (phone_verified_at IS NOT NULL AND phone_e164 IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS registration_onboarding_expires_idx
    ON registration_onboarding (expires_at)
    WHERE completed_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS registration_onboarding_active_telegram_uidx
    ON registration_onboarding (telegram_user_id)
    WHERE telegram_user_id IS NOT NULL AND completed_at IS NULL;

CREATE INDEX IF NOT EXISTS registration_onboarding_email_idx
    ON registration_onboarding (lower(email));

CREATE INDEX IF NOT EXISTS registration_onboarding_username_idx
    ON registration_onboarding (lower(username));

CREATE TABLE IF NOT EXISTS user_onboarding_profiles (
    user_id uuid PRIMARY KEY REFERENCES app_users(id) ON DELETE CASCADE,
    birth_date date NOT NULL,
    city_id text,
    city_name text NOT NULL,
    preferences jsonb NOT NULL,
    teen_mode boolean NOT NULL DEFAULT false,
    phone_e164 text NOT NULL UNIQUE,
    telegram_user_id bigint NOT NULL UNIQUE,
    phone_verified_at timestamptz NOT NULL,
    onboarding_completed_at timestamptz NOT NULL,
    CHECK (char_length(city_name) BETWEEN 1 AND 160),
    CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$')
);

REVOKE ALL ON TABLE registration_onboarding, user_onboarding_profiles FROM PUBLIC;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE registration_onboarding, user_onboarding_profiles FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE registration_onboarding, user_onboarding_profiles FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE registration_onboarding, user_onboarding_profiles TO linkup_api;
    END IF;
END
$$;