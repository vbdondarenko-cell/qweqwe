import XCTest
@testable import LinkUp

final class ServerModelContractsTests: XCTestCase {
    private let userID = UUID()
    private let slotID = UUID()

    private var organizer: SlotOrganizer {
        SlotOrganizer(id: userID, username: "alice", displayName: "Alice", avatarUrl: nil)
    }

    private func slot(
        capacity: Int = 100,
        acceptedCount: Int = 1,
        version: Int64 = 1,
        canonicalPlaceId: UUID? = nil,
        state: SlotState = .filling,
        accessMode: SlotAccessMode = .approval,
        viewerState: SlotViewerState = .none
    ) -> SlotModel {
        let now = Date()
        return SlotModel(
            id: slotID,
            organizer: organizer,
            title: "Coffee",
            activity: "coffee",
            details: "Meet up",
            placeText: "Kontraktova Square",
            zoneText: nil,
            canonicalPlaceId: canonicalPlaceId,
            startAt: now.addingTimeInterval(3_600),
            capacity: capacity,
            acceptedCount: acceptedCount,
            state: state,
            accessMode: accessMode,
            visibility: .publicValue,
            viewerState: viewerState,
            version: version,
            createdAt: now,
            updatedAt: now
        )
    }

    func testPulseDiscoverabilityMatchesServerLifecycleFilter() {
        XCTAssertFalse(SlotState.draft.isPulseDiscoverable)
        XCTAssertTrue(SlotState.published.isPulseDiscoverable)
        XCTAssertTrue(SlotState.filling.isPulseDiscoverable)
        XCTAssertTrue(SlotState.full.isPulseDiscoverable)
        XCTAssertFalse(SlotState.active.isPulseDiscoverable)
        XCTAssertFalse(SlotState.completed.isPulseDiscoverable)
        XCTAssertFalse(SlotState.cancelled.isPulseDiscoverable)
        XCTAssertFalse(SlotState.expired.isPulseDiscoverable)
        XCTAssertFalse(SlotState.moderated.isPulseDiscoverable)
    }

    func testRequestAcceptanceMatchesServerLifecycleFilter() {
        XCTAssertFalse(SlotState.draft.acceptsNewRequests)
        XCTAssertTrue(SlotState.published.acceptsNewRequests)
        XCTAssertTrue(SlotState.filling.acceptsNewRequests)
        XCTAssertFalse(SlotState.full.acceptsNewRequests)
        XCTAssertFalse(SlotState.active.acceptsNewRequests)
        XCTAssertFalse(SlotState.completed.acceptsNewRequests)
        XCTAssertFalse(SlotState.cancelled.acceptsNewRequests)
        XCTAssertFalse(SlotState.expired.acceptsNewRequests)
        XCTAssertFalse(SlotState.moderated.acceptsNewRequests)
    }

    func testRequestActionRequiresEligibleViewerAccessStateAndCapacity() {
        XCTAssertTrue(slot(state: .published).canRequestToJoin)
        XCTAssertTrue(slot(state: .filling).canRequestToJoin)
        XCTAssertFalse(slot(state: .full).canRequestToJoin)
        XCTAssertFalse(slot(capacity: 2, acceptedCount: 2, state: .filling).canRequestToJoin)
        XCTAssertFalse(slot(state: .active).canRequestToJoin)
        XCTAssertFalse(slot(state: .filling, accessMode: .instant).canRequestToJoin)
        XCTAssertFalse(slot(state: .filling, viewerState: .pending).canRequestToJoin)
        XCTAssertFalse(slot(state: .filling, viewerState: .accepted).canRequestToJoin)
        XCTAssertFalse(slot(state: .filling, viewerState: .host).canRequestToJoin)
    }

    func testHostActionsMatchServerLifecycleAndParticipantRules() {
        XCTAssertTrue(slot(state: .published, viewerState: .host).canHostEdit)
        XCTAssertTrue(slot(state: .filling, viewerState: .host).canHostEdit)
        XCTAssertTrue(slot(state: .full, viewerState: .host).canHostEdit)
        XCTAssertFalse(slot(state: .active, viewerState: .host).canHostEdit)
        XCTAssertFalse(slot(state: .filling, viewerState: .none).canHostEdit)

        XCTAssertTrue(slot(acceptedCount: 1, state: .filling, viewerState: .host).canHostStart)
        XCTAssertTrue(slot(acceptedCount: 1, state: .full, viewerState: .host).canHostStart)
        XCTAssertFalse(slot(acceptedCount: 0, state: .filling, viewerState: .host).canHostStart)
        XCTAssertFalse(slot(acceptedCount: 1, state: .published, viewerState: .host).canHostStart)
        XCTAssertFalse(slot(acceptedCount: 1, state: .filling, viewerState: .none).canHostStart)
    }

    func testChatAccessMatchesServerStateAndViewerAuthority() {
        for state in [SlotState.filling, .full, .active] {
            XCTAssertTrue(slot(state: state, viewerState: .host).canUseChat)
            XCTAssertTrue(slot(state: state, viewerState: .accepted).canUseChat)
            XCTAssertFalse(slot(state: state, viewerState: .pending).canUseChat)
            XCTAssertFalse(slot(state: state, viewerState: .none).canUseChat)
        }
        XCTAssertFalse(slot(state: .draft, viewerState: .host).canUseChat)
        XCTAssertFalse(slot(state: .published, viewerState: .host).canUseChat)
        XCTAssertFalse(slot(state: .completed, viewerState: .host).canUseChat)
    }

    func testHostManagementMatchesAcceptedRosterLifecycle() {
        XCTAssertFalse(slot(state: .draft, viewerState: .host).canManageParticipants)
        XCTAssertTrue(slot(state: .published, viewerState: .host).canManageParticipants)
        XCTAssertTrue(slot(state: .filling, viewerState: .host).canManageParticipants)
        XCTAssertTrue(slot(state: .full, viewerState: .host).canManageParticipants)
        XCTAssertTrue(slot(state: .active, viewerState: .host).canManageParticipants)
        XCTAssertFalse(slot(state: .completed, viewerState: .host).canManageParticipants)
        XCTAssertFalse(slot(state: .filling, viewerState: .accepted).canManageParticipants)
    }

    func testLeaveRelationshipRequiresCurrentNonterminalRelationship() {
        XCTAssertTrue(slot(state: .published, viewerState: .pending).canLeaveRelationship)
        XCTAssertTrue(slot(state: .filling, viewerState: .accepted).canLeaveRelationship)
        XCTAssertTrue(slot(state: .active, viewerState: .accepted).canLeaveRelationship)
        XCTAssertFalse(slot(state: .completed, viewerState: .accepted).canLeaveRelationship)
        XCTAssertFalse(slot(state: .cancelled, viewerState: .pending).canLeaveRelationship)
        XCTAssertFalse(slot(state: .filling, viewerState: .none).canLeaveRelationship)
        XCTAssertFalse(slot(state: .filling, viewerState: .host).canLeaveRelationship)
    }

    func testSlotShapeAcceptsServiceMaximumCapacity() {
        XCTAssertEqual(InputContracts.slotCapacityMax, 100)
        XCTAssertTrue(slot(capacity: 100).hasValidServerShape)
    }

    func testSlotShapeRejectsImpossibleCapacityCountAndVersion() {
        XCTAssertFalse(slot(capacity: 101).hasValidServerShape)
        XCTAssertFalse(slot(capacity: 10, acceptedCount: 11).hasValidServerShape)
        XCTAssertFalse(slot(version: 0).hasValidServerShape)
    }

    func testUserProfileShapeUsesAccountContract() {
        let valid = UserProfile(
            id: userID,
            email: "a@example.com",
            username: "alice",
            displayName: "Alice",
            avatarUrl: nil,
            profileVisibility: "PUBLIC",
            language: "uk"
        )
        XCTAssertTrue(valid.hasValidServerShape)

        let invalid = UserProfile(
            id: userID,
            email: "a@example.com",
            username: "alice",
            displayName: "Alice",
            avatarUrl: nil,
            profileVisibility: "PUBLIC",
            language: "de"
        )
        XCTAssertFalse(invalid.hasValidServerShape)
    }

    func testCanonicalPlaceShapeMatchesDatabaseConstraints() {
        let valid = PlaceModel(
            id: UUID(),
            name: "Kontraktova Square",
            category: "square",
            locality: "Kyiv",
            countryCode: "UA",
            latitudeE6: 50_465_300,
            longitudeE6: 30_517_800,
            precisionM: 100
        )
        XCTAssertTrue(valid.hasValidServerShape)

        let invalidCoordinate = PlaceModel(
            id: UUID(),
            name: "Broken",
            category: nil,
            locality: nil,
            countryCode: "UA",
            latitudeE6: 91_000_000,
            longitudeE6: 30_000_000,
            precisionM: 100
        )
        XCTAssertFalse(invalidCoordinate.hasValidServerShape)

        let invalidPrecision = PlaceModel(
            id: UUID(),
            name: "Broken",
            category: nil,
            locality: nil,
            countryCode: "UA",
            latitudeE6: 50_000_000,
            longitudeE6: 30_000_000,
            precisionM: 0
        )
        XCTAssertFalse(invalidPrecision.hasValidServerShape)
    }

    func testMapClusterRequiresConcreteIdentityOnlyForSinglePlaceCluster() {
        let placeID = UUID()
        let single = MapCluster(
            key: "1:1",
            latitudeE6: 50_000_000,
            longitudeE6: 30_000_000,
            placeCount: 1,
            slotCount: 2,
            placeId: placeID,
            placeName: "Place"
        )
        XCTAssertTrue(single.hasValidServerShape)

        let aggregate = MapCluster(
            key: "1:1",
            latitudeE6: 50_000_000,
            longitudeE6: 30_000_000,
            placeCount: 2,
            slotCount: 4,
            placeId: nil,
            placeName: nil
        )
        XCTAssertTrue(aggregate.hasValidServerShape)

        let impossibleAggregate = MapCluster(
            key: "1:1",
            latitudeE6: 50_000_000,
            longitudeE6: 30_000_000,
            placeCount: 2,
            slotCount: 4,
            placeId: placeID,
            placeName: "Leaked identity"
        )
        XCTAssertFalse(impossibleAggregate.hasValidServerShape)
    }

    func testChatMessageShapeUsesSameRuneLimit() {
        let message = ChatMessage(
            id: UUID(),
            slotId: slotID,
            author: organizer,
            text: "hello",
            createdAt: Date()
        )
        XCTAssertTrue(message.hasValidServerShape)

        let oversized = ChatMessage(
            id: UUID(),
            slotId: slotID,
            author: organizer,
            text: String(repeating: "x", count: InputContracts.chatMessageMaxScalars + 1),
            createdAt: Date()
        )
        XCTAssertFalse(oversized.hasValidServerShape)
    }
}
