# LinkUp — CANONICAL INFRASTRUCTURE & DELIVERY WORKFLOW

Status: **active canonical infrastructure and delivery decision**.

This file defines where LinkUp code, runtime, data, push delivery, Android builds and release artifacts live. It must be read together with `PROJECT_RULES.md`, `README.md`, `IMPLEMENTATION_STATUS.md` and the active release runbook before infrastructure/build/deployment work.

If an older document or historical instruction conflicts with this file, `PROJECT_RULES.md` still has the highest priority. This document is the current infrastructure interpretation of those rules.

---

# 1. Canonical architecture

```text
                         source authority
                              GitHub
                               main
                                |
                                | git fetch / reset / pull
                                v
+---------------------------------------------------------------+
|                         Ubuntu server                         |
|                                                               |
|  Go API runtime          Android build host      release work  |
|  systemd service         JDK + SDK + Gradle      APK / AAB     |
|        |                         |                verification  |
+--------|-------------------------|------------------------------+
         |
         +-----------------------> Supabase PostgreSQL/PostGIS
         |                         persistent product data
         |
         +-----------------------> Firebase Cloud Messaging
                                   push delivery only

Android app
    |
    | HTTPS
    v
Ubuntu Go API
```

Canonical split:

- **GitHub** = source-code authority.
- **Ubuntu** = Go API runtime + deployment host + Android build/release host.
- **Supabase** = managed PostgreSQL/PostGIS and canonical persistent product database.
- **Firebase** = FCM push delivery only.
- **Android** = only active mobile platform for now.
- **iOS** = frozen until a separate direct user command.

---

# 2. GitHub — source of truth for code

Repository: `vbdondarenko-cell/qweqwe`.

Rules:

- all source code, migrations, build scripts and canonical documentation live in GitHub;
- current work goes directly to `main` under `PROJECT_RULES.md`;
- Ubuntu receives code from GitHub; Ubuntu is not a competing source repository;
- server-side hotfixes must not become an untracked second source of truth — a real fix must be committed back to `main`;
- GitHub Actions are **not** the canonical build/test/deploy/APK/AAB pipeline;
- Google Cloud Build/Cloud Run/Cloud Deploy are not part of the canonical delivery path.

Canonical code flow:

```text
local/development work
        |
        v
GitHub main
        |
        v
Ubuntu checkout
        |
        +--> Go build/test/deploy
        +--> Android build/test/APK/AAB
```

---

# 3. Supabase — database only, not the app-facing backend

Supabase project: `oavnrlwsfiiehluubwjk`.

Supabase is retained because its managed PostgreSQL/PostGIS layer is enough for the current project and avoids operating the production database manually on Ubuntu.

## 3.1. What is stored there

Canonical persistent LinkUp application data belongs in PostgreSQL/PostGIS in Supabase according to repository migrations and domain contracts.

Examples include account/domain/social state, Slots, memberships, requests, chat state where applicable, entitlement state and other canonical records introduced by repository migrations.

**Important clarification:** persistent product data is not supposed to live in random files on the Ubuntu disk. Ubuntu owns the application/runtime path to the data, while Supabase PostgreSQL is the canonical persistent database.

## 3.2. Access model

```text
Android
   |
   | HTTPS + bearer/session
   v
Go API on Ubuntu
   |
   | server-side DATABASE_URL
   v
Supabase PostgreSQL/PostGIS
```

Rules:

- Android never receives PostgreSQL credentials;
- Android never receives `service_role`, database passwords or privileged SQL access;
- Go is the application-facing authority for auth/domain/data decisions;
- schema changes are forward-only migrations stored in GitHub;
- direct Supabase Data API/Auth/Realtime access must not become a parallel authority unless the project rules are explicitly changed;
- do not create a second production PostgreSQL source of truth on Ubuntu.

Supabase Auth, Data API, Realtime and Storage are not automatically active product authorities merely because the Supabase project exists.

## 3.3. Test database policy

Destructive PostgreSQL integration tests must **never** run against production Supabase data.

When a disposable database is needed and the Ubuntu server is available, prefer a temporary local PostgreSQL database on Ubuntu for destructive migration/integration/race testing instead of creating a paid Supabase branch.

A local PostgreSQL instance used this way is **test-only and disposable**. It must never become the production LinkUp database authority.

---

# 4. Firebase — FCM push notifications only

Firebase is used only as the push-delivery layer through **Firebase Cloud Messaging (FCM)**.

Canonical push flow:

```text
product/domain event
       |
       v
Go API on Ubuntu
       |
       | server decides whether a push is allowed/needed
       v
Firebase Cloud Messaging
       |
       v
Android device
```

Allowed Firebase scope now:

- Android FCM device-token registration plumbing;
- server-side FCM send adapter;
- delivery of push notifications to Android;
- minimum provider configuration required for FCM.

Not canonical without a new direct user decision:

- Firebase Auth;
- Firestore as product database;
- Firebase Realtime Database as product state/realtime authority;
- Firebase Storage as canonical application storage;
- Firebase Functions as backend/domain authority;
- client-side Firebase logic that bypasses the Go API.

The Go backend decides whether a notification exists, which user can receive it and what minimum payload is safe to send.

FCM credentials/server secrets must stay on the server-side secret/config path and never be committed to GitHub or embedded as privileged secrets in the Android app.

## Future iOS note

iOS is currently frozen and no iOS work is authorized.

If iOS is enabled later, the same Firebase push layer can remain in front of mobile push delivery; iOS would additionally require the normal APNs/Firebase configuration. That future compatibility does **not** authorize any iOS code/configuration work now.

---

# 5. Ubuntu — runtime, deploy and Android build server

The dedicated Ubuntu host has three canonical roles:

1. **run the production Go API**;
2. **deploy new Go API builds safely**;
3. **build/test Android and produce APK/AAB artifacts**.

Ubuntu is not the source-control authority and not the production product database.

## 5.1. Canonical server layout

Current scripts use this layout:

```text
/opt/linkup/
├── src/                     # checkout of GitHub main
├── artifacts/<commit>/      # candidate build artifacts
└── bin/
    └── linkup-api           # promoted live Go binary

/etc/linkup/
└── linkup.env               # runtime secrets/config, never committed

/opt/android-sdk/            # Android SDK/toolchain
```

Canonical systemd service:

```text
linkup-api.service
```

## 5.2. Required server toolchain

The Ubuntu build/deploy host must have the repository-pinned/required versions of:

- Git;
- Go;
- JDK 17;
- Android SDK/platform/build tools required by the Android project;
- Gradle Wrapper from the repository;
- standard Linux tooling required by `ops/*` scripts.

Canonical bootstrap/build/deploy scripts live in the repository under `ops/` and must be preferred over undocumented manual server procedures.

---

# 6. Secrets and configuration

Secrets do not belong in Git history.

Server-only configuration includes, as applicable:

- `DATABASE_URL` for Supabase PostgreSQL;
- FCM server credentials/provider configuration;
- recovery/mail credentials;
- Android release keystore path;
- keystore password;
- key alias;
- key password;
- production API URL;
- privacy/terms URLs;
- reset/App Link host;
- any future provider verification secrets.

Runtime environment belongs under the protected Ubuntu configuration path, currently:

```text
/etc/linkup/linkup.env
```

Recommended permissions are restricted to root/runtime service access; secrets must never be echoed into logs, pasted into source files, committed to GitHub or exposed to Android clients.

The Android release keystore must be backed up securely outside the disposable build workspace. Losing the release signing key can make future app updates impossible or significantly harder depending on Play signing configuration.

---

# 7. Canonical Go API deploy flow

A deployment must be based on a real commit from `main`.

```text
GitHub main
    |
    v
Ubuntu fetch/reset to origin/main
    |
    v
Go verify/test/vet/race
    |
    v
build immutable candidate binary
    |
    v
SHA-256 verification
    |
    v
atomic promotion
    |
    v
systemd restart
    |
    v
/livez + /healthz
    |
    +-- green --> keep new binary
    |
    +-- fail ---> rollback previous binary
```

The repository currently provides the canonical build/deploy path through:

```text
ops/build_v1.sh
ops/deploy_v1.sh
ops/linkup-api.service
```

`ops/deploy_v1.sh` must promote a previously built artifact rather than compiling directly into the live binary path.

The previous live Go binary may exist only as a temporary rollback candidate during deployment. After the new binary passes health checks, obsolete rollback material may be removed according to the retention policy below.

---

# 8. Canonical Android build flow

Android is the only active mobile build target.

Every candidate comes from GitHub `main`, not from an uncommitted server workspace.

Minimum debug/candidate gate:

```text
GitHub main
    |
    v
Ubuntu clean checkout/reset
    |
    v
Android unit tests
    |
    v
Android lint
    |
    v
assembleDebug
    |
    v
candidate APK
    |
    v
SHA-256 / integrity checks
```

Release gate adds production configuration/signing and produces both artifacts:

```text
assembleRelease  --> signed APK
bundleRelease    --> signed AAB
```

Purpose:

- **APK** = direct installation/testing/handoff artifact;
- **AAB** = Google Play publishing artifact.

A build is not considered successful merely because an `.apk` or `.aab` file exists. Required tests/lint/build/signing/integrity gates for the active release must pass first.

Current repository build entry point:

```bash
ops/build_v1.sh
```

For release build, the script requires the release configuration/signing variables defined by the Android project and fails closed if required values are missing.

---

# 9. APK/AAB artifact retention — save disk without losing the last good build

The server must not accumulate unlimited APK/AAB/build directories.

At the same time, the last known-good artifact must not be deleted **before** a replacement has successfully passed the required checks.

Canonical lifecycle:

```text
PREVIOUS STABLE
      |
      | keep while new build is being created
      v
NEW CANDIDATE
      |
      +-- build/test fails --> delete failed candidate
      |                      keep previous stable
      |
      +-- build/test green --> verify checksum/signing
                                |
                                v
                             handoff/test
                                |
                  +-------------+-------------+
                  |                           |
                fail                         green
                  |                           |
          delete candidate             promote candidate
          keep previous stable         to CURRENT STABLE
                                              |
                                              v
                              after successful handoff/
                              promotion confirmation,
                              delete obsolete older build
```

Disk-retention target:

- during a build: **1 previous stable + 1 candidate**;
- after a failed build: **1 previous stable**;
- after a successful promotion/handoff: **1 current stable**;
- temporary build/cache files may be cleaned after the build when safe;
- do not retain an unlimited history of per-commit APK/AAB artifacts on Ubuntu.

For the Go runtime, `ops/deploy_v1.sh` already follows the same safety principle: keep a previous binary during the health-check window and remove it after successful promotion.

For Android artifacts, do not delete the previous stable APK/AAB until the replacement has passed required verification and the handoff destination has confirmed a valid copy/checksum.

---

# 10. Artifact handoff to the developer computer

The Ubuntu server is the build machine; the developer computer is an artifact destination, not the source of the build.

Canonical sequence:

1. build APK/AAB on Ubuntu from a known `main` commit;
2. run required tests/lint/signature/integrity checks;
3. calculate SHA-256;
4. copy/download the approved artifact to the developer computer;
5. verify the destination artifact checksum when practical;
6. only then clean obsolete server artifacts under the retention policy.

If a candidate fails testing after handoff, it is not promoted as stable; rebuild from corrected `main`, then remove the failed candidate when it is no longer needed for diagnostics.

---

# 11. End-to-end ownership model

## Product data

```text
canonical persistence = Supabase PostgreSQL/PostGIS
```

## Product/backend logic

```text
canonical server authority = Go API on Ubuntu
```

## Android source

```text
canonical source = GitHub main
canonical implementation = Kotlin + Jetpack Compose
```

## Push notifications

```text
canonical delivery provider = Firebase Cloud Messaging
canonical decision authority = Go API
```

## Build and deploy

```text
canonical host = Ubuntu
source input = GitHub main
Android outputs = APK + AAB
Go output = native Linux binary/systemd service
```

---

# 12. Explicitly excluded architecture

Unless the user directly changes the decision, do not introduce these as alternative authorities or mandatory deployment infrastructure:

- Google Cloud Build;
- Cloud Run;
- Cloud Deploy;
- Artifact Registry;
- Google Cloud Pub/Sub;
- Google Cloud Secret Manager;
- GitHub Actions build/test/deploy workflows;
- Render.com;
- a second production PostgreSQL database on Ubuntu;
- Firebase Auth/Firestore/Realtime Database as LinkUp product authority;
- Docker as a mandatory deployment requirement.

A native Go binary managed by systemd is the canonical backend runtime model.

---

# 13. Current platform boundary

Active mobile platform: **Android only**.

Allowed active work:

- Kotlin/Jetpack Compose Android implementation;
- Go backend;
- Supabase PostgreSQL/PostGIS migrations/data layer;
- Firebase FCM Android push integration;
- Ubuntu build/deploy/runtime work when directly requested;
- Android APK/AAB build/signing/release work.

Frozen until direct command:

- iOS/Swift/SwiftUI;
- Xcode project/signing;
- APNs/iOS Firebase configuration;
- iOS builds/tests/releases.

---

# 14. Operational checklist

Before backend deployment:

- [ ] target commit exists in GitHub `main`;
- [ ] server checkout matches the target commit;
- [ ] required Go gates are green;
- [ ] immutable Go artifact exists;
- [ ] artifact SHA-256 matches the ledger;
- [ ] `/etc/linkup/linkup.env` contains required runtime configuration without secrets in Git;
- [ ] deployment health checks `/livez` and `/healthz` pass;
- [ ] rollback path exists until health checks succeed.

Before Android APK handoff:

- [ ] target commit exists in `main`;
- [ ] Android unit tests are green;
- [ ] lint is green;
- [ ] APK build is green;
- [ ] required integrity/signing checks are green;
- [ ] SHA-256 recorded;
- [ ] previous stable artifact retained until candidate verification completes;
- [ ] approved APK copied to the developer destination;
- [ ] obsolete artifacts removed after successful replacement.

Before AAB/Play handoff:

- [ ] all Android release gates above are green;
- [ ] production API/legal/reset configuration is valid;
- [ ] release signing is configured securely;
- [ ] `assembleRelease` succeeds;
- [ ] `bundleRelease` succeeds;
- [ ] release APK and AAB SHA-256 recorded;
- [ ] signing/certificate evidence verified;
- [ ] approved AAB copied to the release destination;
- [ ] obsolete artifacts removed only after a valid replacement exists.

---

# 15. One-sentence rule

**Code lives in GitHub, Go runs/builds/deploys on Ubuntu, persistent product data lives in Supabase PostgreSQL/PostGIS behind Go, Firebase is only FCM push delivery, Android is the only active client, and Ubuntu produces verified APK/AAB artifacts while retaining the last good build until its replacement is proven good.**
