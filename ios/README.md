# LinkUp iOS — native implementation

Status: iOS work explicitly activated by direct user command on 2026-09-08.

## Platform contract

- Swift + SwiftUI only for the production iOS client.
- Apple SDKs first; no React Native, WebView-first client, Kotlin Multiplatform UI, or parallel domain authority.
- Go API remains the application-facing authority.
- PostgreSQL/PostGIS remains canonical storage behind the Go API.
- The frozen React/TypeScript files at repository root remain the visual/UI source of truth.
- Android source may be read to confirm current API semantics, but it is not an iOS UI implementation template.
- No server/database/service-role secrets belong in the app bundle or Git history.

## Current foundation

The first block establishes:

- an iOS 17 / Swift 5 language-mode project manifest with strict concurrency;
- strict Swift concurrency settings;
- exact frozen LinkUp color, typography-role and radius tokens;
- native shared card/glass/button/chip/avatar/progress/status/state primitives;
- native `Pulse · Map · LINK · Fly · Me` shell;
- central 56×56 LINK create action;
- design-safe Pulse, Map, Fly and Me states without fake social telemetry;
- three-step Create LINK flow with 16 canonical activity labels.

## Build configuration

`LINKUP_API_BASE_URL` is an explicit Xcode build setting and defaults to blank. The iOS client must fail closed until a valid endpoint is supplied; no production URL is hardcoded in source.

`project.yml` is the deterministic XcodeGen manifest for the first server-authored foundation. A generated `.xcodeproj` is intentionally not committed from this Linux host because Xcode is unavailable here.

## Verification boundary

This Ubuntu server has neither `swift` nor `xcodebuild`. XML/YAML/source-diff checks can run here, but iOS compilation, XCTest, simulator, signing and physical-device verification require macOS + Xcode and must not be marked green before execution.

## Next dependency-safe block

1. Apple Keychain session persistence.
2. HTTPS/loopback endpoint validation and bounded `URLSession` transport.
3. Typed Codable models matching the Go v1.0 API.
4. Register/login/logout/recovery/session bootstrap.
5. Real Pulse binding.
6. Idempotent Create/Edit/Request/Approval/Chat/lifecycle binding.
7. iOS unit/UI tests and macOS/Xcode build evidence.

## Native API/session foundation

Published in `44d4073d2ea32676ce14111e7f587a9d3f0ec349`:

- Codable account, Slot, request and chat models;
- release HTTPS-only endpoint validation;
- Apple Keychain bearer persistence;
- ephemeral, no-redirect, no-cookie, no-cache `URLSession` transport;
- 1 MiB streamed response bound and bounded GET retry only;
- cancellation-aware request handling;
- typed Pulse/My LINKs/roster/request/chat reads;
- idempotent Slot mutations and in-process ambiguous chat-send key reuse;
- session bootstrap states for signed-out, signed-in, temporary offline and recoverable failure.

Credential-entry auth publication is still open because the current tool safety boundary rejected the wrapper containing password/reset-token fields. This is recorded as incomplete rather than worked around.

## Current source state

The original foundation/next-block lists above are historical. Current `main` has moved beyond them:

- native register/login/logout/password-reset and profile editing are wired to the Go API;
- RootView is session-state driven with Keychain persistence and foreground revalidation;
- Pulse, Map, Me/My LINKs, Slot detail/edit, approval/roster/removal, lifecycle, block and chat surfaces use typed server data;
- City Context and canonical-place search are bound without prototype fake telemetry;
- durable mutation replay persists exact commands and fails closed on ambiguous/stale outcomes;
- Create LINK uses `POST /v1/slots/drafts` followed by versioned `POST /v1/slots/{slotID}/publish`, with protected process-death recovery and server-side discard;
- the iOS replay window is 20 hours, below the backend default 24-hour idempotency TTL. If a workflow becomes stale without a confirmed draft ID, the client does not replay create outside the safe window;
- Pulse/request/edit/start action eligibility is centralized in native model contracts matching the Go lifecycle rules;
- Pulse and Map quick mutations surface server/durable errors instead of silently swallowing them.

No generated `.xcodeproj`, compile-green, XCTest-green, simulator/device, signing or App Store claim is made from the Ubuntu host. The next authoritative gate is macOS/Xcode: generate the project, compile with strict concurrency, run tests, fix compiler findings, then execute the signed-in end-to-end flows on simulator and device.

## Password-recovery Universal Link gate

Automatic password-reset routing is fail-closed and requires both build settings to agree:

- `LINKUP_RECOVERY_RESET_URL` — exact HTTPS reset URL without query/fragment or custom port;
- `LINKUP_RECOVERY_ASSOCIATED_DOMAIN` — exact `applinks:<host>` value for the same host.

`ios/project.yml` wires `com.apple.developer.associated-domains` through `LinkUp.entitlements`. The repository default is the reserved non-production value `applinks:example.invalid`; release configuration must replace it together with the real reset URL.

The website side remains an external release gate: the configured host must serve a valid `/.well-known/apple-app-site-association` for the final signed app identifier. This Ubuntu source host cannot verify Apple CDN association, device Universal Link delivery, or signing.
