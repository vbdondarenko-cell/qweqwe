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

## 2026-09-07 — v1.0 idempotency replay authorization

Commits:

- replay policy helper: `38c5ddcb4c62646020a9677ae9176ec08074ae65`;
- policy enforcement in `claimIdempotency`: `85ef5006442afac51979dbe61b984bcf1a9752cf`;
- PostgreSQL query-contract tests: `db2a7a0e5cbed0212c3becab0d5f53d3a76dc9d9`.

Implemented:

- mutation replay re-checks current authorization before any internal Slot read;
- old create/edit/cancel/approve/reject/start/complete/remove-member keys do not bypass current host authority;
- REQUEST replay requires the current requester relationship and current block boundary;
- LEAVE replay cannot act as a hidden resource read after access disappears;
- block/rejection/revocation/terminal transitions are evaluated against current database state, not the historical authority that originally created the key.

Execution evidence: source and opt-in PostgreSQL test code committed. PostgreSQL execution remains open.

---

## 2026-09-07 — partial Gradle wrapper recovery

Commits:

- official Gradle v9.6.0 Windows wrapper script: `339bea3357b6f4c40f423fbae1b0249f86fc2074`;
- POSIX wrapper script added: `2e21a0011ef0996655c8da538fec3316bcccc357`;
- probe cleanup + POSIX executable mode: `763a37fc1fe3537e36abb4d7f8fe5149db4d786e`.

Evidence / limitation:

- official upstream Gradle v9.6.0 wrapper JAR is identified as Git blob `b1b8ef56b44f16b14dc800fa8103a6d89abb526f`, 48,462 bytes;
- the available GitHub connector rejects binary blob transfer as UTF-8 and cross-repository blob reuse is rejected by GitHub object scope;
- therefore `android/gradle/wrapper/gradle-wrapper.jar` is still **missing** and Android wrapper/build reproducibility is **not green**;
- temporary provenance/probe files created during the object-transfer check were removed from current main.

Do not claim the wrapper complete until the exact JAR is present and verified.

---

## 2026-09-07 — v1.0 migration-runner concurrency

Commits:

- transaction-scoped advisory lock: `f030b42aa71546ccb48335424d064f0285dacf3e`;
- concurrent/repeat/checksum-drift PostgreSQL test source: `3df32abf0ce40933c9df1f40383e08ef8e45e17c`.

Implemented:

- each migration decision takes a fixed `pg_advisory_xact_lock` before reading the migration ledger;
- lock lifetime is transaction-scoped, so commit/rollback/context cancellation releases it automatically;
- concurrent API startup cannot both decide that the same migration is unapplied;
- opt-in PostgreSQL test starts two concurrent `Apply()` calls, verifies one migration effect/one ledger row, verifies repeat no-op and checksum-drift rejection.

Execution evidence: source/test committed; test requires `LINKUP_TEST_DATABASE_URL` and has not yet run in the current environment.

---

## 2026-09-07 — v1.0 idempotent chat send / ambiguous outcome

Commits:

- migration `000006_chat_idempotency.sql`: `04d46ce950ba14dd57f30d962f9270af95a8bfdb`;
- chat model/service contract: `7a931f8d048e26f58aa4b79defee53e21df3ed9e`, `825108593cf748dbae1603abfde6eb443cabe159`, `298b1faa50d73f85cfe74e62b3d66339b78a3d64`;
- PostgreSQL replay implementation: `63a4bad332a785382e4e9554bbfb527bffc8efc7`;
- HTTP contract: `af1fdce8b6cf15294e6a4ff5b08dc99144f61094`, `aa82d749c036a86cffb9252406b7222c321679dc`;
- Android in-process retry identity: `430ae42337ec989f913b039edc92e257d2b1f741`;
- Compose draft preservation and acknowledgement-race hardening: `f611d62dd92306edbb7e16262efab84483a12974`, `13150752553d9ff74b896744250f5b206a5cd3e1`;
- server-internal key/privacy and stale test-double cleanup: `79823a2d25552bdac3cbfcff2b1e5b81c4b49747`, `dd2687840eaeb5ad5e08acb16208615ac8c5d1a6`, `173a13dd823d0c48b46ac00549a2d60ad0d63049`, `a7c03205fc734511ea5390b3fe66676f997f83fe`.

Implemented:

- every chat send requires an `Idempotency-Key` 16..128 chars;
- `(slot_id, author_id, idempotency_key)` is unique in PostgreSQL;
- same key + same normalized text returns the same canonical message row;
- same key + different text returns HTTP 409 `idempotency_conflict`;
- current chat authorization is checked before replay lookup, so an old send key cannot bypass LEAVE/block/terminal revocation;
- idempotency keys stay server-internal and are not returned to other chat participants;
- Android retains an in-process key across network/5xx ambiguity and explicit Retry, but clears it after acknowledgement or definitive client rejection;
- composer text is not cleared before server acknowledgement; a failure leaves the draft in place;
- Compose only clears after observing a real `Running → Idle` mutation transition, avoiding a coroutine-start/recomposition race.

Scope boundary: this is v1.0 foreground/in-process retry safety. Durable process-death mutation replay remains v1.1 and is not claimed here.

Execution evidence: source/tests committed. Migration `000006`, Go tests, Android tests and device behavior remain unexecuted.

---

## Open v1.0 blockers after the above work

### P0 execution/release gates

- obtain exact official `android/gradle/wrapper/gradle-wrapper.jar`, verify it and execute Android wrapper/build;
- Android compile + unit/lint/instrumentation as required;
- obtain/verify `backend/go.sum` and run full Go test/vet/race gates on the pinned supported Go toolchain;
- execute migrations `000001..000006` on disposable PostgreSQL;
- PostgreSQL integration/concurrency tests for last seat, block races, reset/login races, migration startup, chat idempotency and terminal chat purge;
- verify actual Supabase role grants before live migration application;
- real two-user Android ↔ Go ↔ PostgreSQL v1.0 smoke;
- signing/release AAB;
- backup/recovery + rollback exercise;
- Ukrainian/English/accessibility release checks.

### P1/P2 source/release hardening still open

- review/limit remaining unbounded list/query surfaces where applicable, including pending-request lists;
- complete v1.0 query/index review;
- verify exact Android chat retry behavior with executed tests/device transport faults;
- review remaining v1.0 forms/state restoration/accessibility/localization against release gate.

## Next exact v1.0 block

**Bound remaining v1.0 query surfaces and perform query/index review, while continuing P0 toolchain recovery where the available environment permits.**

Do not start v1.1 capability work while v1.0 release gates above remain open unless the user explicitly changes the release order.

---

## 2026-09-07 — executed v1.0 release-gate verification

This section supersedes the older “unexecuted/toolchain unavailable” notes above where the same gates now have real evidence.

Executed on the Ubuntu release host:

- official Gradle 9.6.0 wrapper JAR is now present in the repository and tracked; SHA-256 `497c8c2a7e5031f6aa847f88104aa80a93532ec32ee17bdb8d1d2f67a194a9c7`;
- `backend/go.sum` is present and Go 1.27.1 is available;
- `./gradlew --no-daemon --stacktrace :app:testDebugUnitTest :app:lintDebug :app:assembleDebug` passed;
- `go test -count=1 ./...`, `go vet ./...` and `go test -race -count=1 ./...` passed;
- PostgreSQL 16.15 was installed locally only for disposable release verification; production Supabase was not mutated;
- canonical migrations `000001..000011` applied successfully to disposable PostgreSQL;
- real PostgreSQL tests passed for accepted roster, block revocation, idempotency replay authorization, terminal chat purge and concurrent last-seat capacity;
- a migration-ledger test bug was found: `LIKE '00000%_%.sql'` counted only migrations 000001..000009. The assertion now matches six-digit migration names and correctly verifies all 11 migrations;
- full PostgreSQL-backed Go race execution passed after the assertion fix;
- v1.0 query surfaces are bounded: Pulse/My LINKs/block/chat via explicit limits, pending requests at 100, accepted roster by v1.0 capacity ceiling 100;
- query/index review confirmed the pending/chat indexes and every current public foreign key in the disposable schema has a valid leading index;
- backup/restore drill passed using a custom-format `pg_dump` and `pg_restore`; restored schema contained 11 migration ledger rows through `000011_android_push_devices.sql` and 18 public tables, then the restore DB and temporary dump were deleted;
- English/Ukrainian live-design string resource files contain the same 35 keys; active v1.0 design screens no longer contain the previously frozen fake account/metric/slot fixtures.

Read-only live Supabase verification:

- runtime `DATABASE_URL` resolves to the expected Supabase project reference;
- active application tables are not granted directly to `anon` or `authenticated`;
- live schema contains the current application tables including `push_devices`, but `public.linkup_schema_migrations` is absent;
- `anon` and `authenticated` still hold broad privileges on `public.spatial_ref_sys`, so the PostGIS hardening represented by migration `000010_v1_database_hardening.sql` is not fully reflected in live state.

The Supabase findings are evidence only. No live database migration or grant change was made because production Supabase writes require explicit authorization.

### Remaining release blockers after executed verification

- production Privacy Policy and Terms HTTPS URLs/content are not supplied; the release build intentionally fails closed without them;
- a signed release AAB is therefore not yet claimable even though API/reset/Firebase/signing inputs are otherwise configured;
- Android runtime/instrumentation and real two-device user-flow verification remain open because the release host has no connected Android device/emulator;
- live Supabase migration ledger and PostGIS grant hardening need an explicitly authorized production migration/reconciliation step;
- production password-recovery delivery and live backup/restore must be treated as production-operation gates, not inferred from source or the local disposable drill.

Do not call v1.0 100% production-ready until those external/live gates are closed with evidence. Source/build/database verification is materially ahead of the historical status notes above.


---

## 2026-09-07 — full-stack audit refresh and migration startup race fix

A fresh audit was run against GitHub `main`, the Ubuntu runtime, live Supabase and the configured Firebase/Android release inputs.

- Server and GitHub `main` were synchronized at `1a04c47c9c829b02791be8904b08c8db040eba96` before this correction; the worktree was clean.
- A brand-new disposable PostgreSQL database exposed a real concurrent-startup defect in the migration runner: two callers could race on `CREATE TABLE IF NOT EXISTS linkup_schema_migrations` before the advisory lock was acquired, producing a PostgreSQL duplicate-type catalog error.
- `migrate.Apply` now serializes migration-ledger creation under the same transaction-scoped advisory lock before any migration decision. The focused concurrent migration test passes after the correction.
- Fresh `go test -count=1 ./...`, `go vet ./...` and `go test -race -count=1 ./...` pass after the fix.
- Fresh Android `:app:testDebugUnitTest :app:lintDebug :app:assembleDebug` passes against the real HTTPS debug endpoint configuration. There is still no connected Android device, AVD or `androidTest` suite on the release host, so physical runtime/instrumentation remains open.
- Live Supabase reports all canonical migrations `000001..000011` in its managed migration history. The earlier note about an absent `public.linkup_schema_migrations` table refers only to the app runner's custom ledger and must not be interpreted as missing Supabase migration history.
- Direct grants on LinkUp application tables are restricted to `linkup_api`; `anon` and `authenticated` do not have direct application-table grants. The remaining concrete Supabase exposure is the PostGIS surface in `public`: `spatial_ref_sys` and public executable `SECURITY DEFINER` `st_estimatedextent` overloads.
- Live data-integrity checks found zero accepted-count mismatches, request/membership overlaps, terminal chat rows, self-blocks or expired unrevoked sessions.
- Firebase service-account project, backend Firebase project and Android Firebase project all match `linkup-4b782`; push registration/delivery configuration initializes successfully in the running API. Live `push_devices` remains empty, so device delivery is not yet end-to-end evidence.
- Production password recovery is not configured: SMTP/reset-delivery environment fields are absent and the public recovery endpoint correctly fails closed with HTTP 503.
- Release API/reset/Firebase/signing properties are present; Privacy Policy and Terms HTTPS properties remain absent, so the signed release AAB gate remains intentionally closed.
- The release runbook migration chain was corrected from the stale `000001..000007` list to the actual `000001..000011` chain.

No production Supabase DDL/grant mutation was performed in this audit.
