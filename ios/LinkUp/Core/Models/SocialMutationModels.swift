import Foundation

struct ChatMessage: Codable, Identifiable, Equatable, Sendable {
    let id: UUID
    let slotId: UUID
    let author: SlotOrganizer
    let text: String
    let createdAt: Date
}

struct CreateSlotBody: Codable, Equatable, Sendable {
    let title: String
    let activity: String
    let details: String?
    let placeText: String
    let zoneText: String?
    let canonicalPlaceId: UUID?
    let startAt: Date?
    let capacity: Int
}

struct EditSlotBody: Encodable, Sendable {
    let expectedVersion: Int64
    let title: String?
    let details: String?
    let placeText: String?
    let zoneText: String?
    let canonicalPlaceId: UUID?
    let clearCanonicalPlaceId: Bool
    let startAt: Date?
    let clearStartAt: Bool
    let capacity: Int?
}

struct ExpectedVersionBody: Encodable, Sendable {
    let expectedVersion: Int64
}

struct ChatSendBody: Encodable, Sendable {
    let text: String
}
