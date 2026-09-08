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
        canonicalPlaceId: UUID? = nil
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
            state: .filling,
            accessMode: .approval,
            visibility: .publicValue,
            viewerState: .none,
            version: version,
            createdAt: now,
            updatedAt: now
        )
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
