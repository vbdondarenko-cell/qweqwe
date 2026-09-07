import Foundation

struct SlotOrganizer: Codable, Identifiable, Equatable, Sendable {
    let id: UUID
    let username: String
    let displayName: String
    let avatarUrl: String?
}

struct SlotModel: Codable, Identifiable, Equatable, Sendable {
    let id: UUID
    let organizer: SlotOrganizer
    let title: String
    let activity: String
    let details: String?
    let placeText: String
    let zoneText: String?
    let canonicalPlaceId: UUID?
    let startAt: Date?
    let capacity: Int
    let acceptedCount: Int
    let state: SlotState
    let accessMode: SlotAccessMode
    let visibility: SlotVisibility
    let viewerState: SlotViewerState
    let version: Int64
    let createdAt: Date
    let updatedAt: Date

    var isTerminal: Bool { state.isTerminal }
    var remainingCapacity: Int { max(0, capacity - acceptedCount) }
}

struct PendingSlotRequest: Codable, Equatable, Sendable {
    let user: SlotOrganizer
    let requestedAt: Date
}

struct ItemsEnvelope<Item: Decodable & Sendable>: Decodable, Sendable {
    let items: [Item]
}
