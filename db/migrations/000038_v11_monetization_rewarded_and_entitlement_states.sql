-- README §6.20/§6.27 + docs/LINKUP_PLUS_MONETIZATION.md §5-§8: this
-- migration's own companion Go change closes the gap that document's own
-- "Current repository implementation status" section already named --
-- premium_grants had no writer of any kind before this block. Neither the
-- rewarded-video quest, purchase verification, nor referral qualification
-- ever actually issued a grant.
--
-- 1. monetization_rewarded_progress needs a quest_started_at column:
--    videos_watched_count/last_video_watched_at are enough to enforce
--    "minimum 4 hours between verified steps", but not the separate
--    "24-hour quest window" rule -- that clock runs from the FIRST
--    verified view of the current 5-view quest, not the most recent one
--    (docs/LINKUP_PLUS_MONETIZATION.md §6).
-- 2. monetization_subscription_receipts.state gains GRACE and
--    BILLING_RETRY, the two entitlement states docs/LINKUP_PLUS_MONETIZATION.md
--    §5 and README §6.20 require alongside ACTIVE/EXPIRED/REVOKED that
--    migration 000009 didn't anticipate. These map to standard Google Play
--    subscription semantics, not an invented meaning: GRACE (payment retry
--    in progress, entitlement still honored) and BILLING_RETRY/account
--    hold (payment retry exhausted, entitlement suspended pending
--    resolution).

ALTER TABLE monetization_rewarded_progress
    ADD COLUMN quest_started_at timestamptz;

ALTER TABLE monetization_subscription_receipts
    DROP CONSTRAINT monetization_subscription_receipts_state_check;
ALTER TABLE monetization_subscription_receipts
    ADD CONSTRAINT monetization_subscription_receipts_state_check
    CHECK (state IN ('ACTIVE', 'GRACE', 'BILLING_RETRY', 'EXPIRED', 'REVOKED', 'REFUNDED'));
