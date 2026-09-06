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

## 3. Approved frozen design reference — ✅ design / 🟡 functionality

Existing files include `App.tsx`, `PulseScreen.tsx`, `MapScreen.tsx`, `CreateLinkScreen.tsx`, `FlyScreen.tsx`, `MeScreen.tsx`, `NotificationsPanel.tsx`, `SlotCard.tsx`, `BottomNav.tsx`, shared UI primitives and `tailwind.config.js`.

Represented surfaces include Pulse, Map, LINK/Create, Fly, Me, Notifications, Slot cards/details, Reliability/BUMP/Passport/settings concepts.

**These files are not production-functionality evidence. They use local demo/prototype state and remain untouched as the visual contract.**

## 4. Repository/build/security foundation — 🟠

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

## 5. Account/session/security foundation — 🟠

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

Commit `5f430737fac07e84aea765e113d2ed5e9b0f1937`: bounded fixed-window auth limiter, direct peer IP + route key, `429` + `Retry-After`, bounded memory and configurable limits.

### Password recovery/reset

Commits:

- domain/HTTP/reset transaction: `28307b7b87e3c813a0e38aca962ad668ef9c4cdc`;
- TLS-only SMTP adapter: `34362616f1c197b3bbe6b1dcd22a5ab4ff3be72d`;
- SMTP env contract: `07c8a8e2535ec863f16c29297ade47883af75878`.

Endpoints:

- `POST /v1/auth/recovery/request`;
- `POST /v1/auth/recovery/reset`.

Security behavior includes one-time hashed reset tokens, expiry, previous-token invalidation, all-session revocation after password reset, TLS-only SMTP, no plaintext fallback, URL-encoded token and fail-closed behavior when delivery is not configured.

### Server-authoritative block controls

Base block API in `6f11f02b3f1f742851e38f76feff97db69d6f861`:

- `GET /v1/me/blocks`;
- `PUT /v1/me/blocks/{userID}`;
- `DELETE /v1/me/blocks/{userID}`;
- bearer authorization;
- self-block rejected;
- duplicate block idempotent at DB level.

Approval hardening in `054b223c65bb5e98147d7eb227b88b426553c32b` makes Block transactional with social state: pending host/requester relation is removed, accepted host/member relation is revoked, and affected Slot `accepted_count/state/version` are corrected atomically.

### Android account/session client

Implemented:

- Android Keystore AES-256-GCM encrypted bearer persistence;
- local expiry/decryption/key-loss clearing;
- register/login/logout/Me/recovery API calls;
- HTTPS requirement outside emulator/loopback;
- process-death bootstrap with `Checking`, `SignedOut`, `SignedIn`, `OfflineSession`, `RecoverableError`;
- 401 clears revoked local session while temporary network loss preserves a still-valid bearer.

Relevant commits: `8c852bf446943b8e83c4954d007b6eebaa72d080`, `2bea65b4c19e5125270adf9fd769e741ec3f8989`.

## 6. Canonical Slot foundation — 🟠

Implemented in commit `2a1fb50728dd47a35f118a1f42d39d70428aca11`.

### Database

`db/migrations/000002_slots.sql` defines canonical `slots`, full lifecycle states, access modes `INSTANT / APPROVAL / WAITLIST`, visibility foundation, capacity + `accepted_count`, server version, Pulse indexes and generic `mutation_idempotency` with finite TTL.

Current foundation create surface creates **PUBLIC + APPROVAL + FILLING** Slots. Instant/Waitlist remain in canonical domain/data for later capability blocks.

### Go Slot API

Endpoints:

- `POST /v1/slots`;
- `GET /v1/slots/{slotID}`;
- `PATCH /v1/slots/{slotID}`;
- `POST /v1/slots/{slotID}/cancel`;
- `GET /v1/pulse`.

Implemented: mandatory mutation `Idempotency-Key`, SHA-256 request fingerprint, finite idempotency retention, server UUID/version/state, host-only edit/cancel, optimistic `expectedVersion`, capacity edit invariant, FILLING/FULL normalization, CANCEL state transition, block-aware Pulse/Get and explicit HTTP error mapping.

**Current foundation idempotency is effect-idempotent and returns the current canonical resource on replay. Exact historical response replay remains part of durable-offline/realtime hardening.**

## 7. Approval social loop — 🟠

Implemented in commit `054b223c65bb5e98147d7eb227b88b426553c32b`.

### Database

`db/migrations/000003_approval.sql` adds:

- `slot_requests` with unique `(slot_id,user_id)` pending invariant;
- `slot_memberships` with unique `(slot_id,user_id)` accepted invariant;
- lookup indexes.

### Server behavior

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
- all critical mutations use idempotency boundary;
- Pulse/Get expose viewer relationship;
- Block revokes host/member/requester relationship transactionally.

Source tests cover pending/withdrawal, approve-to-FULL, full rejection, LEAVE reopen, START/COMPLETE and Store/HTTP compatibility.

## 8. Basic Zero-Trace Coordination Chat — 🟠

Implemented in commit `b9ea7481c78f8f9a61ae3b8cb06cc6449a6feb9b`.

### Database

Created `db/migrations/000004_chat.sql`:

- `slot_messages` with UUID message id, Slot FK, author FK, text, server timestamp;
- DB `CHECK` keeps trimmed body length between 1 and 2000 chars;
- recent-thread index `(slot_id, created_at DESC, id DESC)`;
- PostgreSQL trigger `slots_terminal_message_purge` physically deletes all Slot message rows whenever state transitions to `COMPLETED`, `CANCELLED`, `EXPIRED` or `MODERATED`.

### Go domain/API

Created:

- `backend/internal/chat/model.go`;
- `backend/internal/chat/service.go`;
- `backend/internal/chat/service_test.go`;
- `backend/internal/postgres/chat_store.go`;
- `backend/internal/httpserver/chat_handlers.go`;
- `backend/internal/httpserver/chat_handlers_test.go`.

Endpoints:

- `GET /v1/slots/{slotID}/chat/messages`;
- `POST /v1/slots/{slotID}/chat/messages`.

Implemented behavior:

- max message size: 2000 Unicode code points;
- max recent response: 100 messages;
- server generates message UUID and timestamp comes from PostgreSQL;
- author response includes id, username, display name and avatar;
- read returns a bounded recent slice in chronological order;
- authorization is re-evaluated server-side for every read/send;
- host or current accepted participant only;
- pending requester, stranger/non-member and user after LEAVE are denied even with a known `slotId`;
- terminal/non-chat Slot states return explicit closed state;
- accepted participant authorization checks host/member block relation;
- recent thread filters messages authored by identities with a block relation to the current viewer in either direction;
- host/member LEAVE/block revocation therefore removes future chat authorization;
- terminal state transition physically purges ephemeral rows in canonical PostgreSQL, not merely hides them;
- no realtime/offline-send semantics are falsely claimed for this basic chat block.

Source tests cover service text/limit bounds and HTTP success/forbidden/closed/error mapping. PostgreSQL authorization/purge still requires integration execution before it can be marked verified.

## 9. Verification state

### Actually verified in the available local environment

- earlier pure-Go liveness/session-token tests passed before external dependencies were introduced;
- standalone standard-library rate-limiter scratch test passed;
- standard-only block/recovery code was syntax/parse checked during development.

### NOT yet honestly verified

The current local execution environment cannot resolve external hosts and does not have required external Go/Android dependencies cached. Therefore these gates remain open:

- `go mod tidy` and generated/verified `backend/go.sum`;
- full `go test ./...` after `pgx` + `x/crypto` + Slot/Approval/Chat code;
- PostgreSQL integration tests/race execution;
- migration execution against disposable PostgreSQL, including terminal chat purge trigger;
- applying migrations to canonical Supabase (not requested/deployed yet);
- Android Gradle compile/instrumentation;
- real Android ↔ Go ↔ PostgreSQL two-user smoke;
- release signing/build.

Do **not** mark these green without real evidence.

## 10. What still does NOT exist — active Android/Go Version 1

### Account/UI remaining

- breached/common-password blocklist integration;
- multi-session management UI;
- actual Compose Login/Register/Recovery surfaces in frozen visual language;
- app navigation wired to `SessionCoordinator`;
- production SMTP credentials/provider smoke;
- real DB migration/account smoke.

### Foundation social loop remaining

The server-side foundation path now exists for account → Slot → REQUEST → APPROVE/REJECT → membership → LEAVE → START/COMPLETE/CANCEL → basic chat.

Still missing before this can count as a working Android social network:

- Android production models/API binding for Pulse/LINK/Slot/Approval/Chat;
- Android Compose surfaces wired to those real states without redesign;
- loading/content/empty/error states in production Android flow;
- real two-user Android ↔ Go ↔ PostgreSQL smoke;
- actual build/database/device verification.

### Later Version 1 blocks

- durable Android mutation outbox + transactional backend outbox;
- realtime snapshot/ordered deltas/reconnect;
- City Context/PostGIS locality;
- real Map/viewport/Places integration;
- Waitlist/host control V2;
- Chat V2/realtime/system messages;
- push notifications;
- BUMP/Reliability, City BPM, Vibe/Lasso/Hotspots, swarms;
- Fly production functionality;
- Me 2.0/Squad/Guardian/Ghost production functionality;
- AR/ranking;
- venue/BLE/media/adaptive systems;
- LinkUp+ billing/travel/host/discovery/privacy/identity/rewarded access;
- full ecosystem hardening.

### iOS — ⛔

All iOS work remains intentionally frozen and excluded from current Android/Go readiness.

## 11. README historical claims

README statements saying Android Event Core, Approval, chat, BUMP or historical migrations `000012/000013/000016/000018/000020/000022` were already implemented are **not current repository evidence**. Only code/migrations physically present here and recorded in this ledger count.

## 12. Worklog

### 2026-09-06 — Design/platform contract finalized

- design frozen;
- Android active;
- Kotlin + Go mandated;
- iOS frozen.
- Commit: `ce12cbf075f2b911ac1c73577efffb79c20a0a55`.

### 2026-09-06 — Repository/Android/Go/DB foundation

- `.gitignore`, Go API health foundation, Android project/design tokens, accounts migration, bearer primitive, migration runner, initial ledger.

### 2026-09-06 — Account/session application foundation

- config/UUID, Argon2id, pgx account store, register/login/logout/Me API, Android secure session/API client.
- Key commits: `3031c359388383b40d1169c045e10854209b148e`, `e72c3ceb9ea694638a55895a8293a475f86993ca`, `7e62c4779745c92ff04f10e52eca338619d8a951`, `8c852bf446943b8e83c4954d007b6eebaa72d080`.

### 2026-09-06 — Account/security hardening

- auth rate limit: `5f430737fac07e84aea765e113d2ed5e9b0f1937`;
- password reset domain/HTTP: `28307b7b87e3c813a0e38aca962ad668ef9c4cdc`;
- Android bootstrap/recovery: `2bea65b4c19e5125270adf9fd769e741ec3f8989`;
- TLS-only SMTP: `34362616f1c197b3bbe6b1dcd22a5ab4ff3be72d`;
- block controls: `6f11f02b3f1f742851e38f76feff97db69d6f861`.

### 2026-09-06 — Canonical Slot foundation

- migration `000002_slots.sql`;
- Slot domain/service/pgx store;
- create/read/Pulse/edit/cancel API;
- optimistic version + idempotency + block-aware discovery;
- source tests.
- Commit: `2a1fb50728dd47a35f118a1f42d39d70428aca11`.

### 2026-09-06 — Approval social loop

- migration `000003_approval.sql`;
- REQUEST / withdraw;
- pending requester list;
- APPROVE / REJECT;
- atomic capacity/last seat;
- accepted membership + LEAVE/reopen;
- START / COMPLETE;
- viewer relationship states;
- block relationship revocation integrated with Slot state;
- source tests.
- Commit: `054b223c65bb5e98147d7eb227b88b426553c32b`.

### 2026-09-06 — Basic Zero-Trace Chat

- migration `000004_chat.sql`;
- accepted/host-only bounded chat read/send;
- PostgreSQL-authored timestamp;
- blocked-identity thread filtering;
- terminal physical purge trigger;
- chat HTTP/service tests in source.
- Commit: `b9ea7481c78f8f9a61ae3b8cb06cc6449a6feb9b`.

## 13. Current production readiness

**Android + Go Version 1 production readiness: 0%.**

Reason: the server-side mandatory foundation social loop is now represented in source, but external dependency/build/DB verification remains open and Android still does not expose the real flow. Until Kotlin production binding and a real two-user Android/Go/PostgreSQL smoke exist, this is not a production-ready social network.

## 14. Next exact work block

**Android production social-flow binding without redesign:**

1. add Kotlin canonical Slot/Viewer/PendingRequest/Chat models matching Go JSON;
2. extend `LinkUpApiClient` with Pulse/create/get/edit/cancel/request/leave/pending/approve/reject/start/complete/chat methods;
3. generate strong random `Idempotency-Key` values for every critical mutation;
4. parse server state/version/viewer relationship without client authority;
5. add repository/coordinator state for Pulse/Slot detail/request/host controls/chat;
6. preserve existing frozen React/TS visual contract when adding Compose surfaces;
7. implement loading/content/empty/error and manual refresh baseline;
8. wire session bootstrap into Android app navigation;
9. then run build/device/API verification when an environment with dependencies/database is available.
