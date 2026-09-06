CREATE TABLE slots (
    id uuid PRIMARY KEY,
    host_id uuid NOT NULL REFERENCES app_users(id) ON DELETE RESTRICT,
    title text NOT NULL,
    activity text NOT NULL,
    details text,
    place_text text NOT NULL,
    zone_text text,
    start_at timestamptz,
    capacity integer NOT NULL,
    accepted_count integer NOT NULL DEFAULT 0,
    state text NOT NULL,
    access_mode text NOT NULL,
    visibility text NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    cancelled_at timestamptz,
    started_at timestamptz,
    completed_at timestamptz,
    CHECK (char_length(title) BETWEEN 1 AND 120),
    CHECK (char_length(activity) BETWEEN 1 AND 64),
    CHECK (details IS NULL OR char_length(details) <= 2000),
    CHECK (char_length(place_text) BETWEEN 1 AND 240),
    CHECK (zone_text IS NULL OR char_length(zone_text) <= 160),
    CHECK (capacity BETWEEN 2 AND 10000),
    CHECK (accepted_count BETWEEN 0 AND capacity),
    CHECK (version >= 1),
    CHECK (state IN ('DRAFT','PUBLISHED','FILLING','FULL','ACTIVE','COMPLETED','CANCELLED','EXPIRED','MODERATED')),
    CHECK (access_mode IN ('INSTANT','APPROVAL','WAITLIST')),
    CHECK (visibility IN ('PUBLIC','LINKS','SELECTED','CITY','LASSO','TRAVEL_CORRIDOR','PRIVATE'))
);

CREATE INDEX slots_host_updated_idx ON slots (host_id, updated_at DESC, id);
CREATE INDEX slots_public_pulse_idx
    ON slots (created_at DESC, id)
    WHERE visibility='PUBLIC' AND state IN ('PUBLISHED','FILLING','FULL');
CREATE INDEX slots_public_schedule_idx
    ON slots (start_at, id)
    WHERE visibility='PUBLIC' AND state IN ('PUBLISHED','FILLING','FULL') AND start_at IS NOT NULL;

CREATE TABLE mutation_idempotency (
    actor_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    idempotency_key text NOT NULL,
    operation text NOT NULL,
    request_hash bytea NOT NULL,
    resource_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (actor_id, idempotency_key),
    CHECK (char_length(idempotency_key) BETWEEN 16 AND 128),
    CHECK (char_length(operation) BETWEEN 1 AND 80),
    CHECK (expires_at > created_at)
);

CREATE INDEX mutation_idempotency_expiry_idx ON mutation_idempotency (expires_at);
