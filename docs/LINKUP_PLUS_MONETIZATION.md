# LinkUp+ — Monetization Contract

Status: **canonical product contract for LinkUp+ monetization**, sourced from the user-provided `LinkUp Monetization Model.md` on 2026-09-07.

This is an early **v1.2** foundation. Active release remains **v1.0** and must not be blocked by this work. `PROJECT_RULES.md` and `README.md` keep higher priority for release ordering, security, server authority and platform boundaries.

## 1. Paid premium plans

| Plan | Customer price | Effective monthly price | 12-month amount |
|---|---:|---:|---:|
| Monthly | 149.99 UAH/month | 149.99 UAH | 1,799.88 UAH |
| Annual prepaid | 1,199.88 UAH/year | 99.99 UAH/month | 1,199.88 UAH |

Annual saving versus twelve monthly payments: **600.00 UAH / 33.3%**.

External Google Play product IDs are intentionally **not invented here**. Paid entitlement becomes active only after server-side Play Billing verification is implemented and configured.

## 2. Rewarded premium day

Product policy:

- one credited video every **4 hours**;
- **5** credited videos per reward cycle;
- source model describes the journey as **20 hours**;
- reward: **1 day of premium**;
- premium-day claim: at most **once per 7 days**;
- approximate monthly exposure: **~4 free premium days**.

Canonical tracking state:

- `last_video_watched_at`;
- `videos_watched_count` (`0..5`);
- `last_free_premium_claimed_at`.

A client callback is never sufficient proof. A video counts only after a server-side rewarded-ad verifier accepts a provider receipt/event. Raw provider receipts/tokens must not be persisted or logged; only hashes and normalized verified facts may be stored.

### Timing note

The supplied model states both “one video every 4 hours” and “5 videos = 20 hours”. If video #1 were immediately eligible, five views separated by four-hour gaps could span 16 hours from first to fifth. The product contract retains the supplied **20-hour** user-facing target; provider activation semantics must be finalized before rewarded verification is enabled so the implementation does not silently change this rule.

## 3. Referral program

A referral qualifies **only after the invited user pays for a verified subscription**, and the payment must happen within **14 days of the invited account registration**.

Progressive milestones:

| Qualified referrals | Inviter reward | Triggering invitee reward |
|---:|---:|---:|
| 1 | 1 premium day | 1 premium day |
| 3 | 7 premium days | 3 premium days |
| 5 | 30 premium days | 7 premium days |
| 10 | 90 premium days + badge/status | 7 premium days |

Implementation interpretation: each milestone is awarded **once** per inviter. The qualifying invitee whose verified paid conversion crosses the milestone receives the invitee-side reward for that milestone. This prevents replaying lower milestones for the same paid event.

The source also calls for a monthly referral leaderboard and an additional monthly reward, but does **not specify the reward amount or exact ranking policy**. The system may expose verified monthly counts/ranking later, but must not invent or automatically grant an unspecified leaderboard reward.

## 4. Server authority / anti-fraud

- Android never grants premium locally.
- Billing/rewarded/referral qualification is decided by Go.
- purchase/ad tokens are treated as secrets and never logged;
- provider receipts are stored only as hashes after verification;
- duplicate provider receipts must be idempotent;
- refunded/revoked paid periods must be represented server-side;
- premium grants must never be shortened by a later grant;
- referral self-invites are forbidden;
- one invitee can bind to at most one inviter;
- direct Supabase `anon`/`authenticated` access to monetization tables is forbidden.

## 5. Android UX contract

LinkUp+ is entered from **Me**, without changing the canonical `Pulse · Map · LINK · Fly · Me` navigation.

The screen must show real server state plus:

- monthly/annual plans and 600 UAH annual saving;
- current premium status/expiry;
- rewarded progress (`x/5`), 4-hour cadence, next availability/weekly claim timing when known;
- referral milestones and verified qualified-referral count;
- clear unavailable/fail-closed messaging while Play Billing or rewarded verification adapters are not configured.

No fake “premium activated”, fake ad completion, fake referral count or client-authoritative entitlement state is allowed.
