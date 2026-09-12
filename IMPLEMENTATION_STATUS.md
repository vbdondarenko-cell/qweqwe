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

## 42. 2026-09-11 — v1.1 Chat V2: MEMBER_JOINED / MEMBER_LEFT system messages, and a same-transaction ordering fix

Continues §41 directly, closing the `MEMBER_JOINED`/`MEMBER_LEFT` gap it deferred.

Implemented:

- migration `000026`: replaces `slot_messages_system_event_type_check` to allow `MEMBER_JOINED`/`MEMBER_LEFT` alongside `SLOT_STARTED`. No other shape change; `000025`'s CHECK already covers these rows;
- `chat.SystemEventMemberJoined`/`SystemEventMemberLeft` added to `internal/chat/model.go`;
- wired `emitSystemChatMessageTx` into all 8 production call sites that add or remove a `slot_memberships` row: `SlotStore.Approve` (v1.0 APPROVAL accept, shared with v1.1) → `MEMBER_JOINED`; `SlotStore.Leave`'s accepted-leave branch (shared by APPROVAL and INSTANT) → `MEMBER_LEFT`; `removeMemberTx` (host removal, shared by APPROVAL and INSTANT) → `MEMBER_LEFT`; `V11SlotStore.Join` (INSTANT immediate admission) → `MEMBER_JOINED`; `waitlistRequest`'s auto-admit branch → `MEMBER_JOINED`; `waitlistLeave`'s member-leave branch → `MEMBER_LEFT`; `waitlistRemoveMember` → `MEMBER_LEFT`; and `promoteOldestWaitlistTx`'s successful-promotion return → `MEMBER_JOINED`. The last one is a single change point that covers all 4 of its callers (leave-triggered promotion, host-removal-triggered promotion, capacity-expansion promotion, and block-triggered promotion in `block_store.go`) for free.

**Scope decision, stated honestly:** the block-triggered bulk membership/request removal in `block_store.go` (a CTE-based bulk `UPDATE` across every Slot two users share, not a per-user loop) deliberately does **not** emit `MEMBER_LEFT`. Restructuring it into a per-row loop to attribute individual system messages raises a real privacy question — does announcing "X left" in a chat thread, timed exactly when a block occurred, leak that a block happened? — worth its own reviewed decision rather than folding into this mechanical wiring block. Its WAITLIST seat-promotion loop (which already existed) does still get `MEMBER_JOINED` for the promoted candidate, via the shared `promoteOldestWaitlistTx` change above; only the "member left because of the block" notice itself is withheld.

**Real defect found and fixed while adding real test coverage, not left for later:** `promoteOldestWaitlistTx`'s emission and the leave/remove branch's `MEMBER_LEFT` emission are in the same transaction and, for a leave-that-triggers-a-promotion, previously shared the exact same passed-in `now` for `created_at` — every timestamp elsewhere in this codebase intentionally reuses one `now` per transaction, but `emitSystemChatMessageTx` can now insert more than one row per transaction, and identical `created_at` values left their relative chat order to an incidental UUID tiebreak instead of the order the events actually happened in. Caught by a new integration test asserting exact message order, not just presence. Fixed by having `emitSystemChatMessageTx` use `clock_timestamp()` (advances per call, even inside one transaction) for `created_at` instead of accepting `now` as a parameter at all; every other table's timestamps are unchanged.

**Stale test fixtures found and fixed, same pattern as §17/case in this session's earlier blocks:** `accepted_test.go`'s hand-rolled `TEMP TABLE` fixture for `removeMemberTx` predated `emitSystemChatMessageTx`'s `INSERT INTO slot_messages`; without a temp `slot_messages` table the insert silently targeted the real table and failed on a non-UUID test id. Fixed by adding the temp table. `v1_social_integration_test.go`'s exact chat-message-count/order assertions (`two_user_lifecycle_chat_purge_and_revocation`, `cancel_physically_purges_chat`) predated `MEMBER_JOINED` on `Approve()`; updated them to assert the new, correct message set rather than loosening them.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean;
- `go test -count=1 ./...` and `go test -race -count=1 ./...` — all packages pass;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000026` applied cleanly;
- three new/extended integration tests pass: `TestV11ChatSystemMessageOnSlotStarted` (now also asserts the `MEMBER_JOINED` from `Approve()` in correct chronological order before `SLOT_STARTED`), `TestV11ChatSystemMessageOnMemberLeftAndRemoved` (two approvals → leave → host removal, asserts all 4 SYSTEM messages in order with correct subjects), `TestV11ChatSystemMessageOnInstantAndWaitlistJoin` (INSTANT `Join`, then WAITLIST auto-admit ×2 + queue + leave-triggered promotion, asserting the exact `[JOINED, JOINED, LEFT, JOINED]` order that exercises the `clock_timestamp()` fix);
- the full pre-existing `internal/postgres` suite remains green under `-race` against the rebuilt disposable database, including WAITLIST FIFO/race/block/expiry and the corrected v1.0 exact-count assertions;
- `go mod tidy` — no diff.

Not done in this block, and not claimed: block-triggered `MEMBER_LEFT` (see scope decision above — needs its own privacy-policy review, not a mechanical follow-on); Android UI for any `SYSTEM` message kind (no Android SDK in this environment, per RULE 2/3); README §6.7's remaining bullets (reconnect thread convergence beyond what already exists, stronger revocation-handling verification, activation/expiry/retention rules beyond what §25/existing purge already provide).

Next: decide and implement (or explicitly defer with reasoning) block-triggered `MEMBER_LEFT`; then the read-side WAITLIST-expiry gap from §40, or move on to README §6.8 Notifications. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 43. 2026-09-11 — v1.1 Chat V2: block-triggered MEMBER_LEFT (closes §42's deferral)

Resolves the privacy question §42 deliberately left open before implementing anything, rather than deferring again by default.

**Decision:** the notice is safe to add. It reads exactly like a voluntary leave ("X left") — it never says a block caused it — so it discloses nothing beyond what every other participant already learns the instant the blocked user's name silently disappears from the roster and stops being able to send/read chat (README §8.1 already requires blocked users filtered from every active chat/social/realtime layer, which this repository's chat read path already enforced before this block). Cross-checked against README §8.1 Privacy and §8.3 Anti-stalking: neither prohibits a same-content-as-voluntary-leave departure notice; §8.3 is about location/route/position inference, not membership departure text.

Implemented in `block_store.go`'s `blockPairTx`: before the existing bulk CTE-based `DELETE`/`UPDATE` across every Slot the two users share (a set-based statement, not a per-user loop, so it cannot itself call `emitSystemChatMessageTx` per row), a new `SELECT` captures the exact `(slot_id, user_id)` membership pairs about to be removed. After the bulk statement commits its changes within the same transaction, a loop emits one `MEMBER_LEFT` per captured pair, before the existing WAITLIST-promotion loop (so the chat order for a block that both removes a member and promotes their replacement is `LEFT` then `JOINED`, consistent with every other leave-then-promote path in this codebase). A pending-only request removal (nobody was ever a member) correctly emits nothing.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean;
- `go test -count=1 ./...` and `go test -race -count=1 ./...` — all packages pass;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000026` applied;
- `block_store_test.go`'s temp-table contract test extended: asserts exactly one `MEMBER_LEFT`/subject=`member` row for the removed accepted membership, zero rows for the pending-only removal, and — critically — that the idempotent block repeat does **not** duplicate the notice (the second call finds no membership left to remove, so its pre-fetch is empty);
- new `TestV11ChatSystemMessageOnBlockRemoval` (disposable-DB integration): host blocks one of two accepted members; both host and the remaining member see the `MEMBER_LEFT` notice with the correct subject; the blocked member themselves gets `ErrForbidden`, never the notice about their own departure;
- the full pre-existing `internal/postgres` suite remains green under `-race` against the rebuilt disposable database, including `TestV1SocialCorePostgresIntegration`'s existing block subtest and all WAITLIST/block-promotion tests (no regression to the promotion-order coverage already added in §42);
- `go mod tidy` — no diff.

This closes every `MEMBER_JOINED`/`MEMBER_LEFT` call site identified across §41-§43: `Approve`, `Leave`, `removeMemberTx`, `Join` (INSTANT), `waitlistRequest`, `waitlistLeave`, `waitlistRemoveMember`, `promoteOldestWaitlistTx` (covering leave/removal/capacity-expansion/block-triggered promotion), and now block-triggered removal itself. Not done or claimed: Android UI for any `SYSTEM` message kind (no Android SDK in this environment, per RULE 2/3).

Next: the read-side WAITLIST-expiry gap from §40, or README §6.8 Notifications, or the remaining §6.7 bullets (reconnect thread convergence, activation/retention rules beyond existing purge). Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 44. 2026-09-11 — closes the §40 read-side WAITLIST-expiry gap

Closes the gap §40 explicitly deferred: `Get`/`Pulse`/`ListMine` still read an expired WAITLIST queue position as `PENDING` until some other mutation on the Slot happened to touch the queue, even though `promoteOldestWaitlistTx`/`waitlistRequest` already treat it as gone for mutation purposes.

**Scope decision:** touched only the three `V11SlotStore` read methods actually reachable from the production HTTP surface (`Get`, `ListPulse`, `ListMine` — all wired via `cmd/api/main.go`'s `V11SlotStore`, which overrides the v1.0 `SlotStore` equivalents that remain unused in production). Left untouched: `getV11SlotInternalSQL` (only ever read immediately after the caller's own fresh mutation inside the same transaction, so it cannot observe someone else's stale expiry); the v1.0-only queries in `slot_store.go`/`my_slots.go` (dead code in the current v1.1-wired production path, and touching v1.0's frozen baseline for a v1.1-only concept was avoided in §40 for the same reason); `citymap_store.go` and `realtime_viewer_store.go` (Map and realtime-feed viewer-state surfaces — same class of gap, deliberately left for a follow-up rather than folded in here).

Implemented, in `v11_slot_store.go`:

- `getV11SlotSQL`/`listV11PulseSQL`: the `CASE...WHEN...THEN 'PENDING'` computation now requires `s.access_mode<>'WAITLIST' OR r.created_at>$cutoff` — an expired WAITLIST row reads as `NONE`, not `PENDING`. The `WHERE`-clause visibility grant (can the viewer see the Slot at all) is deliberately untouched: losing display accuracy on relationship state is a small, safe display fix; losing row visibility outright is a different and larger change this block does not make;
- `listV11MySlotsSQL`'s `REQUESTED` view: the row-selection predicate itself gets the same `OR r.created_at>$cutoff` condition, so an expired position drops out of "My Requests" entirely — unlike `Get`/`Pulse`, this list's entire reason to include the row was "you are pending here", so once that is no longer true the row should not appear at all;
- `waitlistExpiryCutoff()` computes `time.Now().UTC().Add(-s.waitlistRequestTTL)` once per call, reusing the same `waitlistRequestTTL` field the write-side (§40) already carries.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean;
- `go test -count=1 ./...` and `go test -race -count=1 ./...` — all packages pass, no assertion needed updating (a live, non-expired request always satisfies `created_at>cutoff`, and non-WAITLIST slots bypass the condition entirely, so this is additive for every existing scenario);
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000026` applied;
- new `TestV11WaitlistExpiredRequestReadsAsNone`: a queued WAITLIST request backdated past the TTL reads as `NONE` on `Get` and in `Pulse`, disappears from `ListMine("REQUESTED")`, while a live request from a different user on the same Slot is unaffected throughout;
- the full pre-existing `internal/postgres` suite remains green under `-race` against the rebuilt database;
- `go mod tidy` — no diff.

Not done or claimed in this block: Map (`citymap_store.go`) and realtime-feed (`realtime_viewer_store.go`) viewer-state expiry-awareness (same class of gap, explicitly deferred, not silently assumed fixed); Android UI (no SDK in this environment, per RULE 2/3).

Next: README §6.8 Notifications, or the Map/realtime-feed expiry-awareness deferred above, or remaining §6.7 bullets. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 45. 2026-09-11 — v1.1 Notifications foundation (README §6.8): pipeline + two event types, plus a real cross-cutting infrastructure bug found and fixed

Starts README §6.8. Before writing anything, read `internal/chat`, `internal/push` and the existing v1.1 outbox migration (`000014`) to find what already existed rather than assuming: the "domain transaction → transactional outbox" half of the canonical pipeline already existed, and so did the "FCM/APNs adapter" half (`push.Service.NotifyUser`, a working Firebase sender). What was missing was everything in between — dedupe, TTL, quiet-hours, frequency-cap, preferences, and a projector to connect the two existing ends. That is the scope of this block.

**A genuine pre-existing rule violation found while surveying, and fixed:** `internal/httpserver/approval_handlers.go` called `push.Service.NotifyUser` directly from three HTTP handlers (`requestSlot`, `approveRequest`, `rejectRequest`), which is exactly what README §6.8 states must never happen ("push is never called directly from an HTTP handler"). No existing test exercised this. Replaced the `requestSlot` and `approveRequest` direct calls with proper outbox-driven projection (below); left `rejectRequest`'s direct call in place with an inline comment explaining why (see scope decision).

Implemented:

- migration `000027`: `notification_preferences` (per-category toggles, quiet-hours window as start/end minute-of-day, IANA `timezone_name`, all defaulting to README §6.8's stated defaults) and `notification_deliveries` (the in-app/domain notification record — `source_event_id` `UNIQUE` is the dedupe boundary; `push_outcome` is a closed enum matching `internal/notification.PushOutcome` exactly);
- `internal/notification` (new domain package, no DB/network dependency): `Type`/`Category` mapping, `FrequencyCapped`/`QuietHoursExempt` per README's stated exemptions (MESSAGE/SECURITY never frequency-capped; only SECURITY/ACCOUNT/SYSTEM exempt from quiet hours and from preference toggles), `DeepLink` matching README's routing table exactly, `Preferences.InQuietHours` (DST-safe: converts to the user's `*time.Location` and compares wall-clock minute-of-day, verified against a real Ukraine DST transition in `Europe/Kyiv`, not just asserted), and `Decide` (expiry → preference → quiet-hours → frequency-cap, in that order, returning the terminal `PushOutcome` or `""` for "deliver");
- `NotificationProjector` (`internal/postgres/notification_store.go`): reuses the existing `RealtimeOutboxStore` connector-cursor mechanism (built in `000014`, previously exercised only by its own dedicated test — this block is its first real consumer) rather than inventing a second one. Recognizes exactly two outbox event types: `slot.membership_added` → `EVENT` "You're in!" to the new member (covers approval, INSTANT/WAITLIST auto-admit and WAITLIST promotion, the same set of call sites `emitSystemChatMessageTx` covers, for the same reason — one INSERT, one signal), and `slot.request_created` → `EVENT` "New request" to the host. Every other event type is checkpointed as Skipped, not silently ignored. Capability-gated per recipient, fail-closed (`notifications` disabled ⇒ no `notification_deliveries` row at all, not a suppressed one).
- Config: `LINKUP_NOTIFICATION_TTL` (14d), `LINKUP_NOTIFICATION_FREQUENCY_CAP_MAX` (5), `LINKUP_NOTIFICATION_FREQUENCY_CAP_WINDOW` (24h), `LINKUP_NOTIFICATION_POLL_INTERVAL` (5s), `LINKUP_NOTIFICATION_BATCH_SIZE` (200) — all documented in `.env.example`;
- `cmd/api/main.go`: an in-process ticker goroutine drives `ProcessBatch` on the same shutdown context as the HTTP server. Guarded against the typed-nil-interface trap explicitly (a nil `*push.Service` passed as the `pusher` interface parameter would be a non-nil interface wrapping a nil pointer, defeating the projector's own nil check and panicking on first send — branches explicitly instead of relying on the implicit conversion).

**Scope decision, stated honestly:** only two event types are wired. `rejectRequest`'s notification is deliberately NOT moved to the outbox path: `slot.request_removed`'s trigger has no way to know whether the row was removed by host rejection or by the requester's own withdrawal (`slot_requests` carries no "removed by" actor for the trigger to record), so a notification built from it alone could wrongly tell a user who withdrew their own request that it was rejected — left as a direct call rather than built wrong. `EVENT_REMINDER`, `EVENT_RECOMMENDATION`, `MESSAGE` grouping/collapse, `FRIEND_*`, and admin `PROMO` campaigns are README §6.8 scope not attempted in this block.

**A real, previously-latent bug in existing (not previously exercised) infrastructure, found and fixed, not worked around:** writing the first real Postgres integration tests against `NotificationProjector` surfaced that `DELETE FROM slots` (used only by three integration-test cleanup helpers — confirmed via repository-wide grep that no production Go code path ever hard-deletes a Slot row) crashes with a foreign-key violation when the Slot still has an accepted membership: the `AFTER DELETE` trigger on the cascade-deleted `slot_memberships` row tries to `INSERT INTO domain_outbox_events` referencing the parent `slot_id`, which by that point in the same cascading `DELETE` is already gone from the FK's point of view. This was previously invisible because the three cleanup call sites all discarded their errors (`_, _ = pool.Exec(...)` or a `t.Errorf` that doesn't stop the test) — orphaned rows were left behind in a disposable database that gets dropped anyway, with no functional consequence, until this session's connector cursor made "did this exact outbox sequence value ever get consumed" a question that mattered for the first time. Fixed the three cleanup call sites to delete `slot_memberships`/`slot_requests` before the parent `slots` row (side-stepping the cascade-during-cascade ordering entirely) rather than touching the trigger function itself, since production code has no reachable path to this bug today; hardening `linkup_emit_canonical_outbox()` against a real future hard-delete-a-Slot feature is left as explicit follow-up, not silently claimed done.

**A second, more fundamental latent bug found and fixed in `RealtimeOutboxStore.Checkpoint` itself** (`internal/postgres/realtime_outbox_store.go`, built in an earlier block, previously exercised only by its own dedicated test — again, this block is the first real consumer): `Checkpoint` required `event.Sequence == current+1` exactly, but `domain_outbox_events.sequence` is a Postgres `IDENTITY` column, which is **not transactional** — any rolled-back transaction anywhere in the system that happened to touch an outbox-triggering table permanently burns the sequence value it claimed, leaving a numeric gap no row will ever occupy. The strict `current+1` check treated every such gap as `ErrCursorOutOfOrder`, permanently wedging the connector — a real architectural fragility, not just a test artifact, since nothing before this block ever drove a real connector against the full, uncontrolled set of transactions a live system runs. Fixed by replacing the arithmetic check with a real-row check: advancing past `current` is allowed unless a real, uncommitted-for-this-connector row exists strictly between `current` and `event.Sequence` — which still correctly rejects skipping an actual unprocessed event (verified: the existing `TestV11RealtimeOutboxIntegration`'s explicit "cursor gap must be rejected" case, which skips over a real intervening row, still passes unchanged) while tolerating a pure numeric gap from a burned identity value.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on the new/touched files in this block shows no issues (the broader pre-existing `gofmt -l` output across the repository is unrelated pre-existing style drift in files this block did not touch, left alone rather than reformatted as an unrelated-scope change);
- `go test -count=1 ./...` and `go test -race -count=1 ./...` — all packages pass, including 19 new `internal/notification` unit tests (category/frequency-cap/quiet-hours-exemption mapping, deep-link routing table, quiet-hours wraparound/zero-width/non-wraparound windows, a real `Europe/Kyiv` DST-transition case, and full `Decide` precedence/gate coverage) requiring no database;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000027` applied cleanly;
- 6 new `NotificationProjector` integration tests pass: approval+request-created project correctly with push recorded and re-running is a no-op (dedupe proven, not just asserted); preference-disabled and quiet-hours suppression each produce the correct row with no push call; capability-disabled produces no row at all; no-pusher-configured records `SKIPPED_NO_DEVICE`; `recentFrequencyCappedCount`'s SQL is verified directly against seeded historical rows (SENT-only, capped-types-only, window-bounded);
- the `TestV11RealtimeOutboxIntegration`'s existing gap-rejection assertion still passes unchanged after the `Checkpoint` fix;
- the full pre-existing `internal/postgres` suite (WAITLIST, chat SYSTEM messages, v1.0 social lifecycle, capability/realtime/city/catalog integration) remains green under `-race` against the rebuilt disposable database, run twice to rule out the exact class of ordering flakiness this block was diagnosing;
- `go mod tidy` — no diff.

Not done or claimed in this block: `EVENT_REMINDER`/`EVENT_RECOMMENDATION`/`MESSAGE` grouping/`FRIEND_*`/admin `PROMO` campaign event types; `rejectRequest`'s notification staying outside the pipeline (see scope decision); user-facing preference CRUD (HTTP endpoints to read/edit `notification_preferences` — the table and defaults exist, no route yet); hardening `linkup_emit_canonical_outbox()` for a hypothetical future hard-delete-a-Slot production path; Android UI (no SDK in this environment, per RULE 2/3); production Firebase credentials were not configured or exercised — `push.Service.NotifyUser`'s existing "returns nil with zero devices" behavior (pre-existing, not changed here) means `SENT` is optimistic in that specific unconfigured-provider case, documented in code rather than silently assumed accurate.

Next: notification preference CRUD endpoints, or the remaining §6.8 event types, or README §6.9 BUMP + Reliability, or the Map/realtime-feed WAITLIST-expiry-awareness deferred in §44. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 46. 2026-09-11 — v1.1 Notifications: preference CRUD endpoints (closes §45's deferral)

Closes the "user-facing preference CRUD" gap §45 explicitly deferred: `notification_preferences` and its defaults existed and were already read by `NotificationProjector`, but no route let a user actually view or change their own settings.

Implemented:

- `internal/notification/preferences_service.go` (new): `StoredPreferences` — the CRUD wire/storage shape, JSON-tagged, deliberately kept separate from `Preferences` (the shape `Decide` needs, carrying a resolved `*time.Location` rather than a raw zone name — see the type's own doc comment for why the two aren't merged). `DefaultStoredPreferences()` mirrors `DefaultPreferences()` and the migration `000027` column defaults exactly, so a user who never saved anything sees the same values the projector already assumes for them. `Validate()` enforces the same bounds as the table's own CHECK constraints (quiet-hours minutes 0-1439) plus confirms the timezone name actually resolves via `time.LoadLocation` — catching a bad zone name at write time rather than leaving it to `NotificationProjector.loadPreferences`'s existing fail-closed-to-UTC fallback, so a user rarely experiences that silent downgrade at all. `PreferencesStore` interface (`Get`/`Update`) and `PreferencesService` (validates before writing, so the store is never responsible for shape rules `Validate` already covers).
- `internal/postgres/notification_preferences_store.go` (new): `NotificationPreferencesStore` implementing `PreferencesStore` against the existing `notification_preferences` table — `Get` returns `DefaultStoredPreferences()` on `pgx.ErrNoRows` (no saved row is every user's real starting state, not an error); `Update` is a single `INSERT ... ON CONFLICT (user_id) DO UPDATE` upsert.
- `internal/httpserver/notification_preferences_handlers.go` (new): `getNotificationPreferences` (`GET`) and `updateNotificationPreferences` (`PUT`, `notification.ErrInvalidPreferences` → 400 `invalid_preferences`), following the exact existing handler/route-registration pattern used by `push_handlers.go`. Routes registered in `server.go`: `GET /v1/me/notifications/preferences` and `PUT /v1/me/notifications/preferences`, both gated by `s.requireAuth(s.requireCapability(capability.Notifications, ...))` — the same capability the push-registration and notification-delivery paths already require, so preference CRUD is fail-closed along with the rest of the feature rather than independently exposed.
- `cmd/api/main.go`: wires `postgres.NewNotificationPreferencesStore(pool)` → `notification.NewPreferencesService(store)` → `httpserver.Dependencies.NotificationPreferences`. No new config/env vars needed for this block.

**Scope decision:** no admin/bulk preference endpoints and no push-token-specific settings were added — only the per-user self-service GET/PUT README §6.8 actually calls for. Preference changes take effect for future projections only (as documented on `StoredPreferences`); already-queued/decided deliveries are not retroactively altered, matching how `NotificationProjector` already reads preferences fresh on every `ProcessBatch` cycle.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file in this block — clean (`cmd/api/main.go` itself was already not `gofmt`-clean before this block, a pre-existing repository-wide condition confirmed via `git stash`/`gofmt -l` on the pre-block revision — left untouched rather than reformatted as an unrelated-scope change);
- `go test -count=1 ./...` — all packages pass, including 5 new `internal/notification` unit tests (`preferences_service_test.go`, against a fake in-memory store: defaults-when-unset, update-then-get round-trip, invalid-input rejection for every out-of-range/empty/unresolvable-timezone case, empty-user-ID rejection, nil-store construction rejection) and 6 new `internal/httpserver` handler tests (`notification_preferences_handlers_test.go`, against a fake `PreferencesStore`: GET defaults, GET fails closed (503) when the service dependency is nil, PUT-then-GET round trip through the real JSON wire shape, PUT rejects invalid body (400 `invalid_preferences`), PUT rejects malformed JSON (400 `invalid_json`), GET propagates a store error as 500);
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000027` applied cleanly; `go test -count=1 ./...` and `go test -race -count=1 ./...` (rebuilt database) both green, including 3 new `NotificationPreferencesStore` integration tests (`notification_preferences_store_integration_test.go`): unset user reads back exactly `DefaultStoredPreferences()`; update-then-get round-trips every field including a non-UTC `timezoneName`; two sequential updates for the same user leave exactly one row (`SELECT count(*)` asserted directly), proving the upsert path never inserts a duplicate;
- the full pre-existing suite (all packages, `-race`, rebuilt disposable database) remains green — no regression;
- `go mod tidy` — no diff (verified by diffing `go.mod`/`go.sum` against a pre-`tidy` backup, not merely by absence of `git diff` output).

Not done or claimed in this block: no Android UI (no SDK in this environment, per RULE 2/3); no admin-facing preference override/bulk endpoint; the remaining §6.8 event types (`EVENT_REMINDER`, `EVENT_RECOMMENDATION`, `MESSAGE` grouping, `FRIEND_*`, admin `PROMO` campaigns) and `rejectRequest`'s notification staying outside the outbox pipeline are still exactly as deferred in §45 — this block did not touch either.

Next: the remaining §6.8 event types, or README §6.9 BUMP + Reliability, or the Map/realtime-feed WAITLIST-expiry-awareness deferred in §44. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 47. 2026-09-11 — Map-side WAITLIST-expiry awareness (closes the `citymap_store.go` half of §44's deferral), plus two real test-fragility bugs found and fixed

**Scope correction, stated honestly:** the plan after §46 was the remaining §6.8 event types (`EVENT_REMINDER`, `FRIEND_REQUEST`/`FRIEND_ACCEPTED`). Before writing code, checked what each actually requires: `FRIEND_REQUEST`/`FRIEND_ACCEPTED` need an entire Friends/Connections domain that does not exist anywhere in this repository (no table, no package, README mentions it only as a one-line nav item) — not a small block. `EVENT_REMINDER` is inherently time-triggered ("starting soon"), but `notification_deliveries.source_event_id` is a `NOT NULL UNIQUE` FK into `domain_outbox_events` (migration 000027) — the schema only supports mutation-triggered notifications, and giving it a synthetic time-triggered source is a real architecture decision, not a quick addition. Rather than force either into an undersized block, picked the well-scoped item already fully specified in §44's own deferral note instead: `citymap_store.go`'s `PlaceSlots` had the same expired-WAITLIST-reads-as-PENDING gap the read side (`Get`/`ListPulse`/`ListMine`) was already fixed for.

Implemented, in `citymap_store.go`:

- `CityMapStore` gained a `waitlistRequestTTL time.Duration` field and `waitlistExpiryCutoff()` method, mirroring `V11SlotStore`'s own (same doc-comment reasoning: display-only staleness, never a visibility-grant change);
- `NewCityMapStore(pool, waitlistRequestTTL)` — signature change, both call sites updated (`cmd/api/main.go` now passes `cfg.WaitlistRequestTTL`, the same config value `V11SlotStore`/`BlockStore` already use for the identical concept; the one existing integration test's constructor call updated to pass a duration);
- `PlaceSlots`' `PENDING` `CASE` branch gained the same `AND (s.access_mode<>'WAITLIST' OR r.created_at>cutoff)` condition already proven correct in `v11_slot_store.go`.

**Scope decision, unchanged from §44:** `realtime_viewer_store.go`'s `slot_requests` reference is a visibility grant (can this viewer receive outbox events for this Slot at all), not a relationship-state display — the same category of change §44 explicitly declined to make ("losing row visibility outright is a different and larger change"). Left untouched, not silently claimed fixed.

**Two real, pre-existing test-fragility bugs found while verifying this block, and fixed, not worked around:** running the full `internal/postgres` suite more than once against the same disposable database (this package's own established, expected verification pattern — one migrated database, multiple `go test` invocations, not a fresh database per invocation) surfaced two latent bugs in `notification_store_integration_test.go` (§45), previously unnoticed because no session had run its tests enough times against one persistent database to expose them:

1. Every test in the file called `ProcessBatch` with a single hardcoded small batch (`ProcessBatch(ctx, 100)`) and assumed that one call would reach the test's own freshly created events. But the `"notifications"` connector's cursor (`connector_cursors`) is one durable row shared by every test in the package that ever runs a `NotificationProjector` — not scoped per test — and the whole `internal/postgres` suite shares one live database across the entire `go test` invocation. As more integration tests accumulate outbox volume ahead of a given test's own events (which grows over the life of the disposable database, and grew enough this session once new tests were added earlier in file-sort order), a bounded single call can exhaust its limit before ever reaching the events the test actually cares about — not a bug in `NotificationProjector` itself (it correctly respects the caller's limit), but a false assumption in the test. Fixed by adding `drainProjector` (repeatedly calls `ProcessBatch` until a short batch signals the connector has caught up to the current end of the outbox — the same guarantee production's own poll ticker provides by calling forever) and using it wherever a test needs to guarantee it reaches its own events; the final "re-running must not duplicate" assertions correctly stay single bounded calls, since by that point the connector is already caught up.
2. `TestNotificationProjectorRecentFrequencyCappedCount` seeded synthetic `domain_outbox_events` rows using fixed literal UUIDs. `domain_outbox_events.subject_user_id` is `ON DELETE SET NULL`, not `CASCADE` (migration 000014), so a synthetic row outlives the test's own host user being cleaned up — the literal `event_id` (a `UNIQUE` column) collided with the same test's own leftover row from an earlier run the moment it ran twice against one disposable database, failing with `duplicate key value violates unique constraint "domain_outbox_events_event_id_key"`. Fixed by generating a fresh UUID per run (`identifier.NewUUID()`, this repository's own convention) instead of literals, plus an explicit `t.Cleanup` deleting each synthetic row for hygiene.

Neither bug is reachable from production code or from a single clean test run against a freshly created database (both require the same disposable database to be reused across multiple `go test` invocations without being recreated) — but that reuse is this package's own normal, documented verification workflow (migrate once, run tests many times), so both were real landmines for the next person who ran this suite a few times in a row, not hypothetical.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000027` applied cleanly;
- new `TestV11MapPlaceSlotsWaitlistExpiryAwareness` (disposable-DB integration, `v11_map_waitlist_expiry_integration_test.go`): a WAITLIST Slot with a canonical place, two seats filled, a third viewer queued — reads `PENDING` on the Map before expiry, `NONE` after backdating the request past the TTL, while the Slot itself stays visible on the Map throughout (visibility grant unaffected, only the relationship display);
- to directly reproduce and confirm the fix for the two test-fragility bugs, ran the **full `go test -count=1 ./...` suite three consecutive times against the same, non-recreated, progressively-accumulating database** (the exact condition that originally surfaced both failures) — all three green, including `internal/postgres`;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff (diffed against a pre-tidy backup).

Not done or claimed in this block: `realtime_viewer_store.go`'s visibility-grant gap (deliberately out of scope, same as §44); `EVENT_REMINDER`/`FRIEND_REQUEST`/`FRIEND_ACCEPTED` (require real architecture/domain work, not attempted here — see scope correction above); Android UI (no SDK in this environment, per RULE 2/3).

Next: a scoped design for `EVENT_REMINDER` (likely: relax `notification_deliveries.source_event_id`'s FK, or have a time-based scanner emit a genuine synthetic outbox event so the existing dedupe/FK model stays intact — a real decision, not attempted without one), or README §6.9 BUMP + Reliability, or `realtime_viewer_store.go`'s visibility-grant gap. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 48. 2026-09-11 — v1.1 BUMP proof baseline (README §6.9): mutual server-verified confirmation + reliability bands (Go backend only)

Starts README §6.9. `Android Keystore-backed device proof` is explicitly out of reach in this environment (no Android SDK in any session since §38) — this block builds everything README requires that does not depend on it, and states that gap rather than hiding it.

**Design decision, stated up front:** README's "Client tap alone can never increase trust/reliability" is the load-bearing rule for this whole feature. A BUMP is therefore never a single write: it is two independent one-directional claims (`bump_submissions`, one row per `(slot_id, submitter_id, counterpart_id)`) that must BOTH exist before a `bump_confirmations` row — the only thing that ever credits reliability — is created. One person tapping "I met them" does nothing observable to either user's reliability by itself; it only becomes real once the other side also claims the same pairing for the same Slot. This is real mutual server-side verification, not a client-trusted flag, and is the closest honest approximation of the intent behind "server-verified proof" achievable without a real Android device-proof signature.

Implemented:

- migration `000028_v11_bump.sql`: `bump_challenges` (server-issued, single-use anti-replay nonce, TTL-bound — the foundation-block stand-in for the not-yet-reachable Keystore device proof), `bump_submissions` (the one-directional claim, PK `(slot_id, submitter_id, counterpart_id)` — this PK *is* the "one-event/one-contribution" anti-farm boundary README asks for), `bump_confirmations` (the verified mutual pair, PK `(slot_id, user_lo_id, user_hi_id)` with `user_lo_id<user_hi_id` normalizing the unordered pair to exactly one row), `reliability_events` (immutable per-user audit trail), `user_reliability` (aggregated private exact count + public coarse band, kept current in the same transaction that credits an event);
- `internal/bump` (new domain package, no DB dependency): `Band` enum (`NEW/BUILDING/RELIABLE/TRUSTED`) and `ComputeBand` — thresholds stated as an explicit initial policy in the doc comment, not claimed as final product tuning; `Service` validates/trims input in front of a `Store` interface the postgres package implements;
- `internal/postgres/bump_store.go`: `isEligibleBumpParticipantTx` is the shared anti-farm gate — a user must be the host or an accepted member of a Slot that has actually reached `ACTIVE` or `COMPLETED` before they can issue a challenge or be named as a counterpart; `IssueChallenge` enforces it and mints a single-use nonce; `Confirm` consumes the nonce (`FOR UPDATE`, rejects unknown/expired/already-consumed/wrong-owner nonces), re-checks both sides' eligibility and the bidirectional block relationship, records the directed claim (`ON CONFLICT DO NOTHING` — a repeat claim is a harmless no-op, never a duplicate contribution), and only when the reverse direction already exists does it insert the confirmation row and credit both users' `reliability_events`/`user_reliability` — gated by `tag.RowsAffected()==1` on the confirmation insert so a concurrent duplicate can never double-credit;
- `internal/httpserver/bump_handlers.go` + routes, all gated by the existing `capability.Bump` key (seeded disabled since §32, unchanged by this block): `POST /v1/slots/{slotID}/bump/challenge`, `POST /v1/slots/{slotID}/bump/confirm`, `GET /v1/me/reliability`, `GET /v1/me/bump-vault` (a user's own history of verified mutual BUMPs, README "BUMP Vault baseline");
- `cmd/api/main.go`/`.env.example`: `LINKUP_BUMP_CHALLENGE_TTL` (default 10m) wired into `bump.NewService`.

**Scope decisions, stated honestly:**
- No admin/other-user-facing exposure of reliability bands was added — `GET /v1/me/reliability` is self-only in this block. README's "private/public reliability bands" language implies a public-facing surface eventually (e.g. on another user's profile); no such profile-viewing endpoint currently exists in this repository to attach it to, so it is left for when that surface exists rather than bolted on speculatively.
- The nonce is the anti-replay mechanism, not a general `Idempotency-Key`-header mutation per README §10.2's usual pattern: a nonce is single-use by design, so retrying a dropped network response means calling `IssueChallenge` again, not replaying the same request. True idempotency of the actual *contribution* lives in `bump_submissions`' own primary key instead, which is safe under retry with a fresh nonce.
- Real Android Keystore-backed device proof, anti-farm behavior beyond the deterministic checks above (e.g. velocity/heuristic abuse detection), and any UI are not attempted — no Android SDK in this environment, per RULE 2/3.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean (`cmd/api/main.go` remains the pre-existing not-`gofmt`-clean file established in §46/§47, untouched further);
- `go test -count=1 ./...` — all packages pass, including 8 new `internal/bump` unit tests (`ComputeBand` boundary cases, service-layer input validation/trimming, limit clamping) requiring no database;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000028` applied cleanly;
- 7 new `BumpStore` integration tests pass: eligibility requires host/accepted membership on an `ACTIVE`/`COMPLETED` Slot and rejects a nonexistent Slot as `ErrNotFound`; a single one-directional claim does not verify and does not move the submitter's own reliability (the core "client tap alone" guarantee, checked directly against the database, not just asserted); the matching reverse claim verifies the pair and credits both users to count=1/`BUILDING`, with the vault surfacing the confirmed entry; an unknown, wrong-owner, replayed (already-consumed), or genuinely expired nonce is each rejected as `ErrInvalidChallenge`; a counterpart who was never actually in the Slot is rejected as `ErrNotEligible`; a blocked pair is rejected as `ErrForbidden`; a repeated one-directional claim via a fresh nonce is a no-op that does not create a second `bump_submissions` row and does not inflate the count; an unknown user's reliability reads as zero/`NEW` by default;
- one genuine edge case found and fixed during this verification, not worked around: a test using a 1-nanosecond challenge TTL tripped the migration's own `expires_at>issued_at` CHECK, because Postgres `timestamptz` only carries microsecond precision — `issued_at` and `issued_at+1ns` round to the identical stored instant. Not a production risk (`durationEnv`'s integer-seconds parsing path and the 10-minute default keep any realistic configured TTL far above that floor), but the test was corrected to a realistic sub-second TTL instead of leaving a misleading false positive in the suite;
- `go test -count=1 ./...` run three consecutive times against the same, non-recreated, accumulating disposable database (the reuse pattern that has previously surfaced real bugs in this session, per §47) — all three green;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff (diffed against a pre-tidy backup).

Not done or claimed in this block: Android Keystore device proof, Android UI, or any device-side anti-farm signal (no Android SDK in this environment); public/other-user-facing reliability band exposure (no profile-viewing endpoint exists yet to attach it to); velocity/heuristic anti-farm beyond the deterministic eligibility/uniqueness/block checks above; forgiveness/reliability-modifier interactions from README §6.26 (that section's own future block).

Next: `realtime_viewer_store.go`'s visibility-grant gap (§44/§47's remaining deferral), or a scoped `EVENT_REMINDER` design, or continuing README §6.9 (after-check flow refinements, private/public band exposure once a profile-viewing surface exists). Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 49. 2026-09-11 — a real production-blocking regression found in already-shipped Notifications (§45-§48), fixed; then EVENT_REMINDER (README §6.8) built on the corrected foundation

**A genuine, previously-undetected production bug, found while scoping `EVENT_REMINDER`, fixed before building on top of it — not worked around:** designing `EVENT_REMINDER` required `ReminderScanner` to call `linkup_enqueue_outbox`, which meant checking what the `linkup_api` role (the role the Go API actually authenticates as in production, per README §3.3) can execute. Reading migration history end to end: `000014` granted `linkup_api` exactly what a connector consumer needs on `connector_cursors`/`connector_delivery_receipts` (`SELECT/INSERT/UPDATE` and `SELECT/INSERT`). `000021` ("realtime outbox privilege hardening") later `REVOKE ALL`'d both from `linkup_api` entirely, on the stated assumption that "connector checkpoint writes belong to a dedicated future worker role, never to the API runtime role" — but no such worker role or separate binary was ever created anywhere in this repository; `cmd/api` is still the only server binary, and its own in-process ticker (`NotificationProjector`, wired since `000027`/§45) is the thing that calls `RealtimeOutboxStore.Checkpoint` in production, under whatever role `DATABASE_URL` authenticates as.

Confirmed empirically, not just by reading SQL: created a local `linkup_api` role, applied migrations through `000028`, connected as that role, and reproduced `permission denied for table connector_cursors` on the exact statement `Cursor()`/`Checkpoint()` issue. Every test in this repository's history connects as the Postgres superuser (`LINKUP_TEST_DATABASE_URL`), which bypasses role grants entirely — so this was invisible to the full green regression suite the Notifications feature was verified under in §45-§48. **In real Supabase production deployment, the entire Notifications pipeline shipped in §45-§48 would have failed on its very first checkpoint call.** This also retroactively means an existing test, `TestV11RealtimeOutboxIntegration`'s "runtime api role cannot mutate outbox" subtest, had been asserting the regression itself as correct behavior (`canCursorWrite`/`canReceiptWrite`/`canEnqueue` all required `false`) since `000021` — it was silently locking in the bug it should have caught.

Fixed, in order:

- migration `000029_v11_notification_connector_grants.sql`: restores exactly what `000014` originally granted on `connector_cursors`/`connector_delivery_receipts`, plus `EXECUTE` on `linkup_enqueue_outbox` (needed by the new `ReminderScanner` below) — re-verified empirically against the same local `linkup_api` role that reproduced the failure: every previously-failing statement now succeeds;
- `TestV11RealtimeOutboxIntegration`'s subtest renamed and its assertions corrected to the intentional model (`domain_outbox_events` stays `SELECT`-only and the trigger function/sequence stay uncallable/unusable directly — only the *trigger* is allowed to write outbox rows — but `connector_cursors`/`connector_delivery_receipts`/`linkup_enqueue_outbox` must be writable/callable), with a comment explaining why, so this class of regression cannot silently reappear;
- new `TestNotificationConnectorGrantsWorkUnderTheAPIRole` (opt-in via `LINKUP_TEST_LINKUP_API_DATABASE_URL`, skips otherwise): the test that would have caught the original regression — connects as `linkup_api` for real and exercises `Cursor`/`ListAfter`/`Checkpoint`/`linkup_enqueue_outbox` end to end, not superuser-bypassed.

**Only after establishing the foundation was actually sound did `EVENT_REMINDER` get built on it.** Design: every other notification type is mutation-triggered (a DB trigger already emits the outbox row); a reminder is time-triggered (`slots.start_at` crossing a threshold), and `notification_deliveries.source_event_id` is a `NOT NULL UNIQUE` FK into `domain_outbox_events`, so rather than relaxing that constraint — which every shipped type still depends on for its dedupe boundary — a new component emits a genuine synthetic outbox event through the exact same canonical path (`linkup_enqueue_outbox`) every trigger already uses, so the existing dedupe/connector/projector pipeline needs no structural change, only a new recognized event type.

Implemented:

- migration `000030_v11_event_reminder.sql`: `slot_reminder_emissions(slot_id PK)` — the scanner's own idempotency boundary ("has a reminder already been queued for this Slot"), independent of and prior to per-recipient event emission;
- `internal/postgres/reminder_scanner.go`: `ReminderScanner.ScanAndEmit` finds Slots with `start_at` newly within `leadTime` of now, `state IN ('PUBLISHED','FILLING','FULL')`, not yet claimed; for each, claims via `INSERT ... ON CONFLICT DO NOTHING` on `slot_reminder_emissions` (loses the race under a hypothetical multi-replica deployment → skip, not an error) and, only if it won the claim, emits one `slot.starting_soon` outbox event per current recipient (host + every accepted member, de-duplicated defensively even though a host is never expected to also hold a membership row) inside the same transaction as the claim. A Slot whose `start_at` already passed before this ever ran is deliberately never reminded late (`start_at > now()` excludes it) — a "starting soon" notice sent after the fact would be wrong, not just late, so silently not sending it is correct;
- `internal/postgres/notification_store.go`: `buildStartingSoonCandidate` recognizes `slot.starting_soon` (the one recognized type a DB trigger never emits — `event.SubjectUserID` already names the single intended recipient, set by the scanner itself, so there is no separate "who gets notified" decision to make here, unlike the trigger-driven candidates) and produces an `EVENT_REMINDER`/"Starting soon" candidate; `internal/notification.TypeEventReminder`'s `Category`/`FrequencyCapped`/`QuietHoursExempt`/`DeepLink` mappings already existed from §45's original model and needed no changes;
- `cmd/api/main.go`: `ReminderScanner` wired with its own ticker (config: `LINKUP_EVENT_REMINDER_LEAD_TIME` 30m, `LINKUP_EVENT_REMINDER_POLL_INTERVAL` 1m, `LINKUP_EVENT_REMINDER_BATCH_SIZE` 100), same shutdown-context pattern as the notification projector's.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000030` applied cleanly;
- the permission fix verified empirically twice: once directly against a hand-built `linkup_api` role reproducing then resolving the original failure, and again via the new opt-in `TestNotificationConnectorGrantsWorkUnderTheAPIRole` test;
- 4 new `ReminderScanner` integration tests: emits exactly one event per recipient (host + 2 members) and is idempotent on re-scan with no duplicate events; a Slot outside the lead-time window is untouched; a Slot whose `start_at` already passed is never claimed (proving the "never late" design decision, not just asserting it); invalid construction/scan-limit inputs are rejected;
- new `TestNotificationProjectorEventReminder`: a `slot.starting_soon` event (seeded via the same `linkup_enqueue_outbox` call the scanner itself makes) produces an `EVENT_REMINDER`/`SENT` `notification_deliveries` row with the correct `app://slot/{slotId}` deep link and a real push call;
- `TestV11RealtimeOutboxIntegration`'s corrected role-privilege subtest passes against the restored grants;
- `go test -count=1 ./...` run three consecutive times against the same, non-recreated, accumulating disposable database — all three green, including every test above and the opt-in role test;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database (with the `linkup_api` role present so migrations' conditional grants actually execute, matching real deployment) — green;
- `go mod tidy` — no diff.

Not done or claimed in this block: Android UI/client for `EVENT_REMINDER` (no SDK in this environment); `EVENT_RECOMMENDATION`/`MESSAGE` grouping/`FRIEND_*`/admin `PROMO` campaigns (still exactly as deferred in §45); `realtime_viewer_store.go`'s visibility-grant gap (§44/§47's remaining deferral, untouched by this block); a genuinely separate worker-role/process for the connector (this block restores the grants the actually-shipped single-process architecture needs — it does not build the role-separation architecture `000021`'s comment aspired to, which would require a real second binary/deployable this repository does not have).

Next: `realtime_viewer_store.go`'s visibility-grant gap, or `EVENT_RECOMMENDATION`/`FRIEND_*` (the latter still blocked on a nonexistent Friends/Connections domain — see §47's scope correction), or README §6.9's remaining after-check/public-band-exposure work. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 50. 2026-09-11 — closes the `realtime_viewer_store.go` visibility-grant gap deferred across §44/§47/§49 — with an honest scope correction found while writing the test

Closes the last of the three WAITLIST-expiry-awareness gaps identified back in §44: `viewerRealtimeSQL`'s `EXISTS(slot_requests)` branch — which grants a viewer realtime visibility into a Slot's own lifecycle events (`slot.created`/`updated`/`state_changed`) because they hold a WAITLIST queue position — never carried the same TTL awareness the read side (`Get`/`ListPulse`/`ListMine`, §44) and the Map (`PlaceSlots`, §47) already do. `RealtimeViewerStore` gained the same `waitlistRequestTTL`/`waitlistExpiryCutoff()` field/method those two already use, and the `EXISTS(slot_requests)` branch gained the identical `AND (s.access_mode<>'WAITLIST' OR r.created_at>cutoff)` condition proven correct in both prior fixes. `NewRealtimeViewerStore`'s signature changed accordingly; `cmd/api/main.go` now passes `cfg.WaitlistRequestTTL` (the same value every other WAITLIST-aware store already receives).

**An honest scope correction found while writing the test, not glossed over:** the first test attempt used a normal PUBLIC Slot and failed — not because the fix was wrong, but because `viewerRealtimeSQL` has a *separate* `PUBLIC` branch (`s.visibility='PUBLIC' AND s.state IN ('PUBLISHED','FILLING','FULL')`) that already grants blanket visibility to every viewer whenever a Slot is public and in that state range, which completely masks any effect from the `slot_requests` branch. Investigating further: every domain transition that moves a Slot *out* of `PUBLISHED`/`FILLING`/`FULL` (`Start`, `Cancel`) unconditionally `DELETE`s all of that Slot's `slot_requests` rows first — so in the **current** product surface (only `PUBLIC` visibility is ever created; README §4.3 states wider visibility is v1.1 scope not yet exposed), this branch has **no reachable production impact on its own today**: either the Slot is still `PUBLIC`+`PUBLISHED`/`FILLING`/`FULL` (and the `PUBLIC` branch already grants everyone visibility, live-WAITLIST-or-not), or it has moved past that range (and `slot_requests` has already been purged for every requester, expired or not). The fix is still correct and worth having — for the moment non-`PUBLIC` visibility ships, and for defensive consistency with the two already-proven fixes — but is stated here as a fix made ahead of an actual exposure, not one closing a live gap, since claiming otherwise would not match what was actually found.

The test itself proves the branch's own logic is correct despite that: it directly sets `slots.visibility='PRIVATE'` via SQL (the column and its `CHECK` constraint already support non-`PUBLIC` values per migration `000002`; no domain method sets one yet) to isolate the `slot_requests` branch from the `PUBLIC` branch, then shows a live WAITLIST position grants visibility, an expired one (backdated past the TTL, same helper used throughout this WAITLIST-expiry work) does not, and every other access path (host, both accepted members) is completely unaffected throughout — plus that the connector cursor still advances across the now-hidden event so pagination cannot get stuck, mirroring the existing blocked-stranger assertion in `TestV11RealtimeViewerFeedIntegration`.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean (`cmd/api/main.go` remains the pre-existing not-`gofmt`-clean file established in §46, untouched further);
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000030` applied cleanly;
- new `TestV11RealtimeViewerWaitlistExpiryVisibility` passes, proving the branch's logic correct via the isolation technique above;
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (with the `linkup_api` role present, so §49's migration `000029` grants and the opt-in role test both continue exercising real production privileges) — all three green;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff.

Not done or claimed in this block: no live production exposure was closed (see the scope correction above — none existed to close); non-`PUBLIC` visibility itself (`LINKS`/`SELECTED`/`CITY`/`LASSO`/`TRAVEL_CORRIDOR`/`PRIVATE`, README §4.3) is still entirely unimplemented at the domain/API level, only the schema and this one query branch are now ready for it; Android UI (no SDK in this environment, per RULE 2/3).

Next: `EVENT_RECOMMENDATION` (still needs a real recommendation-candidate design, not attempted casually), or `FRIEND_*` (blocked on a nonexistent Friends/Connections domain — §47's scope correction), or README §6.9's remaining after-check/public-band-exposure work, or beginning non-`PUBLIC` visibility itself now that one of its downstream consumers is ready. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 51. 2026-09-11 — grant-regression audit (negative result, recorded); WAITLIST queue position/expiry in the host pending list (README §6.6)

**Before picking a next feature, audited every other `REVOKE ALL ... FROM linkup_api` across the migration history for the same class of bug §49 found** (a hardening migration revoking something an in-process caller actually needs): read `000020` (capability registry — re-grants `SELECT`, correct: no production Go path ever writes `capability_registry`), `000022` (City Context/canonical places — re-grants `SELECT` on reference tables and full read/write on `city_context_locks`, correct: `cmd/catalog-import`, the only writer of `localities`/`canonical_places`, connects via its own separate `LINKUP_CATALOG_DATABASE_URL`, never `linkup_api`), `000023` (`linkup_emit_city_slot_outbox` — trigger-only, correctly never granted to `linkup_api`, matching the same trigger-vs-callable-function distinction the corrected `000021`/§49 test now documents), and `000024` (`linkup_schema_migrations` — only `cmd/migrate` touches it, via its own separate `LINKUP_MIGRATION_DATABASE_URL`, confirmed by grep, never `linkup_api`). **Result: no further instances found.** `000021` (fixed in §49) was the only hardening migration that revoked something a component living inside `cmd/api`'s own process actually depends on; every other one correctly matches what actually runs under that role. Recorded here as a negative result worth having on file, not silently discarded — it closes the question of "are there more of these" rather than leaving it open.

**Implemented, closing part of README §6.6's "complete roster/request/waitlist states":** the host's pending-requests list (`GET /v1/slots/{slotID}/requests`) previously returned the identical shape for APPROVAL and WAITLIST Slots — just requester + `requestedAt` — giving a host managing a WAITLIST queue no visibility into FIFO position or which positions are already stale. `slot.PendingRequest` gains `QueuePosition *int` (1-based, WAITLIST only, nil for APPROVAL) and `Expired bool` (mirrors the same `waitlistExpiryCutoff` TTL check already applied everywhere else in this WAITLIST-expiry work — §40/§44/§47/§50 — always false for APPROVAL). `V11SlotStore.ListPending` overrides the inherited v1.0 `SlotStore.ListPending` to compute both via `ROW_NUMBER() OVER (ORDER BY r.created_at ASC, u.id ASC)` — the exact ordering `promoteOldestWaitlistTx` already promotes from, so the displayed position always matches what would actually happen on the next promotion. APPROVAL-mode Slots get the same query shape but with `QueuePosition`/`Expired` left at their zero values, matching v1.0's original contract exactly. No HTTP handler change was needed: `listPendingRequests` already serializes the domain type directly, so the new JSON fields (`queuePosition`, `expired`) flow through automatically.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000030` applied cleanly (no new migration in this block — pure Go/query change);
- new `TestV11ListPendingWaitlistQueuePositionAndExpiry`: three requesters queue in FIFO order on a capacity-2 WAITLIST Slot (both seats filled first); backdates the *oldest* (position 1) request past the TTL — deliberately the only realistic scenario, since FIFO order is entirely a function of `created_at` and backdating any other position would also reorder it ahead of the others, a test-harness artifact of rewriting `created_at` rather than something time passing can actually produce — and confirms positions `1,2,3` in the correct FIFO order with only position 1 reporting `Expired=true`;
- new `TestV11ListPendingApprovalModeHasNoQueueFields`: the regression check — an APPROVAL Slot's pending list always reports `QueuePosition=nil`/`Expired=false`;
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (with the `linkup_api` role present, exercising §49's grants and its role test throughout) — all three green;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff.

Not done or claimed in this block: no admin/host action to manually reorder or bump a WAITLIST queue position (README does not ask for one — FIFO stays server-authoritative and non-bypassable per §37's own "host WAITLIST controls do not expose manual Accept" decision); Android UI (no SDK in this environment, per RULE 2/3); the broader recommendation/Friends/non-`PUBLIC`-visibility items from §50's "Next" remain untouched.

Next: `EVENT_RECOMMENDATION` (still needs a real recommendation-candidate design), or `FRIEND_*` (blocked on a nonexistent Friends/Connections domain), or README §6.9's remaining after-check/public-band-exposure work, or beginning non-`PUBLIC` visibility now that `realtime_viewer_store.go` is ready for it. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 52. 2026-09-11 — first non-`PUBLIC` visibility mode implemented: `PRIVATE` (README §4.3)

**Implemented:** `PRIVATE` is the first of README §4.3's six additional visibility modes (`LINKS`/`SELECTED`/`CITY`/`LASSO`/`TRAVEL_CORRIDOR`/`PRIVATE`) to actually ship, chosen because it needs no new domain — unlike `LINKS` (needs a nonexistent Friends domain) or `CITY`/`LASSO`/`TRAVEL_CORRIDOR` (need deeper City Context/Map/Fly integration). Design, made explicit rather than silently assumed: `PRIVATE` is a *discoverability* gate, not an *access-control* gate — a `PRIVATE` Slot is excluded from every surface that lists Slots a viewer hasn't been given the ID for, but anyone handed the Slot ID directly can still `Request()`/`Join()` it ("shared ID = invite"). This required zero postgres-layer changes on the discovery side: `Get`'s stranger branch, `ListPulse`, Map `Viewport`/`PlaceSlots`, and the realtime viewer already hard-require `visibility='PUBLIC'` (confirmed by grep across `slot_store.go`, `v11_slot_store.go`, `citymap_store.go`, `realtime_viewer_store.go`, `realtime_city_store.go`, `idempotency_replay.go`), so a `PRIVATE` Slot is automatically invisible to strangers on all of them without touching a single query. `Request()`/`Join()`/`Approve()` implementations never reference `visibility` at all, so direct-ID access needed no change either.

Code changes, v1.1 hosting path only (`CreateDraft`) — v1.0's legacy `Create` explicitly keeps rejecting non-`PUBLIC` visibility, since v1.0 clients have no UI concept of it and the v1.0 `Store` implementation was never audited for it:
- `slot/model.go`: `VisibilityPrivate Visibility = "PRIVATE"` constant (with a doc comment explaining the discoverability-vs-access-control distinction above); `CreateInput.Visibility *Visibility` (nil-safe, defaults to `VisibilityPublic`).
- `slot/service.go`: v1.0 `Create` gains an explicit guard rejecting any non-nil, non-`PUBLIC` `Visibility` with `ErrInvalidState`; `normalizeCreate` (v1.1 draft path only — deliberately *not* `normalizeEdit`, since visibility is not editable after creation in this block) uppercases/trims and validates against `validVisibility`; new `effectiveVisibility`/`validVisibility` helpers, the latter a closed set (`PUBLIC`, `PRIVATE` today) so an unrecognized mode like `SELECTED` is rejected with `ErrInvalidInput` rather than silently stored.
- `slot/v11_hosting.go`: `CreateDraft` now stores `effectiveVisibility(in.Visibility)` instead of a hardcoded `VisibilityPublic`; the idempotency fingerprint hash gains a `Visibility` field so a request with the same key but a different visibility value is correctly treated as a *different* request (matching the existing pattern for every other varying field).
- `httpserver/slot_handlers.go` + `v11_hosting_routes.go`: `createSlotRequest.Visibility *slot.Visibility` (`json:"visibility"`), wired into both `Create` and `CreateDraft` calls.
- Postgres store layer needed no changes: `v11_slot_store.go`/`v11_hosting_store.go`'s `INSERT` statements already pass `string(candidate.Visibility)` through generically.

New tests:
- `slot/service_test.go`: `TestLegacyCreateRejectsNonPublicVisibility` — v1.0 `Create` rejects `PRIVATE` with `ErrInvalidState`.
- `slot/v11_hosting_test.go`: `TestCreateDraftPreservesConfiguredVisibility` (a `PRIVATE` draft stores and returns `PRIVATE`) and `TestCreateDraftRejectsUnknownVisibility` (`SELECTED` — an as-yet-unimplemented mode — is rejected with `ErrInvalidInput`, not silently accepted).
- `postgres/v11_private_visibility_integration_test.go` (new): `TestV11PrivateVisibilitySlot` — end-to-end against real PostgreSQL: a host publishes a `PRIVATE`, `AccessApproval` Slot; a stranger's `ListPulse` and Map `PlaceSlots` never include it; a stranger's `Get()` by ID returns `ErrNotFound` (no existence leak); the host's own `Get()` always succeeds (`ViewerHost`); the same stranger, handed the Slot ID directly, successfully `Request()`s it, after which their own `Get()` becomes visible as `ViewerPending` via the pre-existing `slot_requests`-EXISTS branch; after `Approve()`, the same user's `Get()` reads `ViewerAccepted`. `TestV11PrivateVisibilityRejectedForLegacyCreate` — the v1.0 `Create` regression check against real PostgreSQL.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean (confirmed `internal/slot/accepted.go`, `remove_member.go`, `service_limits_test.go` are pre-existing `gofmt`-dirty files untouched by this block — not in this block's `git status`, so not newly introduced);
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000030` applied cleanly (no new migration in this block — verified by grep that `slots.visibility`'s original `CHECK` constraint from `000002_slots.sql` already allows all seven README §4.3 values, `PRIVATE` included, since the column was first created; only the Go domain/service layer was gating it down to `PUBLIC`-only, so this block only had to relax that gate);
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (with the `linkup_api` role present) — all three green;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff.

Not done or claimed in this block: `Visibility` is still create-time-only — no `Edit`-time toggle exists yet (a real gap: a host cannot flip a Slot to/from `PRIVATE` after publishing, only choose it at creation); the remaining five README §4.3 modes (`LINKS`/`SELECTED`/`CITY`/`LASSO`/`TRAVEL_CORRIDOR`) remain entirely unimplemented and blocked on their own missing domains (Friends, City Context integration, Fly/geo-corridor); no host-side "who can currently see this via direct ID" audit trail exists — a `PRIVATE` Slot ID shared outside the app (screenshot, copy-paste) is permanently requestable by whoever has it, which is the deliberate v1.1 design, not a bug, but is worth restating plainly since it has real privacy implications; Android UI (no SDK in this environment, per RULE 2/3).

Next: an `Edit`-time visibility toggle (the clearest immediate gap this block leaves open), or `EVENT_RECOMMENDATION`, or `FRIEND_*` (blocked on a nonexistent Friends/Connections domain — also the prerequisite for `LINKS` visibility), or README §6.9's remaining after-check/public-band-exposure work. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 53. 2026-09-11 — closes §52's own "Edit-time visibility toggle" gap

**Implemented:** a host can now change a Slot's `Visibility` (`PUBLIC`↔`PRIVATE`) via `PATCH /v1/slots/{slotID}`, but — deliberately, mirroring the existing `AccessMode`-after-`DRAFT` restriction already in `V11SlotStore.Edit` — only while the Slot is still `DRAFT`. Reasoning stated explicitly rather than assumed: changing *who can discover* a Slot after it is already `PUBLISHED`/`FILLING`/`FULL` is a bigger decision than a plain field edit should silently make — flipping to `PRIVATE` could yank visibility out from under strangers who already found it on Pulse/Map (their existing relationship, e.g. a pending request, is untouched — visibility only gates *new* discovery — but the sudden discoverability change itself is the kind of decision this block declines to make implicitly), and flipping to `PUBLIC` could surface something a host built as "shared ID = invite only". A host who needs to change visibility after publishing must still cancel and recreate, exactly as §52 already stated for the general case; this block only turns that into a `DRAFT`-scoped capability, not a Blanket one.

Code changes:
- `slot/model.go`: `EditInput.Visibility *Visibility` (doc comment explains the `DRAFT`-only restriction and its reasoning); `CreateInput.Visibility`'s doc comment updated to point at it instead of stating "not supported by this block".
- `slot/service.go`: `normalizeEdit` gains the same normalize/validate treatment `AccessMode` already gets (uppercases, trims, checks `validVisibility`, rejects with `ErrInvalidInput` for an unimplemented mode); `Visibility` added to `normalizeEdit`'s "at least one field set" check. The idempotency fingerprint already hashes the whole `EditInput` struct, so `Visibility` was automatically included with no separate change needed.
- `postgres/v11_slot_store.go` (`V11SlotStore.Edit`): `patch.Visibility != nil && state != "DRAFT"` → `ErrInvalidState`, exactly mirroring the pre-existing `AccessMode` check one line above it; `visibility=CASE WHEN $20 THEN $21 ELSE visibility END` added to the `UPDATE slots` statement, guarded the same way every other optional field already is. The v1.0 `SlotStore.Edit` is intentionally left untouched — it already silently ignores `patch.AccessMode` today (a pre-existing, not newly-introduced, inconsistency), and it is never reached by the production API regardless: `cmd/api/main.go` always wires `V11SlotStore` (which embeds and overrides the base store) as the one `Store` implementation handlers use.
- `httpserver/slot_handlers.go`: `editSlotRequest.Visibility *slot.Visibility` (`json:"visibility"`), wired into `slot.EditInput{...}`.

New tests:
- `slot/service_test.go`: `TestEditPassesVisibilityThroughToStore` (confirms the domain layer passes a `Visibility` patch through to the store unmodified) and `TestEditRejectsUnknownVisibility` (an unimplemented mode like `SELECTED` is rejected with `ErrInvalidInput`, not silently accepted) — both against the existing in-memory `memoryStore` test double, since the `DRAFT`-only enforcement itself lives in the postgres layer, not the domain `Service`.
- `postgres/v11_private_visibility_integration_test.go`: new `TestV11EditVisibilityAllowedOnlyWhileDraft`, end-to-end against real PostgreSQL — a `DRAFT` Slot's default `PUBLIC` visibility is edited to `PRIVATE` (succeeds), the draft is published and keeps `PRIVATE` (confirms the edited value survives `PublishDraft`, not just the initial `CreateDraft` value), then an attempt to edit visibility back to `PUBLIC` after publish is rejected with `ErrInvalidState`, and a final `Get()` confirms the rejected edit did not silently apply anyway (visibility is still `PRIVATE`).

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000030` applied cleanly (no new migration — `slots.visibility`'s original `CHECK` constraint already allows every value this block writes);
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (with the `linkup_api` role present) — all three green;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff.

Not done or claimed in this block: visibility still cannot be changed once a Slot is published — this is the deliberate scope limit explained above, not an oversight; the remaining five README §4.3 modes (`LINKS`/`SELECTED`/`CITY`/`LASSO`/`TRAVEL_CORRIDOR`) remain entirely unimplemented; Android UI (no SDK in this environment, per RULE 2/3).

Next: `EVENT_RECOMMENDATION` (still needs a real recommendation-candidate design), or `FRIEND_*` (blocked on a nonexistent Friends/Connections domain — also the prerequisite for `LINKS` visibility), or README §6.9's remaining after-check/public-band-exposure work. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 54. 2026-09-11 — closes the "public-band-exposure" half of README §6.9's remaining gap

**Before picking this, audited what was actually missing.** §48 already implemented `Band`/`Reliability`/`VaultEntry`/`reliability_events` — most of README §6.9 ("reliability events", "private/public reliability bands", "BUMP Vault baseline" were all already real, tested, and running against PostgreSQL. But `GET /v1/me/reliability` only ever let a user read their *own* `Reliability` (exact `VerifiedBumpCount` included) — there was no way for anyone to see *another* user's Band at all. The "private/public" split the `Band` type's own doc comment already described existed only in the domain model, never as two actual API surfaces. `EVENT_RECOMMENDATION` and `FRIEND_*` remain blocked exactly as stated in §50-§53 (real design/domain work, not attempted casually); README §6.9's other remaining bullet, "after-check flow", has no elaboration anywhere in the document beyond that one phrase and was deliberately left alone this block rather than invented from two words — the risk of building the wrong thing and calling it done is exactly what RULE 2 exists to prevent.

**Implemented:** `GET /v1/users/{userID}/reliability-band` — a new, public counterpart to the existing self-only `GET /v1/me/reliability`. It returns *only* the coarse `Band` for any user, never the exact `VerifiedBumpCount` (that stays visible solely to the user themselves via the existing endpoint), and is gated by the same block relationship every other cross-user-visible surface in this codebase already enforces (`user_blocks`, checked both directions, mirroring `Confirm`'s existing block check in the same store).

Code changes:
- `bump/model.go`: `Store.PublicBand(ctx, viewerID, targetUserID) (Band, error)`; `Service.PublicBand` (trims/validates both IDs, delegates).
- `postgres/bump_store.go`: `BumpStore.PublicBand` — skips the block check for a self-lookup (so a stray self-block row can never lock a user out of their own public band), otherwise checks `user_blocks` both directions and returns `ErrForbidden`; reads `user_reliability.band`, defaulting to `BandNew` (`pgx.ErrNoRows`) for a user with no credited history yet, exactly mirroring `Reliability`'s existing default.
- `httpserver/bump_handlers.go` + `server.go`: new handler `getUserReliabilityBand` and route, capability-gated the same as every other BUMP endpoint (`capability.Bump`), auth required.

New tests:
- `bump/model_test.go`: `TestServicePublicBandTrimsAndForwardsArgs`, `TestServicePublicBandRejectsEmptyInput`, `TestServicePublicBandPropagatesForbidden`.
- `httpserver/bump_handlers_test.go`: `TestGetUserReliabilityBandReturnsOnlyBand` (asserts the response body contains `band` but never `verifiedBumpCount` — the actual privacy boundary, not just a status code), `TestGetUserReliabilityBandRequiresAuth`, `TestGetUserReliabilityBandFailsClosedWhenServiceUnavailable`, `TestGetUserReliabilityBandPropagatesForbiddenForBlockedPair`.
- `postgres/bump_store_integration_test.go` (end-to-end against real PostgreSQL): `TestBumpStorePublicBandReflectsCreditedReliability` — a user with no history reads `BandNew` publicly; after a real mutual BUMP confirmation between two other users (the same `IssueChallenge`/`Confirm` flow §48 already proved), the public `Band` for one of them is fetched by an unrelated third party and shown to exactly match that user's own private `Reliability().Band` (`BandBuilding` after one credited BUMP) — proving the public projection stays in sync with real credited reliability, not a separate/stale computation. `TestBumpStorePublicBandRejectsBlockedViewer` — blocked in either direction, `ErrForbidden`. `TestBumpStorePublicBandAllowsSelfLookupRegardlessOfBlocks` — the self-lookup skip cannot misbehave.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000030` applied cleanly (no new migration — `user_reliability`/`user_blocks` already exist from `000028`/earlier);
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (with the `linkup_api` role present) — all three green;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff.

Not done or claimed in this block: README §6.9's "after-check flow" bullet remains unimplemented — deliberately, since the document gives no further specification of what it means beyond the two words, and inventing behavior for it risks building something that does not match the intended product design (a "fake" real feature, which is its own kind of RULE 2 violation even if the code itself is genuine and tested); this endpoint exposes Band to any authenticated, non-blocked user with no additional relationship requirement (e.g. not restricted to co-participants of a shared Slot) — stated as a design choice matching the Band doc comment's own framing ("the coarse, public-facing reliability signal"), not silently assumed; Android UI (no SDK in this environment, per RULE 2/3).

Next: `EVENT_RECOMMENDATION` (still needs a real recommendation-candidate design), or `FRIEND_*` (blocked on a nonexistent Friends/Connections domain), or a real specification for README §6.9's "after-check flow" (this session will not invent one unprompted). Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 55. 2026-09-11 — new `internal/friend` domain: Friends/Links (README §4.3 prerequisite, §5.4, §6.8 FRIEND_REQUEST/FRIEND_ACCEPTED)

**User instruction driving this block:** after being asked directly whether everything in README.md was done and answering honestly (no — a large majority of README §6 was unimplemented, verified by grep before answering, not from memory), the user asked to fully implement everything in README.md and the other `.md` files. Given the true scope (dozens of major v1.1 features, a full Android app, real device/production verification this environment cannot perform), the user was asked one clarifying question about how to handle the unbuildable/unverifiable Android side in this SDK-less environment and chose "backend + Android best-effort" — write Kotlin for new server-backed features too, clearly flagged as uncompiled/unverified here. This block is backend-only; no Android code was written in it (nothing in this block has an Android-facing surface complex enough yet to justify a client stub — the next block that adds a concrete endpoint contract Android would call is where that starts).

**Why this block first:** `FRIEND_*` was the most-repeated "Next" item across §50-§54 (blocked on a nonexistent Friends/Connections domain) and is also the explicit prerequisite for README §4.3's `LINKS` visibility mode — building it unblocks two previously-stuck backlog items at once, not just one.

**Implemented — the full request/accept/reject/cancel/remove lifecycle, real and tested against PostgreSQL, not just source presence:**

- **`internal/friend/model.go`** (new domain package): `Store` interface + `Service`. Deliberately does NOT reuse the Slot domain's generic `mutation_idempotency`/`claimIdempotency` mechanism (that mechanism's replay-authorization logic is hardcoded per Slot operation in `authorizeIdempotencyReplayTx` and not meant to be generalized) — instead every mutation is designed to be naturally idempotent through its own DB constraints and conditional state transitions, the same pattern `internal/blocklist`'s Block/Unblock already use in this codebase. `Outcome` (`REQUESTED`/`ACCEPTED`/`ALREADY_PENDING`) reports what `Request` actually did, because requesting someone who already has a pending request out to you is not a second parallel request — it is an immediate mutual match, resolved without a redundant accept step (a real product decision, stated explicitly here rather than silently chosen).
- **`db/migrations/000031_v11_friends.sql`**: `friend_requests` (directed, `PENDING`/`ACCEPTED`/`REJECTED`/`CANCELLED`, a partial unique index enforcing at most one `PENDING` row per ordered pair — the reverse direction is deliberately unconstrained, since a real mutual-match state), `friendships` (symmetric, `user_lo_id`/`user_hi_id` canonicalized exactly like `bump_confirmations`, 000028), a new `friends` capability key (fail-closed, off by default like every other v1.1 capability), and the standard `anon`/`authenticated`/`linkup_api` grant/revoke block. No new idempotency table and no new outbox grant needed: `linkup_enqueue_outbox` was already unconditionally granted to `linkup_api` as of 000029.
- **`internal/postgres/friend_store.go`**: `FriendStore`. `Request` locks and checks (in order) target existence, block relationship (`blockedPairTx`, reused as-is from `slot_store.go`), existing friendship, then a reverse-pending row (auto-accepts it if found), then a forward-pending row (no-ops if found), before inserting a new `PENDING` row and emitting a `friend.requested` outbox event via `linkup_enqueue_outbox` in the same transaction — the identical pattern `ReminderScanner` (§49/§30) established for a non-trigger-driven Go-side outbox emission. `Accept`/`Reject`/`Cancel` are each a single conditional `UPDATE ... WHERE status='PENDING'`, idempotent-tolerant on replay (already-resolved state reads as success, not `ErrNotFound`) but a genuinely nonexistent request still errors. `Remove` is an unconditional `DELETE`, idempotent regardless of prior state, matching `Unblock`'s existing contract.
- **`internal/postgres/notification_store.go`**: `buildFriendRequestedCandidate`/`buildFriendAcceptedCandidate` added to `NotificationProjector.buildCandidate`'s explicit allow-list. `notification.TypeFriendRequest`/`TypeFriendAccepted` (category, deep link `app://me/requests`/`app://me/connections`, quiet-hours/frequency-cap behavior) already existed fully-formed in `internal/notification/model.go` from earlier foundation work — this block only had to wire the outbox event types into the switch, confirmed by reading that file before writing anything, not assumed.
- **`internal/httpserver/friend_handlers.go` + `server.go`**: 8 new capability-gated (`capability.Friends`) endpoints — send/cancel/accept/reject a request, list friends/incoming/outgoing, remove a friend.
- **`internal/capability/model.go`**: new `Friends` key, added to `knownKeys`.
- **`cmd/api/main.go`**: `FriendStore`/`friend.Service` wired into `Dependencies`.

**A real bug this block's own test caught, corrected honestly rather than adjusting the product to pass a wrong test:** the first version of `TestFriendStoreReverseRequestAutoAcceptsAndNotifiesOriginalRequester` asserted user B (who completes a mutual match by requesting back) receives *zero* pushes from the whole flow. It failed — B had received exactly one real, correct "New friend request" push for A's original, genuine request, sent *before* B's reverse call auto-accepted it. That notification is correct product behavior, not a duplicate or a leak; the test's assumption was wrong, not the code. Fixed by asserting B receives exactly that one original push and never a second "Friend request accepted" push for their own action, rather than asserting silence.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean (`cmd/api/main.go` remains the pre-existing not-`gofmt`-clean file established in §46; confirmed by `gofmt -d` that this block's own added lines follow that file's existing dense style, not newly-introduced worse formatting);
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000031` applied cleanly;
- new unit tests: `internal/friend/model_test.go` (11 tests — input validation, trimming, error propagation, self-pair short-circuit) and `internal/httpserver/friend_handlers_test.go` (15 tests — every handler's success/auth/service-unavailable/error-mapping paths);
- new integration tests, `internal/postgres/friend_store_integration_test.go` (9 tests) against real PostgreSQL: request creates `PENDING` + projects a real `FRIEND_REQUEST` push+notification row with the correct deep link; a reverse request auto-accepts into a real, symmetric friendship and projects a real `FRIEND_ACCEPTED` push to the *original* requester only; duplicate forward requests are idempotent (no duplicate row); blocked pairs are rejected both directions; already-friends is rejected; Accept/Reject/Cancel are idempotent on replay but error on a genuinely nonexistent request; Remove is idempotent from either side;
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (with the `linkup_api` role present) — all three green;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff.

Not done or claimed in this block: `LINKS` visibility (README §4.3) is NOT wired to this domain yet — `friend.Store.AreFriends` exists as the query surface a future block will consume, but no Slot discovery query currently calls it; no rate limiting on outgoing friend requests (not specified anywhere, not invented here); no Android Kotlin client code (no Android-facing contract in this block complex enough to justify one yet, per the user's "backend + Android best-effort" instruction — the next block that adds one starts the Android side); the `friends` capability stays disabled by default, matching every other v1.1 capability's rollout gate; Android UI/real build (no SDK in this environment, per RULE 2/3); no production deployment or verification of any kind.

Next: wire `LINKS` visibility (README §4.3) onto this domain now that `AreFriends` exists — the natural, dependency-ordered continuation; or `EVENT_RECOMMENDATION`; or README §6.9's "after-check flow" if the user provides a real specification. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 56. 2026-09-11 — `LINKS` visibility wired to Slot discovery (README §4.3, closing §55's own gap); first Android best-effort client code

**Backend — `LINKS` visibility (README §4.3's second non-PUBLIC mode, after PRIVATE in §52):**

- `internal/slot/model.go`: `VisibilityLinks Visibility = "LINKS"`, same "discoverability gate, not access-control gate" contract as `VisibilityPrivate` — the friendship check lives entirely in the postgres query layer, not the Slot domain, since the Slot domain has and gets no dependency on `internal/friend`.
- `internal/slot/service.go`: `validVisibility` now accepts `LINKS` too. The v1.0 legacy `Create` guard (`*in.Visibility != VisibilityPublic → ErrInvalidState`) already covers `LINKS` generically — its own comment already said "and any future visibility mode" — confirmed by reading it before writing anything, not assumed; no change needed there. The `CHECK` constraint on `slots.visibility` (migration `000002`) already allowed `LINKS` from day one, exactly as confirmed for `PRIVATE` in §52.
- `internal/postgres/v11_slot_store.go`: `getV11SlotSQL` and `listV11PulseSQL` (the two discovery queries `V11SlotStore` — the only `Store` implementation `cmd/api/main.go` actually wires — overrides) each gain a `LINKS` branch: visible when `s.state IN ('PUBLISHED','FILLING','FULL')`, a real row exists in `friendships` for `(host, viewer)` (canonicalized via `LEAST`/`GREATEST` on the two UUIDs — confirmed empirically against a live PostgreSQL that `LEAST`/`GREATEST` order `uuid` values identically to the lexicographic string order `friend_store.go`'s Go-side `canonicalPair` already uses, before relying on it), and the same `user_blocks` check every other branch already applies — a block always wins even between real friends, proven by test, not assumed.
- **Explicit, stated scope limit, not silently left implicit:** Map (`citymap_store.go`), the realtime viewer (`realtime_viewer_store.go`), the city realtime channel (`realtime_city_store.go`), and `idempotency_replay.go`'s `slot.request` replay-authorization check are **NOT** updated for `LINKS` in this block — only `Get`/`ListPulse` are. A host can create a `LINKS` Slot today and it will correctly hide/reveal on those two surfaces, but a friend viewing the Map or realtime feed would not yet see it there. Recorded here as a real, known gap (unlike §50's PRIVATE-adjacent finding, which turned out to have no reachable production impact) — this one does, the moment `friends`/any visibility-aware feature ships to real users.

New tests: `internal/slot/v11_hosting_test.go`'s `TestCreateDraftPreservesConfiguredLinksVisibility` (unit); `internal/postgres/v11_links_visibility_integration_test.go`'s `TestV11LinksVisibilitySlot` (end-to-end against real PostgreSQL — a stranger cannot see a LINKS Slot on Pulse/Get; a real friendship made through `internal/friend.Service` makes it visible on both; a block between the two afterward hides it again on both, proving blocking overrides friendship rather than the friendship check being a standalone bypass) and `TestV11LinksVisibilityRequestReachableByDirectID` (a stranger with the Slot ID directly can still `Request()` it — the same "shared ID = invite" contract `LINKS` inherits from `PRIVATE`).

**Android — first best-effort client code this session, explicitly unverified (no SDK/Gradle network access in this environment to compile or run any of it):**

Triggered by the user's standing "backend + Android best-effort" instruction from earlier in this session, now that a concrete, real backend contract exists worth writing a client for. Two categories of change:

1. **A real bug fix, not just new capability:** `android/.../network/SocialModels.kt`'s `SlotVisibility` enum only ever had `PUBLIC`. Both places that parse a Slot response (`LinkUpApiClient.kt`, `DurableSocialApi.kt`) call `SlotVisibility.valueOf(json.getString("visibility"))`, which throws for any value not in the enum. Since the real backend has emitted `PRIVATE` since §52 and now `LINKS` since this block, any Android build against that backend before this fix would crash parsing the very first `PRIVATE`/`LINKS` Slot it encountered — this was a live defect independent of anything new in this block, found by reading the existing code before writing anything, not by running it (impossible here). Fixed by extending the enum to `{PUBLIC, PRIVATE, LINKS}`.
2. `visibility: SlotVisibility? = null` added to `CreateSlotInput`/`EditSlotInput`, wired into both places that build the create/edit JSON body (`LinkUpApiClient.kt` and `DurableSocialApi.kt` — this codebase duplicates that body-building logic between the two, an existing structural fact, not something this block introduced or fixed).
3. New `android/.../network/FriendApiClient.kt` + `FriendModels.kt`: a client for all 8 `internal/httpserver/friend_handlers.go` endpoints from §55, following `MonetizationApiClient.kt`'s established self-contained style (own `requestOnce`, reuses the package-`internal` `validatedApiRoot`/`readUtf8Bounded` helpers). Not wired into any UI screen or a DI container — this codebase has no existing Friends UI surface to extend, and inventing one was out of scope for a client-layer block (also `PROJECT_RULES.md`'s "existing design is frozen" — a new screen is a design decision, not a client-contract one).
4. New unit tests in `DurableHostingApiTest.kt` (`draft create includes visibility when configured`, `draft create omits visibility from body when not configured`) extending the existing `createDraft` test pattern.

**Every line of Android code in this block is unverified**: not compiled, not run, no Gradle/SDK access in this environment. It was written by close, careful mirroring of existing, working Android code in this same repository (`MonetizationApiClient.kt`'s exact request/retry/error-parsing shape, `LinkUpApiClient.kt`'s exact JSON field names and `parseSlot` structure) and manual review for syntax/brace correctness, not verified execution — stated plainly rather than implied otherwise. Production readiness for anything client-side remains 0% regardless.

Executed evidence (backend only — see above for why Android has none):

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched Go file — clean;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000031` applied cleanly (no new migration — `LINKS` was already in the `slots.visibility` `CHECK` constraint);
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (with the `linkup_api` role present) — all three green;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff.

Not done or claimed in this block: `LINKS` on Map/realtime/city-realtime/idempotency-replay (stated above as a real, known gap, not swept under "future work" vaguely); Android Friends UI (no screen exists to extend); Android build/compile/run verification of any kind (no SDK); production deployment or verification.

Next: extend `LINKS` visibility to Map/realtime (the concrete gap this block leaves open), or `EVENT_RECOMMENDATION`, or README §6.9's "after-check flow" given a real specification. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 57. 2026-09-11 — closes §56's own explicitly stated gap: `LINKS` visibility now covers Map, realtime viewer, city realtime channel, and idempotency-replay

**Every surface §56 named as an open, known gap is closed in this block, not partially.** All four: `citymap_store.go` (`Viewport` + `PlaceSlots`), `realtime_viewer_store.go` (`viewerRealtimeSQL`), `realtime_city_store.go` (`cityRealtimeSQL`), and `idempotency_replay.go` (`slot.leave`'s PUBLIC-visibility fallback branch) now gain the identical `LINKS` sibling branch `v11_slot_store.go` got in §56: `s.visibility='LINKS' AND s.state IN ('PUBLISHED','FILLING','FULL') AND EXISTS(friendships row, LEAST/GREATEST-canonicalized) AND NOT EXISTS(blocked)`.

- `citymap_store.go`: `Viewport`'s `visible_places` CTE and `PlaceSlots`'s `WHERE` clause each gain the branch, both keyed on the query's own `viewerID` parameter.
- `realtime_viewer_store.go`: `viewerRealtimeSQL`'s lifecycle-event branch (`slot.created`/`slot.updated`/`slot.state_changed`) gains a sibling `OR` branch alongside its existing `PUBLIC` one.
- `realtime_city_store.go`: `cityRealtimeSQL` operates on the outbox event's JSON *payload* (`e.payload->>'visibility'`), not the live `slots` row, since it is a per-viewer coarse "something changed here, go re-fetch" signal, not a full Slot read — but it still joins the live `slots` row (`s.host_id`) for the friendship `EXISTS` check, and the existing `NOT EXISTS(user_blocks)` at the outer level already applies to the new branch too without needing to be duplicated (confirmed by reading the surrounding `AND`/`OR` structure and SQL operator precedence carefully before editing, not assumed).
- `idempotency_replay.go`: the `slot.leave` case's fallback ("no current relationship, but the Slot is still generally visible, so a replay may still read it back") gains the same `LINKS` branch. This is the one path that is a real, live risk if skipped: a `LEAVE` idempotency replay for a `LINKS` Slot from a real friend would otherwise incorrectly `ErrForbidden` instead of returning the current state.

**A real bug in this block's own first draft, caught and fixed, not glossed over:** the very first version of the new `idempotency_replay_test.go` case (`TestIdempotencyReplayAuthorizationSQL`, the *existing* test, not the new one) started failing with `ERROR: operator does not exist: uuid = text` the moment the new `LINKS`-branch SQL was added to `idempotency_replay.go`. Root cause: that test's temp-table fixture never declared a `pg_temp.friendships` table, so the new query's unqualified `FROM friendships` resolved to the *real*, already-migrated `public.friendships` (`uuid` columns) from earlier tests run against the same disposable database in this session, colliding with the temp fixture's `text`-typed `slots.host_id`. Not a production bug (real `slots.host_id` and real `friendships` columns are both genuinely `uuid`) — a test-fixture gap, exposed by the new code, fixed by adding `CREATE TEMP TABLE friendships(...) ON COMMIT DROP` to that test's setup so it shadows the real table exactly like its sibling `slots`/`slot_requests`/etc. declarations already do.

New tests, all end-to-end against real PostgreSQL:

- `internal/postgres/v11_links_visibility_integration_test.go`: `TestV11LinksVisibilityMap` (Viewport cluster counts and PlaceSlots both exclude a LINKS Slot from a non-friend and include it — via a real `internal/friend.Service` friendship, not a shortcut — for a friend) and `TestV11LinksVisibilityRealtimeViewer` (a stranger's `PullViewer` never surfaces a LINKS Slot's `slot.created`; a real friend's does).
- `internal/postgres/idempotency_replay_test.go`: `TestIdempotencyReplayAuthorizationSQLLinksVisibility` — a stranger is denied `slot.leave` replay on a LINKS Slot even though PUBLIC would have allowed it (LINKS is not a weaker PUBLIC, it requires a real friendship row); a real friendship allows it; a block between the two still wins; leaving the `PUBLISHED/FILLING/FULL` state range removes the fallback entirely, same as the pre-existing PUBLIC case.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000031` applied cleanly (no new migration — this block only changed application-layer SQL, no schema);
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (with the `linkup_api` role present) — all three green;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff.

Not done or claimed in this block: `LINKS` visibility (README §4.3) is now fully wired across every discovery/replay surface this codebase has, but the remaining four modes (`SELECTED`/`CITY`/`LASSO`/`TRAVEL_CORRIDOR`) remain entirely unimplemented; no Android work this block (no new client-visible contract — Map/realtime/replay are all server-internal correctness, nothing a client calls differently); Android UI/real build (no SDK in this environment); no production deployment or verification of any kind.

Next: `EVENT_RECOMMENDATION` (still needs a real recommendation-candidate design), or README §6.9's "after-check flow" given a real specification, or one of the remaining four README §4.3 visibility modes if the user wants to prioritize that over other backlog. Go-backend-first until Android SDK/Gradle network access is available in this environment.

## 58. 2026-09-11 — first real production deployment this session: merge to `main`, migrations 000028-000031 applied to production Supabase, real Android build verification on the Ubuntu server, and a genuine live bug found and fixed

**How this block started:** the user gave direct SSH credentials to the production Ubuntu server (`ubuntu@89.168.108.176`) and said access was now unrestricted ("руки розв'язані" — hands untied). Given the weight of that ("Ubuntu deployment: do not touch until direct deployment/server command" had been the standing rule all session), this was treated as exactly that direct command, but not as a blank check to improvise: before touching anything, two real ambiguities were surfaced back to the user via a clarifying question rather than guessed —

1. **Supabase project mismatch.** The Supabase MCP connector was initially attached to a project (`link2027` / `mexxvdcgksmdwhssmdra`) that does **not** match `PROJECT_RULES.md`/`INFRASTRUCTURE.md`'s documented canonical project (`oavnrlwsfiiehluubwjk`). No migration or query was run against either project until this was resolved. The user reconnected and confirmed `oavnrlwsfiiehluubwjk` — verified again via `list_projects` before proceeding, not just taken on faith.
2. **Deploy source conflict.** `INFRASTRUCTURE.md` §7 states plainly: "A deployment must be based on a real commit from `main`" — and the server's checkout genuinely only tracks `main` (confirmed via `git status`/`git log` over SSH: 7 commits behind `origin/main`, clean working tree). But this session's entire body of work (§46-§57) lived on `claude/verification-push-main-xsofu0`, and the harness-level branch instruction for this session explicitly forbade pushing to `main` without explicit permission. Asked the user directly rather than resolving this conflict unilaterally either direction; they chose to merge and push to `main`.

**What actually happened, in order, each step verified before the next:**

1. Confirmed real SSH connectivity and read the server's actual state first (systemd status, git log, service file, artifact layout) before changing anything — the server has been running `linkup-api.service` continuously for 20 days serving real traffic (`GET /v1/pulse`, `POST /v1/slots/.../request` returning 200 in the live log), not a throwaway box.
2. `git fetch` confirmed `claude/verification-push-main-xsofu0` was a clean fast-forward of `origin/main` (no divergence, no merge conflict risk) — merged with `git merge --ff-only` and pushed to `main` (commit `e76cb5b`, 61 files, all of §46-§57's work).
3. Applied the four migrations production was missing (`000028_v11_bump`, `000029_v11_notification_connector_grants`, `000030_v11_event_reminder`, `000031_v11_friends`) to `oavnrlwsfiiehluubwjk` via the Supabase MCP's `apply_migration`, one at a time, each in the same order/content as the repository files — not paraphrased or reconstructed from memory. Verified afterward via `list_migrations` (all four now in the ledger) and a direct table-existence query (all 8 new tables present). **Migration `000029` is not merely "new" — it restores `linkup_api`'s grants on `connector_cursors`/`connector_delivery_receipts` that an earlier hardening migration (`000021`) revoked and never restored; that regression had been live in production since whenever `000021` was first applied there, silently breaking the Notifications connector's cursor checkpointing the entire time.** This deploy is the first point at which that fix actually reached the real database, not just this session's own disposable test databases.
4. Noticed, while verifying grants, that `linkup_api` already holds full `SELECT/INSERT/UPDATE/DELETE` on every one of the new tables — broader than what each migration's own explicit `GRANT` statements ask for. Root cause: a pre-existing `ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT ... TO linkup_api` on the production database (not something this session or migration set up) auto-grants full CRUD to `linkup_api` on every new table the `postgres` role creates, regardless of a migration's own narrower intent. Recorded here as an honest observation, not fixed in this block — changing a standing default-privilege policy is a separate infrastructure decision from shipping a schema migration, and every migration's own explicit grants remain correct minimal-privilege documentation of what the application code actually needs even though the database is not currently enforcing that minimum.
5. Ran the repository's own canonical build pipeline on the server (`ops/build_v1.sh`, unmodified) rather than improvising a deploy: `go mod verify`, `go test ./... -count=1`, `go vet ./...`, `go test -race ./... -count=1`, `go build`, **then a real Android debug build** (`./gradlew testDebugUnitTest lintDebug assembleDebug`) against the server's real Android SDK/JDK17 toolchain (`/opt/android-sdk`) — something this session's own sandbox has never had access to. **BUILD SUCCESSFUL in 2m 50s.** This is the first real compiler/test verification this session's Android Kotlin code (§56's `FriendApiClient.kt`/`FriendModels.kt`, the `SocialModels.kt`/`LinkUpApiClient.kt`/`DurableSocialApi.kt` visibility-field changes, and the new `DurableHostingApiTest.kt` cases) has ever received — it was written by careful pattern-mirroring and manual review, explicitly flagged "unverified" in §56/§57's own worklog text, and that caveat is now resolved: it compiles, its unit tests pass, and `lintDebug` passed. SHA-256 checksums verified for both the `linkup-api` binary and the debug APK.
6. Promoted (`ops/promote_v1.sh`) and deployed (`ops/deploy_v1.sh`) the verified candidate. The deploy script's own health-check loop passed (`/livez`+`/healthz` on `127.0.0.1:8080`), and this was independently re-verified directly rather than trusting the script's own exit code alone: `sudo systemctl status` (active, running, correct new PID), `curl https://linkupapp-ua.duckdns.org/livez` over the real public HTTPS domain, and `git log -1` on the server confirming the exact commit (`e76cb5b`) is what's actually running.
7. Confirmed via a direct query against the production `capability_registry` that every capability — including the newly-added `friends` and `bump` — is `enabled=false`. **This deploy changes zero user-visible behavior for any real user right now**: every v1.1 feature shipped this session stays behind its fail-closed capability gate exactly as designed, so shipping the code and schema carries none of the risk a behavior change would.

**A real, live bug found during post-deploy verification, not glossed over:** `sudo journalctl -u linkup-api --since '2 min ago'` showed `NotificationProjector`'s background polling ticker failing every cycle (~5s) with `ERROR: prepared statement "stmtcache_..." already exists (SQLSTATE 42P05)`. Investigated rather than dismissed: production's `DATABASE_URL` connects through Supabase's PgBouncer pooler in **transaction-pooling mode** (`aws-1-eu-west-1.pooler.supabase.com:6543`, confirmed by reading the actual redacted connection string on the server) — a mode that is well-documented as incompatible with pgx's default `QueryExecModeCacheStatement` (server-side named prepared statements cached per logical connection): transaction pooling can hand two different logical connections the same physical backend mid-session, so a second `PREPARE` of an identical statement name collides with one already cached there. This is a **pre-existing bug in `internal/postgres/pool.go`**, present since the pool was first written, most likely masked until now because migration `000029` (this same block, step 3 above) is what let `NotificationProjector`'s `Checkpoint`/`Cursor` calls actually reach real queries for the first time — before that grant fix, every attempt failed instantly on a permission error, before ever exercising the query path that triggers the pooler collision. Fixed: `pool.go` now sets `cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol` (the standard, Supabase-documented fix for pgx behind a transaction-mode pooler), factored into a separately unit-testable `buildPoolConfig` so the fix has a real regression guard (`TestBuildPoolConfigUsesSimpleProtocol`) rather than being verifiable only by re-reading the source. Verified no `pgx.Batch`/`SendBatch` usage exists anywhere in this codebase that `SimpleProtocol` mode could break. This fix was built, tested (3x + race + mod tidy, same standard as every other block), merged to `main` (commit `36e0475`), and redeployed through the exact same verified pipeline as step 5-6 above (`ops/build_v1.sh` → `ops/promote_v1.sh` → `ops/deploy_v1.sh`, health-checked) before this block was considered done — not left as a known issue in a shipped state. **Confirmed live, not just assumed fixed:** `git log -1` on the server shows `36e0475` running; `sudo journalctl -u linkup-api` was checked twice more after this redeploy (once covering the 25 seconds immediately after restart, once covering the following 2 minutes) — zero occurrences of the `42P05`/"already exists" error in either window, versus one every ~5 seconds before the fix. `capability_registry` re-checked one final time: still 0 of 13 capabilities enabled.

Executed evidence, this session:

- Backend: `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean; the full `go test -count=1 ./...` suite three consecutive times against the same accumulating disposable database (`linkup_api` role present) — all green; `go test -race -count=1 ./...` — green; `go mod tidy` — no diff; new `TestBuildPoolConfigUsesSimpleProtocol`.
- Real production deployment: `main` at commit `e76cb5b` then the `pool.go` fix commit, both built via `ops/build_v1.sh` (Go verify/test/vet/race + real Android `testDebugUnitTest`/`lintDebug`/`assembleDebug` against a real SDK), both promoted via `ops/promote_v1.sh`, both deployed via `ops/deploy_v1.sh` with a passing health-check loop, both independently re-verified live (`systemctl status`, `/livez`, `/healthz`, public HTTPS, `git log -1` on the server matching exactly).
- Production Supabase (`oavnrlwsfiiehluubwjk`): migrations `000028`-`000031` applied and confirmed in the ledger; all 8 new tables confirmed present; `linkup_api`'s grants confirmed (and the broader-than-intended default-privilege behavior recorded honestly, not hidden); `capability_registry` confirmed every capability including `friends`/`bump` still `enabled=false`.
- Post-deploy log verification caught and fixed a real bug (`SQLSTATE 42P05`) that would otherwise have shipped silently.

**Honest production-readiness restatement, per this session's standing rule (RULE 6, never inflated):** this is the first block where "production" stopped being purely hypothetical — real code and real schema are now genuinely running on the real Ubuntu server against the real Supabase database, independently re-verified rather than taken on a script's word. The "real Android build" half of this session's long-standing 0% anchor is now genuinely satisfied — not from this sandbox (still has no SDK), but from the server's own toolchain, compiling and testing this session's actual Kotlin changes for the first time. What is **still** not done, so overall v1.1 feature readiness is **not** "production-verified": no physical two-user device test has occurred (only `assembleDebug` + unit tests + lint, no device or emulator interaction); every v1.1 capability, including everything shipped this session, remains deliberately disabled, so no real user is exposed to any of it yet; the release APK/AAB signing path (`LINKUP_BUILD_RELEASE=1`) was not exercised, only the debug path; no Play Store or physical distribution has happened. The backend deployment itself, independently verified end-to-end, is real and live — that specific claim, and only that one, can now be made without qualification.

Next: consider whether to enable any capability for real users (a product decision, not a backend one — this session will not flip a capability flag without the user explicitly asking for that specific feature to go live), or continue backend feature work (`EVENT_RECOMMENDATION`, remaining README §4.3 visibility modes, README §6.9's "after-check flow" given a specification), or a real two-user physical device smoke test if the user wants to pursue that next.

## 59. 2026-09-11 — `SELECTED` visibility implemented (README §4.3's third non-PUBLIC mode, after PRIVATE §52 and LINKS §55-§57)

**Implemented:** README §4.3's "Selected people" mode — the host picks an explicit, individual allow-list of users at creation time, distinct from LINKS's "anyone who is a real mutual friend" rule. Same discoverability-gate contract every other mode here already has: `Request()`/`Join()`/`Approve()` never reference the allow-list, so a stranger not on the list but handed the Slot ID directly can still reach it.

- `db/migrations/000032_v11_selected_visibility.sql`: `slot_selected_viewers(slot_id, user_id)`, `linkup_api` granted `SELECT, INSERT` only (no `UPDATE`/`DELETE` — see scope note below).
- `internal/slot/model.go`: `VisibilitySelected` constant; `CreateInput.SelectedUserIDs []string`; `Slot.SelectedUserIDs []string` (`omitempty`), explicitly documented as populated only right after creation, not on every subsequent read — a stated, deliberate gap, not a silent one.
- `internal/slot/service.go`: `validVisibility` accepts `SELECTED`; new `normalizeSelectedUserIDs` (non-empty required, each a well-formed UUID, deduplicated, capped at `MaxPendingRequests`, sorted for a stable idempotency fingerprint) wired into `normalizeCreate`, invoked only when the effective visibility is `SELECTED` — otherwise any accidentally-provided list is discarded rather than stored against a Slot whose visibility never reads it. `normalizeEdit` explicitly rejects `SELECTED` as an edit target (distinct from rejecting a genuinely unknown mode): `EditInput` has no field to update an existing allow-list, so silently accepting it would let a host switch a Slot to SELECTED with nobody actually selected — an undiscoverable Slot with no error, which is worse than refusing outright. The v1.0 legacy `Create` guard already rejects any non-PUBLIC visibility generically, so `SELECTED` is covered there with no separate change, confirmed by test.
- `internal/postgres/v11_hosting_store.go`: `CreateDraft` bulk-inserts the allow-list via `pgx.CopyFrom` in the same transaction as the Slot row; a new `attachSelectedUserIDsTx` helper re-reads the *persisted* allow-list (never the in-memory candidate, so a replayed idempotent call returns the real stored state) and populates `Slot.SelectedUserIDs` on both the fresh-create and replay return paths.
- `internal/postgres/v11_slot_store.go`: `getV11SlotSQL`/`listV11PulseSQL` each gain a third `OR` branch alongside PUBLIC/LINKS: `visibility='SELECTED' AND state in range AND EXISTS(allow-list row) AND NOT blocked`.
- `internal/httpserver/slot_handlers.go` + `v11_hosting_routes.go`: `selectedUserIds` JSON field wired into both create endpoints (`createSlotRequest` is shared by both `/v1/slots` and `/v1/slots/drafts`).

**Explicit, stated scope limit, matching how LINKS itself shipped in two blocks (§56 domain+Get/Pulse, then §57 Map/realtime/replay):** this block only covers `Get`/`ListPulse`, mirroring exactly where LINKS started. Map (`citymap_store.go`), the realtime viewer, the city realtime channel, and `idempotency_replay.go`'s `slot.leave` fallback are **NOT** updated for `SELECTED` in this block — a real, known gap recorded here, not swept under "future work" vaguely, following the exact template §56/§57 already established for tracking this kind of incremental rollout honestly.

New tests:
- `internal/slot/v11_hosting_test.go`: `TestCreateDraftPreservesSelectedVisibilityAndAllowList`, `TestCreateDraftRejectsSelectedVisibilityWithoutAllowList`, `TestCreateDraftRejectsSelectedVisibilityWithInvalidUserID`, `TestCreateDraftIgnoresSelectedUserIDsForOtherVisibilities`.
- `internal/slot/service_test.go`: new `TestEditRejectsSelectedVisibility`, distinct from the pre-existing `TestEditRejectsUnknownVisibility` (which used to use `"SELECTED"` as its example of an unimplemented mode — now genuinely stale since SELECTED is implemented for Create, just not Edit, so that test was updated to use `"CITY"` instead, and the SELECTED-specific rejection got its own dedicated test with the accurate reason).
- `internal/postgres/v11_selected_visibility_integration_test.go`: `TestV11SelectedVisibilitySlot` (end-to-end against real PostgreSQL — a non-selected stranger excluded from Pulse/Get; the selected user included on both; a block afterward overrides selection, same as LINKS's block-wins proof; a non-selected stranger with the Slot ID directly can still `Request()`) and `TestV11SelectedVisibilityRejectedForLegacyCreate`.

Executed evidence, this session:

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean;
- fresh disposable PostgreSQL 16 + PostGIS, migrations `000001..000032` applied cleanly (`slots.visibility`'s original `CHECK` constraint already allowed `SELECTED` since migration `000002`, confirmed before writing this block, not assumed — only the new allow-list table needed a migration);
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (with the `linkup_api` role present) — all three green;
- `go test -race -count=1 ./...` against a freshly rebuilt disposable database — green;
- `go mod tidy` — no diff.

Not done or claimed in this block: `SELECTED` on Map/realtime/city-realtime/idempotency-replay (stated above); no way to edit an existing SELECTED Slot's allow-list after creation (`EditInput` rejects `SELECTED` outright — a host who selected the wrong people must cancel and recreate); `Get`/`ListPulse`/`ListMine` do not re-populate `SelectedUserIDs` on every read, only immediately after creation; only `CITY`/`LASSO`/`TRAVEL_CORRIDOR` remain among README §4.3's six additional modes.

**Deployed to production, same session, following the exact §58 pipeline again — not left sitting in GitHub only:** given the user's standing "hands untied" server authorization from §58 and repeated instruction to keep working, this block was pushed through the same real pipeline rather than stopping at GitHub. `claude/verification-push-main-xsofu0` → fast-forward merge → push to `main` (commit `9bb25c2`); migration `000032` applied to production Supabase (`oavnrlwsfiiehluubwjk`) and confirmed (`slot_selected_viewers` exists); `ops/build_v1.sh` run on the server again — real Go verify/test/vet/race plus a real Android `testDebugUnitTest`/`lintDebug`/`assembleDebug` (gradle reused most tasks as `UP-TO-DATE` since only backend files changed this block, but still `BUILD SUCCESSFUL`); promoted and deployed via `ops/promote_v1.sh`/`ops/deploy_v1.sh`; independently re-verified — `git log -1` on the server shows `9bb25c2`, `/livez`/`/healthz` both `ok`, `sudo journalctl` clean of `error`/`panic`/`fatal` in the 30 seconds after restart, and `capability_registry` re-confirmed at 0 of 13 enabled. Same zero-user-impact property as §58: this ships real code and schema live without changing any real user's experience, since `SELECTED` (and every other v1.1 feature) stays behind its disabled capability gate.

Next: extend `SELECTED` to Map/realtime/replay (the concrete gap this block leaves open, mirroring §57's continuation of §56), or `EVENT_RECOMMENDATION`, or README §6.9's "after-check flow" given a specification.

## 60. 2026-09-11 — closes §59's own explicitly stated gap: `SELECTED` visibility now covers Map, realtime viewer, city realtime channel, and idempotency-replay (full parity with `LINKS`)

**Every surface §59 named as an open, known gap is closed in this block, not partially — the exact continuation §57 already did for `LINKS`, now repeated for `SELECTED`.** All four: `citymap_store.go` (`Viewport` + `PlaceSlots`), `realtime_viewer_store.go` (`viewerRealtimeSQL`), `realtime_city_store.go` (`cityRealtimeSQL`), and `idempotency_replay.go` (`slot.leave`'s fallback branch) now gain a `SELECTED` sibling branch alongside the `PUBLIC`/`LINKS` branches already there: `s.visibility='SELECTED' AND s.state IN ('PUBLISHED','FILLING','FULL') AND EXISTS(a slot_selected_viewers row for this viewer) AND NOT EXISTS(blocked)`.

- `citymap_store.go`: `Viewport` and `PlaceSlots` each gain the `SELECTED` `OR`-branch as a direct sibling of `LINKS`'s, checking `EXISTS(SELECT 1 FROM slot_selected_viewers v WHERE v.slot_id=s.id AND v.user_id=$N::uuid)` in place of the friendship check — no new block-check needed, each visibility branch here already carries its own.
- `realtime_viewer_store.go`: `viewerRealtimeSQL`'s lifecycle-event branch gains the same `SELECTED` sibling, with its own inline `NOT EXISTS(user_blocks)`, matching the shape `LINKS` already has there. First insertion attempt mis-nested the new block *inside* `LINKS`'s own parens instead of as a sibling (a flawed line-counting heuristic in the Python-based insertion script, given this file's tab-indented SQL); caught immediately by re-reading the file, reverted cleanly via `git checkout --`, and redone correctly using an assert-verified exact line index rather than a heuristic.
- `realtime_city_store.go`: `cityRealtimeSQL` operates on the outbox event's JSON *payload* (`e.payload->>'visibility'`/`e.payload->>'previousVisibility'`), same as `LINKS` there — the `SELECTED` branch checks both current and previous payload visibility against `('PUBLISHED','FILLING','FULL')` states, `AND EXISTS(slot_selected_viewers)`, with the existing outer `NOT EXISTS(user_blocks)` already covering this new branch too (confirmed by reading the surrounding `AND`/`OR` nesting and operator precedence before editing, not assumed) — no separate block-check duplicated, exactly like `LINKS` didn't need one either.
- `idempotency_replay.go`: `slot.leave`'s fallback gains a `SELECTED` sibling term to `PUBLIC`/`LINKS`, with its own inline `AND NOT EXISTS(blocked)`, matching that branch's per-term (not shared) block-check convention.
- `idempotency_replay_test.go`: proactively added `CREATE TEMP TABLE slot_selected_viewers(...)` to **both** pre-existing test fixtures (`TestIdempotencyReplayAuthorizationSQL` and `TestIdempotencyReplayAuthorizationSQLLinksVisibility`), not just the new one — avoiding, before it could happen, the exact `uuid`/`text` collision bug §57 hit and fixed for `friendships` (an unqualified bare table reference in the SQL under test resolving to the real migrated table from earlier tests in the same session database instead of the fixture's own text-typed temp table).

New tests:
- `internal/postgres/v11_selected_visibility_integration_test.go`: `TestV11SelectedVisibilityMap` (Viewport cluster counts and PlaceSlots both exclude a SELECTED slot for a non-selected stranger, include it for the selected user) and `TestV11SelectedVisibilityRealtimeViewer` (PullViewer excludes `slot.created` for a stranger, includes it for the selected user) — direct mirrors of `TestV11LinksVisibilityMap`/`TestV11LinksVisibilityRealtimeViewer`.
- `internal/postgres/idempotency_replay_test.go`: `TestIdempotencyReplayAuthorizationSQLSelectedVisibility`, mirroring `TestIdempotencyReplayAuthorizationSQLLinksVisibility` — a non-selected stranger denied, the selected user allowed, a block overriding selection, and the fallback disappearing outside `PUBLISHED`/`FILLING`/`FULL`.
- No new dedicated test was written directly against `cityRealtimeSQL`'s SQL text — matching exactly the level of verification §57 itself used for `LINKS` on that same file (careful manual read of the nesting/precedence plus `gofmt`/`build`/`vet`, no bespoke test harness for that one query), not a lower bar invented for this block.

Executed evidence, this session:

- `gofmt -l` on every touched file (`citymap_store.go`, `realtime_viewer_store.go`, `realtime_city_store.go`, `idempotency_replay.go`, `idempotency_replay_test.go`, `v11_selected_visibility_integration_test.go`) — clean;
- `go build ./...`, `go vet ./...` — clean;
- the new/mirrored tests run individually first (`TestV11SelectedVisibility*`, `TestIdempotencyReplayAuthorizationSQL*`) — all pass;
- the full `go test -count=1 ./...` suite against the same disposable PostgreSQL 16+PostGIS (`linkup_api` role present), run three consecutive times — all three green;
- `go test -race -count=1 ./...` — the block's own new/touched tests pass cleanly in isolation under `-race`; the full-suite run under `-race` additionally showed 5 failures, all in `TestFriendStoreRequestCreatesPendingAndNotifiesTarget`, `TestFriendStoreReverseRequestAutoAcceptsAndNotifiesOriginalRequester`, and three `TestNotificationProjector*` cases — none in a file this block touched. Root-caused, not shrugged off: the container's wall clock read `23:01 UTC` at that moment, inside the notification pipeline's default quiet-hours window (`23:00`-`08:00 UTC`), and these particular tests call the real projector without forcing `now` into a fixed instant the way `TestNotificationProjectorSuppressedByQuietHours` deliberately does — pre-existing, environment-time-dependent flakiness, not a regression from this block. Recorded honestly rather than silently re-run until green.
- `go mod tidy` — no diff.

Not done or claimed in this block: the pre-existing quiet-hours wall-clock test flakiness just found (real, but out of this block's scope — a fix would mean auditing every notification-projector test for a forced `now`, a separate piece of work); any Android change (this block was backend-only, no UI surface exposes an allow-list picker yet); any change to `SELECTED`'s own still-open gaps from §59 (no allow-list edit after creation; `SelectedUserIDs` not re-populated on every read). With this block, `SELECTED` reaches exactly the same discovery/replay parity `LINKS` has: both now cover `Get`/`ListPulse`/Map/realtime-viewer/city-realtime/idempotency-replay. Among README §4.3's six additional visibility modes, only `CITY`/`LASSO`/`TRAVEL_CORRIDOR` remain unimplemented, and all three need City Context/Fly domain integration this session has not scoped.

No new migration is required for this block (only SQL string constants and Go query logic change; the runtime table these queries read, `slot_selected_viewers`, was already created by migration `000032` in §59).

**Deployed to production, same session, following the exact §58/§59 pipeline again — independently re-verified, not taken on a script's word:** `claude/verification-push-main-xsofu0` → fast-forward merge → push to `main` (commit `ab7ab0a`). On the server (`89.168.108.176`, `/opt/linkup/src`): `git fetch`+fast-forward merge to `ab7ab0a`; `ops/build_v1.sh` run — real Go build/test/vet/`-race` (all packages green, no `FAIL`, matching the sandbox's own run) plus a real Android `testDebugUnitTest`/`lintDebug`/`assembleDebug` (`BUILD SUCCESSFUL`, mostly `UP-TO-DATE` since only backend files changed this block); `ops/promote_v1.sh ab7ab0a...` then `sudo ops/deploy_v1.sh` (the deploy script's first health-check line showed a transient `curl: (7) Failed to connect` while the service was still restarting, then its retry loop succeeded — the same benign transient §58/§59 already documented, not a real failure, confirmed by the script reaching its final success markers rather than its rollback trap). Independently re-verified after deploy, not just from the script's own output: server `git log -1` shows `ab7ab0a` exactly; `systemctl status linkup-api` shows `active (running)`; both `/livez` and `/healthz` return `{"status":"ok"}`; `journalctl` for the 60 seconds after restart has zero `error`/`panic`/`fatal` lines; `capability_registry` on production Supabase (`oavnrlwsfiiehluubwjk`) re-queried directly and confirmed all 13 capabilities still `enabled:false`. Same zero-user-impact property as §58/§59: real code now live, no real user's experience changes, since `SELECTED` (and every other v1.1 feature) stays behind its disabled capability gate.

Next: `EVENT_RECOMMENDATION`, README §6.9's "after-check flow" given a specification, or `CITY`/`LASSO`/`TRAVEL_CORRIDOR` if City Context/Fly integration gets scoped — or, separately from feature work, auditing the notification-projector test suite for the quiet-hours wall-clock flakiness just found so it stops being a live trap for the next `-race` run.

## 61. 2026-09-11 — closes `SELECTED` visibility's last two stated gaps from §59/§60: allow-list editable while DRAFT, and re-populated on every host read

**Session/environment note, stated plainly rather than left implicit:** this session runs in a different execution environment than §46-§60 (no inherited SSH session to the Ubuntu server, no attached Supabase MCP project connection). This block is real, tested, backend-only work, pushed to this session's own designated branch (`claude/loving-pasteur-ugv572`) rather than `main` or the earlier `claude/verification-push-main-xsofu0` — a harness-level instruction for this specific session pins its branch and explicitly withholds permission to push elsewhere without the user asking directly, which takes priority over RULE 7's general "direct to main" default the same way §58 itself already established that a direct, explicit instruction can override a standing rule. **No production deployment claim is made in this block** — this environment was not given Ubuntu/Supabase production access, so unlike §58-§60, "deployed to production" is not stated here at all rather than assumed or reused from an earlier block's evidence.

**What was actually missing, confirmed by reading the code before writing anything:** `Slot.SelectedUserIDs`'s own doc comment (§59) already stated the two real gaps: (1) "no way to edit an existing SELECTED Slot's allow-list after creation" (a host who picked the wrong people had to cancel and recreate the whole Slot), and (2) "`Get`/`ListPulse`/`ListMine` do not re-populate `SelectedUserIDs` on every read, only immediately after creation." Both closed in this block, not partially.

**Implemented:**

- `internal/slot/model.go`: `EditInput.SelectedUserIDs []string` — read only when `Visibility` is set to `VisibilitySelected` in the same patch (still DRAFT-only, the same gate `Visibility` itself already has); always a full replacement, no partial add/remove. `Slot.SelectedUserIDs`'s doc comment updated to describe the real, current contract: populated on every Create/Edit/Get/ListPulse/ListMine for a SELECTED Slot, but **only for the Slot's own host** — a selected viewer (or anyone else reaching the Slot) never receives the full roster of who else was picked, on any read. This is a deliberate privacy decision stated here explicitly, not a silent side effect: the allow-list is the host's private curation, not a shared participant roster.
- `internal/slot/service.go`: `normalizeEdit` now validates `SelectedUserIDs` via the existing `normalizeSelectedUserIDs` (non-empty, well-formed UUIDs, deduplicated, sorted for a stable idempotency fingerprint — identical validation to `Create`) whenever the patch sets `Visibility` to `SELECTED`; rejects with `ErrInvalidInput` if the list is empty (switching to SELECTED, or re-asserting it, with nobody selected is a clear input error, not a silent no-op); discards any `SelectedUserIDs` the caller supplied when `Visibility` is absent or set to anything else, so the allow-list is never touched by an edit that isn't explicitly re-asserting `SELECTED`.
- `internal/postgres/v11_hosting_store.go`: generalized `attachSelectedUserIDsTx` to accept a small `rowQuerier` interface (`Query(ctx, sql, args...) (pgx.Rows, error)`) satisfied by both `*pgxpool.Pool` and `pgx.Tx`, so the same helper works on a read path with no open transaction and on a write path inside one. Added `attachSelectedUserIDsForHost`, the read-path wrapper that only calls through when `out.Organizer.ID == actorID` — the privacy gate described above.
- `internal/postgres/v11_slot_store.go`:
  - `Get`, `ListPulse`, `ListMine` each now call `attachSelectedUserIDsForHost` after scanning, closing gap (2) for the host on every read (a non-host viewer's `SelectedUserIDs` stays empty exactly as before, `omitempty` hides it in the JSON response the same way it always did).
  - `Edit`: when the patch sets `visibility='SELECTED'` (only reachable, per the pre-existing DRAFT gate immediately above this new code, while the Slot is still `DRAFT`), the existing `slot_selected_viewers` rows for the Slot are deleted and `patch.SelectedUserIDs` bulk-inserted via `pgx.CopyFrom` in the same transaction as the rest of the edit — the identical pattern `CreateDraft` (§59) already established, not a new one invented here. Both the replay branch and the normal-commit branch of `Edit` now call `attachSelectedUserIDsTx` unconditionally (not the host-gated wrapper — `Edit` already enforces `actorID==hostID` earlier in the same function, so there is no non-host caller to guard against, mirroring `CreateDraft`'s own unconditional call).
- `internal/httpserver/slot_handlers.go`: `editSlotRequest.SelectedUserIDs []string` (`json:"selectedUserIds"`), wired into `slot.EditInput{...}` alongside the pre-existing `Visibility` field.

**A real, pre-existing design fact found and worked around, not asserted from memory:** the first draft of this block's own integration test asserted the host would see their freshly-edited `SELECTED` Slot in their own `ListPulse` feed. It failed — reading `listV11PulseSQL` (§56/§57) showed its visibility `WHERE` clause has no `OR s.host_id=$1` branch at all for non-`PUBLIC` visibility; a host's own `PRIVATE`/`LINKS`/`SELECTED` Slot has never appeared in their own Pulse feed, since §52/§56/§59 respectively shipped those modes — Pulse is a discovery feed for other people's Slots, and a host already has `ListMine(view="HOSTING")` (§9/original v1.0 foundation) as their own-Slot management read path. Not a bug introduced or found in this block; the test was corrected to assert against `ListMine(HOSTING)` instead, which is where this re-population gap actually needed to be proven closed for the host.

New tests:
- `internal/slot/service_test.go`: `TestEditRejectsSelectedVisibilityWithoutAllowList` (replaces the now-stale `TestEditRejectsSelectedVisibility`, whose own comment claimed SELECTED was categorically rejected as an edit target — no longer true), `TestEditPassesSelectedVisibilityAndAllowListThroughToStore`, `TestEditIgnoresSelectedUserIDsWhenVisibilityNotChanging`.
- `internal/postgres/v11_selected_visibility_integration_test.go`: `TestV11SelectedVisibilityAllowListEditableWhileDraft` — end-to-end against real PostgreSQL: host creates a DRAFT SELECTED Slot with one allow-listed user, confirms their own `Get()` re-populates it; edits the allow-list to swap in a different user while still DRAFT; publishes; confirms the dropped user loses visibility and the added user gains it; confirms the newly-added (non-host) selected user's own `Get()` never receives the full roster; confirms the host's post-publish `Get()` and `ListMine(HOSTING)` both show the replaced list; confirms editing to `SELECTED` with an empty list is rejected with `ErrInvalidInput` rather than silently dropping the existing one.

Executed evidence, this session (this environment has no inherited SSH/Ubuntu/Supabase-project access, so a local disposable PostgreSQL was provisioned instead — same migration set, same test harness, same evidence standard §46 onward already established, not a lower bar):

- installed `postgresql-16-postgis-3` and started a local PostgreSQL 16 cluster in this sandbox; created a disposable `linkup_test` database and a `linkup_api` role;
- `cmd/migrate` run against it with `LINKUP_MIGRATION_DATABASE_URL` — migrations `000001..000032` applied cleanly, no errors;
- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file (`internal/slot/model.go`, `internal/slot/service.go`, `internal/slot/service_test.go`, `internal/postgres/v11_hosting_store.go`, `internal/postgres/v11_slot_store.go`, `internal/postgres/v11_selected_visibility_integration_test.go`, `internal/httpserver/slot_handlers.go`) — clean;
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (with the `linkup_api` role present) — all three runs show the identical 5 pre-existing failures already documented in §60 (`TestFriendStoreRequestCreatesPendingAndNotifiesTarget`, `TestFriendStoreReverseRequestAutoAcceptsAndNotifiesOriginalRequester`, `TestNotificationProjectorApprovalAndRequestCreated`, `TestNotificationProjectorNoPusherConfigured`, `TestNotificationProjectorEventReminder`) and nothing else. Not assumed pre-existing: reproduced identically (`git stash` back to this block's parent commit, same two representative failing tests, same failures) with the container's wall clock reading `23:2x UTC`, inside the documented `23:00`-`08:00 UTC` quiet-hours window §60 already root-caused — confirmed this block introduces no new failure, rather than re-run-until-green;
- `go test -race -count=1 ./...` — the same 5 known failures, no new ones, no race detector reports;
- `go mod tidy` — no diff.

Not done or claimed in this block: the pre-existing quiet-hours wall-clock test flakiness (still real, still out of scope — a fix means auditing every notification-projector test for a forced `now`, separate work); `SelectedUserIDs` still has no partial add/remove — every edit fully replaces the list; no production deployment or verification of any kind (this environment has no such access this session — stated once here, applies to the whole block); Android UI/build (no SDK in this environment); `CITY`/`LASSO`/`TRAVEL_CORRIDOR` remain the only unimplemented README §4.3 visibility modes; `EVENT_RECOMMENDATION` and README §6.9's "after-check flow" remain unstarted, exactly as every prior block already stated, for the same reasons (no recommendation-candidate design, no specification beyond two words).

**Honest restatement of overall v1.1 scope, since this block was explicitly asked to "finish v1.1 to the very end":** README §6 spans 29 subsections (§6.1-§6.29) — realtime+durable offline, City Context, Map v1, hosting expansion, Waitlist/Host Control V2, Chat V2, Notifications, BUMP/Reliability, City BPM/recommendation ranking, Auto-Swarms, Fly Now/Travel/Motion, Me 2.0/Squad Radar, AR/Ranking V2, cold-start/venue ecosystem, offline real-world ops, safety/accessibility expansion, ephemeral media, adaptive experience, and the full LinkUp+ billing/travel/host/discovery/privacy/identity/forgiveness/rewarded-access surface, plus final ecosystem hardening — on top of a real, compiled, device-tested Android client and a real production deployment/verification pass. The great majority of that surface has no code in this repository at all yet (§12 of this file's own "Later Version 1 blocks" list, essentially unchanged since it was written). This is a genuinely multi-month, many-block scope for a real engineering team, not something any single work session finishes — stated here plainly rather than declared closer to "done" than the evidence supports (RULE 6). This block closes one small, real, fully-tested gap in already-shipped `SELECTED` visibility; the next dependency-safe block continues the same way.

Next: `EVENT_RECOMMENDATION` (still needs a real recommendation-candidate design — none invented here), `CITY`/`LASSO`/`TRAVEL_CORRIDOR` visibility (all three need City Context/Fly domain integration not yet scoped), README §6.9's "after-check flow" given a real specification, or the pre-existing notification-projector quiet-hours test flakiness (§60/§61, real but out of either block's own scope). Continuing block-by-block, backend-first, in this session's designated branch.

## 62. 2026-09-11 — realtime reconnect/first-run cursor bootstrap (README §6.2); `CITY` visibility fully implemented in one pass (README §4.3's fourth non-`PUBLIC` mode, after PRIVATE/LINKS/SELECTED); live Android enum bug fixed

**User instruction driving this block:** given a large enumerated status list ("Є" 42 items / "Немає" 37 items) and told to take groups of 5 and implement them, reporting exactly which were completed. The requested first five were `CITY`/`LASSO`/`TRAVEL_CORRIDOR` visibility and "full BUMP flow"/"full Reliability flow." Each was audited against actual repository code before deciding what to do, not assumed from the enumerated list's own framing (§55's own standing practice) — see the per-item accounting at the end of this entry for what that audit found.

### Part A — Realtime reconnect/first-run cursor bootstrap (README §6.2)

**Real gap found, not previously stated in any worklog entry:** `PullViewer`/`PullCity` already convergedcorrectly and efficiently from any starting cursor (§scanned-forward design, bounded per call, cursor always advances even across a page with zero visible events) — but a client with **no prior cursor at all** (fresh install, cleared local storage, a process-death with lost state) had no way to start anywhere except `after=0`, meaning its first sync would work forward through the *entire* realtime history it has no use for, one bounded page at a time, before ever reaching "now." README §6.2's own reconnect model lists "obtain authoritative snapshot/cursor" as step 2, distinct from replaying deltas (step 3) — this was never actually implemented as a distinct capability.

**Implemented — a bootstrap/"resync" cursor, backend and Android:**

- `internal/realtime/feed.go` + `city_feed.go`: `ViewerFeedStore`/`CityFeedStore` gain `CurrentCursor(ctx) (int64, error)`; `FeedService`/`CityFeedService` gain a validated `CurrentCursor(ctx, viewerID)` wrapper (viewerID is checked for the same auth-adjacent shape every other read here has, even though the underlying value carries no viewer-specific content to protect).
- `internal/postgres/realtime_viewer_store.go`: `RealtimeViewerStore.CurrentCursor` — a single `SELECT COALESCE(max(sequence),0) FROM domain_outbox_events`, satisfying both interfaces at once (both channels already share the identical sequence space, confirmed by reading `PullCity`'s own scan query before relying on it).
- `internal/httpserver/realtime_routes.go` + `server.go`: two new capability-gated endpoints, `GET /v1/realtime/cursor` and `GET /v1/realtime/city/cursor`.
- Android: `RealtimeApi`/`CityRealtimeApi` gain `currentCursor(): Long`; `RealtimeApiClient`/`CityRealtimeApiClient` implement it against the new endpoints, mirroring each file's own existing request/retry/error-parsing shape exactly. `RealtimeCursorStore`/`CityRealtimeCursorStore` gain `hasSynced(userId): Boolean` (distinguishes "never persisted" from "persisted a genuine 0" — `load()` alone cannot). `RealtimeCoordinator`/`CityRealtimeCoordinator` gain `bootstrapIfNeeded(userId)`: a no-op once any cursor exists, otherwise fetches and persists the current server cursor directly, skipping historical replay entirely. **Stated limitation, not silently glossed over:** `RealtimeCursorStore.save` only ever advances, so on a genuinely empty outbox (cursor==0 — realistically only a brand-new deployment with zero domain events ever) the persist is a no-op and `bootstrapIfNeeded` would re-fetch on every call until the first real event ever occurs; costs at most a few redundant network calls in that one unlikely case, never an incorrect cursor.
- Every Android file in this block is unverified in the same sense every prior Android block has stated: no SDK/Gradle in this environment, written by careful mirroring of existing working code and manual review, not compiled or run here.

New tests:
- `internal/realtime/feed_test.go` + new `city_feed_test.go`: `TestFeedServiceCurrentCursor`/`TestCityFeedServiceCurrentCursor` and their empty-viewer/store-failure counterparts.
- `internal/httpserver/realtime_routes_test.go` + `realtime_city_routes_test.go`: auth/service-unavailable/success/error-mapping tests for both new handlers.
- `internal/postgres/v11_realtime_viewer_integration_test.go`: `TestV11RealtimeCurrentCursorBootstrap` — end-to-end against real PostgreSQL: creates a Slot (real pre-existing history), bootstraps a fresh viewer's cursor to the current live position, proves the very next pull from that cursor sees nothing (the pre-existing history is correctly skipped), then proves a genuinely new event afterward (an Edit) *is* still reached going forward from the bootstrapped cursor — proving this is a real fast-forward, not an off-by-one that would also hide the next real event.
- Android: `RealtimeCoordinatorTest.kt`/`CityRealtimeCoordinatorTest.kt` gain `bootstrapIfNeeded adopts server cursor...`/`...is a no-op once a cursor already exists`/`a real event after bootstrap is still reached going forward` cases, with `FakeRealtimeApi`/`FakeCityRealtimeApi` and `MemoryCursorStore`/`MemoryCityCursorStore` extended to implement the new interface members.

### Part B — `CITY` visibility (README §4.3's fourth non-`PUBLIC` mode)

**Design, decided by reading existing infrastructure before writing anything:** README §4.3 lists "City-only" without further elaboration. This repository already has real, tested City Context infrastructure (`internal/citycontext`, `city_context_locks` — a user's current locality lock with its own freshness/expiry policy, README §6.3) built in an earlier session. Rather than inventing a new per-Slot "which city" field, `CITY` visibility is defined as: **discoverable to a viewer only while both the viewer and the host currently hold a live, unexpired `city_context_locks` row for the *same* locality, evaluated fresh on every read** — never "same city as when the Slot was created." This needed **no new migration or table at all**: `CITY` was already in the `slots.visibility` `CHECK` constraint since migration `000002` (confirmed before writing this block, matching how PRIVATE/LINKS/SELECTED each confirmed the same before their own first line of code), and `city_context_locks` already existed. Either side missing a live lock (never resolved, or expired) fails closed — not discoverable — matching every other visibility mode's own "fail closed" default. Unlike `SELECTED`, there is no allow-list to configure at creation or edit time, so `CreateInput`/`EditInput` needed no new field.

**Implemented, full parity in one pass (unlike LINKS/SELECTED, which each shipped Get/Pulse first and Map/realtime/replay in a separate follow-up block — this block closes all of it at once, now that the exact pattern from those two is well-established):**

- `internal/slot/model.go`: `VisibilityCity Visibility = "CITY"`.
- `internal/slot/service.go`: `validVisibility` accepts `CITY`; falls through the existing generic non-`SELECTED` path in `normalizeCreate`/`normalizeEdit` (no allow-list handling needed).
- `internal/postgres/v11_slot_store.go`: `getV11SlotSQL` and `listV11PulseSQL` each gain a `CITY` branch: `EXISTS(city_context_locks vcl JOIN city_context_locks hcl ON hcl.locality_id=vcl.locality_id WHERE vcl.user_id=viewer AND vcl.expires_at>now() AND hcl.user_id=s.host_id AND hcl.expires_at>now())`.
- `internal/postgres/citymap_store.go`: `Viewport`'s `visible_places` CTE and `PlaceSlots`'s `WHERE` clause each gain the identical branch.
- `internal/postgres/realtime_viewer_store.go`: `viewerRealtimeSQL`'s lifecycle-event branch gains the sibling branch (its own inline block check, matching that file's per-term convention already established for LINKS/SELECTED).
- `internal/postgres/realtime_city_store.go`: `cityRealtimeSQL` gains the sibling branch against the outbox payload's `visibility`/`previousVisibility` fields, same shape as LINKS/SELECTED there.
- `internal/postgres/idempotency_replay.go`: `slot.leave`'s fallback gains the sibling branch.
- `internal/postgres/idempotency_replay_test.go`: `CREATE TEMP TABLE city_context_locks(...)` added to all three existing fixtures (proactively — avoiding, before it could happen, the exact `uuid`/`text` real-table-shadowing collision §57/§60 already hit and fixed for `friendships`/`slot_selected_viewers`).
- Android: `SocialModels.kt`'s `SlotVisibility` enum only ever had `{PUBLIC, PRIVATE, LINKS}` — **a real, live client bug found and fixed, not just new capability:** the backend has emitted `SELECTED` since backend worklog §59, and neither call site that parses a Slot response (`LinkUpApiClient.kt`, `DurableSocialApi.kt`, both via `SlotVisibility.valueOf(...)`) was ever updated for it — any real build against this backend since §59 would have crashed on the very first `SELECTED` (now also `CITY`) Slot it parsed. Fixed by extending the enum to `{PUBLIC, PRIVATE, LINKS, SELECTED, CITY}`. No other Android change needed: both create/edit body-builders already serialize `input.visibility?.let { body.put("visibility", it.name) }` generically, not hardcoded per value.

New tests:
- `internal/slot/v11_hosting_test.go`: `TestCreateDraftPreservesConfiguredCityVisibility`.
- `internal/slot/service_test.go`: `TestEditPassesCityVisibilityThroughToStore`; `TestEditRejectsUnknownVisibility` updated to use `"LASSO"` instead of `"CITY"` as its unimplemented-mode example (the same test previously used `"SELECTED"`, then `"CITY"`, each swapped out the block that implemented it — same stale-test pattern §59 already documented and fixed once).
- `internal/postgres/v11_city_visibility_integration_test.go` (new file): `TestV11CityVisibilitySlot` (end-to-end against real PostgreSQL — a viewer with no city lock at all excluded; a viewer locked to a *different* locality excluded; a viewer locked to the *same current* locality as the host included on Get/Pulse; a block between two same-city people still wins; a viewer with no lock at all can still `Request()` a known Slot ID directly), `TestV11CityVisibilityRejectedForLegacyCreate`, `TestV11CityVisibilityMapAndRealtimeViewer` (Map `Viewport`/`PlaceSlots` and the realtime viewer feed all honor the identical same-locality-lock rule).
- `internal/postgres/idempotency_replay_test.go`: `TestIdempotencyReplayAuthorizationSQLCityVisibility` — no lock denied, different-locality lock denied, same-locality lock allowed, a block overriding it, an *expired* same-locality lock denied (proving "currently," not "ever was"), and the fallback disappearing outside `PUBLISHED`/`FILLING`/`FULL`.

Executed evidence, this session (same disposable-PostgreSQL setup as §61 — no inherited SSH/Supabase-project access in this environment):

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched Go file — clean;
- migrations `000001..000032` applied cleanly (no new migration in this block at all);
- the full `go test -count=1 ./...` suite run three consecutive times against the same, non-recreated, accumulating disposable database (`linkup_api` role present) — all three runs show only the same 5 pre-existing quiet-hours failures §60/§61 already documented, reproduced identically, nothing new;
- `go test -race -count=1 ./...` — same 5 known failures, no new ones, no race reports;
- `go mod tidy` — no diff;
- one real regression caught and fixed mid-block, not left broken: adding `CITY` to `service_test.go`'s example of an unimplemented mode broke `TestEditRejectsUnknownVisibility` the moment `CITY` actually became implemented — caught by the full-suite run, fixed by switching that test's example to `LASSO` (still genuinely unimplemented) and adding a dedicated `TestEditPassesCityVisibilityThroughToStore` in `CITY`'s place.

### Part C — audit of "full BUMP flow" / "full Reliability flow" (the requested items 4-5): no new backend work, found already substantially real

Audited `internal/bump` (`model.go`, `postgres/bump_store.go`, migration `000028_v11_bump.sql`, `httpserver/bump_handlers.go`) against README §6.9's own bullet list before assuming anything was missing:

- **BUMP proof baseline** ✅ — `IssueChallenge`/`Confirm` two-sided flow, single client tap can never move reliability alone (`ConfirmResult.Verified` only true once both directions exist).
- **Anti-replay** ✅ — single-use, expiring `bump_challenges` nonce, consumed on use.
- **Anti-farm / one-event/one-contribution** ✅ — `bump_submissions`' own primary key `(slot_id, submitter_id, counterpart_id)` makes a second claim against the same counterpart for the same Slot impossible even with a fresh nonce; `requireAuth` (the shared middleware wrapping every authenticated route, BUMP's included) already applies a generic per-user `UserLimiter` rate limit — confirmed by reading `server.go`'s middleware chain, not assumed.
- **Reliability events / private+public bands / BUMP Vault baseline** ✅ — `reliability_events`, `user_reliability`, `Vault`, and the public/private `Band` split (backend worklog §54) are all real and tested.
- **Android Keystore-backed device proof** ❌ — explicitly documented in `Challenge`'s own doc comment as a stand-in awaiting a real Android build/device this environment cannot produce or verify; unchanged this block, not newly discovered.
- **"After-check flow"** ❌ — README gives no elaboration beyond the two words, exactly as backend worklog §54/§55 already stated; still not invented here, for the same reason RULE 2 gives every prior block: guessing at unspecified product behavior risks shipping something that only looks like the feature.

**Nothing was implemented for BUMP/Reliability in this block** — not because the item was skipped, but because auditing it first (rather than assuming the enumerated status list's "Немає" framing was itself ground truth) showed the backend-buildable majority of it already real and tested from an earlier session, and the two remaining pieces are genuinely blocked (no Android build access) or genuinely unspecified (no product answer to build against), not newly-discovered gaps this block could close honestly.

### `LASSO`/`TRAVEL_CORRIDOR` visibility: deliberately not attempted this block, a real design question raised to the user instead of guessed

Both need a live geometry-membership test — "is this viewer's current position inside the host-drawn shape" — that `CITY` did not: `CITY` only ever compares two `city_context_locks.locality_id` values (a coarse, already-privacy-reviewed identifier), never a raw coordinate. `LASSO`/`TRAVEL_CORRIDOR` cannot be evaluated the same way without persisting *some* form of the viewer's actual point (even approximate/rounded) beyond the resolved locality — this repository currently persists no such thing (`city_context_locks` stores only a `locality_id`, never a raw `latitude_e6`/`longitude_e6`). Building that storage is a real product/privacy decision (retention, precision/rounding, TTL, exposure surface) directly touching README §8's cross-version privacy/anti-stalking non-negotiables and this file's own repeated "no public exact stranger coordinates" language — not something to invent unilaterally the way a purely relational check (friendship row, allow-list row, locality-ID match) could be. Raised back to the user as a direct question in this same turn rather than guessed at or silently deferred.

Executed evidence for this whole block is listed in Part A/B above; Part C added no code and needed none.

Not done or claimed in this block: `LASSO`/`TRAVEL_CORRIDOR` visibility (a real design question was raised instead of guessing); `EVENT_RECOMMENDATION` and README §6.9's "after-check flow" remain unstarted for the same reasons every prior block already gave; the pre-existing notification-projector quiet-hours flakiness remains unfixed and out of this block's scope; Android build/compile/run verification of any kind (no SDK in this environment); no production deployment or verification of any kind (no SSH/Supabase-project access in this environment this session).

**Per-item accounting against the user's requested five, stated plainly rather than a blanket "done":**
1. `CITY` visibility — **done**, full parity, tested end-to-end.
2. `LASSO` visibility — **not done**; blocked on a real design decision now put to the user.
3. `TRAVEL_CORRIDOR` visibility — **not done**; same blocker as `LASSO`.
4. "Full BUMP flow" — **already done in an earlier session**, confirmed by audit; the only remaining pieces are blocked (Android hardware/build) or unspecified (after-check flow), not new work this block skipped.
5. "Full Reliability flow" — same as 4.

Next: the user's answer on `LASSO`/`TRAVEL_CORRIDOR`'s location-persistence design, or the next groups of 5 from the enumerated backlog, or `EVENT_RECOMMENDATION` if a recommendation-candidate design is provided.

## 63. 2026-09-12 — `LASSO`/`TRAVEL_CORRIDOR` visibility implemented (closes §62's own question): README §4.3 visibility model now fully real

**User's answer to §62's question:** a new, separate, privacy-safe point table (`city_context_points`) — rounded/coarsened, sharing `city_context_locks`'s own freshness/expiry discipline exactly. Built as specified, nothing invented beyond it.

**Implemented:**

- **`db/migrations/000033_v11_selected_viewers_delete_grant.sql`**: fixes a real, live privilege gap this session found while auditing grants before starting new geometry work — migration `000032` granted `linkup_api` only `SELECT`/`INSERT` on `slot_selected_viewers`, but §61's allow-list-replacement `Edit` code does `DELETE FROM slot_selected_viewers` first. Every test for that feature passed anyway because this repository's test harness connects as the `postgres` superuser, not `linkup_api` — the missing grant was never actually exercised until this session added a real privilege-check test for it (see below). Not a production failure found in the wild; a real gap found by reading grants before relying on them, exactly the kind of check RULE 2 exists for.
- **`db/migrations/000034_v11_city_context_points.sql`**: `city_context_points(user_id PK, latitude_e6, longitude_e6, permission_class, accuracy_m, observed_at, expires_at)` — `linkup_api` gets `SELECT/INSERT/UPDATE`, no `DELETE` (an upsert-per-user row, like `city_context_locks`, never an application-deleted log).
- **`internal/postgres/city_context_store.go`**: `Apply` (the real `citycontext.Service.Resolve` implementation) now also upserts a coarsened point into `city_context_points` in the *same transaction* as the lock, sharing its exact `expires_at` — they can never drift out of sync. `roundToPointGrid`/`roundToGrid`: rounds to the nearest ~111m grid (1000 E6 units), rounding toward the nearest line (bounded ±half a cell), never storing the raw observation. This runs unconditionally on every successful resolve, regardless of which visibility modes a Slot ever uses it for.
- **`internal/slot/model.go`**: `VisibilityLasso`/`VisibilityTravelCorridor`; `CreateInput.LassoPolygonWKT *string`; `CreateInput.CorridorLineWKT *string` + `CorridorRadiusM *int`; `Slot` gains the same three fields as a host-only echo (mirroring `SelectedUserIDs`'s exact privacy contract — a non-host viewer never sees the shape). Neither mode is editable yet (`EditInput` has no field for either) — matches `SELECTED`'s own original first-block scope before §61 added its allow-list edit.
- **`internal/slot/service.go`**: `validVisibility` now accepts all seven modes — README §4.3's full set is real. `normalizeCreate`/`normalizeEdit` rewritten as a `switch` over effective visibility: `LASSO` requires a plausible WKT `POLYGON` (prefix + length-bound sanity check, real geometry validity is PostGIS's job via `st_geomfromtext` at query time, not reimplemented here); `TRAVEL_CORRIDOR` requires both a WKT `LINESTRING` and a `1..50000` meter radius, together or not at all. `normalizeEdit` rejects both as edit targets outright (no field to configure either shape via edit).
- **`internal/postgres/v11_hosting_store.go`**: `CreateDraft`'s `INSERT` gains the three new nullable `slots` columns; `attachLassoAndCorridorTx`/`attachLassoAndCorridorForHost` mirror `attachSelectedUserIDsTx`/`ForHost` exactly (read straight from `slots`, no side table — unlike `SELECTED` there is only one polygon/route per Slot, not a list).
- **`internal/postgres/v11_slot_store.go`**: `Get`/`ListPulse`/`ListMine`/`Edit` all wire in the new attach helpers (Get/ListPulse/ListMine host-gated, Edit unconditional — actor is already host by then). `getV11SlotSQL`/`listV11PulseSQL` each gain `LASSO` (`public.st_contains(st_geomfromtext(polygon), viewer's point)`) and `TRAVEL_CORRIDOR` (`public.st_dwithin(st_geomfromtext(line)::geography, viewer's point::geography, radius)`) branches, joined against a fresh (`expires_at>now()`) `city_context_points` row for the viewer. **Scope explicitly limited to Get/ListPulse in this block** — Map/realtime-viewer/city-realtime-channel/idempotency-replay parity is deferred, matching exactly how `LINKS`/`SELECTED` each shipped discovery first and full parity in a dedicated follow-up block (§56→§57, §59→§60). Unlike those two, this isn't a "next block" placeholder: the city-realtime channel's own dispatch model is locality-scoped (an event only reaches a viewer whose *locked locality* matches the Slot's place's locality), which doesn't naturally fit `LASSO`/`TRAVEL_CORRIDOR` at all (a lasso can span or miss locality boundaries independent of any lock) — extending it needs its own design thought, not just a mechanical repeat of the `CITY` pattern, so it is named here as a real, harder-than-`CITY` follow-up rather than assumed identical.

New tests:
- `internal/postgres/city_context_store_test.go`: `TestRoundToGridNeverStoresRawValue`, `TestRoundToPointGridBoundsErrorToHalfAGridCell`.
- `internal/postgres/city_context_store_integration_test.go`: `TestV11CityContextStoreApplyWritesRoundedPoint` — a real `citycontext.Service.Resolve()` call (not a direct row insert) proves the point is coarsened (never the exact raw value), shares the lock's exact `expires_at`, and upserts (never accumulates a second row on a repeat resolve). This is also the first integration test this session found that exercises `CityContextStore.Apply` through a real `Resolve()` call at all — every prior City Context integration test inserted `city_context_locks` rows directly, a pre-existing coverage gap noted here, not fully closed (closing it in general is separate work).
- `internal/slot/v11_hosting_test.go` + `service_test.go`: create/edit unit tests for both modes (valid shape accepted and echoed, missing/malformed shape rejected, radius-out-of-range rejected, corridor line-without-radius/radius-without-line rejected, fields ignored for other visibilities, edit rejection for both). `TestEditRejectsUnknownVisibility` now uses a literal `"BOGUS_MODE"` instead of a real README term — every real mode is implemented now, so (as this test's own comment history already shows for `SELECTED` then `CITY` then `LASSO` each getting swapped out the moment they shipped) there is no longer a real "still unimplemented" mode left to reuse as the example.
- `internal/postgres/v11_lasso_travel_corridor_visibility_integration_test.go` (new file): `TestV11LassoVisibilitySlot`/`TestV11TravelCorridorVisibilitySlot` — end-to-end against real PostgreSQL+PostGIS: a viewer with no live point excluded; a viewer whose point falls outside the polygon/corridor excluded; a viewer inside included on Get/Pulse but never shown the host's own shape; the host's own read shows it; a block still wins even for a viewer geometrically inside the shape; a stranger with the Slot ID directly can still `Request()`. Plus both modes' legacy-create rejection regression tests.
- `internal/postgres/v11_city_context_privileges_integration_test.go`: extended with a `city_context_points` grant check (`SELECT/INSERT/UPDATE`, no `DELETE`).
- `internal/postgres/v11_selected_visibility_integration_test.go`: new `TestV11SlotSelectedViewersLinkupApiPrivileges` — a direct regression guard for the `000033` grant fix, so a future regression here is caught by a real privilege check instead of silently passing again the way it did before this test existed.
- Android: `SocialModels.kt`'s `SlotVisibility` enum extended to include `LASSO`/`TRAVEL_CORRIDOR` (same live-crash-prevention reasoning as the `SELECTED`/`CITY` fix two blocks ago). No `CreateSlotInput`/`EditSlotInput` wiring added for either mode's shape fields — there is no Android UI to configure a polygon or corridor route yet, and inventing one is a design decision out of a client-parsing block's scope (matching `SELECTED`'s own "not wired into any UI screen" stance from its introducing block).

**A real, non-product test-infrastructure bug found and fixed mid-block, disclosed rather than hidden:** two of this block's own new integration tests (`TestV11CityVisibilitySlot`, from §62, and `TestV11CityVisibilityMapAndRealtimeViewer`) registered their `t.Cleanup` calls in the wrong order relative to Go's LIFO cleanup execution — a locality/canonical_place delete was registered to run *before* the `slots`/`city_context_locks` rows referencing it were removed, so the delete silently no-op'd on an FK violation (`_, _ = pool.Exec(...)`, error deliberately ignored per this codebase's own established cleanup-helper convention) and permanently orphaned rows in this session's shared disposable database — accumulating enough over repeated runs to make a later test's own locality resolution pick the wrong overlapping row and fail non-deterministically. Root-caused by inspecting the actual leftover rows rather than guessed at; fixed by reordering registration in both tests (and additionally by giving the new `TestV11CityContextStoreApplyWritesRoundedPoint` a geographically distant, non-overlapping test locality so it can never be shadowed by another test's same-region fixture regardless of cleanup ordering elsewhere). This is a bug in this session's own test fixtures, not in any shipped product code, product migration, or product behavior — stated plainly rather than left implicit, since RULE 2's "no fake/hidden gaps" spirit applies to disclosure here too, not just to product functionality.

Executed evidence, this session (same disposable-PostgreSQL-in-sandbox setup as §61/§62 — no inherited SSH/Supabase-project access):

- installed environment unchanged from §61/§62 (PostGIS already present); **reset the disposable database from scratch** partway through this block once the cleanup-ordering bug above was found and fixed, to clear accumulated garbage rather than test against a polluted fixture;
- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean;
- migrations `000001..000035` applied cleanly (three new migrations this block: `000033` grant fix, `000034` points table, `000035` slots columns);
- the full `go test -count=1 ./...` suite run three consecutive times against the freshly-reset disposable database (`linkup_api` role present) — all three runs show only the same 5 pre-existing quiet-hours failures already documented in §60/§61/§62, reproduced identically, nothing new;
- `go test -race -count=1 ./...` — same 5 known failures, no new ones, no race reports;
- `go mod tidy` — no diff.

Not done or claimed in this block: `LASSO`/`TRAVEL_CORRIDOR` on Map/realtime-viewer/city-realtime-channel/idempotency-replay (a real, harder-than-`CITY` follow-up, explained above — not a mechanical repeat of the `CITY` pattern); neither mode is editable after creation; Android has no UI or input-wiring for either mode's shape, only the parsing-crash fix; no production deployment or verification of any kind (this environment still has no such access); `EVENT_RECOMMENDATION` and README §6.9's "after-check flow" remain unstarted for the same reasons every prior block already gave; the pre-existing notification-projector quiet-hours test flakiness remains unfixed and out of scope.

**README §4.3's visibility model is now fully implemented** for the first time: Public, Friends/Links, Selected people, City-only, Lasso/geo-scoped, Travel corridor, and Private/invite-only all have real, tested Go/PostgreSQL implementations (Map/realtime/replay parity varies by mode as documented above and in §56/§57/§59/§60 for the earlier ones). This closes a section of the README that had zero non-PUBLIC implementation as recently as this session's own start.

Next: extend `LASSO`/`TRAVEL_CORRIDOR` to Map/realtime/idempotency-replay (the city-realtime-channel design question named above still needs answering first); `EVENT_RECOMMENDATION`; README §6.9's "after-check flow" given a specification; or the next groups of 5 from the user's enumerated backlog.

## 64. 2026-09-12 — chat messages now generate real `MESSAGE` push notifications; grouping/collapse implemented; Chat V2/deep-links/BUMP audited (mostly already real)

**User instruction driving this block:** the next 5 items from the earlier enumerated backlog — "Full Notifications delivery flow", "Notifications deep-links", "Notifications delivery metrics", "Full Chat V2", `EVENT_RECOMMENDATION`. Each audited against actual code before assuming the enumerated list's own framing was accurate (the same practice §55/§62 already established) — see the per-item accounting at the end.

### Audit findings, before any code was written

- **Deep links** — already fully real (`internal/notification.DeepLink`), matching README §6.8's routing table exactly for every canonical type, with its own dedicated test (`TestDeepLinkMatchesReadmeRoutingTable`). Nothing to build.
- **Chat V2** (README §6.7) — realtime delivery (`slot.chat_message_created` already emits a realtime invalidation, confirmed via existing tests), offline/idempotent send retry (chat send already goes through Android's durable mutation outbox, `DurableSocialApi.kt`), duplicate convergence (idempotency key + `ON CONFLICT`), activation/expiry/retention (host/accepted-only gating + terminal physical purge, both real since v1.0/§8), system messages (`SLOT_STARTED`/`MEMBER_JOINED`/`MEMBER_LEFT`, real since earlier v1.1 blocks), kick/block/revocation (`authorizeChatTx` re-checks membership/block live on every call). All already real. **The one genuine gap:** chat messages never generated a push notification at all — `buildCandidate`'s own allow-list comment already said this outright ("extending it to more of README §6.8's canonical types (EVENT_RECOMMENDATION, MESSAGE grouping, admin PROMO campaigns) is future work"), confirmed by reading the switch statement, not assumed.
- **"Full BUMP flow"/"Full Reliability flow"** — re-confirmed unchanged from §62's own audit (proof baseline, anti-replay, anti-farm, reliability events, bands, vault all real; only Android Keystore proof and the unspecified "after-check flow" remain, both already-known gaps).
- **"Notifications delivery metrics"** (targeted/sent/delivered/opened/CTR) and **admin campaigns** — genuinely do not exist (confirmed: zero `campaign`-related code anywhere except one comment naming it as future work). This is its own substantial domain (admin roles, segmentation, scheduling, audited campaign CRUD) that this block does not attempt — named honestly as still-missing, not built partially and called done.
- **`EVENT_RECOMMENDATION`** — still has no recommendation-candidate design; not invented here, same reasoning every prior block gave.

### Implemented — MESSAGE notifications + grouping/collapse (both real gaps closed together, since they compound: without grouping, MESSAGE would spam a push per message in a burst)

- **A real, necessary schema change, not additive-only:** `notification_deliveries.source_event_id` was `UNIQUE` on its own, meaning at most one recipient could ever be notified per outbox event — true of every event type wired so far (each had exactly one recipient) but wrong for a chat message, which can have many simultaneous recipients. Migration `000036` drops that constraint and replaces it with `UNIQUE (source_event_id, user_id)`. Checked before making this change: `ReminderScanner`'s own existing per-recipient-event-emission pattern (its own comment already explains why it works around today's one-row-per-event limit) stays correct under the new, strictly looser constraint — every event it emits already has a single, distinct `user_id`.
- **`internal/notification/model.go`**: `Type.Groupable() bool` — true only for `MESSAGE` and `EVENT` (README §6.8's own two grouping bullets); `EVENT_REMINDER` and everything else excluded (a scheduled one-shot has no "burst" to collapse, and no other type has a natural same-thread identity).
- **`internal/notification/decide.go`**: `OutcomeSuppressedGrouped`; `DecisionInput` gains `LastGroupAnchorAt *time.Time` + `GroupWindow time.Duration`; `Decide` gains the grouping gate, last in the pipeline (after expiry/preference/quiet-hours/frequency-cap) so a notification that would already be suppressed for a more fundamental reason is never mis-reported as merely grouped. Semantics: a **fixed, non-overlapping** window per (recipient, Slot, Type) — the first notification in a burst is let through and becomes the new anchor; everything else within `GroupWindow` of that anchor collapses; the next one after the window elapses is a real, new notification and starts the next window. Explicitly *not* a rolling window (which could suppress forever under sustained traffic) — stated as a deliberate design choice, not assumed.
- **`internal/config/config.go`**: `NotificationGroupWindow` (default 5 minutes, `LINKUP_NOTIFICATION_GROUP_WINDOW` override) — same config pattern as `NotificationFrequencyCapWindow`.
- **`internal/postgres/notification_store.go`**:
  - `buildCandidate` → `buildCandidates` (`[]*candidate`, not `*candidate`): the one, necessary shape change for fan-out. Every existing single-recipient builder function is untouched — a new `oneOrNone` helper wraps each one's existing return into the new shape, so this is a one-line change at each of the 5 existing call sites, not a rewrite of any of them.
  - `projectOne` now loops over every candidate an event fans out to, calling a new `projectCandidate` (the old per-recipient body, extracted) for each — the outbox cursor still advances exactly once per event, after all its recipients are processed, not once per recipient.
  - New `buildChatMessageCandidates`: recipients are every current host/accepted participant of the Slot except the sender, block-excluded the same way `ChatStore.ListRecent` already filters blocked authors. The notification body deliberately never echoes the message's own text — a zero-trace chat product (README §6.7) should not leak message content into an externally-visible, provider-retained push payload.
  - New `lastGroupAnchor`: finds the most recent *non*-grouped delivery for (user, Slot, Type) to feed `Decide`'s window check.
  - **A real, distinct bug found and fixed while building this, not glossed over:** the `INSERT INTO notification_deliveries` never set `created_at` explicitly, relying on the column's `DEFAULT now()` — indistinguishable from correct in production (real wall clock either way), but silently wrong for grouping, which reads `created_at` back as the window anchor: a test-controlled fake clock (`projector.now`) would compute `Decide`'s `Now` against one clock while the anchor it just wrote reflected a completely different one (the database server's real wall time), especially since this sandbox's real date is now later than assorted historical fixture dates elsewhere in this test suite. Found by a failing test, not assumed from reading the code alone — root-caused by comparing the actual timestamps involved, not re-run until green. Fixed by inserting `p.now()`'s own value into `created_at` explicitly, matching `expires_at`'s own existing convention of always deriving from the same in-request `now`.

New tests:
- `internal/notification/decide_test.go`: `TestDecideGroupingSuppressesWithinWindow`, `...DeliversOnceWindowElapses`, `...NoAnchorAlwaysDelivers`, `...WindowZeroOrNegativeDisablesGrouping`, `...DoesNotApplyToUngroupableTypes`.
- `internal/notification/model_test.go`: `TestGroupableOnlyMessageAndEvent`.
- `internal/postgres/notification_store_integration_test.go`: `TestNotificationProjectorChatMessageFanOutAndGrouping` — end-to-end against real PostgreSQL: a host's chat message notifies both other accepted participants (never the sender); a block from one recipient toward the host stops that recipient's future notifications while the other's continue; a second message inside the grouping window collapses (`SUPPRESSED_GROUPED`, no second push); a third message after the window elapses produces a real second push.

Executed evidence, this session (same disposable-PostgreSQL-in-sandbox setup as §61-§63):

- `go build ./...`, `go vet ./...` — clean; `gofmt -l` on every new/touched file — clean (`cmd/api/main.go`'s two changed call sites remain within that file's own pre-existing not-gofmt-clean dense style, confirmed by `gofmt -d`, not a new issue introduced);
- migrations `000001..000036` applied cleanly (one new migration this block);
- all 9 existing `NewNotificationProjector(...)` call sites across `cmd/api/main.go` and 2 test files updated for the new `groupWindow` parameter;
- the full `go test -count=1 ./...` suite run three consecutive times against the same disposable database (`linkup_api` role present) — all three runs show only the same 5 pre-existing quiet-hours failures already documented in §60-§63, reproduced identically, nothing new; the new chat-notification test itself confirmed independently stable across 5 repeated runs in isolation before the full-suite run;
- `go test -race -count=1 ./...` — same 5 known failures, no new ones, no race reports;
- `go mod tidy` — no diff.

Not done or claimed in this block: admin campaigns (segmentation, scheduling, audited CRUD, `targeted`/`sent`/`delivered`/`opened`/CTR analytics) — a real, substantial, still-entirely-missing domain, not attempted here; `EVENT_RECOMMENDATION` — still no recommendation-candidate design; README §6.9's "after-check flow" — still unspecified; Android has no notification-tap/deep-link-routing/FCM-receiver code at all to extend (unlike the `SlotVisibility` enum fix two blocks ago, there was no existing client contract here to find broken — this is a first-time gap, not a regression, and building a whole new Android notification-handling surface is out of a backend-block's scope); no production deployment or verification of any kind (no Ubuntu/Supabase access in this session); the pre-existing notification-projector quiet-hours test flakiness remains unfixed and out of scope.

**Per-item accounting against the five requested items:**
1. "Full Notifications delivery flow" — **partially done this block**: the core dedupe→TTL→quiet-hours→frequency-cap→grouping→push pipeline was already real; this block closes the one concrete gap in it (MESSAGE never wired) and adds grouping. Admin campaigns remain a separate, unstarted domain.
2. "Notifications deep-links" — **already done** in an earlier session; confirmed by audit and existing test, nothing new needed.
3. "Notifications delivery metrics" — **not done**; this is the admin-campaign analytics surface, which does not exist at all.
4. "Full Chat V2" — **already substantially done** in earlier sessions; this block closes the one real gap found (MESSAGE notifications), everything else audited and confirmed real.
5. `EVENT_RECOMMENDATION` — **not done**; still blocked on a real recommendation-candidate design this session will not invent.

Next: admin campaigns (a real, large, dependency-heavy domain — admin role/authorization concept, segmentation, scheduling, analytics, audit events — worth a direct scoping conversation before starting, given its size relative to every block so far this session); `EVENT_RECOMMENDATION` or README §6.9's "after-check flow" given a specification; `LASSO`/`TRAVEL_CORRIDOR` Map/realtime/replay parity; or the next groups of 5 from the user's enumerated backlog.

## 65. 2026-09-12 — Pulse relevance ranking (README §6.10 Layer 2): user interests + additive `sort=relevance`

**Context:** continuing the "next 5 items" batch from §64 (this block finishes the ranking-pipeline work that was already in progress at the start of this session, rather than starting a new batch of 5 — the previous block's own "Next" line named it explicitly: "the next groups of 5 from the user's enumerated backlog" was deferred in favor of finishing what was already mid-flight).

### What existed before this block, and the one real gap in it

`slot.PulseSort` (`RECENCY`/`RELEVANCE`), the `Store.ListPulse` signature threading it through, `internal/httpserver`'s `sort` query param wiring, and `internal/postgres`'s `listV11PulseRelevanceSQL` (ranking by `s.activity = ANY(viewer's interests)` before the existing recency tiebreak) were all already written earlier in this session. But `listV11PulseRelevanceSQL` referenced `app_users.interests` — a column that did not exist anywhere in the schema. `sort=relevance` would have failed at the SQL level on first real use. This was identified and explicitly flagged as a blocking gap before this block started, not discovered here.

### Implemented — the missing `interests` field, end to end

- **`db/migrations/000037_v11_account_interests.sql`**: `app_users.interests text[] NOT NULL DEFAULT '{}'` + `CHECK (cardinality(interests) <= 20)` as a second line of defense (the real validation lives in Go, matching how `slot.normalizeSelectedUserIDs` validates its own array input rather than pushing that into a CHECK subquery). No new `linkup_api` grant needed: `app_users` predates the v1.0 database-hardening migration (`000005`) and `linkup_api` already owns it as a table, so a new column is automatically covered by existing ownership — confirmed by grepping every other migration for a `GRANT ... app_users` line and finding none anywhere in the migration history.
- **`internal/account/model.go`**: `User.Interests []string` (never nil on read — defaults to an empty slice); `ProfilePatch.Interests *[]string` — nil means "leave unchanged", a non-nil empty slice means "clear all interests" (deliberately different from `slot.normalizeSelectedUserIDs`, where an empty allow-list is invalid — a user having no declared interests is a completely normal, valid state).
- **`internal/account/service.go`**: `normalizeInterests` — lowercases/trims each tag the same way `slot.Service` normalizes a Slot's own `Activity` field (the field it's matched against), rejects a blank tag, deduplicates, caps at `maxInterests` (20, mirroring the DB-side ceiling), sorted for a stable representation. Wired into `UpdateProfile`'s existing validation chain.
- **`internal/postgres/account_store.go`**: `interests` column added to every `User`-returning query (`FindByLogin`, `Authenticate`, `UpdateProfile`'s `RETURNING`) and to `UpdateProfile`'s `CASE WHEN` update pattern, following the exact style already used there for `display_name`/`avatar_url`/`profile_visibility`/`language`.
- **`internal/httpserver/auth_handlers.go`**: `patchMeRequest.Interests *[]string` wired into `PATCH /v1/me`, in the same dense one-line style already established in this file (confirmed via `gofmt -d` that this file was already not-gofmt-clean before this block, from an earlier session — not something introduced now; new edits kept consistent with it rather than reformatting the whole file).

New tests:
- `internal/account/service_test.go`: `TestUpdateProfileNormalizesInterests` (case/whitespace/dedupe, and that a non-nil empty slice clears), `TestUpdateProfileRejectsBlankInterestTag`, `TestUpdateProfileRejectsTooManyInterests`.
- `internal/httpserver/auth_handlers_test.go`: `TestPatchMeNormalizesAndPersistsInterests`, `TestPatchMeRejectsBlankInterestTag` — through the real HTTP handler, not just the service layer.
- `internal/postgres/v11_pulse_relevance_ranking_integration_test.go` (new): `TestV11PulseRelevanceSortRanksInterestMatchAheadOfRecency` — against real PostgreSQL: a viewer sets `interests=["board games"]`; a host creates an older Slot with `activity="board games"` and then a strictly newer Slot with `activity="chess"`. Under the default (`sort` omitted) the newer Slot ranks first, proving RECENCY is completely unaffected by interests. Under `sort=relevance` the older, interest-matching Slot ranks first instead. Also confirms `sort=ReLeVaNcE` (mixed case) is accepted, matching `slot.Service.ListPulse`'s existing case-insensitive parsing.

Executed evidence (same disposable-PostgreSQL-in-sandbox setup as §61-§64, started fresh this block: `pg_ctlcluster` was down at block start, started it, set a local password on the `postgres` role for TCP auth since peer auth doesn't apply to this container's OS user, then ran `cmd/migrate` against the same `linkup_test` database the last several blocks already used):

- `go build ./...`, `go vet ./...` — clean;
- `gofmt -l` on every new/touched file — clean, except `internal/account/store.go`, `internal/httpserver/auth_handlers.go`, and `internal/httpserver/slot_handlers_test.go`, all three already not-gofmt-clean before this block (confirmed via `git diff`/`gofmt -d` — `store.go` wasn't touched at all this block, the other two keep the file's existing dense style rather than reformatting around a one-line edit);
- migrations `000001..000037` applied cleanly (one new migration this block); `assertMigrationCount` floor bumped to `37` in the new test file, matching the pattern already used everywhere else (a `<` floor check, so it doesn't need bumping again by every future migration);
- the full `go test -count=1 ./...` suite run three consecutive times — all three runs show only the same 5 pre-existing quiet-hours failures already documented since §60, reproduced identically, nothing new. **Verified this time, not just asserted:** `git stash`ed this block's entire diff and re-ran the two `TestFriendStore.../TestNotificationProjectorApprovalAndRequestCreated` failures in isolation against the unmodified tree — they fail identically with zero changes from this block applied, confirming they are a pre-existing artifact of these particular tests using real `time.Now()` as the notification clock while this sandbox's real wall-clock time (05:xx UTC) falls inside the default quiet-hours window (23:00-08:00 UTC), not anything this block introduced;
- `go test -race -count=1 ./...` — same 5 known failures, no new ones, no race reports;
- `go mod tidy` — no diff.

Not done or claimed in this block: `Registration` has no `interests` field — a new user starts with none and sets them afterward via `PATCH /v1/me`, matching how every other optional profile field already works; README §6.10's Layer 3 (opt-in personalization from history) — still requires explicit consent infrastructure that doesn't exist, not attempted; no additional Layer 2 signals beyond activity/interest match (time-window proximity, freshness decay, City Context distance) — the doc comment on `listV11PulseRelevanceSQL` already named these as valid future Layer 2 signals not yet attempted, unchanged this block; no production deployment or verification of any kind (no Supabase/SSH access in this session); the pre-existing notification-projector quiet-hours test flakiness remains unfixed and out of scope (re-confirmed, not newly discovered, this block).

**README §6.10's Recommendation/ranking pipeline now has a working, tested Layer 2** for the first time: `GET /v1/pulse?sort=relevance` is a real, additive, opt-in ranking mode; the long-standing recency default is provably unaffected (the new integration test asserts both orderings against the same two fixture Slots in the same test run).

Next: admin campaigns (still flagged as worth a direct scoping conversation given its size, unanswered since §64); `EVENT_RECOMMENDATION` or README §6.9's "after-check flow" given a specification; `LASSO`/`TRAVEL_CORRIDOR` Map/realtime/replay parity; additional Layer 2 ranking signals; or the next groups of 5 from the user's enumerated backlog.

## 66. 2026-09-12 — LinkUp+ billing foundation: corrected pricing catalog, rewarded-quest grants, purchase verification, referral qualification/milestones (README §6.20/§6.27, docs/LINKUP_PLUS_MONETIZATION.md)

**User instruction driving this block:** "Продовжуй" after §65, under the session's own standing instruction ("продовжуй далі працювати, тестувати все буду коли ти повністю зробиш v1.1 до самого кінця" — keep working autonomously through the rest of v1.1). Rather than picking a shallow slice across many unrelated backlog items, this block closes one item in depth — item 25 of the "Немає" list ("Повний LinkUp+ billing") — because auditing it first surfaced a genuinely severe, previously undocumented gap (below), and the session's own established practice is to close a found real gap rather than skip past it.

### Audit findings, before any code was written

`internal/monetization` already existed (pre-dating this session — one commit in its history, the repo's initial import) with a `Service`/`Store`/postgres/httpserver wiring already in place, and `docs/LINKUP_PLUS_MONETIZATION.md` already documents its own known gaps under "10. Current repository implementation status." Read that document in full and cross-checked its claims against the actual code (not trusted at face value):

- **Confirmed accurate:** the Go catalog really was outdated (only `monthly`/`annual`, old prices `149.99`/`1199.88` UAH, no `WEEKLY`/`THREE_MONTH`, no feature-availability catalog) — exactly as that document's own §10 says.
- **A gap that document's own §10 does NOT mention, found by grepping every monetization table for its writers:** `premium_grants` had **zero writers anywhere in the codebase**. Referral binding (`PUT /v1/me/referral`) only ever wrote `monetization_referral_codes`/`monetization_referrals` — it never qualified a referral (`qualified_at` never set) or awarded a milestone. Nothing verified a rewarded-ad view or a purchase; `monetization_rewarded_receipts`/`monetization_subscription_receipts`/`monetization_referral_milestone_awards` were all write-only schema with no Go code ever inserting into them. **No user could ever actually receive LinkUp+ premium access through any path** — the entire reward-granting side of the feature was schema-only. This is a more severe gap than "the catalog has stale prices," and closing only the catalog would have left the paywall showing a completely non-functional product.

### Implemented

- **`db/migrations/000038_v11_monetization_rewarded_and_entitlement_states.sql`**: adds `monetization_rewarded_progress.quest_started_at` (needed to enforce the 24h quest window from the *first* verified view, separately from the 4h-between-steps rule, which the existing `last_video_watched_at` alone can't distinguish); extends `monetization_subscription_receipts.state`'s CHECK to add `GRACE`/`BILLING_RETRY` (README §6.20/docs §5's required entitlement states beyond the original `ACTIVE/EXPIRED/REVOKED/REFUNDED`), mapped to standard Google Play subscription semantics (GRACE still entitled, BILLING_RETRY/account-hold not) rather than an invented meaning. No new `linkup_api` grant needed (same reasoning as §65: ownership predates/matches every other migration in this history, which has never granted any monetization table to `linkup_api` either).
- **`internal/monetization/model.go`** (new, split out of the growing `service.go`): corrected `Plan`/`Catalog` shape — four plans (`WEEKLY`/`MONTHLY`/`THREE_MONTH`/`ANNUAL`) with prices, effective-monthly, and savings-vs-monthly computed to match docs §2's table exactly (verified by unit test against the literal table values, not re-derived from a differently-rounding formula); a `Benefit`/feature-availability catalog covering the six still-unbuilt LinkUp+ areas (Travel Pro, Host Power Tools, Advanced Discovery, Privacy&QoL, Identity/Analytics, Forgiveness) — every entry `Available:false`, confirmed by grepping the whole backend for each area's canonical terms (Mega-Slot, Stealth Slot, Ghost Mode, Guardian Auto-Ping, Hex-Aura, No-Strike) and finding nothing, so this is an honest "not yet" list, not a stub; `RewardedVerifier`/`PurchaseVerifier` pluggable-provider interfaces (same optional-dependency shape as `account.Service.ConfigureRecovery`) so the fail-closed boundary docs §6/§8 requires ("client `watched=true`/purchase-state is never authority") holds regardless of which ad network/store is eventually plugged in; `Store` interface gains `RecordRewardedView`/`RecordPurchase`.
- **`internal/monetization/service.go`**: `ConfigureRewarded`/`ConfigurePurchases` (capability flags flip on only once a real verifier is configured — confirmed by a test that configuring a purchase verifier, and only that, also flips `ReferralQualification` on, since qualification is derived from a real paid-purchase signal rather than needing its own separate wiring); `SubmitRewardedView`/`VerifyPurchase` — both re-verify with the configured provider before anything reaches the Store, hash the provider's own verified identity (never the raw receipt/token) for idempotency, and validate the verifier's response shape before trusting it (rejects empty provider/product/purchase IDs, an unknown state string, or `periodEnd <= periodStart`).
- **`internal/postgres/monetization_store.go`**: `RecordRewardedView` — idempotent by `(provider, receipt_hash)` (a replay is a silent no-op before touching progress at all); enforces the 4h/24h/cooldown rules with an explicit `FOR UPDATE` lock on the progress row (serializing concurrent submissions for the same user, matching this codebase's existing `CreateSession`/`ResetPassword` locking style rather than relying on `Serializable` isolation); a view arriving after the 24h window restarts the quest at count 1 instead of erroring (a deliberate, stated choice — the ad was genuinely watched, so it shouldn't be discarded); the 5th qualifying view atomically inserts a `REWARDED` `premium_grants` row and resets progress. `RecordPurchase` — upserts `monetization_subscription_receipts` (idempotent + updatable, so a later revoke/renewal of the *same* purchase token converges correctly); `ACTIVE`/`GRACE` grant or extend a `PAID` grant keyed to that exact purchase (so a duplicate verification of the same token never double-grants — confirmed by `premium_grants`'s own pre-existing `UNIQUE(source, source_key)`); any other state retracts that specific grant (never touching the audit trail in `monetization_subscription_receipts`, which is only ever updated, not deleted); on `ACTIVE`, `qualifyReferralAndAwardMilestones` qualifies a pending referral (if any, and only once — the `WHERE qualified_at IS NULL` guard) and awards every newly-crossed milestone exactly once (the milestone table's own `(inviter_id, milestone)` primary key is the idempotency guard, including under a race between two invitees of the same inviter — documented in code as a stated, acceptable trade-off: correctness never depends on which one wins that race, only fairness-of-attribution might).
- **`internal/httpserver`**: new `POST /v1/me/monetization/rewarded-views` and `POST /v1/me/monetization/purchases` handlers, both re-verifying via the Service (never trusting the request body's claims — there are none to trust, the body is just an opaque receipt/token); a new `MonetizationLimiter` (`internal/config`'s new `LINKUP_MONETIZATION_RATE_*` settings, default 20/hour/user) applied to referral-bind and both new endpoints, independent of the existing general `UserLimiter` — closing docs §8.2's explicit "entitlement-sensitive endpoints need a separate abuse budget" requirement, which nothing enforced before this block.

New tests:
- `internal/monetization/service_test.go`: rewritten `TestCatalogMatchesProductContract` against the corrected 4-plan/benefits shape; new fail-closed/verification/hashing tests for `SubmitRewardedView`/`VerifyPurchase` (`TestSubmitRewardedViewFailsClosedWithoutVerifier`, `...RejectsUnverifiedReceipt`, `...RecordsVerifiedProviderAndHashesReceipt`, `...PropagatesStoreErrors`, `TestVerifyPurchaseFailsClosedWithoutVerifier`, `...RejectsMalformedVerifiedResult`, `...RecordsNormalizedStateAndEnablesReferralQualification`).
- `internal/httpserver/monetization_handlers_test.go` (new file — this handler had no tests at all before this block): capability-disabled/success/error-mapping/invalid-JSON/auth-required cases for both new endpoints plus the existing `getMonetization`.
- `internal/postgres/monetization_store_integration_test.go` (new): `TestMonetizationStoreRewardedQuestGrantsPremiumAfterFiveVerifiedViews` (too-soon rejection, correctly-spaced 5-view grant, replay-safety, cooldown rejection, all against real PostgreSQL with hand-controlled timestamps passed directly to the Store — no need for a settable clock since the Store's own signature already takes `now` explicitly); `TestMonetizationStoreRewardedQuestWindowExpiryRestartsRatherThanErrors`; `TestMonetizationStorePurchaseGrantsEntitlementAndQualifiesReferral` (ACTIVE purchase grants PAID premium, qualifies a pending referral, awards milestone 1 to both inviter and triggering invitee exactly once, replay-safe, and a later REVOKED verification retracts only the PAID grant while leaving the referral-milestone grant untouched).

Executed evidence (same disposable-PostgreSQL-in-sandbox setup as §61-§65; the sandbox's Postgres cluster and local `postgres`-role password had to be brought back up again this block after a fresh container):

- `go build ./...`, `go vet ./...` — clean;
- `gofmt -l` on every new/touched file — clean, except `internal/account/store.go`/`internal/httpserver/auth_handlers.go`/`internal/httpserver/slot_handlers_test.go` (untouched this block, already known not-gofmt-clean) and `cmd/api/main.go` (touched, but its diff stays within that file's own pre-existing dense one-line style, confirmed by comparing against the pre-edit version via `git stash`);
- migration `000038` applied cleanly (`assertMigrationCount` floor bumped to 38 in the new rewarded-quest test, matching the established `<` floor-check pattern elsewhere so it never needs bumping again by a future migration);
- the full `go test -count=1 ./...` suite run three consecutive times — all three runs show only the same 5 pre-existing quiet-hours failures already documented since §60 (re-confirmed via `git stash` against the unmodified tree in §65, not re-litigated here), nothing new; every new monetization test passed on every run;
- `go test -race -count=1 ./...` — same 5 known failures, no new ones, no race reports;
- `go mod tidy` — no diff.

Not done or claimed in this block: **no real ad-network or store adapter is plugged in** — `RewardedVerifier`/`PurchaseVerifier` are real, tested interfaces with real store-side logic behind them, but `cmd/api/main.go` never calls `ConfigureRewarded`/`ConfigurePurchases` (there is no AdMob/Google Play/App Store credential or SDK access in this session, and inventing a fake "verifier" that always succeeds would defeat the entire fail-closed design this block just built) — so in the actual running server today, `SubmitRewardedView`/`VerifyPurchase` still return `ErrCapabilityDisabled` exactly as before, honestly; wiring a real adapter is real future work, not a loose end silently left in this block's own code. Android/iOS client updates (§6.20's own "Required synchronization" list: plan-duration parsing, launch prices, benefits section, purchase/restore UI) are not attempted — iOS is frozen per README §0.2 and out of scope regardless; Android has real existing LinkUp+ UI/client code (`MonetizationApiClient.kt`, `LinkUpPlusScreen.kt`) that would need updating for the new 4-plan/benefits shape, named here as a real, still-open gap rather than silently left for someone else to notice. `GRACE`/`BILLING_RETRY` entitlement states are schema-ready and the Store's `RecordPurchase` branches correctly on them, but nothing exercises them from a real provider yet (no adapter is configured) — the postgres integration test only exercises `ACTIVE`/`REVOKED` directly, not through a live subscription lifecycle. The manual-entitlement audit log (docs §8.6) and Layer-3-style anomaly/anti-fraud detection (docs §8.3/§8.4) are real, still-unbuilt pieces of this same section, named honestly rather than attempted partially. Admin campaigns remain unstarted (still flagged since §64/§65 as worth a scoping conversation, not raised again as a question this block per the standing "keep working" instruction — proceeding autonomously instead).

Next: a real ad-network/store adapter (blocked on credentials this session doesn't have); Android LinkUp+ client updates for the new plan/benefits shape; the manual-entitlement audit log and anti-fraud/anomaly detection (docs §8.3/§8.4/§8.6); or the next backlog items (`EVENT_RECOMMENDATION`/README §6.9 still blocked on an unspecified design; `Auto-Swarms`/`Fly`/`Me 2.0`/`AR` are large, genuinely under-specified product surfaces — bullet lists of required capabilities, not field-level specs — that will need either a scoping/design pass or narrowing to a concretely-specifiable slice before real code, the same reasoning already applied to `EVENT_RECOMMENDATION`).

## 67. 2026-09-12 — Bill Splitter calculator (README §6.18)

**Context:** continuing autonomously after §66 per the session's standing "keep working through v1.1" instruction. Scanned §6.11-§6.19 (Auto-Swarms, Fly Now/Travel/Motion, Me 2.0/Squad Radar, AR/Ranking V2, Cold start/venue ecosystem, Offline real-world operations, Safety/accessibility expansion, Ephemeral media, Adaptive experience) for a concretely-specifiable slice before writing any code — most of that surface is a bullet list of required capabilities (clustering, activity recognition, AR anchors, venue fraud detection, BLE mesh, guardian links) with no field-level spec, the same kind of gap that has repeatedly made this session defer `EVENT_RECOMMENDATION`/README §6.9 rather than invent behavior. Bill Splitter (§6.18) stood out as the one fully self-contained, unambiguous item in that whole range: "calculator with deterministic minor-unit rounding," explicitly "not a payment processor" — a pure function plus an authorization boundary this codebase already has, nothing to invent.

### Implemented

- **`internal/chat`** (grouped with chat rather than a new top-level package, matching README §6.18's own framing of Bill Splitter as one of several in-Slot coordination tools alongside chat/media): `BillSplit`/`ParticipantShare` types; `Store.SplitBillRoster` (authorizes actorID with the *exact* boundary `Send`/`ListRecent` already use — host or a currently-accepted member, block-excluded — then returns the current roster); `Service.SplitBill(ctx, actorID, slotID, totalMinor, participantIDs)` — defaults to the whole roster when `participantIDs` is empty, rejects any requested participant not actually on the roster, rejects fewer than 2 (deduplicated) participants, and computes an even split via the largest-remainder method: `total/n` as each share's base, the `total%n` leftover minor units going one each to the alphabetically-first *n* participant IDs (participants are always sorted before splitting) — a stated, arbitrary-but-fully-deterministic tie-break, not left implicit.
- **`internal/postgres/chat_store.go`**: `SplitBillRoster` — reuses `authorizeChatTx` unchanged, then queries host ∪ `slot_memberships` for that Slot, block-excluded from the actor's own view (identical filter to `ListRecent`'s).
- **`internal/httpserver`**: new `POST /v1/slots/{slotID}/bill-split` — stateless: returns the computed split, never persists it or touches any payment/ledger table (the guardrail this section names explicitly).

New tests:
- `internal/chat/service_test.go`: deterministic-rounding + sum-equals-total, rejects a non-roster participant, rejects <2 participants (including a deduplicated single ID), rejects invalid/out-of-bounds totals, propagates authorization errors from the Store.
- `internal/httpserver/chat_handlers_test.go`: HTTP-level success + invalid-total rejection.
- `internal/postgres/v11_bill_split_integration_test.go` (new): against real PostgreSQL — a stranger is forbidden entirely; host and an accepted member can both split; a non-roster participant listed explicitly is rejected; a member who left drops out of the default roster; a block from the host still wins over being an otherwise-accepted member (mirroring chat's own existing invariant, checked directly via `SplitBillRoster` rather than through `SplitBill`'s own ≥2-participant rule, since blocking-plus-an-earlier-leave in the same test can legitimately leave only one person on the roster).

Executed evidence (same disposable-PostgreSQL-in-sandbox setup as §61-§66):

- `go build ./...`, `go vet ./...`, `gofmt -l` on every touched file — all clean, no exceptions this block (nothing here touched any of the session's known pre-existing not-gofmt-clean files);
- the full `go test -count=1 ./...` suite run three consecutive times — only the same 5 pre-existing quiet-hours failures already documented since §60, nothing new; every new Bill Splitter test passed on every run;
- `go test -race -count=1 ./...` — same 5 known failures, no new ones, no race reports;
- `go mod tidy` — no diff (no new migration or dependency this block).

Not done or claimed in this block: no persistence — a computed split is not saved anywhere or posted as a chat message automatically (the caller can already do that themselves via the existing `POST .../chat/messages` with the returned numbers; a dedicated `BILL_SPLIT` SYSTEM message type, matching the existing `SLOT_STARTED`/`MEMBER_JOINED`/`MEMBER_LEFT` pattern, is real, disclosed future work, not attempted here since it needs its own schema/rendering-contract decision); no custom per-participant weights/shares (doc only specifies an even split); every other §6.11-§6.19 item remains unstarted, named honestly rather than attempted partially, for the reason given above (bullet-list-only specs needing either a scoping pass or a narrower concretely-specifiable slice, the same standard already applied to `EVENT_RECOMMENDATION`).

Next: a real ad-network/store adapter for §66's monetization work (blocked on credentials); Android LinkUp+ client updates; a narrower concretely-specifiable slice of one of `Auto-Swarms`/`Fly`/`Me 2.0`/`Squad Radar`/`AR` (each needs a scoping pass first — bullet lists, not field-level specs); or the next backlog items.
