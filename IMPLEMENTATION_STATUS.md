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

`db/migrations/000001_accounts.sql` defines:

- `app_users`;
- case-insensitive unique email/username indexes;
- `user_sessions` with hashed opaque-token boundary;
- `password_reset_tokens`;
- `user_blocks`;
- PUBLIC/HIDDEN profile visibility;
- uk/en language baseline;
- FK/check/index constraints.

### Go account/session API

Implemented:

- `POST /v1/auth/register`;
- `POST /v1/auth/login`;
- `POST /v1/auth/logout`;
- `GET /v1/me`;
- `PATCH /v1/me`;
- Argon2id password hashing with encoded parameters and configurable production calibration floor;
- cryptographically random 256-bit opaque bearer tokens; only SHA-256 bearer digest is persisted server-side;
- session expiry/revocation checks;
- normalized account identifiers and generic wrong-credential response;
- bounded JSON request bodies + unknown-field rejection;
- server-side current-session logout.

### Auth abuse protection

Implemented in commit `5f430737fac07e84aea765e113d2ed5e9b0f1937`:

- bounded fixed-window auth rate limiter;
- key = direct peer IP + auth route;
- `429` + `Retry-After`;
- bounded map with idle pruning;
- fail-closed behavior when bounded storage is saturated;
- configurable limit/window/idle TTL/max entries.

Config documentation: `a440536eee634220ade555835b3bca875052d5f1`.

### Password recovery/reset

Implemented in commits:

- domain/HTTP/reset transaction: `28307b7b87e3c813a0e38aca962ad668ef9c4cdc`;
- TLS-only SMTP adapter: `34362616f1c197b3bbe6b1dcd22a5ab4ff3be72d`;
- SMTP env contract: `07c8a8e2535ec863f16c29297ade47883af75878`.

Endpoints:

- `POST /v1/auth/recovery/request`;
- `POST /v1/auth/recovery/reset`.

Security behavior:

- opaque one-time reset tokens, hash persisted only;
- finite expiry;
- newer reset request invalidates previous unused reset token;
- unknown account does not create a different successful-domain response;
- successful reset changes password and revokes all active sessions transactionally;
- TLS required for SMTP: STARTTLS or implicit TLS, minimum TLS 1.2;
- no plaintext SMTP fallback;
- reset link token is URL-encoded;
- header injection rejected;
- recovery fails closed when delivery is not configured.

### Server-authoritative block controls

Implemented in commit `6f11f02b3f1f742851e38f76feff97db69d6f861`:

- `GET /v1/me/blocks`;
- `PUT /v1/me/blocks/{userID}`;
- `DELETE /v1/me/blocks/{userID}`;
- bearer authorization;
- self-block rejected;
- duplicate block idempotent at DB level;
- block-list summaries returned from server;
- block relationship is ready to be enforced by subsequent social queries.

### Android account/session client

Implemented:

- Keystore-backed AES-256-GCM encrypted bearer persistence;
- local expiry/decryption/key-loss clearing;
- register/login/logout/Me API client;
- HTTPS requirement outside emulator/loopback development;
- recovery/reset API calls;
- process-death/session bootstrap coordinator with `Checking`, `SignedOut`, `SignedIn`, `OfflineSession`, `RecoverableError` states;
- 401 clears revoked local session; temporary network loss preserves still-valid local bearer.

Relevant commits:

- secure session/API client: `8c852bf446943b8e83c4954d007b6eebaa72d080`;
- session bootstrap/recovery client: `2bea65b4c19e5125270adf9fd769e741ec3f8989`.

## 6. Canonical Slot foundation — 🟠

Implemented in commit `2a1fb50728dd47a35f118a1f42d39d70428aca11`.

### Database

Created `db/migrations/000002_slots.sql`:

- canonical `slots` table;
- lifecycle states `DRAFT / PUBLISHED / FILLING / FULL / ACTIVE / COMPLETED / CANCELLED / EXPIRED / MODERATED`;
- access modes `INSTANT / APPROVAL / WAITLIST`;
- visibility enum foundation;
- capacity + `accepted_count` invariant;
- server `version`;
- host/public Pulse indexes;
- generic `mutation_idempotency` table with request SHA-256 and finite TTL/index.

The current foundation create surface creates **PUBLIC + APPROVAL + FILLING** Slots as required by the README foundation flow. Instant/Waitlist stay in the canonical data model for later capability blocks.

### Go Slot domain/API

Created:

- `backend/internal/slot/model.go`;
- `backend/internal/slot/service.go`;
- `backend/internal/slot/service_test.go`;
- `backend/internal/postgres/slot_store.go`;
- `backend/internal/httpserver/slot_handlers.go`;
- `backend/internal/httpserver/slot_handlers_test.go`.

Endpoints:

- `POST /v1/slots`;
- `GET /v1/slots/{slotID}`;
- `PATCH /v1/slots/{slotID}`;
- `POST /v1/slots/{slotID}/cancel`;
- `GET /v1/pulse`.

Implemented behavior:

- authenticated host create;
- mandatory mutation `Idempotency-Key` (printable 16–128 chars);
- canonical SHA-256 request fingerprint;
- idempotency conflict if a key is reused for a different operation/request;
- configurable idempotency retention (`LINKUP_IDEMPOTENCY_TTL`, default 24h);
- real server-generated Slot UUID/version/state;
- host-only edit/cancel authorization;
- optimistic `expectedVersion` conflict handling;
- edit capacity cannot fall below current `accepted_count`;
- `FILLING ↔ FULL` normalization when capacity changes;
- terminal-state cancellation rejection;
- CANCEL is a state transition, not hard delete;
- PUBLIC Pulse excludes cancelled/terminal/non-public Slots;
- Pulse/Get apply block relationship filter in **both directions**;
- explicit HTTP error mapping for invalid input, authorization, missing Slot, stale version, invalid state, and idempotency conflict.

Tests are present in source for:

- PUBLIC+APPROVAL+FILLING create contract;
- create idempotency-key requirement;
- edit normalization/version requirement;
- stale version conflict;
- cancel version transition;
- HTTP create → Pulse → edit → stale-edit `409` → cancel flow.

**Important:** at this foundation stage a replayed mutation is effect-idempotent and returns the current canonical resource. Exact historical response replay is still part of the later durable-offline/realtime hardening block.

## 7. Verification state

### Actually verified in the available local environment

- earlier pure-Go liveness/session-token tests passed before external dependencies were introduced;
- standalone standard-library rate-limiter scratch test passed;
- standard-only block/recovery code was syntax/parse checked during development.

### NOT yet honestly verified

The current local execution environment cannot resolve external hosts and does not have the required external Go/Android dependencies cached. Therefore the following gates remain open:

- `go mod tidy` and generated/verified `backend/go.sum`;
- full `go test ./...` after `pgx` + `x/crypto` + new Slot code;
- PostgreSQL integration tests;
- migration execution against disposable PostgreSQL;
- applying migrations to canonical Supabase (not requested/deployed yet);
- Android Gradle compile;
- Android instrumentation tests;
- real Android ↔ Go ↔ PostgreSQL smoke;
- release signing/build.

Do **not** mark the above green without real evidence.

## 8. What still does NOT exist — active Android/Go Version 1

### Account/UI remaining

- breached/common-password blocklist integration;
- multi-session management UI;
- actual Compose Login/Register/Recovery screens in the frozen visual language;
- application navigation wired to `SessionCoordinator`;
- production SMTP credentials/provider smoke;
- real DB migration/account smoke.

### Foundation social loop remaining

- REQUEST / withdraw;
- host APPROVE / REJECT;
- accepted membership;
- atomic last-seat allocation under concurrency;
- LEAVE;
- START / COMPLETE;
- pending/accepted/host roster summaries;
- accepted-only ephemeral chat;
- terminal physical chat purge;
- Android production Pulse/LINK/host-control binding;
- real two-user end-to-end smoke.

### Later Version 1 blocks

- durable mutation outbox + transactional outbox;
- realtime snapshot/ordered deltas/reconnect;
- City Context/PostGIS locality;
- real Map/viewport/Places integration;
- Waitlist/host control V2;
- Chat V2;
- push notifications;
- BUMP/Reliability, City BPM, Vibe/Lasso/Hotspots, swarms;
- Fly production functionality;
- Me 2.0/Squad/Guardian/Ghost production functionality;
- AR/ranking;
- venue/BLE/media/adaptive systems;
- LinkUp+ billing/travel/host/discovery/privacy/identity/rewarded access;
- full ecosystem hardening.

### iOS — ⛔

All iOS implementation remains intentionally frozen and excluded from current Android/Go readiness.

## 9. README historical claims

README sections that say Android Event Core, Approval, chat, BUMP or historical migrations `000012/000013/000016/000018/000020/000022` were already implemented are **not current repository evidence**. Only code/migrations physically present in this repository and recorded here count.

## 10. Worklog

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

- bounded auth rate limit: `5f430737fac07e84aea765e113d2ed5e9b0f1937`;
- password reset domain/HTTP: `28307b7b87e3c813a0e38aca962ad668ef9c4cdc`;
- Android bootstrap/recovery client: `2bea65b4c19e5125270adf9fd769e741ec3f8989`;
- TLS-only SMTP adapter: `34362616f1c197b3bbe6b1dcd22a5ab4ff3be72d`;
- block controls: `6f11f02b3f1f742851e38f76feff97db69d6f861`.

### 2026-09-06 — Canonical Slot foundation

- migration `000002_slots.sql`;
- Slot domain/service/pgx store;
- create/read/Pulse/edit/cancel API;
- optimistic version + idempotency + block-aware discovery;
- source tests for service/HTTP flow.
- Commit: `2a1fb50728dd47a35f118a1f42d39d70428aca11`.

## 11. Current production readiness

**Android + Go Version 1 production readiness: 0%.**

Reason: production-oriented account/security/Slot foundations now exist, but external dependency build/DB/device verification is still open and the mandatory real social loop (`register → create Slot → REQUEST → APPROVE → chat → START/COMPLETE/CANCEL`) is not yet end-to-end. Documentation/scaffolding/design do not count toward readiness.

## 12. Next exact work block

**Approval social loop:**

1. add request/membership schema and unique invariants;
2. implement REQUEST + withdraw via LEAVE semantics;
3. host pending-request list with identity summary;
4. implement APPROVE/REJECT;
5. make last-seat approval atomic and update `accepted_count/state` under transaction lock;
6. accepted participant LEAVE + `FULL → FILLING` reopen;
7. implement host START/COMPLETE;
8. enforce block checks on every transition;
9. add idempotency/version/race-oriented tests;
10. then bind these server states to Android production models/API without changing the approved design.
