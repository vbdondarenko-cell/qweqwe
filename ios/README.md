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

- an iOS 17 / Swift 5.10 project manifest;
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
