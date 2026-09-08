import Foundation
import XCTest
@testable import LinkUp

final class DraftPublishWorkflowStoreTests: XCTestCase {
    private let fixedNow = Date(timeIntervalSince1970: 2_000_000_000)
    private let owner = String(repeating: "b", count: 64)

    func testStorePersistsExactWorkflowAndDraftIdentity() async throws {
        let root = temporaryRoot()
        defer { try? FileManager.default.removeItem(at: root) }
        let store = DraftPublishWorkflowStore(baseDirectory: root)
        let initial = workflow()
        try await store.save(initial)
        let roundTrip = try await store.load()
        XCTAssertEqual(roundTrip, initial)

        let draftID = UUID()
        let advanced = initial.recordingDraft(id: draftID, version: 4)
        try await store.save(advanced)
        let loaded = try await store.load()
        XCTAssertEqual(loaded?.draftID, draftID)
        XCTAssertEqual(loaded?.draftVersion, 4)
        XCTAssertEqual(loaded?.createBody, initial.createBody)
        XCTAssertEqual(loaded?.createKey, initial.createKey)
        XCTAssertEqual(loaded?.publishKey, initial.publishKey)
    }

    func testNeedsAttentionPersistsWithoutChangingStableKeys() async throws {
        let root = temporaryRoot()
        defer { try? FileManager.default.removeItem(at: root) }
        let store = DraftPublishWorkflowStore(baseDirectory: root)
        let initial = workflow().recordingDraft(id: UUID(), version: 2)
        try await store.save(initial)
        try await store.markNeedsAttention()

        let loaded = try await store.load()
        XCTAssertEqual(loaded?.requiresAttention, true)
        XCTAssertEqual(loaded?.createKey, initial.createKey)
        XCTAssertEqual(loaded?.publishKey, initial.publishKey)
        XCTAssertEqual(loaded?.cancelKey, initial.cancelKey)
        XCTAssertEqual(loaded?.draftID, initial.draftID)
        XCTAssertEqual(loaded?.draftVersion, initial.draftVersion)
    }

    func testWorkflowRejectsIdentityTamperingAndIncompleteDraftPair() {
        let valid = workflow()
        XCTAssertTrue(valid.hasValidShape)

        let tampered = DraftPublishWorkflow(
            ownerFingerprint: valid.ownerFingerprint,
            requestIdentity: String(repeating: "0", count: 64),
            createBody: valid.createBody,
            createKey: valid.createKey, publishKey: valid.publishKey, cancelKey: valid.cancelKey,
            draftID: nil, draftVersion: nil, createdAt: valid.createdAt, requiresAttention: false
        )
        XCTAssertFalse(tampered.hasValidShape)

        let incomplete = DraftPublishWorkflow(
            ownerFingerprint: valid.ownerFingerprint, requestIdentity: valid.requestIdentity,
            createBody: valid.createBody, createKey: valid.createKey, publishKey: valid.publishKey,
            cancelKey: valid.cancelKey, draftID: UUID(), draftVersion: nil,
            createdAt: valid.createdAt, requiresAttention: false
        )
        XCTAssertFalse(incomplete.hasValidShape)
    }

    func testManualResolveRequiresAttentionAndConfirmedDraftIdentity() {
        let base = workflow()
        XCTAssertFalse(base.canManuallyResolve)
        XCTAssertFalse(base.markingNeedsAttention().canManuallyResolve)

        let confirmed = base.recordingDraft(id: UUID(), version: 3).markingNeedsAttention()
        XCTAssertTrue(confirmed.canManuallyResolve)
    }

    func testPreparingRetryOnlyAdvancesDraftVersionAndClearsAttention() throws {
        let draftID = UUID()
        let initial = workflow().recordingDraft(id: draftID, version: 2).markingNeedsAttention()
        let retry = try XCTUnwrap(initial.preparingRetry(version: 5))

        XCTAssertEqual(retry.draftID, draftID)
        XCTAssertEqual(retry.draftVersion, 5)
        XCTAssertFalse(retry.requiresAttention)
        XCTAssertEqual(retry.createBody, initial.createBody)
        XCTAssertEqual(retry.createKey, initial.createKey)
        XCTAssertEqual(retry.publishKey, initial.publishKey)
        XCTAssertEqual(retry.cancelKey, initial.cancelKey)
        XCTAssertEqual(retry.requestIdentity, initial.requestIdentity)
    }

    func testWorkflowRejectsReusedOperationKeys() {
        let valid = workflow()
        let invalid = DraftPublishWorkflow(
            ownerFingerprint: valid.ownerFingerprint, requestIdentity: valid.requestIdentity,
            createBody: valid.createBody, createKey: valid.createKey, publishKey: valid.createKey,
            cancelKey: valid.cancelKey, draftID: nil, draftVersion: nil,
            createdAt: valid.createdAt, requiresAttention: false
        )
        XCTAssertFalse(invalid.hasValidShape)
    }

    func testWorkflowAutoResumeUsesSameTwentyHourSafetyWindow() {
        let fresh = workflow(createdAt: fixedNow.addingTimeInterval(-60))
        let stale = workflow(createdAt: fixedNow.addingTimeInterval(-(durableMutationReplayWindow + 1)))
        XCTAssertTrue(fresh.canAutoResume(at: fixedNow))
        XCTAssertFalse(stale.canAutoResume(at: fixedNow))
    }

    func testCorruptWorkflowFileFailsClosed() async throws {
        let root = temporaryRoot()
        defer { try? FileManager.default.removeItem(at: root) }
        let file = root.appendingPathComponent("LinkUp/DraftPublish/v1.json")
        try FileManager.default.createDirectory(at: file.deletingLastPathComponent(), withIntermediateDirectories: true)
        try Data("not-json".utf8).write(to: file)

        let store = DraftPublishWorkflowStore(baseDirectory: root)
        do {
            _ = try await store.load()
            XCTFail("Corrupt workflow must fail closed")
        } catch let error as DraftPublishWorkflowStoreError {
            guard case .corrupt = error else {
                return XCTFail("Unexpected workflow error: \(error)")
            }
        }
    }

    private func workflow(createdAt: Date? = nil) -> DraftPublishWorkflow {
        let body = Data(#"{"title":"Coffee","activity":"coffee","placeText":"Podil","capacity":4}"#.utf8)
        let request = APIRequest(method: .post, path: "/v1/slots/drafts", body: body)
        return DraftPublishWorkflow(
            ownerFingerprint: owner,
            requestIdentity: MutationIdentity.digest(for: request),
            createBody: body,
            createKey: UUID(),
            publishKey: UUID(),
            cancelKey: UUID(),
            draftID: nil,
            draftVersion: nil,
            createdAt: createdAt ?? fixedNow,
            requiresAttention: false
        )
    }

    private func temporaryRoot() -> URL {
        FileManager.default.temporaryDirectory
            .appendingPathComponent("LinkUpTests-DraftWorkflow-\(UUID().uuidString)", isDirectory: true)
    }
}
