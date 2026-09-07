# LinkUp v1.0 — ACTIVE IMPLEMENTATION WORKLOG

> Append-only working ledger for the active **v1.0 Core Social Network** release.
>
> Read before every v1.0 implementation block together with `PROJECT_RULES.md`, `README.md`, `IMPLEMENTATION_STATUS.md` and relevant `REPOSITORY_AUDIT.md` findings. The purpose is to avoid repeating a block that has already been implemented or hardened.
>
> `IMPLEMENTATION_STATUS.md` remains the consolidated repository-wide factual status ledger. This file records the active v1.0 work stream in smaller chronological blocks and must never be used to claim tests/builds that were not actually executed.

## Active rules

- Active release: **v1.0** only.
- Design: frozen; no redesign.
- Android: Kotlin + Jetpack Compose.
- Backend/domain authority: Go.
- Database: PostgreSQL/PostGIS, forward-only migrations.
- iOS: frozen.
- Work: direct to `main`.
- Ubuntu/live Supabase deployment: not touched without direct user deployment instruction.
- Production readiness increases only with real execution evidence.

## v1.0 baseline already present before this worklog

The repository already contains production-oriented source for:

- registration/login/logout/session/recovery/reset;
- profile edit + block controls;
- PUBLIC + APPROVAL Slot create/read/edit/cancel;
- optional Slot date/time;
- Pulse real-data API/client surface;
- REQUEST / APPROVE / REJECT;
- accepted roster;
- accepted LEAVE;
- host participant removal;
- START / COMPLETE;
- basic accepted-only ephemeral chat + terminal PostgreSQL purge trigger;
- Android Auth/Pulse/LINK/Slot/Edit/Chat/Me routing;
- Hosting / Joined / Requests dashboard;
- foreground bounded chat polling.

These features remain subject to compile/integration/device verification gates recorded below.

---

## 2026-09-07 — v1.0 recovery/password/database boundary hardening

Commit: `c3f8df7153d8c483315df1fecb1bb51a8e563366`

Implemented:

- SMTP timeout now bounds the full SMTP operation, not only TCP connect;
- cancellation/deadline propagates to stalled SMTP greeting/TLS/commands;
- implicit TLS uses context-aware handshake;
- recovery reset URL must be an absolute HTTPS application link;
- reset URL rejects userinfo, existing query and fragment;
- empty reset token rejected;
- SMTP timeout has a safety ceiling;
- Argon2id has upper safety bounds in addition to the project security floor;
- hostile encoded Argon2 parameters are rejected before expensive Argon work;
- oversized login password is rejected before expensive verify work;
- integer-second duration parsing rejects `time.Duration` overflow;
- PostgreSQL connection/config errors returned by the pool boundary no longer wrap raw pgx/DSN errors;
- `.env.example` documents HTTPS recovery origin and accepted Argon2 bounds;
- regression-test sources added for these boundaries.

Execution evidence: source changes committed. Full Go test graph was **not** executed in the available environment.

---

## 2026-09-07 — v1.0 login/password-reset race serialization

Commit: `14cf7a3e7cb250fc5adcb08330f9de7f9c134298`

Implemented:

- successful password verification is bound to session creation using the verified `password_hash`;
- PostgreSQL `CreateSession` takes the account row `FOR UPDATE` and refuses session creation if password hash changed after verification;
- password-reset creation locks the same account row before invalidating older reset tokens and creating a replacement;
- password reset locks the account before password update/session revocation;
- reset consumes its token, changes password, revokes all sessions and invalidates remaining unused reset tokens in the same transaction;
- fake store/HTTP test contracts updated;
- regression test simulates password rotation between verify and session insert.

Security property: a login verified against the old password cannot create a fresh session after a concurrent reset has changed that password.

Execution evidence: source changes committed. PostgreSQL concurrency test execution remains open.

---

## 2026-09-07 — v1.0 block/social mutation lock ordering

Commit: `4e0a696c18ac3d380020aaa85e413ad02cda63cd`

Implemented:

- Block uses the same Slot-first lock order as REQUEST/APPROVE/LEAVE style mutations;
- all non-terminal Slots hosted by either side are locked deterministically by Slot id before block publication/cleanup;
- existing pending requests and accepted memberships between the pair are revoked transactionally;
- pending-only relationship cleanup now advances Slot `version`;
- accepted cleanup decrements `accepted_count` and reopens `FULL → FILLING`;
- each affected Slot advances version once per real cleanup;
- repeated block with no remaining relationship does not advance Slot version again;
- opt-in PostgreSQL query-contract test source added.

Execution evidence: source changes committed. Real concurrent PostgreSQL execution remains open.

---

## 2026-09-07 — v1.0 Supabase/Data API role boundary

Commit: `4bc1e379dfbf601436fbff0007d34df773fc1f18`

Added forward-only migration:

- `db/migrations/000005_api_role_boundary.sql`;
- removes direct privileges on current canonical LinkUp tables from `PUBLIC`;
- removes direct table/function privileges from Supabase `anon` and `authenticated` roles when those roles exist;
- removes direct execution privilege for the terminal-chat purge function;
- adds deny-by-default privileges for future tables/functions created by the same migration owner.

Important: RLS is **not** force-enabled here because the deployment DB application-role/owner model has not yet been verified; enabling FORCE RLS without a correct Go policy could break the canonical Go API. This migration was added to Git only and has **not** been applied to live Supabase.

---

## 2026-09-07 — v1.0 Android transport/session revocation hardening

Commits:

- bounded API response body: `c1c6c4fade41bc2c0424740656dd86d2d6d2f937`;
- authenticated 401 routing: `7d3b85e42c1e74272917dc4776b57423a0ebd1aa`;
- response-boundary tests: `91bcf0cb05820afeafaf1ff90fa0612f4c6ecec8`.

Implemented:

- all Android API response/error bodies are capped at 1 MiB before JSON parsing;
- oversized response produces explicit `response_too_large` protocol failure instead of unbounded `readText()` allocation;
- authenticated HTTP 401 clears secure local bearer state immediately;
- `SessionCoordinator` subscribes to the API revocation callback and transitions routing to `SignedOut` from any authenticated endpoint, not only `/v1/me` bootstrap;
- byte-boundary tests include UTF-8 multi-byte content.

Execution evidence: Kotlin source/tests committed. Android Gradle/JVM tests remain unexecuted until the build toolchain gate is restored.

---

## Open v1.0 blockers after the above work

### P0 execution/release gates

- complete reproducible Android Gradle wrapper (`gradlew`, `gradlew.bat`, verified wrapper JAR);
- Android compile + unit/lint/instrumentation as required;
- obtain/verify `backend/go.sum` and run full Go test/vet/race gates;
- execute migrations `000001..000005` on disposable PostgreSQL;
- PostgreSQL integration/concurrency tests for last seat, block races, reset/login races and terminal chat purge;
- verify actual Supabase role grants before live migration application;
- real two-user Android ↔ Go ↔ PostgreSQL v1.0 smoke;
- signing/release AAB;
- backup/recovery + rollback exercise;
- Ukrainian/English/accessibility release checks.

### P1 source/security work still open

- idempotency replay must re-authorize current access before returning a resource after block/reject/revocation;
- verify exact replay semantics for LEAVE separately from host/request operations;
- review/limit remaining unbounded list/query surfaces where applicable;
- verify migration-runner concurrency behavior;
- complete v1.0 query/index review.

## Next exact v1.0 block

**Idempotency replay authorization**, then **build/toolchain recovery and executed verification**.

Do not start v1.1 capability work while v1.0 release gates above remain open unless the user explicitly changes the release order.
