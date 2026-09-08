import Foundation

indirect enum JSONValue: Decodable, Equatable, Sendable {
    case object([String: JSONValue])
    case array([JSONValue])
    case string(String)
    case number(Double)
    case bool(Bool)
    case null

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() { self = .null; return }
        if let value = try? container.decode(Bool.self) { self = .bool(value); return }
        if let value = try? container.decode(Double.self) { self = .number(value); return }
        if let value = try? container.decode(String.self) { self = .string(value); return }
        if let value = try? container.decode([String: JSONValue].self) { self = .object(value); return }
        if let value = try? container.decode([JSONValue].self) { self = .array(value); return }
        throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unsupported JSON value.")
    }
}

struct RealtimeEvent: Decodable, Equatable, Sendable, Identifiable {
    let sequence: Int64
    let eventId: UUID
    let eventType: String
    let aggregateType: String
    let aggregateId: UUID
    let subjectUserId: UUID?
    let slotId: UUID?
    let payload: JSONValue
    let occurredAt: Date

    var id: UUID { eventId }

    var hasValidServerShape: Bool {
        guard sequence > 0,
              (3...96).contains(eventType.unicodeScalars.count),
              (1...48).contains(aggregateType.unicodeScalars.count),
              case .object = payload else { return false }
        // Go Event.Valid rejects only time.Time's exact zero value (0001-01-01T00:00:00Z).
        return occurredAt.timeIntervalSince1970 != -62_135_596_800
    }
}

struct RealtimeBatch: Decodable, Equatable, Sendable {
    let cursor: Int64
    let events: [RealtimeEvent]

    func isValid(after: Int64, limit: Int) -> Bool {
        guard after >= 0, cursor >= after, (1...200).contains(limit), events.count <= limit else { return false }
        var previous = after
        var seen = Set<UUID>()
        for event in events {
            guard event.hasValidServerShape,
                  event.sequence > previous,
                  event.sequence <= cursor,
                  seen.insert(event.eventId).inserted else { return false }
            previous = event.sequence
        }
        return true
    }
}

struct RealtimeInvalidationHints: Equatable, Sendable {
    let refreshPulse: Bool
    let refreshProfile: Bool
    let refreshRelationships: Bool
    let slotIDs: Set<UUID>
    let chatSlotIDs: Set<UUID>
}

struct RealtimePull: Equatable, Sendable {
    let fromCursor: Int64
    let nextCursor: Int64
    let events: [RealtimeEvent]
    let hints: RealtimeInvalidationHints
}
