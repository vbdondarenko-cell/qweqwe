-- Forward-only LinkUp+ monetization foundation.
-- Active release remains v1.0; this schema is an early v1.2 foundation and
-- does not make v1.0 depend on billing/rewarded/referral capabilities.
-- Go remains the only app-facing authority. No Android/Supabase client role
-- receives direct access to canonical monetization state.

CREATE TABLE premium_grants (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    source text NOT NULL CHECK (source IN ('PAID', 'REWARDED', 'REFERRAL')),
    source_key text NOT NULL,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, source_key),
    CHECK (char_length(source_key) BETWEEN 8 AND 256),
    CHECK (ends_at > starts_at)
);

CREATE INDEX premium_grants_user_end_idx
    ON premium_grants (user_id, ends_at DESC);

CREATE TABLE monetization_rewarded_progress (
    user_id uuid PRIMARY KEY REFERENCES app_users(id) ON DELETE CASCADE,
    videos_watched_count smallint NOT NULL DEFAULT 0
        CHECK (videos_watched_count BETWEEN 0 AND 5),
    last_video_watched_at timestamptz,
    last_free_premium_claimed_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE monetization_rewarded_receipts (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    provider text NOT NULL,
    receipt_hash bytea NOT NULL,
    watched_at timestamptz NOT NULL,
    verified_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, receipt_hash),
    CHECK (char_length(provider) BETWEEN 2 AND 64)
);

CREATE INDEX monetization_rewarded_receipts_user_idx
    ON monetization_rewarded_receipts (user_id, watched_at DESC);

CREATE TABLE monetization_subscription_receipts (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    provider text NOT NULL,
    product_id text NOT NULL,
    purchase_token_hash bytea NOT NULL,
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    state text NOT NULL CHECK (state IN ('ACTIVE', 'EXPIRED', 'REVOKED', 'REFUNDED')),
    verified_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, purchase_token_hash),
    CHECK (char_length(provider) BETWEEN 2 AND 64),
    CHECK (char_length(product_id) BETWEEN 1 AND 200),
    CHECK (period_end > period_start)
);

CREATE INDEX monetization_subscription_receipts_user_idx
    ON monetization_subscription_receipts (user_id, period_end DESC);

CREATE TABLE monetization_referral_codes (
    user_id uuid PRIMARY KEY REFERENCES app_users(id) ON DELETE CASCADE,
    code text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (code ~ '^[A-Z0-9]{6,20}$')
);

CREATE TABLE monetization_referrals (
    invitee_id uuid PRIMARY KEY REFERENCES app_users(id) ON DELETE CASCADE,
    inviter_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    referral_code text NOT NULL REFERENCES monetization_referral_codes(code),
    accepted_at timestamptz NOT NULL DEFAULT now(),
    qualifying_deadline timestamptz NOT NULL,
    qualified_at timestamptz,
    qualifying_receipt_id uuid REFERENCES monetization_subscription_receipts(id) ON DELETE SET NULL,
    CHECK (invitee_id <> inviter_id),
    CHECK (qualifying_deadline > accepted_at)
);

CREATE INDEX monetization_referrals_inviter_qualified_idx
    ON monetization_referrals (inviter_id, qualified_at DESC)
    WHERE qualified_at IS NOT NULL;

CREATE TABLE monetization_referral_milestone_awards (
    inviter_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    milestone smallint NOT NULL CHECK (milestone IN (1, 3, 5, 10)),
    triggering_invitee_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    inviter_reward_days smallint NOT NULL CHECK (inviter_reward_days > 0),
    invitee_reward_days smallint NOT NULL CHECK (invitee_reward_days > 0),
    awarded_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (inviter_id, milestone)
);

-- Direct Supabase/Data API access remains forbidden. The runtime Go DB role is
-- deployment configuration and receives only the minimum grants it needs.
REVOKE ALL ON TABLE
    premium_grants,
    monetization_rewarded_progress,
    monetization_rewarded_receipts,
    monetization_subscription_receipts,
    monetization_referral_codes,
    monetization_referrals,
    monetization_referral_milestone_awards
FROM PUBLIC;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE
            premium_grants,
            monetization_rewarded_progress,
            monetization_rewarded_receipts,
            monetization_subscription_receipts,
            monetization_referral_codes,
            monetization_referrals,
            monetization_referral_milestone_awards
        FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE
            premium_grants,
            monetization_rewarded_progress,
            monetization_rewarded_receipts,
            monetization_subscription_receipts,
            monetization_referral_codes,
            monetization_referrals,
            monetization_referral_milestone_awards
        FROM authenticated;
    END IF;
END
$$;
