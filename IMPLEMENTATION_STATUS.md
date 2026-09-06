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

### Exists

- real `.gitignore` excluding `.env`, signing keys/keystores, APK/AAB, build outputs, logs/temp/IDE files;
- Go module/API process foundation;
- `/livez` and DB-aware `/healthz`;
- HTTP timeouts + graceful shutdown;
- request IDs and structured method/path/status/latency logging without bearer headers/request bodies;
- environment config;
- forward-only migration runner with `linkup_schema_migrations` checksum ledger and checksum-drift rejection;
- Android Kotlin/Compose project shell;
- frozen design colors transferred 1:1 to Kotlin.

### Relevant commits

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

Commit `5f430737fac07e84aea765e113d2ed5e9b0f1937`:

- bounded fixed-window auth rate limiter;
- direct peer IP + route key;
- `429` + `Retry-After`;
- bounded memory with idle pruning;
- configurable limit/window/idle TTL/max entries.

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

Base block API added in `6f11f02b3f1f742851e38f76feff97db69d6f861`:

- `GET /v1/me/blocks`;
- `PUT /v1/me/blocks/{userID}`;
- `DELETE /v1/me/blocks/{userID}`;
- bearer authorization;
- self-block rejected;
- duplicate block idempotent at DB level.

Approval hardening in `054b223c65bb5e98147d7eb227b88b426553c32b` makes Block transactional with social state: pending requests between the pair are deleted, accepted memberships between host/member are removed, and affected Slot `accepted_count/state/version` are corrected atomically.

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

`db/migrations/000002_slots.sql` defines:

- canonical `slots` table;
- lifecycle `DRAFT / PUBLISHED / FILLING / FULL / ACTIVE / COMPLETED / CANCELLED / EXPIRED / MODERATED`;
- access modes `INSTANT / APPROVAL / WAITLIST`;
- visibility foundation;
- capacity + `accepted_count` invariant;
- server version;
- host/public Pulse indexes;
- generic `mutation_idempotency` table with request SHA-256 and finite TTL/index.

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

Created `db/migrations/000003_approval.sql`:

- `slot_requests` with unique `(slot_id,user_id)` pending-request invariant;
- `slot_memberships` with unique `(slot_id,user_id)` accepted-membership invariant;
- user/request and user/membership lookup indexes.

### Server behavior

Added canonical viewer relationship states:

- `NONE`;
- `PENDING`;
- `ACCEPTED`;
- `HOST`.

Added endpoints:

- `POST /v1/slots/{slotID}/request`;
- `POST /v1/slots/{slotID}/leave`;
- `GET /v1/slots/{slotID}/requests`;
- `POST /v1/slots/{slotID}/requests/{userID}/approve`;
- `POST /v1/slots/{slotID}/requests/{userID}/reject`;
- `POST /v1/slots/{slotID}/start`;
- `POST /v1/slots/{slotID}/complete`.

Implemented invariants:

- APPROVAL request creates pending relation only, never accepted membership;
- duplicate pending request rejected;
- accepted user cannot request again;
- host cannot request own Slot;
- bidirectional block check before request/approval;
- host-only pending-request list with requester `id/@username/displayName/avatar`;
- host-only approve/reject;
- APPROVE executes under Slot row `FOR UPDATE`, so concurrent last-seat approvals serialize and `accepted_count` cannot exceed capacity;
- APPROVE atomically inserts membership, removes pending request, updates `accepted_count/state/version`;
- last seat moves Slot to `FULL`;
- pending LEAVE withdraws request;
- accepted LEAVE deletes membership, decrements count and reopens `FULL → FILLING`;
- START is host-only, requires at least one accepted participant, clears remaining pending requests and moves Slot to `ACTIVE`;
- COMPLETE is host-only and requires `ACTIVE`;
- create/request/approve/reject/leave/start/complete/cancel all use the generic idempotency boundary;
- Pulse/Get include viewer relationship for Android action rendering;
- active/terminal Slot is not normal public discovery, while host/accepted relationship access can still resolve the resource where required;
- Block immediately revokes pending/accepted social relationship transactionally.

Source tests now cover PUBLIC+APPROVAL create, pending state, withdrawal, approve-to-FULL, full-capacity rejection, accepted LEAVE reopen, START/COMPLETE lifecycle and existing Slot HTTP flow/interface compatibility.

## 8. Verification state

### Actually verified in the available local environment

- earlier pure-Go liveness/session-token tests passed before external dependencies were introduced;
- standalone standard-library rate-limiter scratch test passed;
- standard-only block/recovery code was syntax/parse checked during development.

### NOT yet honestly verified

The current local execution environment cannot resolve external hosts and does not have required external Go/Android dependencies cached. Therefore these gates remain open:

- `go mod tidy` and generated/verified `backend/go.sum`;
- full `go test ./...` after `pgx` + `x/crypto` + Slot/Approval code;
- PostgreSQL integration tests/race execution;
- migration execution against disposable PostgreSQL;
- applying migrations to canonical Supabase (not requested/deployed yet);
- Android Gradle compile/instrumentation;
- real Android ↔ Go ↔ PostgreSQL smoke;
- release signing/build.

Do **not** mark these green without real evidence.

## 9. What still does NOT exist — active Android/Go Version 1

### Account/UI remaining

- breached/common-password blocklist integration;
- multi-session management UI;
- actual Compose Login/Register/Recovery surfaces in frozen visual language;
- app navigation wired to `SessionCoordinator`;
- production SMTP credentials/provider smoke;
- real DB migration/account smoke.

### Foundation social loop remaining

- accepted-only ephemeral chat;
- bounded recent thread read/send API;
- pending/stranger/left authorization rejection;
- terminal physical chat purge for `COMPLETED/CANCELLED/EXPIRED/MODERATED`;
- Android production Pulse/LINK/host/request/chat binding;
- real two-user end-to-end smoke.

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

## 10. README historical claims

README statements saying Android Event Core, Approval, chat, BUMP or historical migrations `000012/000013/000016/000018/000020/000022` were already implemented are **not current repository evidence**. Only code/migrations physically present here and recorded in this ledger count.

## 11. Worklog

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
- source tests updated for expanded Store contract.
- Commit: `054b223c65bb5e98147d7eb227b88b426553c32b`.

## 12. Current production readiness

**Android + Go Version 1 production readiness: 0%.**

Reason: the server-side account + Slot + Approval foundation now exists, but external dependency/build/DB/device verification remains open and the mandatory foundation social loop still lacks accepted-only ephemeral chat and real Android production binding. Documentation/scaffolding/design do not count toward readiness.

## 13. Next exact work block

**Basic Zero-Trace Coordination Chat:**

1. create forward-only chat migration;
2. add bounded ephemeral `slot_messages` schema;
3. add PostgreSQL trigger that physically deletes chat rows when Slot becomes `COMPLETED`, `CANCELLED`, `EXPIRED` or `MODERATED`;
4. implement accepted-only/host read+send authorization;
5. deny pending/stranger/left users even if they know `slotID`;
6. bound message length and recent-thread response;
7. server-create message timestamp/author identity;
8. add block-aware chat authorization/revocation;
9. add service/HTTP authorization tests;
10. then start Android production binding to the frozen design without redesign.
