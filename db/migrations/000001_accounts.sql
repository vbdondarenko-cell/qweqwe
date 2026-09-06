BEGIN;

CREATE TABLE app_users (
    id uuid PRIMARY KEY,
    email text NOT NULL,
    username text NOT NULL,
    display_name text NOT NULL,
    password_hash text NOT NULL,
    avatar_url text,
    profile_visibility text NOT NULL DEFAULT 'PUBLIC'
        CHECK (profile_visibility IN ('PUBLIC', 'HIDDEN')),
    language text NOT NULL DEFAULT 'uk'
        CHECK (language IN ('uk', 'en')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (char_length(trim(email)) BETWEEN 3 AND 320),
    CHECK (char_length(username) BETWEEN 3 AND 32),
    CHECK (char_length(display_name) BETWEEN 1 AND 80)
);

CREATE UNIQUE INDEX app_users_email_ci_uq
    ON app_users (lower(trim(email)));

CREATE UNIQUE INDEX app_users_username_ci_uq
    ON app_users (lower(username));

CREATE TABLE user_sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    device_label text,
    CHECK (expires_at > created_at)
);

CREATE INDEX user_sessions_user_active_idx
    ON user_sessions (user_id, expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE password_reset_tokens (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    CHECK (expires_at > created_at)
);

CREATE INDEX password_reset_tokens_user_active_idx
    ON password_reset_tokens (user_id, expires_at)
    WHERE used_at IS NULL;

CREATE TABLE user_blocks (
    blocker_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    blocked_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

CREATE INDEX user_blocks_blocked_idx ON user_blocks (blocked_id, blocker_id);

COMMIT;
