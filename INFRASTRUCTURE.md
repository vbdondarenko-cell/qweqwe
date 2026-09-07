# LinkUp — CANONICAL INFRASTRUCTURE

Status: **active canonical infrastructure decision**.

This file clarifies the current infrastructure split and must be read together with `PROJECT_RULES.md` and `README.md` before deployment/build work.

## Canonical stack

### GitHub

- Source-control authority.
- Work is committed directly to `main` under the current project rules.
- GitHub Actions are not used for build, test, deploy, migrations, APK or AAB generation unless the user explicitly changes that rule.

### Supabase

Project: `oavnrlwsfiiehluubwjk`.

Supabase is retained as the managed **PostgreSQL/PostGIS** infrastructure and canonical persistent database for LinkUp.

- PostgreSQL/PostGIS is the source of truth for persistent application data.
- Database schema changes are forward-only migrations from the repository.
- The Go API remains the application-facing authority for auth/domain/data decisions.
- Android must not receive database credentials, `service_role`, database passwords or direct privileged SQL access.
- Supabase Auth, Data API, Realtime and Storage are not canonical product authorities unless the user explicitly changes this later.
- Do not create a second canonical PostgreSQL database on the Ubuntu server.

### Firebase

Firebase is used **only for push notifications through Firebase Cloud Messaging (FCM)**.

Allowed Firebase scope:

- FCM device registration/integration;
- server-originated push delivery;
- Android notification delivery plumbing required by FCM.

Not canonical / not allowed without a new direct user decision:

- Firebase Auth;
- Firestore as product database;
- Firebase Realtime Database as product database/realtime authority;
- Firebase Storage as canonical media/data store;
- Firebase Functions as backend/domain authority;
- using Firebase to bypass the Go API or Supabase PostgreSQL source of truth.

The Go backend decides whether a push should exist and sends only the minimum notification payload required by the product contract.

### Ubuntu server

The dedicated Ubuntu server is the canonical deployment/build host for the active Android + Go product.

Its intended responsibilities are:

1. run/deploy the LinkUp Go API;
2. hold the runtime environment/configuration required by the Go API;
3. connect server-side to the managed Supabase PostgreSQL/PostGIS database;
4. build Android APK artifacts;
5. build Android AAB artifacts;
6. run permitted Go/Android build/test/release commands during deployment/release work;
7. produce verified artifact hashes/signature evidence before handoff.

The Ubuntu server is **not** the canonical database host and must not become a second source of truth.

Do not install or depend on a local production PostgreSQL instance for LinkUp unless the user explicitly changes the architecture.

## Explicitly excluded infrastructure

Unless the user directly changes the decision, LinkUp does not use these as canonical build/deploy infrastructure:

- Google Cloud Build;
- Cloud Run;
- Cloud Deploy;
- Artifact Registry;
- Google Cloud Pub/Sub;
- Google Cloud Secret Manager;
- GitHub Actions build/test/deploy workflows;
- Render.com.

Docker is not required as the canonical deployment model. The Go API may be deployed as a native Linux binary/service.

## Data flow

```text
Android app
    |
    | HTTPS API
    v
Ubuntu Go API
    |
    +---- PostgreSQL/PostGIS ----> Supabase
    |
    +---- push adapter ----------> Firebase Cloud Messaging
```

Android does not talk directly to PostgreSQL with privileged credentials.

FCM is a delivery channel, not a domain authority.

## Android release flow

Canonical release/build direction:

```text
GitHub main
    |
    v
Ubuntu build/deploy host
    |
    +--> Go build/test/runtime deployment
    |
    +--> Android Gradle build
           |
           +--> APK
           +--> AAB
           +--> signing verification
           +--> SHA-256 verification
```

APK/AAB artifacts are handed off directly after verification according to `PROJECT_RULES.md` and the active release runbook.

## Current infrastructure state

As of 2026-09-07:

- Supabase is retained and the current LinkUp migration chain has been installed there;
- Firebase remains reserved for FCM push integration;
- the dedicated Ubuntu host has been cleaned of the previous quantbot/BingX application files/services and is intended for the fresh LinkUp deploy/build environment;
- v1.0 remains the active release until its release gates are closed.
