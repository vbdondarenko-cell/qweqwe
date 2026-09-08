import Foundation

actor RealtimeCoordinator {
    private let api: LinkUpAPI
    private let cursors: RealtimeCursorStore
    private var offeredCursorByUser: [UUID: Int64] = [:]

    init(api: LinkUpAPI, cursors: RealtimeCursorStore) {
        self.api = api
        self.cursors = cursors
    }

    func pull(userID: UUID, limit: Int = 100) async throws -> RealtimePull {
        guard (1...200).contains(limit) else {
            throw APIError.protocolViolation("Invalid realtime batch limit.")
        }
        let from = await cursors.load(userID: userID)
        let batch = try await api.pullRealtime(after: from, limit: limit)
        offeredCursorByUser[userID] = max(offeredCursorByUser[userID] ?? from, batch.cursor)
        return RealtimePull(
            fromCursor: from,
            nextCursor: batch.cursor,
            events: batch.events,
            hints: realtimeHints(for: batch.events)
        )
    }

    func acknowledge(userID: UUID, cursor: Int64) async throws {
        guard cursor >= 0 else { throw APIError.protocolViolation("Invalid realtime cursor.") }
        let persisted = await cursors.load(userID: userID)
        guard cursor > persisted else { return }
        let offered = offeredCursorByUser[userID] ?? persisted
        guard cursor <= offered else {
            throw APIError.protocolViolation("Cannot acknowledge an unseen realtime cursor.")
        }
        await cursors.save(userID: userID, cursor: cursor)
        if cursor >= offered { offeredCursorByUser.removeValue(forKey: userID) }
    }

    func reset(userID: UUID) async {
        offeredCursorByUser.removeValue(forKey: userID)
        await cursors.clear(userID: userID)
    }
}

func realtimeHints(for events: [RealtimeEvent]) -> RealtimeInvalidationHints {
    var slotIDs = Set<UUID>()
    var chatSlotIDs = Set<UUID>()
    var refreshPulse = false
    var refreshProfile = false
    var refreshRelationships = false

    for event in events {
        if let slotID = event.slotId { slotIDs.insert(slotID) }
        switch event.eventType {
        case "slot.created", "slot.updated", "slot.state_changed":
            refreshPulse = true
        case "slot.request_created", "slot.request_removed", "slot.membership_added", "slot.membership_removed":
            refreshPulse = true
            refreshRelationships = true
        case "slot.chat_message_created":
            if let slotID = event.slotId { chatSlotIDs.insert(slotID) }
        case "user.block_created", "user.block_removed":
            refreshPulse = true
            refreshProfile = true
            refreshRelationships = true
        case "user.profile_changed":
            refreshProfile = true
        default:
            refreshPulse = true
            if event.slotId != nil { refreshRelationships = true }
        }
    }

    return RealtimeInvalidationHints(
        refreshPulse: refreshPulse,
        refreshProfile: refreshProfile,
        refreshRelationships: refreshRelationships,
        slotIDs: slotIDs,
        chatSlotIDs: chatSlotIDs
    )
}
