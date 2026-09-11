# LinkUp — IMPLEMENTATION STATUS / WORKLOG

> **Purpose:** factual implementation ledger for this repository. Read this file together with `PROJECT_RULES.md` and `README.md` before every implementation block so completed work is not repeated and README historical claims are not confused with current repository fact.
>
> **Mandatory workflow:** after every work block record files/capabilities actually added, evidence actually obtained, commit SHA(s), remaining work, and the next dependency-safe block.
>
> `PROJECT_RULES.md` has highest priority. **Existing design is frozen. Android + Go are active. iOS is frozen until direct user command.**

## 1. Status legend

- ✅ **EXISTS** — physically present in current `main`.
- ✅ **VERIFIED** — present and tested/validated with stated evidence.
- 🟡 **DESIGN ONLY** — approved visual contract exists; production Kotlin/Go/data flow does not.
- 🟠 **FOUNDATION ONLY** — production-oriented implementation exists but is not yet end-to-end production-complete.
- ❌ **DOES NOT EXIST** — no production implementation in current `main`.
- ⛔ **FROZEN** — intentionally untouched until direct user command.

## 2. Canonical constraints

| Area | Status | Rule |
|---|---:|---|
| Existing React/TS LinkUp design | ✅ | Frozen visual/UI contract. No redesign, cleanup, replacement or reinterpretation. |
| Android production client | 🟠 | Kotlin + Jetpack Compose only. |
| Backend/domain authority | 🟠 | Go. |
| Database | 🟠 | PostgreSQL/PostGIS via forward-only migrations. |
| React/TypeScript | 🟡 | Design reference only; no new production domain authority. |
| iOS | ⛔ | No Swift/SwiftUI/Xcode/assets/signing/tests/build/parity work. |
| Git | ✅ | Work directly in `main`; no PR/feature branch unless explicitly requested. |
| Ubuntu deployment | ⛔ | Do not touch until direct deployment/server command. |
| GitHub Actions / Google Cloud Build | ⛔ | Not part of canonical delivery flow. |

## 3. Frozen design reference — ✅ design

The original React/TypeScript files remain untouched and are the canonical visual reference: `App.tsx`, `PulseScreen.tsx`, `MapScreen.tsx`, `CreateLinkScreen.tsx`, `FlyScreen.tsx`, `MeScreen.tsx`, `NotificationsPanel.tsx`, `SlotCard.tsx`, `BottomNav.tsx`, shared UI primitives and `tailwind.config.js`.

Production Android now uses the same black/red token language and the same core Pulse/LINK/card/detail interaction structure where that capability is active. Fake prototype values such as hardcoded city/BPM/reliability/demo online data are **not** copied into production Kotlin before their server capabilities exist.

## 4. Repository / build / process foundation — 🟠

Exists:

- real `.gitignore` excluding `.env`, signing keys/keystores, APK/AAB, build outputs, logs/temp/IDE files;
- Go module/API process foundation;
- `/livez` and DB-aware `/healthz`;
- HTTP timeouts + graceful shutdown;
- request IDs and structured method/path/status/latency logging without bearer headers/request bodies;
- environment config;
- forward-only migration runner with `linkup_schema_migrations` checksum ledger and checksum-drift rejection;
- Android Kotlin/Compose project shell;
- frozen design colors transferred 1:1 to Kotlin.

Relevant commits:

- `.gitignore`: `12f211971413468ac82d2dca4cee5a4865c4b3d5`
- Go process foundation: `fce37365716fefcab2cf28b421607eb7e833ba93` → `9047a558e3916b87442512869082244c248e400a`
- migration runner: `fea9929a8a72efbb86c6cbe156e44537aad6bfd7`
- Android project/token layer: `c0bf43e01027face9ca12d9daf2685558e56544c` → `0fd7cf6671585adff53aa76c788ac7463dc1ce01`

Open build-infrastructure issues are recorded in Verification below. Do not assume Android is reproducibly buildable only because Gradle metadata exists.

## 5. Account / session / security foundation — 🟠

### PostgreSQL

`db/migrations/000001_accounts.sql` defines `app_users`, case-insensitive email/username uniqueness, hashed opaque sessions, password-reset tokens, `user_blocks`, PUBLIC/HIDDEN profile visibility, uk/en language baseline and supporting constraints/indexes.

### Go account/session API

Implemented:

- `POST /v1/auth/register`;
- `POST /v1/auth/login`;
- `POST /v1/auth/logout`;
- `GET /v1/me`;
- `PATCH /v1/me`;
- Argon2id password hashing with encoded/configurable parameters;
- 256-bit opaque bearer tokens with SHA-256-only server persistence;
- expiry/revocation checks;
- normalized identifiers and generic wrong-credential responses;
- bounded JSON bodies + unknown-field rejection.

### Auth abuse protection

Commit `5f430737fac07e84aea765e113d2ed5e9b0f1937` adds bounded fixed-window auth limiting, direct peer IP + route keys, `429` + `Retry-After`, bounded memory and configurable limits.

### Password recovery/reset

Commits:

- domain/HTTP/reset transaction: `28307b7b87e3c813a0e38aca962ad668ef9c4cdc`;
- TLS-only SMTP adapter: `34362616f1c197b3bbe6b1dcd22a5ab4ff3be72d`;
- SMTP env contract: `07c8a8e2535ec863f16c29297ade47883af75878`.

Endpoints:

- `POST /v1/auth/recovery/request`;
- `POST /v1/auth/recovery/reset`.

Security behavior includes one-time hashed reset tokens, expiry, previous-token invalidation, all-session revocation after password reset, TLS-only SMTP, no plaintext fallback, URL-encoded reset token and fail-closed behavior when delivery is not configured.

### Server-authoritative block controls

Base block API in `6f11f02b3f1f742851e38f76feff97db69d6f861`:

- `GET /v1/me/blocks`;
- `PUT /v1/me/blocks/{userID}`;
- `DELETE /v1/me/blocks/{userID}`;
- bearer authorization;
- self-block rejection;
- duplicate block idempotency at DB level.

Approval hardening in `054b223c65bb5e98147d7eb227b88b426553c32b` makes host/member/requester block effects transactional with social state: relevant pending/accepted relationship is revoked and affected Slot `accepted_count/state/version` is corrected atomically.

### Android account/session client

Implemented:

- Android Keystore AES-256-GCM bearer persistence;
- local expiry/decryption/key-loss clearing;
- register/login/logout/Me/recovery calls;
- HTTPS required outside emulator/loopback;
- process-death bootstrap with `Checking`, `SignedOut`, `SignedIn`, `OfflineSession`, `RecoverableError`;
- `401` clears revoked local session while temporary transport loss can retain an unexpired local bearer.

Relevant commits: `8c852bf446943b8e83c4954d007b6eebaa72d080`, `2bea65b4c19e5125270adf9fd769e741ec3f8989`.

## 6. Canonical Slot foundation — 🟠

Implemented in `2a1fb50728dd47a35f118a1f42d39d70428aca11`.

`db/migrations/000002_slots.sql` defines canonical `slots`, full lifecycle states, access modes `INSTANT / APPROVAL / WAITLIST`, visibility foundation, capacity + `accepted_count`, server version, Pulse indexes and generic `mutation_idempotency` with finite TTL.

Current foundation create surface creates **PUBLIC + APPROVAL + FILLING** Slots. Instant/Waitlist remain in canonical domain/data for later capability blocks.

Endpoints:

- `POST /v1/slots`;
- `GET /v1/slots/{slotID}`;
- `PATCH /v1/slots/{slotID}`;
- `POST /v1/slots/{slotID}/cancel`;
- `GET /v1/pulse`.

Implemented: mandatory mutation `Idempotency-Key`, SHA-256 request fingerprint, finite idempotency retention, server UUID/version/state, host-only edit/cancel, optimistic `expectedVersion`, capacity edit invariant, FILLING/FULL normalization, CANCEL state transition, block-aware Pulse/Get and explicit HTTP error mapping.

**Current foundation idempotency is effect-idempotent and returns the current canonical resource on replay. Exact historical response replay remains part of durable-offline/realtime hardening.**

## 7. Approval social loop — 🟠

Implemented in `054b223c65bb5e98147d7eb227b88b426553c32b`.

`db/migrations/000003_approval.sql` adds unique pending `slot_requests`, unique accepted `slot_memberships` and lookup indexes.

Viewer relationship states: `NONE / PENDING / ACCEPTED / HOST`.

Endpoints:

- `POST /v1/slots/{slotID}/request`;
- `POST /v1/slots/{slotID}/leave`;
- `GET /v1/slots/{slotID}/requests`;
- `POST /v1/slots/{slotID}/requests/{userID}/approve`;
- `POST /v1/slots/{slotID}/requests/{userID}/reject`;
- `POST /v1/slots/{slotID}/start`;
- `POST /v1/slots/{slotID}/complete`.

Implemented invariants:

- APPROVAL request is pending only;
- duplicate request and accepted re-request rejected;
- host cannot request own Slot;
- bidirectional host/requester block check;
- host-only requester identity list;
- host-only approve/reject;
- APPROVE serializes on Slot `FOR UPDATE`, atomically inserts membership/removes request/updates count+state+version;
- last seat moves Slot to `FULL`;
- pending LEAVE withdraws; accepted LEAVE decrements count and reopens `FULL → FILLING`;
- START requires accepted participant, clears pending requests and moves to `ACTIVE`;
- COMPLETE requires `ACTIVE`;
- critical mutations use the idempotency boundary;
- Pulse/Get expose viewer relationship;
- Block revokes the relevant host/member/requester relationship transactionally.

Source tests cover pending/withdrawal, approve-to-FULL, full rejection, LEAVE reopen, START/COMPLETE and Store/HTTP compatibility.

## 8. Basic Zero-Trace Coordination Chat — 🟠

Implemented in `b9ea7481c78f8f9a61ae3b8cb06cc6449a6feb9b`.

`db/migrations/000004_chat.sql` adds `slot_messages`, a 1..2000 trimmed-body DB check, recent-thread index and PostgreSQL terminal purge trigger.

Endpoints:

- `GET /v1/slots/{slotID}/chat/messages`;
- `POST /v1/slots/{slotID}/chat/messages`.

Implemented behavior:

- max message size 2000 Unicode code points;
- max recent response 100;
- PostgreSQL-created timestamp and server-generated UUID;
- author id/username/displayName/avatar response;
- bounded recent messages returned chronologically;
- every read/send reauthorizes host/current accepted membership;
- pending/stranger/left denied even with known `slotId`;
- terminal states close chat;
- block relation is checked for accepted participant vs host and blocked-author messages are filtered for the current viewer;
- terminal transition to `COMPLETED/CANCELLED/EXPIRED/MODERATED` physically deletes canonical PostgreSQL chat rows.

Realtime/offline retry/system messages are **not** falsely claimed here; they remain Chat V2 work.

## 9. Android production social binding — 🟠

### Canonical Kotlin data/API layer

Implemented in `375f087d7b038bf1e2bdefe322938330217b0358`.

Created:

- `core/network/SocialModels.kt`;
- `core/network/SocialApi.kt`;
- expanded `LinkUpApiClient.kt`;
- `core/social/SocialCoordinator.kt`;
- `SocialCoordinatorTest.kt` source tests.

Kotlin now models the server contract directly:

- Slot lifecycle `DRAFT/PUBLISHED/FILLING/FULL/ACTIVE/COMPLETED/CANCELLED/EXPIRED/MODERATED`;
- `INSTANT/APPROVAL/WAITLIST` access enums;
- `NONE/PENDING/ACCEPTED/HOST` viewer relationship;
- server `version`, organizer, capacity/count, PendingRequest and Chat models.

`LinkUpApiClient` now implements:

- Pulse;
- create/get/edit/cancel Slot;
- request/leave;
- host pending request list;
- approve/reject;
- start/complete;
- chat read/send.

Every critical Android mutation creates a cryptographically strong UUID-shaped `Idempotency-Key`. Kotlin does not locally invent Slot state/version; it accepts the canonical server response. No blind mutation retry was added.

`SocialCoordinator` owns explicit `Idle/Loading/Empty/Content/Failure` read states plus mutation state, serializes UI double-tap mutations locally with a Mutex, reconciles Pulse from server-returned state and clears chat/pending local state after relationship/terminal changes.

### Native foundation UI / routing

Implemented in `3c686fda1c438f10b2819cac46b931687ea04d0a`.

Added Kotlin/Compose surfaces:

- `AuthScreen` — real Login/Register/Recovery request;
- `PulseScreen` — real server list, search/category filtering, loading/empty/error/manual refresh;
- native `SlotCard` matching frozen dark/elevated/status/progress/action structure;
- `CreateLinkScreen` — 3-step real PUBLIC + APPROVAL foundation creation flow;
- `SlotDetailScreen` — viewer-state actions + host pending/Accept/Decline/START/COMPLETE/CANCEL controls;
- `EditSlotScreen` — host edit with canonical `expectedVersion` and accepted-count capacity floor;
- `ChatScreen` — bounded real chat load/send/manual refresh;
- `MeScreen` — real account data, block list/unblock, logout;
- `LinkUpApp` — session bootstrap routing and Pulse/Map/LINK/Fly/Me navigation shell;
- `MainActivity` — real dependency wiring.

Important design/data rule:

- existing React/TS design files were not modified;
- production Kotlin uses frozen LinkUp colors/card geometry/interaction language;
- hardcoded prototype city/BPM/reliability/demo online values were deliberately not copied as production data;
- Map/Fly navigation positions remain visible but clearly report that their required production capability blocks are not active yet, rather than pretending with fake behavior.

### Android network/build safety added with UI block

- debug uses `http://10.0.2.2:8080` only for emulator development;
- cleartext traffic is enabled only for debug manifest placeholder;
- release has cleartext disabled;
- release API URL comes from Gradle property `LINKUP_API_BASE_URL`;
- no production endpoint is hardcoded;
- blank release API URL fails closed in `MainActivity` instead of silently using a fake server.

## 10. Verification state

### Actually verified in the available execution environment

- earlier pure-Go liveness/session-token tests passed before external dependencies were introduced;
- standalone standard-library rate-limiter scratch test passed;
- standard-only block/recovery code was syntax/parse checked during development.

### Source tests now present but not yet executed in final dependency graph

- Go account/session/password/idempotency/Slot/Approval/Chat tests;
- Android `SocialCoordinatorTest` for real-data/empty Pulse, server-returned PENDING+version and terminal chat-state clearing.

### NOT yet honestly verified

The available execution environment has not provided the complete external Go/Android dependency/build chain. These gates remain open:

- `go mod tidy` and generated/verified `backend/go.sum`;
- full `go test ./...` with `pgx` + `x/crypto` + all new code;
- PostgreSQL integration/race tests;
- migrations `000001..000004` execution against disposable PostgreSQL, including terminal chat purge trigger;
- applying migrations to canonical Supabase (not performed in this work block);
- Android Gradle compile/unit tests/instrumentation;
- complete reproducible Gradle wrapper scripts/JAR validation;
- real Android ↔ Go ↔ PostgreSQL two-user smoke;
- release signing/AAB.

**Never mark these green without real execution evidence.**

## 11. Active foundation gaps

Server source now represents the mandatory account → Slot → REQUEST → APPROVE/REJECT → membership → LEAVE → START/COMPLETE/CANCEL → basic chat path, and Android source now binds that path.

Still incomplete before foundation can be called working/green:

- compile/test the current Go graph;
- execute PostgreSQL migrations/integration/race tests;
- compile/test Android;
- finish reproducible Gradle wrapper/build infrastructure;
- verified Android App Links for password reset (manual link/code completion added in section 20);
- device verification of direct Block user actions added in worklog section 19;
- device verification of Basic Me profile editing added in worklog section 19;
- verify/fix Compose compile/runtime issues found by real build;
- real two-user Android ↔ Go ↔ PostgreSQL smoke;
- production SMTP configuration smoke when deployment is explicitly allowed.

## 12. Later Version 1 blocks — ❌ unless noted design-only

Still required by the full README before final Version 1 Done:

- durable Android mutation outbox + transactional backend outbox;
- realtime snapshot/ordered deltas/reconnect/convergence;
- City Context/PostGIS locality and privacy-safe location;
- real Map/viewport/Places integration;
- Waitlist/host-control V2;
- Chat V2/realtime/system messages/stronger revocation-retention;
- notifications/push;
- BUMP/Reliability (design only exists for parts);
- City BPM/Vibe/Lasso/Hotspots/swarms;
- Fly Now/Travel/Motion production behavior (design exists);
- Me 2.0/Social Passport/Squad/Guardian/Ghost production behavior;
- AR/ranking;
- venue/BLE/offline proof;
- safety/accessibility expansion;
- ephemeral media/translation/audio;
- adaptive systems/weather/asset matching;
- LinkUp+ billing/travel/host/discovery/privacy/identity/rewarded access;
- full ecosystem/security/restore hardening.

### iOS — ⛔

All iOS implementation remains intentionally frozen and excluded from current Android/Go readiness until direct user instruction.

## 13. README historical claims

README text that says historical Android Event Core, Approval, chat, BUMP or migrations `000012/000013/000016/000018/000020/000022` were already implemented is **not current repository evidence**. Only code/migrations physically present in this repository and recorded here count.

## 14. Worklog

### 2026-09-06 — Design/platform contract finalized

- design frozen;
- Android active;
- Kotlin + Go mandated;
- iOS frozen.
- Commit: `ce12cbf075f2b911ac1c73577efffb79c20a0a55`.

### 2026-09-06 — Repository/Android/Go/DB foundation

- `.gitignore`, Go API health foundation, Android project/design tokens, accounts migration, bearer primitive, migration runner.

### 2026-09-06 — Account/session/security

- account/session/Argon2/pgx/API client foundation;
- auth rate limit `5f430737fac07e84aea765e113d2ed5e9b0f1937`;
- password reset `28307b7b87e3c813a0e38aca962ad668ef9c4cdc`;
- Android bootstrap/recovery `2bea65b4c19e5125270adf9fd769e741ec3f8989`;
- TLS-only SMTP `34362616f1c197b3bbe6b1dcd22a5ab4ff3be72d`;
- block controls `6f11f02b3f1f742851e38f76feff97db69d6f861`.

### 2026-09-06 — Canonical Slot foundation

- migration `000002_slots.sql`, Slot domain/store/API, optimistic version, idempotency, block-aware discovery.
- Commit: `2a1fb50728dd47a35f118a1f42d39d70428aca11`.

### 2026-09-06 — Approval social loop

- migration `000003_approval.sql`, REQUEST/withdraw, pending list, approve/reject, atomic last seat, membership/LEAVE, START/COMPLETE, viewer state.
- Commit: `054b223c65bb5e98147d7eb227b88b426553c32b`.

### 2026-09-06 — Basic Zero-Trace Chat

- migration `000004_chat.sql`, host/accepted bounded chat, blocked-author filtering, terminal physical purge trigger.
- Commit: `b9ea7481c78f8f9a61ae3b8cb06cc6449a6feb9b`.

### 2026-09-06 — Android social API binding

- canonical Kotlin social models/API/coordinator/tests.
- Commit: `375f087d7b038bf1e2bdefe322938330217b0358`.

### 2026-09-06 — Android foundation social UI

- Auth/Pulse/LINK/Create/Detail/Edit/Chat/Me/app routing;
- debug/release network boundary;
- no React/TS design modifications.
- Commit: `3c686fda1c438f10b2819cac46b931687ea04d0a`.

## 15. Current production readiness

**Android + Go Version 1 production readiness: 0%.**

Reason: substantial real source now exists on both server and Android for the mandatory foundation path, but the current dependency graph, PostgreSQL migrations, Android build and real two-user device/server flow have not yet been executed successfully. Source implementation without build/DB/device evidence is not production readiness.

## 16. Next exact work block

**Foundation verification/build-completeness + remaining Account/Me safety surfaces:**

1. audit/fix Gradle wrapper/scripts/JAR so Android build is reproducible without GitHub Actions;
2. obtain/generate `backend/go.sum` and run full Go tests when dependency access is available;
3. add Android reset-password completion/deep-link path;
4. add Android block action from Slot identity and profile edit UI;
5. add targeted Android unit tests for routing/action-state behavior;
6. run Android compile/test and fix all compiler findings;
7. run migrations/integration/race tests in allowed PostgreSQL environment;
8. run real two-user foundation smoke;
9. only after foundation is green move to transactional outbox/realtime/offline/City Context dependency block.


## 17. 2026-09-06 — Repository audit / transport and auth stabilization

Audited current `main` starting at `77349f8`, independently of historical Linkup-Plus claims.

Changes:
- corrected obsolete `New()` call in Go HTTP health test to `New(Dependencies{})`;
- JSON request decoding now consumes the whole 64 KiB bounded body and rejects trailing values/garbage/oversized whitespace before mutation;
- authentication storage failures now return generic 503, not credential-revoking 401; actual invalid credentials remain 401;
- Android validates the parsed API URI authority, rejects HTTP host-prefix impersonation and embedded credentials/query/fragment;
- Android disables automatic redirects and transport caching;
- non-JSON HTTP errors preserve their status/request ID; malformed successful JSON gets explicit `protocol_error`;
- logout always updates local session routing even when remote logout fails; UI handles the error.

Regression test sources: `request_boundary_test.go`, `ApiEndpointTest.kt`.
Validation performed: source review and `git diff --check`. **Go/Android tests have not run.**
Environment: Java 17 is available; Go, Gradle, Kotlin compiler and Android SDK were not found. Go download attempt did not pass network approval. No substitute checks are counted as compilation. `go.sum`, wrapper and all previous build/DB/device gates remain open. No server deployment or database mutation performed.

Production readiness remains **0%** under this ledger's verified end-to-end criterion.

## 18. 2026-09-06 — Android response invalidation / audit findings

- Mutation double taps are dropped while another action is running rather than queued; create/edit routing uses an explicit successful return.
- Generation/request tokens prevent old selected/Pulse/chat/pending replies from restoring disposed state or overriding newer mutation state.
- Terminal/access updates invalidate outstanding thread reads; sign-in UI is keyed by account and disposal clears all social data.
- Cancellation propagates and releases the action lock; successful sends deduplicate and bound the local recent list.
- Six additional deterministic suspended-response regression tests were added (nine social coordinator tests total).
- `REPOSITORY_AUDIT.md` records audit scope, corrected defects, prioritized remaining SQL/auth/Android/build issues and precise verification limitations.

Executed checks: diff whitespace/conflict check, Android XML parsing, preservation of design/iOS/existing migration files. Kotlin/Go tests remain unexecuted because toolchains are unavailable; no readiness increase or foundation-green claim.

## 19. 2026-09-06 — Account/Me functionality (user-directed feature phase)

Current user instruction: read all repository Markdown and add functionality now; existing audit fixes will be handled in a later phase. This supersedes the previous next-block recommendation to start with build/security fixes. Audit findings remain open; no uncommitted replay-access correction was published.

Added native Android flows using existing canonical Go endpoints:
- Me → Edit profile: display name, avatar URL (empty removes it), PUBLIC/HIDDEN preference and uk/en preferred language;
- save calls PATCH /v1/me and updates session profile only from the same account's server response;
- profile form has save/error states and retained draft fields across configuration recreation;
- Slot host identity → Block user;
- host pending requester identity → Block user;
- confirmation uses the selected server user ID; success clears social state, reloads Pulse and invalidates the block-list snapshot; failure remains visible in the dialog.

Files: EditProfileScreen.kt, MeScreen.kt, SlotDetailScreen.kt, LinkUpApp.kt, SessionCoordinator.kt. Existing React/TS visual reference, Go implementation, migrations and iOS unchanged.

Validation: source/API wiring review and git diff --check passed. Android build/device verification not executed. Preferred language persistence does not claim full UI localization; avatar URL editing does not claim image upload/rendering. Existing server concurrency/replay issues remain in REPOSITORY_AUDIT.md for the later correction phase.

Next feature: password-reset completion surface using existing reset endpoint. Production readiness remains 0% under the ledger's verified end-to-end criterion.


## 20. 2026-09-06 — Password reset completion functionality

- Auth → Forgot password → I have a reset link → Change password now calls the existing POST /v1/auth/recovery/reset endpoint.
- Accepts a pasted HTTPS reset link or canonical 32-byte base64url reset code. The pasted URL is parsed locally and never fetched; only the token goes to the configured API.
- New password + confirmation validation, server busy/error state, success message and return to Login are connected.
- Reset code/password fields are memory-only and cleared on success/back; no saved-state credential persistence was added.
- PasswordResetInputTest.kt adds valid code/link and malformed/duplicate-token test cases; source tests are not claimed executed.
- Profile/block feature block was published directly on main as 5c05a6251f5e515bc834d00dfdede1e0bdbe73c6.

Validation: git diff --check and source/call-site review passed. No build/deploy was run. Automatic verified App Link opening remains unimplemented; the manual reset completion path is now present. SMTP delivery depends on existing server configuration. Android/Go Version 1 verified production readiness remains 0% pending end-to-end evidence.

Next functionality in the active foundation: date/time selection for create/edit, then Hosting/Joined navigation using real server data. Existing audit fixes are reserved for the later correction phase per the user's current instruction.

## 21. 2026-09-07 — Native Slot date/time functionality

- Create step 2 now offers native date/time selection; step 3 previews the selected instant.
- Edit supports setting/changing startAt and clearing it through existing clearStartAt semantics; unchanged start time is omitted from the patch.
- Slot detail shows full date, local time, UTC offset and viewer time zone.
- New SlotScheduleField reuses existing UI colors and Android pickers. Schedule choice survives form configuration recreation; open picker dialogs are disposed with the screen.
- Local time conversion rejects DST gaps and explicitly asks which offset to use during repeated clock times. API continues serializing UTC Instant through the existing Go startAt contract.
- SlotScheduleTest adds ordinary half-hour-zone conversion, spring gap and autumn overlap cases.

Validation: source/API wiring review and git diff --check passed; Kotlin unit/device tests not run (toolchain unavailable). No server, migration, iOS or React/TS design modifications. This is optional start-time functionality, not full City Context/scheduled-expiry policy.

Next functionality: server-backed Hosting/Joined navigation, including ACTIVE Slots. Existing audit corrections remain deferred by user direction. Verified production readiness stays 0% pending full end-to-end execution.


## 22. 2026-09-07 — Current Hosting/Joined/Requests dashboard

- Me → My LINKs opens Hosting, Joined and Requests using authenticated GET /v1/me/slots?view=HOSTING|JOINED|REQUESTED (default HOSTING).
- Go derives the actor only from the bearer session. PostgreSQL selects the actor's own hosting, accepted memberships or pending requests and excludes blocks in either direction.
- Hosting/Joined include ACTIVE Slots absent from Pulse. Cancelled/completed Slots are excluded; Requests excludes ACTIVE Slots. Results are ordered by update time and bounded to the latest 100 per view; history and pagination are not implemented.
- Android reuses SlotCard and existing detail/manage/chat flows, loads again when returning from detail, and provides loading/content/empty/error/retry states. Request tokens discard responses from previous views and disposed accounts.
- Added HTTP test cases for bearer/default/invalid-view/ignored actor override, service relationship cases and Android delayed-response cases. Production uses the real PostgreSQL store; test doubles are only test fixtures.

Validation: source review and git diff --check passed. Go/Kotlin tests, PostgreSQL execution and Android device navigation remain unexecuted because the required local toolchains are unavailable. No deployment, live DB changes, iOS or design-reference changes. Existing audit fixes remain deferred per user instruction.

Next: extend current LINK management with participant visibility consistent with README privacy rules. Verified production readiness remains 0% pending full Android/Go end-to-end evidence.


## 23. 2026-09-07 — Host accepted-participant roster

README §9 requires clearer roster/request/accepted summaries. This block adds the host's current accepted list alongside existing pending requests, extending the canonical Kotlin/Go flow.

- Authenticated GET /v1/slots/{slotID}/accepted returns organizer-shaped identities only (ID, username, display name, optional avatar URL); no email, location or account metadata.
- PostgreSQL checks host ownership, current lifecycle, memberships and bidirectional blocks in a single statement snapshot. Missing/non-owned/terminal Slots return 404; a current empty roster returns items: []. Pending users, accepted members and strangers cannot read this host-only endpoint.
- Android Host controls → Accepted participants → Show/Refresh displays real API data with loading, empty, failure and retry states. Identity rows use the existing Block user confirmation/API flow.
- Selection/mutation changes invalidate the list; delayed responses cannot restore an old roster after completion, navigation or account disposal. Show reloads after a mutation; this block does not claim realtime roster updates or participant removal without blocking.
- Added HTTP/service contract tests, an Android suspended-response regression test, and PostgreSQL production-query cases using isolated temporary tables. PostgreSQL test opt-in: LINKUP_TEST_DATABASE_URL; it checks query behavior, not migration correctness.

Executed: git diff --check and source/interface/call-site review. Go, Gradle and Kotlin executables remain unavailable; none of the added tests or Android device flows were executed. No server connection, deployment, live database mutation, migration, iOS or React/TS design change.

Next functionality: continue README host-management scope with authorized participant removal and explicit state/capacity transitions. Existing audit corrections remain deferred by user direction. Verified production readiness stays 0% until Android/Go end-to-end gates have evidence.


## 24. 2026-09-07 — Host removes an accepted participant

README §10 host-remove/version-conflict/FULL→FILLING scope now has a Kotlin/Go path:

- POST /v1/slots/{slotID}/members/{userID}/remove requires bearer authentication, Idempotency-Key and a positive expectedVersion. The fingerprint binds Slot, target participant and expected version.
- PostgreSQL locks the Slot row, verifies current host and bidirectional blocks, rejects stale versions/non-current states, deletes membership, decrements accepted_count, increments version and reopens FULL to FILLING atomically. ACTIVE remains ACTIVE. The existing chat authorization reads membership under the Slot lock, so requests authorized after committed removal are denied.
- Replay handling rechecks host/block authorization and skips a second deletion/count update. No permanent ban or block is created: the user may request again while the Slot accepts requests.
- Android accepted roster offers Remove with confirmation, uses the selected Slot version, displays errors through existing mutation state and reloads the roster after success. A conflict requires refreshing the LINK before retrying.
- Added service/HTTP/Android test cases and extended the opt-in PostgreSQL transaction test with real remove/chat-authorization calls, stale-version/non-host checks, replay branch, count/version/state and revoked chat assertions. This test exercises the transaction helper, not the complete idempotency claim or concurrent connections.

Validation: source/interface review and git diff --check passed. Tests/build remain unexecuted (Go/Gradle/Kotlin/Android SDK unavailable); PostgreSQL concurrency/device evidence remains open. No production DB, deployment, migrations, iOS or design-reference changes. Existing audit work remains deferred by user instruction.

Next: continue host-management requirements, including waitlist and request expiry only with their canonical domain foundations. Verified production readiness remains 0% pending executed end-to-end evidence.


## 25. 2026-09-07 — Foreground chat polling and reconnect baseline

README §§4.9/6 coordination delivery and reconnect scope now has lifecycle-aware Android polling over the existing authorized Go recent-thread endpoint.

- Opening chat starts sequential snapshot reads while the Activity is RESUMED. Pause/stop/disposal cancels the polling job; resume starts a fresh read. Navigation/disposal invalidates pending chat results. Transport cancellation still follows the existing HTTP client's cancellation/timeout behavior; this does not claim immediate socket abortion.
- ChatPollingPolicy defaults to 5-second polling and capped retry delays up to 60 seconds. These are configurable operational defaults, not measured capacity/latency guarantees. Successful reads reset retry delay; authorization/closed/not-found/client-invalid responses stop automatic retries for that foreground session. Manual refresh and later resume can recheck access.
- Existing content remains on screen during a normal refresh. Failed reads replace content with an explicit error; send is disabled until an authorized snapshot succeeds. Server remains authoritative for every read/send.
- Sending invalidates older snapshots; refresh skips while a mutation is active so an in-flight old snapshot cannot erase the acknowledged send. Existing bounded recent lists remain in use.
- Added injected-delay polling/backoff/cancellation tests plus suspended-snapshot/send and access-revocation coordinator tests.

Executed: source/lifecycle/call-site review and git diff --check. Kotlin tests, compilation, device foreground/background/network behavior and two-client convergence remain unexecuted because toolchains are unavailable. No new dependency, backend endpoint, deployment, migration, iOS or canonical design-reference change. This is polling, not an ordered realtime stream, durable offline outbox or push notification implementation.

Next: continue durable coordination delivery foundations with persisted send identity and server idempotency before claiming offline retry. Verified production readiness remains 0% pending executed end-to-end evidence.

## 26. 2026-09-07 — current executed v1.0 verification status

Earlier sections that state the Gradle wrapper, Go tests, PostgreSQL execution or Android build are unavailable are historical and superseded by this dated evidence.

Green with executed evidence on the Ubuntu release host:

- Gradle wrapper/JAR tracked and runnable;
- Android debug unit tests, lint and assemble;
- Go unit/integration graph, vet and race suite;
- disposable PostgreSQL migrations `000001..000011`;
- PostgreSQL social lifecycle, revocation, replay, purge and last-seat concurrency tests;
- bounded-query and foreign-key index review;
- disposable database custom-format backup + restore drill;
- current live-design English/Ukrainian resource-key parity;
- current frozen-design layer bound to real v1.0 state/callbacks instead of the former fake profile/Pulse/Fly fixtures.

Not green / requires external production evidence:

- release Privacy Policy and Terms URLs/content;
- signed production AAB after legal URLs are supplied;
- Android device/instrumentation and physical two-user end-to-end runtime verification;
- live Supabase migration-ledger reconciliation and `spatial_ref_sys` privilege hardening (read-only audit found the ledger absent and broad PostGIS grants still present);
- production recovery-delivery smoke and production backup/restore operational evidence.

No percentage is assigned here: release status is gate-based. v1.0 remains **NOT DONE** until every required release gate above is closed, even though the previously unexecuted build/test/database gates are now green.


## 27. 2026-09-07 — audited current v1.0 state

Current executed state superseding older audit assumptions:

- GitHub/server source parity verified before the audit correction.
- Fresh Go unit/integration, vet and race gates are green after fixing concurrent migration-ledger creation.
- Fresh Android debug unit/lint/assemble gate is green.
- Supabase managed migration history contains `000001..000011`; core application-table grants are Go-API-only through `linkup_api`.
- Live database integrity checks are green for current v1.0 invariants.
- Ubuntu API/Caddy/HTTPS/App Links are healthy and the API systemd sandbox reports an `OK` exposure score.
- Firebase project/config identity is consistent across service account, backend and Android build configuration; physical FCM delivery remains unverified because no device token is registered.

Remaining production gates are concrete rather than source-completeness claims: physical two-user Android regression/instrumentation, production SMTP password recovery, Privacy/Terms HTTPS endpoints, signed release AAB verification, and remediation/acceptance of the remaining Supabase PostGIS public-surface advisories. FCM device-token registration and delivered push are early v1.1 §6.8 foundation, not a v1.0 release gate; bounded/manual refresh is sufficient for v1.0.


## 28. 2026-09-07 — production database boundary closed for v1.0 clients

Current verified state after production remediation:

- canonical repository migration chain is now `000001..000013`; fresh disposable PostgreSQL execution and PostgreSQL-backed Go tests pass;
- Supabase managed production history includes both new hardening changes;
- `anon` and `authenticated` no longer have `USAGE` on the `public` schema, so they cannot resolve LinkUp or PostGIS objects through the Supabase Data API;
- `linkup_api` retains `public` schema usage and application-table authority; live API health remains green;
- Go unit/integration, vet and race gates are green; Android debug unit/lint/assemble is green.

Still not production-complete: SMTP password recovery is unconfigured, physical two-user/device runtime evidence is absent, and release legal URLs are intentionally deferred. The signed release APK/AAB must remain blocked until those v1.0 release inputs/gates are closed. Real FCM token registration and delivered push remain unverified v1.1 §6.8 evidence only and do not block v1.0.


## 29. 2026-09-08 — iOS explicitly activated / native SwiftUI foundation

The user's direct 2026-09-08 instruction explicitly activates iOS work. The earlier conditional iOS freeze is therefore no longer the active platform gate for the work recorded below. Existing Android, Go, PostgreSQL and frozen React/TypeScript sources remain preserved; this activation does not create a second backend/domain authority.

Published directly to `main`:

- `27c321b2a4617314beb9c9ff16744d39e319c639` — native iOS SwiftUI design/application foundation;
- `44d4073d2ea32676ce14111e7f587a9d3f0ec349` — native iOS API transport/session/social-contract foundation.

Current iOS source now uses Swift + SwiftUI with an iOS 17 / Swift 5 language-mode target plus strict concurrency, an XcodeGen project manifest, frozen LinkUp visual tokens/primitives, native `Pulse · Map · LINK · Fly · Me` navigation, design-safe Pulse/Map/Fly/Me surfaces and the three-step Create LINK surface with all 16 canonical activity labels. The compiler version is supplied by Xcode rather than encoded as an unsupported `SWIFT_VERSION=5.10` language-mode value.

Security/network foundation now present in source:

- release HTTPS-only endpoint validation with debug loopback exception and no embedded credentials/query/fragment/base-path ambiguity;
- Apple Keychain bearer persistence using this-device-only accessibility and local expiry cleanup;
- ephemeral `URLSession`, redirects/cookies/cache disabled, bounded timeouts and 1 MiB streamed response limit;
- GET-only bounded retry; mutations are never blindly transport-retried;
- cancellation stops retry work;
- typed Codable account/Slot/chat models matching the current Go API authority;
- typed Pulse/My LINKs/roster/request/chat reads;
- UUID `Idempotency-Key` on critical Slot mutations;
- ambiguous in-process chat resend reuses the same idempotency key until acknowledged or definitively rejected;
- session bootstrap distinguishes signed-out, server-revoked, temporary offline and recoverable secure-storage/server states.

No fake production social telemetry was introduced: inactive BPM/reliability/BUMP/Passport/Map/Fly values remain unavailable/empty rather than fabricated. Create LINK publish remains disabled in the visual shell until real session/API binding is wired.

Executed on the Ubuntu host: repository/source parity checks, `git diff --check`, `Info.plist` XML parsing and `project.yml` YAML parsing. The host has neither `swift` nor `xcodebuild`; therefore Swift compilation, XCTest execution, simulator/device behavior, signing and App Store artifacts are **not verified and must not be marked green**.

The current tooling blocked publication of the credential-entry auth wrapper containing password/reset-token fields. No bypass was attempted. Consequently registration/login/recovery UI-to-API binding is not claimed complete even though Keychain/session/transport primitives exist.

Font roles are mapped to Outfit/Inter/JetBrains Mono, but the repository does not currently contain the corresponding bundled font resources, so exact iOS typography parity is not yet verified.

Next dependency-safe iOS block: compile on macOS/Xcode, fix compiler findings, add the permitted native auth binding, route RootView through canonical session state, bind real Pulse/Create/Me and Slot detail/actions to `LinkUpAPI`, then add coordinator-level stale-response/double-tap tests before any production-readiness claim.

## 30. 2026-09-08 — current native iOS source state (supersedes §29 capability list)

Section 29 remains the historical activation snapshot. The capability statements below supersede its now-stale claims that credential auth, RootView session routing, real social binding and Create LINK publish are still open.

Published directly to `main` after the initial foundation includes:

- `8df0ece` — native account recovery and profile editing;
- `70d1c33` — native Map place discovery;
- `6c48743` — native host-management flow;
- `0ba328a` / `36ad635` — privacy-safe City Context and server-city scheduling time zone;
- `1b48089` — authoritative realtime catch-up;
- `96c6aec` — durable mutation replay;
- `9b43267` / `0eddfec` / `dd4c2da` — organizer blocking and discovery convergence;
- `bfdcded` — safe password-reset URL routing;
- `6ab69c3` — native notifications surface;
- `341c40a` / `8f09695` / `f3de6a3` / `713d7ae` — DRAFT → publish API foundation, protected workflow persistence, restart recovery and Create LINK binding;
- `bbd93aa` — iOS Slot/Pulse action lifecycle aligned to Go server rules;
- `cdb59f6` — visible Pulse/Map social mutation error surface.

Current native iOS source now has:

- registration, login, logout, password-recovery/reset and `/v1/me` account flows through the Go API;
- RootView routing through canonical session state with Keychain-backed bearer recovery and foreground revalidation;
- real Pulse, Map, My LINKs, Slot detail/edit, request/leave, approval/rejection, participant removal, host lifecycle, block and Zero-Trace Chat bindings;
- privacy-safe City Context and canonical-place search without copying prototype fake telemetry;
- durable mutation journaling with owner fingerprinting, bounded replay window and fail-closed ambiguity handling;
- Create LINK using the canonical v1.1 `POST /v1/slots/drafts` → `POST /v1/slots/{id}/publish` workflow instead of the legacy direct-create client path;
- protected DRAFT workflow state with stable create/publish/cancel idempotency keys, process-death/reconnect resume, authoritative version reconciliation and server-side discard;
- client replay cutoff of 20 hours, deliberately below the backend default idempotency TTL of 24 hours; an unconfirmed draft ID is not replayed manually after the safe window because duplicate prevention can no longer be guaranteed;
- shared Slot lifecycle/action contracts matching server Pulse/request/edit/start rules, including removal of ACTIVE Slots from local Pulse reconciliation;
- visible mutation errors on quick-action Pulse/Map surfaces rather than silent failure.

Source verification executed on the Ubuntu host for the latest blocks: repeated repository/origin parity checks, `git diff --check`, Swift source delimiter/call-site scans, no hardcoded production URL/secret findings, and preservation of Android/Go/DB/frozen React/TypeScript sources for iOS-only commits.

Still **not verified**: Swift compilation, XCTest execution, XcodeGen generation under Xcode, simulator/device behavior, APNs/device notification delivery, signing, archive/App Store build and physical multi-account iOS runtime. This Ubuntu host has no `swiftc`, `xcodebuild` or `xcodegen`; none of those gates may be marked green from source review alone.

Exact next iOS verification priority: run XcodeGen and compile/tests on macOS/Xcode, fix compiler/concurrency findings first, then execute signed-in auth/Create/Pulse/Map/Me/Slot/chat flows on simulator and physical device before any iOS production-readiness claim.

## 31. 2026-09-08 — iOS recovery/session/privacy/localization hardening (supersedes §30 verification list)

Section 30 remains the prior source snapshot. Current `main` has additional native iOS hardening and release-gate work:

- `b3595ad` — authoritative Slot reconciliation no longer performs chat/roster reads after the refreshed lifecycle closes those surfaces, preventing terminal 404/closed responses from stalling realtime/durable convergence;
- `9cfd93c` — transient durable mutation and DRAFT→publish workflow journals retain iOS Data Protection and are also excluded from device/iCloud backup; source tests assert the actual file backup-exclusion resource flag;
- `d2d9ac1` — automatic password-reset routing trusts only the configured HTTPS recovery origin/port/path and exact single token parameter; manual pasted-code/link parsing remains a separate local-only path;
- `4461e84` — legacy mutation-key persistence was removed after durable owner-bound journaling became the single mutation replay authority;
- `3140716` — Associated Domains entitlement wiring was added for password-recovery Universal Links with a reserved fail-closed default rather than a fabricated production domain;
- `432e3dd` — iOS runtime build inputs were consolidated into one validated configuration authority instead of independent bundle reads;
- `0dba2bd` — local session deletion/security-storage failures now fail closed rather than silently claiming sign-out/session removal succeeded;
- `1e2b791` — active native v1.0 surfaces gained an English/Ukrainian localization baseline with 297 parity-checked keys, localized client safety/error copy, localized dynamic framing, and a Linux-runnable `ios/scripts/verify_localizations.py` gate.

Executed on the Ubuntu source host for the localization/hardening blocks: repeated origin parity guards, `git diff --check`, plist/YAML parsing where configuration changed, Swift delimiter/source scans, and the localization verifier (`297` en/uk keys, exact key parity, no empty values, active v1.0 static UI literals covered). These are source/configuration checks only.

The iOS localization baseline follows the same release semantics as Android: device/system locale selects app resources, while the server `uk`/`en` account field remains the user's profile language preference and is not promoted into a second client-localization authority.

Still external/unverified: macOS/Xcode compilation and XCTest, generated Xcode project inspection, simulator/physical-device language switching, VoiceOver/Dynamic Type/Reduce Motion QA, Universal Link delivery through the final production domain/AASA/Apple CDN, signing/archive/App Store validation, and physical multi-account iOS smoke.

Push remains a backend dependency rather than an iOS source claim. Current canonical Go push endpoints/store/sender are Android/FCM-only (`/v1/me/push/android`, `platform='ANDROID'`, Firebase sender). No APNs client registration path is added until Go/PostgreSQL expose an authoritative iOS/APNs contract.

## 32. 2026-09-09 — v1.1 server-authoritative Capability Registry foundation

This section supersedes the stale §12 implication that every v1.1 foundation is absent. The active v1.1 rollout architecture now starts with a fail-closed, server-authoritative capability registry while v1.0 remains the mandatory regression baseline.

Implemented in the current work block:

- forward-only migration `000018_v11_capability_registry.sql` with canonical keys `realtime`, `city_context`, `map`, `waitlist`, `chat_v2`, `notifications`, `bump`, `city_bpm`, `swarms`, `fly_now`, `fly_travel`, `fly_motion`;
- every seeded capability is `enabled=false` by default;
- registry rows have a globally monotonic revision, optional effective time, and `ALL` / `USER_ALLOWLIST` rollout scope;
- `anon`, `authenticated` and PUBLIC have no registry write authority; the Go API role has read authority only;
- authenticated `GET /v1/capabilities` returns the server-evaluated snapshot; missing rows, unknown keys, unavailable registry state and client parse failures resolve to disabled;
- the Go server independently gates v1.1 realtime, City Context, Map and Android push-registration endpoints and returns `403 capability_disabled` when the server capability is off, so a modified client UI flag cannot authorize the operation;
- v1.0 account/Slot/request/approval/basic-chat routes are intentionally not dependent on the registry, so a registry outage cannot disable the v1.0 social loop;
- Android starts from an all-disabled snapshot and only starts realtime, push token synchronization/notification permission flow, or Map network behavior after the corresponding server capability evaluates enabled;
- the frozen canonical Map implementation was not edited; an additive wrapper renders an inactive state when `map=false`.

Executed evidence for this work block:

- pre-change baseline: Go `go test ./...`, `go vet ./...`, `go test -race ./...` green; Android `testDebugUnitTest + lintDebug + assembleDebug` green with 64 unit tests;
- targeted Go capability/http/postgres tests green, including disabled/error fail-closed behavior and server-side allowlist enforcement;
- Android targeted/unit suite green with 68 tests after adding capability parser/coordinator coverage;
- post-change full regression is green: Go `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...`; Android `:app:testDebugUnitTest :app:lintDebug :app:assembleDebug` completed `BUILD SUCCESSFUL`;
- fresh disposable PostgreSQL applied canonical migrations `000001..000018`; both the full v1.0 social integration test and the new capability-registry integration test passed; the disposable database was dropped afterward;
- PostgreSQL integration proves default-off state, allowlist isolation and revision advancement on both enable and disable transitions.

No v1.1 capability is enabled by this foundation. Enabling any individual capability remains a later explicit gate after that feature's own implementation and real verification. FCM/push remains v1.1 §6.8 foundation and is not part of the v1.0 Definition of Done.


## 33. 2026-09-10 — v1.1 activation and Android City Context revocation safety

The user's direct “роби v1.1” instruction activates v1.1 development under RULE 8's explicit-order exception. README and PROJECT_RULES now reflect that instruction. This does not close any outstanding v1.0 release gates or enable production capabilities.

Current block starts from main `8372797` (which already contains city realtime and permission UX beyond the historical §32 inventory).

Changed:

- `CityContextCoordinator.kt`: discard cached city on HTTP 401/403, unavailable server City-Lock and revoked local location permission; auth failures now reach the existing session handlers as Failure instead of being hidden inside cached Content.
- After obtaining GPS, check coroutine cancellation and request generation before sending any observation; clear/sign-out or a newer request prevents the obsolete observation from reaching the resolver.
- Recheck location permission before submission; a PRECISE observation cannot be submitted after permission is downgraded to APPROXIMATE or removed.
- Temporary service failures continue to expose cached city with explicit refreshError; canonical city/access authority remains on Go.
- `CityContextCoordinatorTest.kt`: eight additional regression cases cover cached auth denial, resolver expiry, temporary outage, clear during GPS, a superseding request, permission loss/downgrade, a late server response and local permission failure.

Verification actually obtained:

- `git diff --check` passed; inspected coordinator call sites in MainActivity and LinkUpApp, including session invalidation and city realtime's Content gate.
- Attempted `./gradlew :app:testDebugUnitTest --tests com.linkup.app.core.city.CityContextCoordinatorTest`; wrapper download failed with `java.net.SocketException: Network is unreachable` for the pinned Gradle distribution. Tests were NOT executed and Android compilation is NOT verified in this environment.
- No Go/SQL changes, database writes, capability enablement, server deployment, design-reference or iOS modifications in this block.

Remaining: execute the targeted Android tests and full Android regression in a provisioned environment, then verify permission downgrade/sign-out/reconnect on device and two-client city convergence. Continue the README §15 dependency chain against current source, retaining all v1.0 release gates. No defensible numeric production-readiness estimate was obtained from this source-only block; v1.1 is NOT production-complete.


## 34. 2026-09-10 — cancellable Android city-location acquisition

Continues v1.1 README §§6.2–6.3 from main `df446b0` (the published equivalent of the prior local `2b14ca1`, with identical tree).

Changes:

- `LocationObservationAcquisition.kt` is the shared provider-attempt implementation used by `AndroidLocationObservationSource`: one attempt per provider, the existing 12-second per-provider timeout, fallback on unavailable/invalid fixes, immediate propagation of caller cancellation or revoked permission. A provider's own timeout can fall back; an outer operation timeout cannot.
- Android API 30+ `getCurrentLocation` now receives a CancellationSignal linked to coroutine cancellation. Legacy listener registration cleans up on cancellation, including cancellation racing registration, provider-disable, completion and registration failure.
- Device observations preserve their actual capture timestamp: missing timestamps no longer become `now`. Missing, non-finite, zero or negative accuracy is rejected before integer normalization.
- `CityContextCoordinator` retains a settled snapshot separately from transient Loading/refreshing presentation. Cancelling a newer request cannot restore an obsolete in-flight spinner; older responses remain generation-guarded.
- Added 14 unit test cases: 9 acquisition/cancellation/fallback cases, 3 observation-metadata cases and 2 overlapping-load/resolve cancellation cases (including both cached and initially empty state).

Verification:

- `git diff --check` passed; inspected the production acquisition call site, API-level branches, permission-failure handling and all coordinator state publication paths.
- Attempted the three targeted Android test classes via the repository Gradle wrapper. Pinned Gradle distribution download again failed with `java.net.SocketException: Network is unreachable`; no Kotlin compilation or test execution is claimed.
- Native CancellationSignal/listener cleanup requires Android device or instrumentation verification on both API 26–29 and API 30+; coroutine source tests alone do not prove platform GPS teardown.
- API contracts consulted: https://developer.android.com/reference/android/location/LocationManager and https://kotlinlang.org/api/kotlinx.coroutines/kotlinx-coroutines-core/kotlinx.coroutines/with-timeout-or-null.html .

Remaining: execute targeted tests and full Android regression in a provisioned environment; verify GPS cancellation, permission changes, foreground/background transitions and city-realtime recovery on device. City realtime expiry/recovery orchestration still needs its own executed convergence review. No backend/DB, deployment, capability enablement, design or iOS changes. Production readiness is not re-estimated from unexecuted tests; v1.1 release gates remain open.


## 35. 2026-09-10 — city realtime session recovery after City-Lock expiry

Continues README §§6.2–6.3 from main `04d48ce`.

Implemented:

- `CityRealtimeSession.kt` owns the sequential city stream recovery loop. MainActivity now scopes it to signed-in user identity plus server realtime/city-context capabilities and STARTED lifecycle, rather than cancelling it whenever City Context temporarily leaves Content.
- An expired City-Lock can transition through Empty/Loading during the existing authorized resolver flow without cancelling that same recovery. Loss of account, lifecycle or capability still cancels the owner coroutine.
- Initial session, successful expiry recovery and locality changes require canonical Pulse/loaded-Map snapshot refresh before consuming the next delta batch. Snapshot expiry/auth failures reach the same recovery/stop path rather than becoming a permanent failed-refresh loop.
- Ambiguous resolver/network failures trigger spaced GET readback only. No resolver POST is blindly replayed. If readback confirms no lock, or permission/locality is unavailable, polling suspends until explicit resolution/foreground initialization supplies a context.
- HTTP 401 ends the local session; 403 stops the city loop and refreshes capability state. Generic city errors preserve HTTP status in SocialError so retry/access decisions do not depend on a specific provider error message.
- The existing 5-second retry interval and existing cursor acknowledgement/reconciliation contract remain in use. Cached city with a failed/in-progress refresh is no longer reported as a successful city refresh by the MainActivity helper.
- Added 11 CityRealtimeSession tests covering expiry, ambiguous readback, empty/unresolved city, access denial, owner cancellation, failed snapshots and city switch; expanded coordinator assertions for preserved HTTP status.

Verification actually obtained:

- `git diff --check` and production call-site/control-flow inspection passed.
- Attempted `:app:testDebugUnitTest` for CityRealtimeSessionTest and CityContextCoordinatorTest; Gradle distribution download failed with `Network is unreachable`. Neither these tests nor Android compilation executed in this environment.
- No production capabilities enabled, backend/SQL/deployment changes or design/iOS modifications.

Remaining: execute the targeted and full Android regression, then verify two-device expiry/reconnect/airplane-mode and foreground/account/capability cancellation behavior. Continue canonical v1.1 dependency gates after that evidence. Source review does not establish production readiness; all unclosed v1.0/v1.1 release gates remain mandatory.

## 36. 2026-09-10 — server-authoritative Map locality boundary and canonical place search

Continues the active v1.1 City Context → Map dependency chain from main `c973f29a`.

Implemented:

- canonical place search now accepts and returns canonical `localityId`; Android Map search forwards the fresh server City Context locality identity instead of filtering by a human-readable locality label;
- Go validates locality IDs before place-search persistence access, while the PostgreSQL store filters by `canonical_places.locality_id`; legacy text locality remains only as a compatibility fallback when no canonical locality ID is supplied;
- Map viewport and place-detail service contracts receive locality identity as a separate server-derived argument, not as a client viewport/query field;
- both Map HTTP routes now require `map` and `city_context` capabilities and resolve a fresh City Context before querying Map data;
- PostgreSQL Map queries require `canonical_places.locality_id` to match the server-derived locality, preventing a modified client bbox/place ID from reading another locality;
- explicit PostgreSQL casts were added to Map positional parameters after real integration execution exposed ambiguous parameter typing (`integer >= text`);
- Android CityNetworkCoordinator now preserves HTTP status in Map errors, allowing access/capability recovery decisions without parsing provider messages;
- the canonical-place PostgreSQL integration cleanup was corrected to run before pool close and to fail visibly on cleanup errors, making repeated execution deterministic.

Executed evidence:

- targeted Go citymap/httpserver/places/postgres tests pass, including fail-closed invalid-locality handling, missing City Context preventing Map-store access, and server-derived locality forwarding for viewport and place-detail reads;
- a disposable native PostgreSQL database proved canonical locality-scoped place search, viewport isolation, place-detail isolation and no cross-locality leakage; the test passed repeatedly and the disposable databases were removed;
- the complete PostgreSQL integration package executed against a fresh disposable database with `LINKUP_TEST_DATABASE_DESTRUCTIVE=1` and returned `FULL_POSTGRES_RC=0`;
- full Go regression is green: `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...`;
- full Android regression is green: `:app:testDebugUnitTest :app:lintDebug :app:assembleDebug` completed `BUILD SUCCESSFUL`;
- `git diff --check` passes; no iOS files, DB migrations, frozen design-reference files, capability enablement or production runtime/deployment were changed in this block.

Remaining: physical Android/device verification of City Context → Map expiry/switch/reconnect behavior and two-client convergence are still required. Continue README §15 with hosting/access/waitlist only after the current City Context/Map dependency evidence is closed; no v1.1 production-readiness claim is implied by source and host tests alone.

## 37. 2026-09-10 — concurrency-safe WAITLIST admission and Android states

Continues README §6.6 from main `34bba8bd`.

Implemented:

- WAITLIST now uses the canonical `slot_requests` queue: free seats auto-admit, overflow remains FIFO pending, ordered by request creation time with deterministic user-ID tie breaking;
- leave, host removal and capacity expansion promote the oldest eligible request while holding the Slot row lock; accepted-count, membership and capacity invariants are updated in the same transaction;
- blocking either side revokes relationships and, for affected WAITLIST Slots, fills newly released seats from the remaining eligible FIFO queue before commit;
- WAITLIST publish/request paths are server fail-closed behind the `waitlist` capability; APPROVAL and INSTANT behavior remains unchanged;
- Android exposes capability-aware INSTANT / APPROVAL / WAITLIST draft selection and honest Join waitlist / Waitlisted / Leave waitlist states without changing the frozen layout;
- host WAITLIST controls do not expose manual Accept, preventing a client from bypassing FIFO admission.

Executed evidence:

- targeted Go slot/httpserver/postgres tests pass, including disabled/enabled WAITLIST capability routing;
- disposable PostgreSQL WAITLIST tests pass (`WAITLIST_DB_RC=0`) for FIFO promotion, host removal, capacity expansion, withdrawal, blocked candidates and concurrent FULL/reopen mutations without oversubscription;
- full Go regression is green in an LF-normalized Linux tree: `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...` (`GO_FULL_LF_RC=0`);
- full Android regression is green on JDK 17 / Android SDK: `:app:testDebugUnitTest :app:lintDebug :app:assembleDebug` completed `BUILD SUCCESSFUL` with 53 actionable tasks; the generated debug APK is present;
- no DB migration, iOS, production capability, production DB/runtime deployment or frozen design-reference change is part of this block.

Remaining: request-expiry/withdrawal hardening, expiry-during-mutation tests, complete optimistic-version conflict UX and two-client/device WAITLIST verification. Then continue README §6.7 Chat V2.

## 38. 2026-09-11 — verification pass finds and fixes a real Block/WAITLIST regression

This block audited "що є / чого не має" against §§37 and 11 before doing any new feature work, per RULE 5. Toolchains not previously available in-session (Go 1.24→toolchain 1.27.1, PostgreSQL 16 + `postgresql-16-postgis-3`) were available this session, so this block executes real backend gates rather than only reading source. Android SDK/Gradle network access and macOS/Xcode remain unavailable in this environment, so no Android/iOS claim is made or attempted here; no Kotlin/Swift/React/TS/design files were touched.

Found by running the full disposable-PostgreSQL integration suite (previously blocked in most prior sessions by unavailable toolchains):

- **Real production bug** in `internal/postgres/block_store.go`: the WAITLIST seat-promotion query added by the §37 concurrency-safe-waitlist block used `SELECT DISTINCT s.id::text … ORDER BY s.id` — the `ORDER BY` expression did not match the `SELECT DISTINCT` list (`uuid` vs `::text`), so PostgreSQL rejected the statement (`42P10`). Every `Block()` call reaches this query unconditionally, so this broke the core Block/unblock path repository-wide, not only WAITLIST-adjacent flows. Fixed by ordering on `s.id::text` to match the selected expression exactly.
- **Stale test fixture** in `block_store_test.go`: the hand-rolled `TEMP TABLE slots` fixture predated the `access_mode` column the new query reads, so `TestBlockPairTxRevokesRelationshipsAndAdvancesAffectedVersions` failed with `column s.access_mode does not exist` independently of the production bug above. Fixed by adding `access_mode` to the fixture and giving existing rows a non-WAITLIST value.
- **Stale test fixture** in `catalog_import_store_integration_test.go`: the locality centroid (`lat 49.44`) fell outside the test's own boundary polygon (`lat 49.0–49.1`), so the `validateLocalityGeometry` `ST_Covers` check added in `fa61832` correctly rejected it and `TestCatalogImportIsIdempotentAndAtomic` failed at the first import with `invalid canonical catalog`. Fixed by moving the centroid inside the polygon.
- **Stale assertion** in `v11_canonical_place_integration_test.go`: `TestV11CanonicalPlaceSlotAndMapIntegration` still asserted that publishing a WAITLIST draft returns `ErrInvalidState` ("queue/promotion semantics not available yet"). That gate no longer exists at the domain layer now that WAITLIST FIFO promotion is fully implemented (§37); the `waitlist` capability gate lives at the HTTP layer, not `slot.Service`, so calling the service directly now succeeds. Updated the assertion to expect a successful publish into `FILLING`, matching the already-covered WAITLIST FIFO integration tests.

No domain/product behavior was added or removed in this block; RULE 9 preserved. `go.mod`/`go.sum` were confirmed already tidy (`go mod tidy` produced no diff).

Executed evidence, this session, this host:

- `go build ./...`, `go vet ./...` — clean;
- `go test -count=1 ./...` — all packages pass;
- `go test -race -count=1 ./...` — all packages pass;
- fresh disposable PostgreSQL 16 + PostGIS 3.4, all canonical migrations `000001..000024` applied cleanly via `cmd/migrate`;
- `go test -race -count=1 ./internal/postgres/...` against that disposable database with `LINKUP_TEST_DATABASE_DESTRUCTIVE=1` — every test passes, including the previously-failing four above; the two subtests requiring the production-only `linkup_api` role correctly self-skip on a disposable database (expected, not a defect: that role is provisioned out-of-band on managed Supabase, not by these migrations);
- disposable database dropped after the run.

Not executed in this block: Android Gradle/SDK build (`ANDROID_HOME` unset, no SDK installed in this environment), Xcode/Swift compilation, live Supabase migration, any deployment. This block does not change Android/iOS/production-readiness claims; it closes a real backend regression discovered only because full PostgreSQL+PostGIS execution was possible this session, and leaves the Go backend regression baseline (v1.0 + v1.1-to-date) demonstrably green end-to-end on this host.

Next: continue README §15/§37 dependency chain — request-expiry/withdrawal hardening and optimistic-version conflict UX for WAITLIST, then §6.7 Chat V2 — only once an environment with Android SDK/Gradle network access is available to keep client-side work honestly verifiable per RULE 2/3.

## 39. 2026-09-11 — iOS re-frozen by direct user command

The user's direct instruction this session ("на айфон ми поки нічого не пишемо, пишемо тільки на андроїд") re-freezes iOS, overriding the 2026-09-08 activation recorded in §§29–31. Recorded as `PROJECT_RULES.md` RULE 4.1. No iOS files were read, changed, built or deleted in this or the preceding block; `ios/` is preserved as-is per RULE 9. All continuing work in this and future blocks, until the user unlocks iOS again, targets Android (Kotlin/Compose) and Go backend only.

## 40. 2026-09-11 — v1.1 WAITLIST request-expiry/withdrawal hardening (Go backend only)

Continues README §6.6 ("request expiry/withdrawal hardening", "expiry during mutation") from main `4bc1543`, per RULE 5 re-read of README/IMPLEMENTATION_STATUS/§37 "Remaining" before starting. Scope was deliberately narrowed and the narrowing is recorded honestly below rather than silently assumed complete.

**Scope decision:** expiry is implemented only for the v1.1 WAITLIST queue (`V11SlotStore`/`BlockStore` promotion paths), not for v1.0 APPROVAL pending requests. README places this requirement under §6.6 "Approval + Waitlist + Host Control V2", but v1.0's `SlotStore.Request/Approve/Reject/ListPending` is the frozen v1.0 regression baseline (RULE 8/9); changing its behavior without an explicit user request was judged out of scope for this block. Also deliberately out of scope: propagating expiry into shared read paths (`Get`/`Pulse`/`Map`/`MyLinks`/realtime viewer feed all compute `PENDING` by row existence, not TTL) — those queries are shared with v1.0 APPROVAL and with already-hardened realtime/idempotency-replay code (REPOSITORY_AUDIT.md P1 findings), so touching them safely needs its own reviewed block, not a side effect of this one.

Implemented:

- `LINKUP_WAITLIST_REQUEST_TTL` config (default `48h`, validated positive like every other TTL) bounds how long a WAITLIST `slot_requests` queue position stays eligible for promotion;
- `promoteOldestWaitlistTx` (used by leave, host removal, capacity expansion and block-triggered promotion — all 4 call sites) now skips and deletes any candidate whose `created_at` is at or past the TTL, evaluated against the same `now` used for the rest of the transaction so an expiry racing a promotion resolves atomically and deterministically ("expiry during mutation");
- `waitlistRequest` no longer returns `ErrDuplicateRequest` for a request whose own prior queue position has expired: the stale row is deleted and the new request proceeds, fixing the concrete "withdrawal hardening" gap (a user stuck behind a queue position that will never promote and that they cannot re-request into);
- a live (non-expired) duplicate request is still rejected, unchanged;
- `V11SlotStore`/`BlockStore` constructors now take `waitlistRequestTTL`; wired from `cfg.WaitlistRequestTTL` in `cmd/api/main.go`.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean;
- `go test -count=1 ./...` and `go test -race -count=1 ./...` — all packages pass;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000024` applied;
- `go test -race -count=1 ./internal/postgres/...` against that database (`LINKUP_TEST_DATABASE_DESTRUCTIVE=1`) — all pass, including two new deterministic tests (`v11_waitlist_expiry_integration_test.go`, timestamps backdated directly rather than sleeping): an expired queue position is skipped, purged, and the next eligible (later-queued but non-expired) candidate is promoted instead; a user can request again once their own expired position is gone, while a still-live duplicate is still rejected — and the existing FIFO/race/block-promotion WAITLIST tests remain green, confirming no regression to already-covered behavior;
- `go mod tidy` — no diff.

Not done in this block, and not claimed: v1.0 APPROVAL request expiry; read-side (`Get`/`Pulse`/`Map`/`MyLinks`/realtime feed) TTL awareness for WAITLIST viewer state — a WAITLIST requester whose position has technically expired still reads as `PENDING` until the next mutation on that Slot touches the queue; Android UI for any of this (`ANDROID_HOME` unset, no SDK in this environment, per RULE 2/3 no unverified client claim is made); optimistic-version conflict UX and two-client/device WAITLIST verification carried over from §37.

Next: either close the read-side WAITLIST-expiry gap as its own reviewed block, or move to README §6.7 Chat V2 — both remain Go-backend-first until an environment with Android SDK/Gradle network access is available.

## 41. 2026-09-11 — v1.1 Chat V2 foundation: SYSTEM messages for Slot lifecycle events (Go backend only)

Continues README §6.7 ("system messages for important Slot lifecycle events") from main `334c4c0`, per RULE 5. Chose this bullet first because it was the one concretely absent (confirmed by reading `internal/chat`, `chat_store.go` and the v1.1 outbox migration before writing any code): idempotent send, block-aware read filtering, terminal physical purge and realtime delivery of `slot.chat_message_created` via the existing v1.1 outbox/viewer-feed path (migration `000014`) were already present; only server-originated lifecycle notices were missing.

**Scope decision, stated honestly:** implemented exactly one event — `SLOT_STARTED` — as a complete, tested vertical slice (schema → domain model → store → lifecycle hook → tests) rather than all plausible lifecycle events at once. `MEMBER_JOINED`/`MEMBER_LEFT` are deliberately deferred: they have several more call sites (APPROVAL accept, INSTANT/WAITLIST auto-admit, WAITLIST promotion ×4, host removal, leave ×2) and would roughly double this block's review surface; extending `system_event_type`'s CHECK constraint in a follow-up forward-only migration is cheap once this shape is proven correct end-to-end. `SLOT_COMPLETED`/`SLOT_CANCELLED` are intentionally *not* planned: the existing terminal purge trigger deletes every `slot_messages` row for the Slot in the same statement that sets the terminal state, so a system notice for those transitions would be inserted and purged before any client could ever read it.

Implemented:

- migration `000025_v11_chat_system_messages.sql`: `slot_messages` gains `kind` (`USER`/`SYSTEM`, default `USER`), nullable `author_id`/`body`/`idempotency_key` (relaxed only for `SYSTEM`), `system_event_type` (checked against an explicit allow-list, currently just `SLOT_STARTED`), and `subject_user_id`. A single shape CHECK enforces the two valid combinations so a half-formed row can never be committed. No behavior changes for existing `USER` rows;
- `internal/chat/model.go`: `MessageKind`, `SystemEventType`, and `Message.Author`/`Message.Subject` become `*Author` (additive/`omitempty` JSON — a `USER` message's wire shape is unchanged);
- `chat_system_messages.go` (new): `emitSystemChatMessageTx`, a `postgres`-package-internal helper (bypasses the `chat` domain package entirely, like `blockPairTx`/`promoteOldestWaitlistTx` already do) that inserts one `SYSTEM` row in the caller's own transaction;
- `slot_store.go`: `hostLifecycle`'s `start` branch (shared unmodified by `V11SlotStore.Start`, so both v1.0 and v1.1 callers get it identically) emits `SLOT_STARTED` with `subject=host` in the same transaction as the `ACTIVE` state write, after clearing pending requests;
- `chat_store.go`: `ListRecent`'s query changes `JOIN app_users` to `LEFT JOIN` (twice, for author and subject) and the Go scan handles all-nullable columns; the existing block filter is unaffected for `SYSTEM` rows because a `NULL` author can never match a blocked/blocker id (`NOT EXISTS` over an unsatisfiable predicate is vacuously true — verified by the new integration test's stranger/host/member coverage, not just asserted);
- the pre-existing `slot_messages_outbox_after_insert` and terminal-purge triggers needed no changes: both already operate on `slot_messages` INSERT/DELETE regardless of `kind`, so `SYSTEM` rows get realtime delivery and terminal purge for free.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean;
- `go test -count=1 ./...` and `go test -race -count=1 ./...` — all packages pass;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000025` applied cleanly (the migration was iterated against this real database — an initial guess at the implicit `CHECK` constraint's default name was wrong and caught immediately by a real failed `ALTER TABLE`, not left as a guess);
- new `TestV11ChatSystemMessageOnSlotStarted` (disposable-DB integration) passes: a `SLOT_STARTED` row appears exactly once, ordered after the preceding user message, for both host and member, with `Author=nil`, `Text=""`, `SystemEventType=SLOT_STARTED`, `Subject=host`; a stranger still gets `ErrForbidden`; `Complete()` afterward purges it along with everything else;
- the full pre-existing `internal/postgres` suite (WAITLIST FIFO/race/block/expiry, v1.0 two-user lifecycle/chat-purge/revocation, capability/realtime/city/catalog integration) remains green under `-race` against the same rebuilt disposable database — no regression to the exact-count chat assertions (`assertChatRows`) or the message-count/ordering assertions that predate this block;
- `go mod tidy` — no diff.

Not done in this block, and not claimed: `MEMBER_JOINED`/`MEMBER_LEFT` (or any other) system event; Android UI for rendering `kind`/`systemEventType`/`subject` (no Android SDK in this environment, per RULE 2/3); realtime delivery of `SYSTEM` messages was not separately re-verified beyond confirming the existing outbox trigger fires unconditionally on INSERT — the v1.1 realtime viewer-feed authorization path for `slot.chat_message_created` was already covered by `TestV11RealtimeViewerFeedIntegration` before this block and was not re-run against a `SYSTEM` row specifically.

Next: extend `system_event_type` with `MEMBER_JOINED`/`MEMBER_LEFT` across their several call sites as a follow-up block, or pick up the read-side WAITLIST-expiry gap from §40, or the remaining §6.7 bullets (reconnect thread convergence, stronger revocation handling verification). All remain Go-backend-first until Android SDK/Gradle network access is available in this environment.
