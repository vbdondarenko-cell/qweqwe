import Foundation
import XCTest
@testable import LinkUp

final class DurableMutationOutboxTests: XCTestCase {
    private let fixedNow = Date(timeIntervalSince1970: 2_000_000_000)
    private let owner = String(repeating: "a", count: 64)
    private let slotID = UUID(uuidString: "5f1c1694-f4a7-4ca5-8433-ec70cfb7e6b6")!

    func testMutationOwnerFingerprintIsOpaqueAndDeterministic() {
        let first = MutationOwnerFingerprint.make(token: "opaque-session-token")
        let second = MutationOwnerFingerprint.make(token: "opaque-session-token")
        XCTAssertEqual(first, second)
        XCTAssertEqual(first?.count, 64)
        XCTAssertFalse(first?.contains("opaque") ?? true)
    }

    func testCommandReplayWindowStopsAutomaticRetryAfterTwentyHours() {
        let now = fixedNow
        let fresh = command(createdAt: now.addingTimeInterval(-60), firstAttemptAt: now.addingTimeInterval(-60))
        let staleAttempt = now.addingTimeInterval(-(durableMutationReplayWindow + 1))
        let stale = command(
            createdAt: staleAttempt.addingTimeInterval(-1),
            firstAttemptAt: staleAttempt
        )
        XCTAssertTrue(fresh.canAutoReplay(at: now))
        XCTAssertFalse(stale.canAutoReplay(at: now))
    }

    func testOutboxPersistsExactCommandAndReusesEquivalentIdentity() async throws {
        let root = temporaryRoot()
        defer { try? FileManager.default.removeItem(at: root) }
        let outbox = DurableMutationOutbox(baseDirectory: root)
        let original = command(createdAt: fixedNow, firstAttemptAt: nil)

        try await outbox.enqueue(original)
        let marked = try await outbox.markAttempt(idempotencyKey: original.idempotencyKey, now: fixedNow.addingTimeInterval(1))
        XCTAssertEqual(marked?.idempotencyKey, original.idempotencyKey)
        XCTAssertNotNil(marked?.firstAttemptAt)

        let reloaded = DurableMutationOutbox(baseDirectory: root)
        let equivalent = try await reloaded.replayableEquivalent(
            ownerFingerprint: owner,
            requestIdentity: original.requestIdentity,
            responseKind: .slot,
            expectedSlotID: slotID,
            now: fixedNow.addingTimeInterval(2)
        )
        XCTAssertEqual(equivalent?.idempotencyKey, original.idempotencyKey)
        XCTAssertEqual(equivalent?.body, original.body)
    }

    func testOutboxFlagsAgedAmbiguousCommandInsteadOfReplayingIt() async throws {
        let root = temporaryRoot()
        defer { try? FileManager.default.removeItem(at: root) }
        let outbox = DurableMutationOutbox(baseDirectory: root)
        let attempted = fixedNow.addingTimeInterval(-(durableMutationReplayWindow + 30))
        let stale = command(createdAt: attempted.addingTimeInterval(-1), firstAttemptAt: attempted)
        try await outbox.enqueue(stale)

        let now = fixedNow
        let pending = try await outbox.pendingReplayable(ownerFingerprint: owner, now: now)
        let unsafe = try await outbox.unsafeAmbiguousCount(ownerFingerprint: owner, now: now)
        XCTAssertTrue(pending.isEmpty)
        XCTAssertEqual(unsafe, 1)
    }

    func testCommandRejectsOversizedBodyAndGetMethod() {
        let now = fixedNow
        let oversizedBody = Data(repeating: 1, count: durableMutationMaxBodyBytes + 1)
        let oversizedRequest = APIRequest(method: .post, path: "/v1/slots", body: oversizedBody)
        let oversized = DurableMutationCommand(
            idempotencyKey: UUID(), ownerFingerprint: owner,
            requestIdentity: MutationIdentity.digest(for: oversizedRequest),
            method: .post, path: "/v1/slots", queryItems: [],
            body: oversizedBody, responseKind: .slot, expectedSlotID: nil,
            createdAt: now, firstAttemptAt: nil
        )
        let getRequest = APIRequest(method: .get, path: "/v1/slots")
        let get = DurableMutationCommand(
            idempotencyKey: UUID(), ownerFingerprint: owner,
            requestIdentity: MutationIdentity.digest(for: getRequest),
            method: .get, path: "/v1/slots", queryItems: [], body: nil,
            responseKind: .slot, expectedSlotID: nil, createdAt: now, firstAttemptAt: nil
        )
        XCTAssertFalse(oversized.hasValidShape)
        XCTAssertFalse(get.hasValidShape)
    }

    func testCommandRejectsTamperedBodyThatNoLongerMatchesIdentity() {
        let original = command(createdAt: fixedNow, firstAttemptAt: nil)
        let tampered = DurableMutationCommand(
            idempotencyKey: original.idempotencyKey,
            ownerFingerprint: original.ownerFingerprint,
            requestIdentity: original.requestIdentity,
            method: original.method,
            path: original.path,
            queryItems: original.queryItems,
            body: Data(#"{"expectedVersion":3,"title":"Changed"}"#.utf8),
            responseKind: original.responseKind,
            expectedSlotID: original.expectedSlotID,
            createdAt: original.createdAt,
            firstAttemptAt: original.firstAttemptAt
        )
        XCTAssertFalse(tampered.hasValidShape)
    }

    private func command(createdAt: Date, firstAttemptAt: Date?) -> DurableMutationCommand {
        let path = "/v1/slots/\(slotID.uuidString.lowercased())"
        let body = Data(#"{"expectedVersion":3,"title":"Coffee"}"#.utf8)
        let request = APIRequest(method: .patch, path: path, body: body)
        return DurableMutationCommand(
            idempotencyKey: UUID(),
            ownerFingerprint: owner,
            requestIdentity: MutationIdentity.digest(for: request),
            method: .patch,
            path: path,
            queryItems: [],
            body: body,
            responseKind: .slot,
            expectedSlotID: slotID,
            createdAt: createdAt,
            firstAttemptAt: firstAttemptAt
        )
    }

    private func temporaryRoot() -> URL {
        FileManager.default.temporaryDirectory
            .appendingPathComponent("LinkUpTests-DurableOutbox-\(UUID().uuidString)", isDirectory: true)
    }
}
