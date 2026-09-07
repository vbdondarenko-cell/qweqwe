# LinkUp Android FCM

Status: **Android push foundation implemented; production delivery requires the server service-account credential.**

Canonical path:

```text
Android Firebase Messaging
        |
        | FCM registration token
        v
Ubuntu Go API
        |
        | encrypted device registry
        v
Supabase PostgreSQL

Go social event -> Ubuntu FCM HTTP v1 sender -> Firebase -> Android notification
```

## Boundaries

- Firebase is used for **FCM push delivery only**.
- Firebase Auth, Firestore and Realtime Database are not LinkUp authorities.
- Android never receives Supabase database credentials or a Firebase service-account private key.
- Go remains the server-side authority for whether a notification exists and who may receive it.
- iOS is not part of this implementation block.

## Android client configuration

Ubuntu `/etc/linkup/build.env` supplies these Gradle properties to Android builds:

- `ORG_GRADLE_PROJECT_LINKUP_FIREBASE_API_KEY`
- `ORG_GRADLE_PROJECT_LINKUP_FIREBASE_APP_ID`
- `ORG_GRADLE_PROJECT_LINKUP_FIREBASE_PROJECT_ID`
- `ORG_GRADLE_PROJECT_LINKUP_FIREBASE_SENDER_ID`

These are Firebase Android client identifiers. The populated file is not stored in Git.

## Server-only runtime configuration

Ubuntu `/etc/linkup/linkup.env` owns:

- `LINKUP_PUSH_TOKEN_KEY_ID`
- `LINKUP_PUSH_TOKEN_KEY_BASE64`
- `LINKUP_FIREBASE_PROJECT_ID`
- `LINKUP_FIREBASE_SERVICE_ACCOUNT_FILE=/etc/linkup/firebase-service-account.json`

The service-account JSON and push-token encryption key must never be committed or bundled into Android.

`firebase-service-account.json` must belong to the Firebase project configured for LinkUp and must have permission to send FCM messages. Recommended file ownership is `root:linkup` with mode `0640` or stricter.

## Token storage

Migration `000011_android_push_devices.sql` stores:

- user/session/installation identity;
- SHA-256 token lookup hash;
- AES-256-GCM token ciphertext;
- 96-bit nonce;
- encryption key id;
- version and lifecycle timestamps.

Raw FCM registration tokens are not persisted in plaintext. Active delivery additionally requires a non-revoked, non-expired LinkUp session.

## API

Authenticated device registration:

```text
PUT /v1/me/push/android
DELETE /v1/me/push/android/{installationID}
```

Logout revokes the LinkUp session and best-effort marks its push registrations revoked. Delivery queries independently filter revoked/expired sessions, so notification authorization does not depend on cleanup succeeding.

## Current social events

The early foundation sends best-effort push for:

- a new approval request to the Slot host;
- APPROVE to the requester;
- REJECT to the requester.

Push delivery occurs outside the canonical social mutation. An FCM outage must not roll back an already committed Slot/request decision.

## Remaining production notification work

The broader notification release contract still includes durable dedupe/outbox behavior, cancellation/starting-soon/reopened-seat events, deep links, preferences, quiet-hours foundation and delivery metrics. Those later-release requirements are not used to block v1.0.
