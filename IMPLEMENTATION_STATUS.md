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
