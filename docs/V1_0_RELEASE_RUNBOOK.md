# LinkUp v1.0 — RELEASE, BACKUP, RECOVERY & ROLLBACK RUNBOOK

Status: canonical operational procedure for the active Android + Go **v1.0** release.

This document defines how v1.0 is verified and released. It is **not execution evidence**. A step counts as passed only when its command/output is recorded in `V1_0_RELEASE_CHECKLIST.md` or the active worklog.

`PROJECT_RULES.md` and `README.md` have higher priority.

## 1. Non-negotiable release boundaries

- Android only; iOS remains frozen.
- Go API is the application-facing auth/domain authority.
- PostgreSQL/PostGIS is the source of truth.
- Migrations are forward-only. Never edit an already-applied migration.
- No GitHub Actions, Cloud Build, Cloud Run, Render or other forbidden deployment path.
- Do not place DB credentials, bearer tokens, SMTP credentials, keystore passwords or service-role secrets in Git.
- Release API, Privacy Policy and Terms URLs must be HTTPS.
- A production Android release must be signed with the configured release keystore.

## 2. Required local release inputs

Keep these outside Git, preferably in a protected local Gradle properties file or environment-backed Gradle properties:

```properties
LINKUP_API_BASE_URL=https://api.example.invalid
LINKUP_PRIVACY_URL=https://example.invalid/privacy
LINKUP_TERMS_URL=https://example.invalid/terms
LINKUP_KEYSTORE_FILE=/secure/path/linkup-release.jks
LINKUP_KEYSTORE_PASSWORD=...
LINKUP_KEY_ALIAS=...
LINKUP_KEY_PASSWORD=...
```

Replace placeholder domains only with the approved real production endpoints. Never commit the resolved secrets.

For backend/database verification:

```bash
export LINKUP_TEST_DATABASE_URL='postgresql://...disposable-test-db...'
export DATABASE_URL='postgresql://...target-runtime-db...'
```

`LINKUP_TEST_DATABASE_URL` must point to a disposable PostgreSQL database for destructive integration/race tests.

## 3. Source integrity preflight

From a clean checkout of `main`:

```bash
git status --short
git rev-parse HEAD
git diff --check
```

Expected:

- clean worktree;
- exact release commit recorded;
- no whitespace/conflict-marker failures.

Confirm forbidden artifacts/secrets are absent:

```bash
git grep -nE '(service_role|BEGIN (RSA|OPENSSH|EC) PRIVATE KEY|LINKUP_KEYSTORE_PASSWORD=|DATABASE_URL=postgresql://[^U])' -- ':!*.md' || true
git ls-files | grep -E '\.(jks|keystore|p12|pem|key|apk|aab)$' && exit 1 || true
```

Review all matches manually. False positives in tests/examples are not release blockers when they contain no real secret.

## 4. Gradle wrapper verification

The repository pins:

- Gradle: `9.6.0`;
- Gradle binary distribution SHA-256: `bbaeb2fef8710818cf0e261201dab964c572f92b942812df0c3620d62a529a01`;
- wrapper JAR SHA-256: `497c8c2a7e5031f6aa847f88104aa80a93532ec32ee17bdb8d1d2f67a194a9c7`.

If `gradle-wrapper.jar` is absent, `gradlew`/`gradlew.bat` bootstraps the pinned JAR and refuses to execute it unless the official checksum matches.

Verify after bootstrap:

```bash
cd android
./gradlew --version
sha256sum gradle/wrapper/gradle-wrapper.jar
```

Windows PowerShell:

```powershell
cd android
.\gradlew.bat --version
Get-FileHash .\gradle\wrapper\gradle-wrapper.jar -Algorithm SHA256
```

## 5. Go verification gate

From `backend/` with the repository-declared supported Go toolchain:

```bash
go mod tidy
go mod verify
gofmt -w $(find . -name '*.go' -type f)
git diff --check
go test ./...
go vet ./...
go test -race ./...
```

Release gate:

- `go.sum` is present and committed if dependency resolution changes it;
- all commands exit 0;
- no test is silently treated as integration evidence when it skipped for missing `LINKUP_TEST_DATABASE_URL`.

## 6. Disposable PostgreSQL migration gate

Apply the complete forward chain to a fresh disposable PostgreSQL database:

```text
000001_accounts.sql
000002_slots.sql
000003_approval.sql
000004_chat.sql
000005_api_role_boundary.sql
000006_chat_idempotency.sql
000007_v1_query_indexes.sql
```

Preferred verification through the Go migration runner:

```bash
LINKUP_TEST_DATABASE_URL="$LINKUP_TEST_DATABASE_URL" go test ./internal/migrate ./internal/postgres -count=1 -v
```

Then inspect migration ledger:

```sql
SELECT name, encode(checksum, 'hex') AS checksum, applied_at
FROM linkup_schema_migrations
ORDER BY name;
```

Required:

- all seven migrations recorded once;
- concurrent migration startup test passes;
- checksum drift test passes;
- terminal chat purge test passes;
- last-seat/capacity tests pass;
- block/request/approve race tests pass;
- password-reset/login race tests pass;
- chat idempotency tests pass;
- v1.0 pending-request ceiling and query/index contracts pass.

## 7. Database backup before production migration

Before applying any new production migration, confirm the managed Supabase backup state and create/retain a logical backup when permitted by the environment.

Example PostgreSQL logical backup:

```bash
pg_dump \
  --format=custom \
  --no-owner \
  --no-privileges \
  --file="linkup-pre-v1.0-$(date -u +%Y%m%dT%H%M%SZ).dump" \
  "$DATABASE_URL"
sha256sum linkup-pre-v1.0-*.dump
```

Store the dump in an approved protected backup location, **not Git and not the Android artifact**. Record:

- UTC timestamp;
- source database identity/environment;
- migration ledger top entry;
- dump SHA-256;
- restore owner/contact;
- retention location.

Do not place the dump or database URL in chat/repository logs.

## 8. Restore drill

A backup is not accepted as recovery evidence until a restore has been tested on a separate disposable database.

Example:

```bash
createdb linkup_restore_drill
pg_restore --no-owner --no-privileges --dbname=linkup_restore_drill /secure/path/linkup-pre-v1.0-....dump
```

Verify at minimum:

```sql
SELECT count(*) FROM app_users;
SELECT count(*) FROM slots;
SELECT count(*) FROM linkup_schema_migrations;
SELECT name FROM linkup_schema_migrations ORDER BY name;
```

Then run API/database smoke tests against the restored disposable database. Record evidence and destroy the drill DB afterward.

## 9. Forward-only database rollback policy

**Never roll back production by editing/deleting an applied SQL migration or by running an unreviewed reverse migration.**

If application release `R2` must be rolled back to `R1`:

1. stop new deployment of `R2`;
2. keep the database at its current forward schema if `R1` is schema-compatible;
3. redeploy the previously verified Go binary/config for `R1`;
4. redeploy the previously verified signed Android artifact only through the normal store/release mechanism where applicable;
5. run health/auth/social smoke checks;
6. if schema correction is required, create a **new forward migration** that restores compatible behavior/data;
7. never destroy newer user data merely to match an older binary.

Every migration in v1.0 must therefore be backward-compatible with the immediately previous application binary for the rollback window, or the release must explicitly document why application rollback is unsafe and require a forward-fix path.

## 10. Go application rollback

For the target Ubuntu runtime, after the user explicitly authorizes deployment:

- retain the previous verified Go binary and its SHA-256;
- retain the previous non-secret configuration version/reference;
- deploy atomically (new file → integrity check → service switch/restart);
- on failed smoke, restore the previous binary and restart;
- confirm `/livez` and `/healthz` plus authenticated v1.0 smoke;
- never print bearer tokens or DB credentials during rollback.

The exact Ubuntu commands are executed only during a user-authorized deployment session.

## 11. Android verification gate

From `android/`:

```bash
./gradlew clean testDebugUnitTest lintDebug assembleDebug
```

Then, on an emulator/device supported by the test matrix, execute required instrumentation/accessibility flows.

Release preflight:

```bash
./gradlew clean test lint bundleRelease
```

The Gradle release task must fail closed if API/legal/signing properties are missing.

## 12. Signed AAB integrity

After `bundleRelease`:

```bash
sha256sum app/build/outputs/bundle/release/app-release.aab
jarsigner -verify -verbose -certs app/build/outputs/bundle/release/app-release.aab
```

If Android SDK build-tools provide `apksigner`, also verify generated APKs used for device testing.

Record:

- source commit SHA;
- versionCode/versionName;
- AAB SHA-256;
- signing certificate SHA-256 fingerprint;
- Gradle version;
- JDK version;
- build UTC timestamp.

Never distribute the keystore/private key.

## 13. Required two-user v1.0 regression

Use two distinct test accounts/devices or isolated app instances A and B.

1. A registers/logs in.
2. B registers/logs in.
3. A creates PUBLIC + APPROVAL Slot.
4. B sees it in real Pulse.
5. B sends REQUEST.
6. A sees B in pending list.
7. Verify B has no chat access while pending.
8. A APPROVES B.
9. Verify capacity and viewer states update server-side.
10. A and B can read/send ephemeral chat.
11. Verify chat duplicate-send replay returns one canonical message.
12. B LEAVES; B immediately loses chat access; capacity reopens correctly.
13. Repeat with a fresh membership; verify host remove also revokes chat.
14. Verify block removes request/membership relationship and prevents subsequent interaction.
15. Verify `FULL → FILLING` when a seat reopens.
16. A STARTS a Slot with at least one accepted member → `ACTIVE`.
17. New stranger REQUEST is denied for ACTIVE.
18. A COMPLETES → `COMPLETED`.
19. Verify chat becomes inaccessible and `slot_messages` rows for the Slot are physically zero.
20. Verify cancelled Slot disappears from Pulse and chat rows purge.
21. Verify process death restores a valid session; revoked/expired session routes SignedOut.
22. Verify transient GET failure retains cached content while definitive 401/403 removes protected state.

Run critical capacity/request/block paths concurrently, not only sequentially.

## 14. Localization and accessibility gate

For both `en` and `uk`:

- Auth/register/recovery/reset;
- Pulse/search/filter;
- Create/Edit LINK;
- Slot detail/host controls;
- My LINKs;
- Chat;
- Me/profile/block/legal/version;
- offline/config/error surfaces.

Verify:

- no clipped primary action text at supported font scales;
- TalkBack names close/back icon-only controls;
- controls remain reachable with large font/display scaling;
- focus order follows visual order;
- buttons expose disabled/enabled state correctly;
- color is not the sole carrier of critical state;
- screen reader does not expose bearer/reset credentials.

## 15. Privacy/security release review

Before release confirm:

- no exact stranger GPS is exposed;
- no fake reliability/online/BPM data appears;
- block enforcement covers discovery, membership and chat;
- bearer/reset raw tokens are absent from logs/DB persistence;
- password recovery is non-enumerating externally;
- Android does not contain DB/service-role credentials;
- API mutations remain server-authoritative and idempotent where required;
- GET retry is bounded and mutations are not blindly retried;
- response bodies, chat thread, Pulse/My LINKs and pending queues are bounded;
- direct Supabase `anon/authenticated/PUBLIC` access boundary is verified before live migration application.

## 16. Release decision

v1.0 can be marked **GREEN / DONE** only when every mandatory item in `V1_0_RELEASE_CHECKLIST.md` has evidence.

If any build, migration, race, restore, two-user, localization/accessibility, legal/signing or integrity gate is missing, the release remains **NOT DONE**, regardless of source completeness.
