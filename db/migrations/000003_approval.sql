CREATE TABLE slot_requests (
    slot_id uuid NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (slot_id, user_id)
);

CREATE INDEX slot_requests_user_created_idx
    ON slot_requests (user_id, created_at DESC, slot_id);

CREATE TABLE slot_memberships (
    slot_id uuid NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    accepted_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (slot_id, user_id)
);

CREATE INDEX slot_memberships_user_accepted_idx
    ON slot_memberships (user_id, accepted_at DESC, slot_id);
