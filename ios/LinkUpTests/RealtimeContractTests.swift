import XCTest
@testable import LinkUp

final class RealtimeContractTests: XCTestCase {
    private let slotID = UUID(uuidString: "5f1c1694-f4a7-4ca5-8433-ec70cfb7e6b6")!
    private let eventA = UUID(uuidString: "11111111-1111-1111-1111-111111111111")!
    private let eventB = UUID(uuidString: "22222222-2222-2222-2222-222222222222")!

    func testBatchDecodesAndValidatesMonotonicViewerFeed() throws {
        let data = Data(#"{
          "cursor":12,
          "events":[
            {
              "sequence":11,
              "eventId":"11111111-1111-1111-1111-111111111111",
              "eventType":"slot.updated",
              "aggregateType":"slot",
              "aggregateId":"5f1c1694-f4a7-4ca5-8433-ec70cfb7e6b6",
              "subjectUserId":null,
              "slotId":"5f1c1694-f4a7-4ca5-8433-ec70cfb7e6b6",
              "payload":{"state":"FILLING","version":4},
              "occurredAt":"2026-09-08T07:00:00Z"
            },
            {
              "sequence":12,
              "eventId":"22222222-2222-2222-2222-222222222222",
              "eventType":"slot.chat_message_created",
              "aggregateType":"message",
              "aggregateId":"33333333-3333-3333-3333-333333333333",
              "slotId":"5f1c1694-f4a7-4ca5-8433-ec70cfb7e6b6",
              "payload":{"messageId":"33333333-3333-3333-3333-333333333333"},
              "occurredAt":"2026-09-08T07:00:01Z"
            }
          ]
        }"#.utf8)

        let batch = try APICoding.decoder().decode(RealtimeBatch.self, from: data)
        XCTAssertTrue(batch.isValid(after: 10, limit: 100))
        XCTAssertEqual(batch.events.map(\.eventId), [eventA, eventB])
        XCTAssertEqual(batch.events.first?.slotId, slotID)
    }

    func testBatchRejectsDuplicateIDsOutOfOrderAndCursorRegression() {
        let first = event(sequence: 11, id: eventA, type: "slot.updated", slotID: slotID)
        let duplicate = event(sequence: 12, id: eventA, type: "slot.updated", slotID: slotID)
        XCTAssertFalse(RealtimeBatch(cursor: 12, events: [first, duplicate]).isValid(after: 10, limit: 100))

        let earlier = event(sequence: 10, id: eventB, type: "slot.updated", slotID: slotID)
        XCTAssertFalse(RealtimeBatch(cursor: 12, events: [first, earlier]).isValid(after: 10, limit: 100))
        XCTAssertFalse(RealtimeBatch(cursor: 9, events: []).isValid(after: 10, limit: 100))
    }

    func testBatchRejectsNonObjectPayload() {
        let invalid = RealtimeEvent(
            sequence: 1,
            eventId: eventA,
            eventType: "slot.updated",
            aggregateType: "slot",
            aggregateId: slotID,
            subjectUserId: nil,
            slotId: slotID,
            payload: .array([]),
            occurredAt: Date()
        )
        XCTAssertFalse(invalid.hasValidServerShape)
        XCTAssertFalse(RealtimeBatch(cursor: 1, events: [invalid]).isValid(after: 0, limit: 100))
    }

    func testInvalidationHintsNeverTreatPayloadAsAuthority() {
        let events = [
            event(sequence: 1, id: eventA, type: "slot.request_created", slotID: slotID),
            event(sequence: 2, id: eventB, type: "slot.chat_message_created", slotID: slotID),
            event(sequence: 3, id: UUID(), type: "user.profile_changed", slotID: nil)
        ]
        let hints = realtimeHints(for: events)
        XCTAssertTrue(hints.refreshPulse)
        XCTAssertTrue(hints.refreshProfile)
        XCTAssertTrue(hints.refreshRelationships)
        XCTAssertEqual(hints.slotIDs, Set([slotID]))
        XCTAssertEqual(hints.chatSlotIDs, Set([slotID]))
    }

    func testBlockEventInvalidatesRelationshipsWithoutLeakingSlotIdentity() {
        let hints = realtimeHints(for: [
            event(sequence: 1, id: eventA, type: "user.block_created", slotID: nil)
        ])
        XCTAssertTrue(hints.refreshPulse)
        XCTAssertTrue(hints.refreshProfile)
        XCTAssertTrue(hints.refreshRelationships)
        XCTAssertTrue(hints.slotIDs.isEmpty)
    }

    func testUnknownSlotEventFallsBackToCanonicalRefresh() {
        let hints = realtimeHints(for: [
            event(sequence: 1, id: eventA, type: "future.slot_event", slotID: slotID)
        ])
        XCTAssertTrue(hints.refreshPulse)
        XCTAssertTrue(hints.refreshRelationships)
        XCTAssertEqual(hints.slotIDs, Set([slotID]))
    }

    @MainActor
    func testCursorStoreIsPerUserAndMonotonic() async {
        let suite = "LinkUpTests.Realtime.\(UUID().uuidString)"
        guard UserDefaults(suiteName: suite) != nil else {
            XCTFail("Unable to create isolated UserDefaults")
            return
        }
        defer { UserDefaults(suiteName: suite)?.removePersistentDomain(forName: suite) }

        let store = RealtimeCursorStore(suiteName: suite)
        let userA = UUID()
        let userB = UUID()
        let initialA = await store.load(userID: userA)
        XCTAssertEqual(initialA, 0)
        await store.save(userID: userA, cursor: 7)
        await store.save(userID: userA, cursor: 3)
        await store.save(userID: userB, cursor: 2)
        let savedA = await store.load(userID: userA)
        let savedB = await store.load(userID: userB)
        XCTAssertEqual(savedA, 7)
        XCTAssertEqual(savedB, 2)
        await store.clear(userID: userA)
        let clearedA = await store.load(userID: userA)
        XCTAssertEqual(clearedA, 0)
    }

    private func event(sequence: Int64, id: UUID, type: String, slotID: UUID?) -> RealtimeEvent {
        RealtimeEvent(
            sequence: sequence,
            eventId: id,
            eventType: type,
            aggregateType: slotID == nil ? "user" : "slot",
            aggregateId: slotID ?? UUID(),
            subjectUserId: nil,
            slotId: slotID,
            payload: .object([:]),
            occurredAt: Date()
        )
    }
}
