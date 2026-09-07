-- Android FCM device registry for the canonical Ubuntu Go API -> Firebase push path.
-- Raw FCM registration tokens are never stored in plaintext. The Go API stores
-- an AES-GCM ciphertext plus a SHA-256 lookup hash and revokes rows on logout.

CREATE TABLE IF NOT EXISTS public.push_devices (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES public.app_users(id) ON DELETE CASCADE,
    session_id uuid NOT NULL REFERENCES public.user_sessions(id) ON DELETE CASCADE,
    platform text NOT NULL CHECK (platform = 'ANDROID'),
    installation_id uuid NOT NULL,
    token_hash bytea NOT NULL CHECK (octet_length(token_hash) = 32),
    token_ciphertext bytea NOT NULL CHECK (octet_length(token_ciphertext) > 0),
    token_nonce bytea NOT NULL CHECK (octet_length(token_nonce) = 12),
    key_id text NOT NULL CHECK (char_length(key_id) BETWEEN 1 AND 64),
    app_version text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    UNIQUE (platform, installation_id),
    UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS push_devices_user_active_idx
    ON public.push_devices (user_id, updated_at DESC)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS push_devices_session_active_idx
    ON public.push_devices (session_id)
    WHERE revoked_at IS NULL;

REVOKE ALL ON TABLE public.push_devices FROM PUBLIC;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE public.push_devices FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE public.push_devices FROM authenticated;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'linkup_api') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE public.push_devices TO linkup_api;
    END IF;
END
$$;
