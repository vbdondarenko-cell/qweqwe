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

Referral-code binding is server-authoritative:

- every authenticated account can obtain one stable referral code;
- an invitee can bind only one inviter;
- binding the same code again is idempotent;
- self-referral is rejected;
- the qualification deadline is calculated from canonical `app_users.created_at + 14 days`, not from the Android clock;
- binding alone grants **no premium**; the referral becomes qualified only after a future verified paid-subscription event.

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
- the user's own referral code;
- one-time inviter-code binding with the canonical qualification deadline;
- referral milestones and verified qualified-referral count;
- clear unavailable/fail-closed messaging while Play Billing or rewarded verification adapters are not configured.

No fake “premium activated”, fake ad completion, fake referral count or client-authoritative entitlement state is allowed.

## 6. Repository implementation status — 2026-09-07

Implemented in `main` as an **early v1.2 production foundation**:

### Database

`db/migrations/000009_linkup_plus_monetization.sql` adds canonical tables for:

- premium grants;
- rewarded progress and hashed verified rewarded receipts;
- verified subscription receipts;
- referral codes and invitee→inviter binding;
- one-time referral milestone awards.

The migration explicitly revokes direct table access from `PUBLIC`, `anon` and `authenticated`. It has **not been applied to production Supabase in this work block**, because the user requested repository-only work.

### Go

New monetization domain/read model:

- exact plan/reward/referral policy constants;
- authenticated `GET /v1/me/monetization`;
- authenticated `POST /v1/me/referral`;
- PostgreSQL-backed premium/reward/referral state;
- stable per-account referral codes;
- server-side 14-day referral binding deadline;
- self-referral and rebinding protection;
- provider capability flags default to `false` so unconfigured payment/ad verification fails closed.

Pure Go policy tests were added for prices, savings, rewarded cadence/cooldown, referral milestones, code normalization and fail-closed capabilities. These tests are **present but not claimed as executed in this repository-only block**.

### Android

Added:

- `MonetizationModels.kt`;
- authenticated `MonetizationApiClient.kt` with bounded responses and GET-only retry;
- native `LinkUpPlusScreen.kt`;
- entry from existing `Me` without changing bottom navigation;
- Ukrainian and English LinkUp+ resources;
- real server-backed premium status, plan prices, rewarded progress/timing, referral count/milestones;
- own referral code and real one-time referral-code submission to Go.

The Android client never creates a premium grant and never treats a local purchase/ad callback as proof.

### Commits in this work block

- data foundation: `d759755783e595d40bec931666237d293207a32e`;
- monetization contract: `3ddb61ff8e5bf37c6d057f436e331c7e054284c7`;
- Go domain/test/store/API wiring: `f8f7828a0c1b7f7cc900cecf5e63a9cfe41ad4d7` → `68933d41af484793502bb1ab7c927be9e1b72f0f`;
- Android model/client/screen/Me wiring/resources: `8036fc6cd854b015009368c3b38a39d7e4115232` → `9cff1579525934b6919947f9a49df532d76f2c0b`.

## 7. Intentionally still locked / not claimed complete

The following require external product/provider configuration and are **not** faked:

- actual Google Play subscription purchase flow and server-side Play purchase verification;
- RTDN/subscription renewal/refund/revocation processing;
- rewarded-ad SDK/provider integration and server-side rewarded receipt verification;
- mutation that increments the 0..5 rewarded counter and grants the verified weekly premium day;
- qualification of a referral after a verified paid receipt and atomic milestone premium grants;
- the monthly referral leaderboard reward, because its exact reward and ranking/tie policy were not specified by the supplied model;
- production migration execution, runtime role grants and end-to-end Android/Go/PostgreSQL verification.

These are v1.2 completion gates and do not change the active v1.0 release order.
