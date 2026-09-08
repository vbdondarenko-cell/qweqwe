import XCTest
@testable import LinkUp

final class DraftHostingContractTests: XCTestCase {
    private let slotID = UUID()
    private let organizerID = UUID()

    func testDraftCreateAcceptsDraftAndAdvancedReplaySnapshots() throws {
        XCTAssertNoThrow(try validateDraftCreateReplay(slot(state: .draft, viewer: .host, version: 1)))
        XCTAssertNoThrow(try validateDraftCreateReplay(slot(state: .filling, viewer: .host, version: 3)))
        XCTAssertNoThrow(try validateDraftCreateReplay(slot(state: .completed, viewer: .host, version: 8)))
    }

    func testDraftCreateRejectsUnauthorizedOrImpossibleDraft() {
        XCTAssertThrowsError(try validateDraftCreateReplay(slot(state: .draft, viewer: .none, version: 1)))
        XCTAssertThrowsError(try validateDraftCreateReplay(slot(state: .draft, viewer: .host, version: 1, acceptedCount: 1)))
    }

    func testPublishReplayRequiresTransitionPastDraftAndVersion() {
        XCTAssertNoThrow(try validateDraftPublishReplay(slot(state: .filling, viewer: .host, version: 2), expectedVersion: 1))
        XCTAssertNoThrow(try validateDraftPublishReplay(slot(state: .cancelled, viewer: .host, version: 7), expectedVersion: 1))
        XCTAssertThrowsError(try validateDraftPublishReplay(slot(state: .draft, viewer: .host, version: 2), expectedVersion: 1))
        XCTAssertThrowsError(try validateDraftPublishReplay(slot(state: .filling, viewer: .host, version: 1), expectedVersion: 1))
    }

    func testCreateBodyRoundTripsWithCanonicalAPICoding() throws {
        let start = Date(timeIntervalSince1970: 2_000_000_000)
        let original = CreateSlotBody(
            title: "Coffee", activity: "coffee", details: "Catch up", placeText: "Podil",
            zoneText: nil, canonicalPlaceId: UUID(), startAt: start, capacity: 4
        )
        let data = try APICoding.encoder().encode(original)
        let decoded = try APICoding.decoder().decode(CreateSlotBody.self, from: data)
        XCTAssertEqual(decoded, original)
    }

    func testCallerStableKeyWinsAndConflictingLegacyKeyFailsClosed() throws {
        let requested = UUID()
        XCTAssertEqual(try resolveDurableMutationKey(requested: requested, legacy: nil), requested)
        XCTAssertEqual(
            try resolveDurableMutationKey(requested: requested, legacy: StoredMutationKey(key: requested, touchedAt: Date())),
            requested
        )
        XCTAssertThrowsError(try resolveDurableMutationKey(
            requested: requested,
            legacy: StoredMutationKey(key: UUID(), touchedAt: Date())
        ))
    }

    private func slot(
        state: SlotState,
        viewer: SlotViewerState,
        version: Int64,
        acceptedCount: Int = 0
    ) -> SlotModel {
        let now = Date()
        return SlotModel(
            id: slotID,
            organizer: SlotOrganizer(id: organizerID, username: "alice", displayName: "Alice", avatarUrl: nil),
            title: "Coffee",
            activity: "coffee",
            details: nil,
            placeText: "Podil",
            zoneText: nil,
            canonicalPlaceId: nil,
            startAt: nil,
            capacity: 4,
            acceptedCount: acceptedCount,
            state: state,
            accessMode: .approval,
            visibility: .publicValue,
            viewerState: viewer,
            version: version,
            createdAt: now,
            updatedAt: now
        )
    }
}
