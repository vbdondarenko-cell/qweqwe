# LinkUp repository audit — 2026-09-06

Scope: `vbdondarenko-cell/qweqwe`, starting main `77349f8`; current repository only. Read PROJECT_RULES.md, README.md (all sections), IMPLEMENTATION_STATUS.md and inspected repository structure, Android/API integration, Go domain/HTTP/storage, migration SQL and verification configuration. This is a **source audit**, not a passed build, live database audit, penetration test or device smoke. All runtime conclusions below remain gated by execution.

## Inventory and delivery decision

After first correction block: 117 tracked files, 43 Go files, 20 Kotlin files, 4 SQL migrations; 19 TSX files remain the canonical design reference. New implementation stays Kotlin/Go; iOS remains frozen. No Ubuntu connection, deployment, live database write, GitHub Actions or Google Cloud build was performed.

**Decision: foundation is not production-ready.** The implementation ledger's 0% verified end-to-end production readiness remains unchanged. This does not mean no code exists: it means no complete Android/Go/database release path has passed its required gates.

## Corrected in this audit

| Finding | Correction | Evidence / limitation |
|---|---|---|
| Health test called `New()` after constructor changed to require Dependencies | Updated call | Source mismatch removed; Go compiler unavailable |
| JSON decoder accepted a valid first object without consuming the rest of the body | Require EOF within the existing 64 KiB limit before mutation | Added tests for multiple values, garbage, oversized suffix and no registration write |
| Storage outage returned 401 from authentication/login | Return generic 503 for storage failures; preserve real 401 | Added outage regression tests without exposing storage error text |
| Android accepted HTTP URLs starting with local-host strings | Parse URI and require exact local host or HTTPS; disallow userinfo/query/fragment | Added endpoint test cases for host suffix and userinfo attacks |
| Redirects/caches were left to transport defaults | Explicitly disabled | Source checked; network transport tests outstanding |
| HTML/empty error responses lost HTTP status during JSON parsing | Preserve status and request ID; successful malformed JSON becomes protocol_error | Source checked; Android transport tests outstanding |
| Failed remote logout cleared token but left signed-in routing / uncaught UI error | Session state changes in finally; UI catches error | Remote session revocation is still not guaranteed on transport failure |
| Mutation Mutex queued a second tap | Use tryLock and explicit successful mutation return for create/edit routing | Added deterministic suspended-request test |
| Late reads/mutations could restore cleared chat, selection or account state | Request/generation invalidation, account-keyed UI disposal clearing | Added late-response/terminal/account-disposal tests |
| Cancellation was swallowed as a product error | Rethrow cancellation and release mutation lock | Added cancellation/lock recovery test |
| Old Pulse response could overwrite a mutation response | Invalidate older read and reconcile against retained list | Added out-of-order snapshot test |

First correction published on main: `7348ea0d0bcf217d7c195dc74b965351803010fb`.

## Remaining release blockers and source findings

Priorities express release impact, not a claim of a reproduced production exploit.

| Priority | Area / source | Finding and required next work |
|---|---|---|
| P0 gate | backend/go.mod; android/gradle | No go.sum; no gradlew/gradlew.bat/wrapper JAR. Pinned toolchain/dependency availability and compatibility are unverified. Restore reproducible toolchains, resolve checksummed dependencies, run Go and Android gates before adding downstream capabilities. Do not guess replacement versions. |
| P0 gate | backend/internal/postgres; db/migrations | No PostgreSQL integration/concurrency suite. Execute all four migrations on a disposable database, then test two-user social loop, rollback, last-seat allocation and terminal purge. Unit fake stores cannot prove SQL transaction safety. |
| P1 | postgres/slot_store.go, replay branches | All mutation replays read through getSlotInternalTx without current visibility/block authorization. A previously valid request key can retrieve a resource after access was revoked. Reauthorize replay separately from successful LEAVE's response semantics; regression must cover blocked actor and terminal/private resource. |
| P1 | postgres/block_store.go | Block deletes requests/memberships without first taking the same Slot locks as REQUEST/APPROVE/LEAVE. Concurrent creation/approval can escape cleanup or deadlock due to lock order. Define common lock order and reproduce both orderings on PostgreSQL. Pending-only block cleanup also does not advance Slot version. |
| P1 | postgres/account_store.go | Concurrent recovery creation does not serialize per account; more than one reset token may remain unused. Login password verification and session insertion can straddle password reset, allowing a session based on the previous password after revocation. Add account-level serialization/credential version check and race tests. |
| P1 | recovery/smtp.go | Timeout only bounds connection establishment; SMTP greeting/TLS/commands lack full-operation deadline/context cancellation. Reset URL validation accepts more than a verified HTTPS application link. Harden timeout and origin contracts with local test server. |
| P1 | postgres/pool.go; cmd/api/main.go | Raw wrapped connection/config errors are logged. Audit driver error redaction, especially malformed DSN; do not assume credentials cannot appear. |
| P1 gate | db/migrations | SQL files do not explicitly define the app-role grants/RLS/Data API exposure boundary. Actual managed database grants are unknown. Verify that mobile/public roles cannot bypass Go API or read password/session/reset/chat rows. No live exposure is asserted from these files alone. |
| P1 | Android session/social | Only bootstrap centrally handles revoked session. Authenticated social 401 needs coordinated account invalidation/routing. Current in-memory guards do not implement durable outbox, exact replay, process-death mutation recovery or realtime revocation. |
| P1 | ChatScreen; LinkUpApiClient | Composer clears draft immediately before send result; send lacks idempotent identity and an ambiguous-outcome flow. Concurrent manual refresh/send needs explicit message reconciliation. Avoid blind retries. Response readText is unbounded. |
| P1 gate | Android account/Me | Password reset completion/deep link, profile editing and direct block action still missing from production surfaces. Existing API methods are not completion evidence. |
| P2 | Android event screens | Start-time input absent from Create/Edit though API models support it; forms use remember rather than process-death-safe draft state. System Back navigation, localization, accessibility and design parity need device/render verification. |
| P2 | postgres/slot_store.go; Pulse | ACTIVE disappears from normal Pulse; there is no hosting/joined list to recover navigation after restart. Request validation does not explicitly enforce visibility; only PUBLIC creation is currently exposed. Pending list is unbounded. |
| P2 | migrate/migrate.go | No serialization between concurrent migration runners. Exercise checksum drift, repeat application and concurrent startup in isolated database. Existing applied migrations stay forward-only. |
| P2 | password/argon2id.go; config | Argon parameters have a floor but no upper work bound; integer-seconds duration multiplication may overflow. Add bounded config/hash validation and focused tests. |

## Product coverage

Account/session, profile API, PUBLIC APPROVAL Slots, Pulse, request/approve/reject/leave, lifecycle and ephemeral text chat have production-oriented source. Their integrated execution is not proven.

Map/Fly are capability messages in Android. Realtime/outbox, City Context/PostGIS locality, Waitlist, notifications, BUMP/reliability, city intelligence, swarms, travel/motion, Me/Squad/Guardian, AR, venue/BLE, media, adaptive features, billing and rewarded access remain incomplete or absent as recorded in IMPLEMENTATION_STATUS.md. README historical statements about green foundations and later migration numbers are not evidence for this repository. No requirement was removed or marked complete.

## Verification actually performed

- Read and cross-checked repository contracts and dependency order.
- Reviewed correction diffs and added targeted Go/Kotlin regression test sources.
- `git diff --check`: passed.
- Parsed tracked Android XML: passed.
- Confirmed correction scope leaves TS/TSX/CSS design, iOS and existing SQL migrations unchanged.
- Confirmed first published GitHub tree matches local commit tree.

Not run: Go compile/test/race/vet, Android compile/unit/lint/instrumentation, dependency vulnerability resolution, PostgreSQL integration, SMTP runtime tests, APK/AAB and two-user device smoke. Environment has Java 17 but no Go/Gradle/Kotlin compiler/Android SDK; Go download did not pass network approval. Tests in the repository are **test source, not green test results**.

Next dependency-safe block: restore build prerequisites; run the new tests and existing suites; fix SQL access/replay and account/block concurrency with executable PostgreSQL tests; finish account safety surfaces. Foundation remains the active block.


## Feature-phase update after the audit

The user explicitly directed feature implementation first and corrections later on 2026-09-06. Profile editing, direct host/requester Block actions, and manual password reset link/code completion now have Android UI/API bindings (IMPLEMENTATION_STATUS sections 19–20). Their prior absence above is historical audit context, not the current source inventory. Device verification, verified App Links and all remaining audit defects stay open; no SQL correction or build recovery is claimed in this feature block.
