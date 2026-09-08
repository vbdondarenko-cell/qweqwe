import Foundation

extension LinkUpAPI {
    func pulse() async throws -> [SlotModel] {
        let response: ItemsEnvelope<SlotModel> = try await client.send(
            APIRequest(method: .get, path: "/v1/pulse")
        )
        return try validatedServerItems(response.items, context: "Pulse Slot")
    }

    func mySlots(_ view: MySlotsView) async throws -> [SlotModel] {
        let response: ItemsEnvelope<SlotModel> = try await client.send(APIRequest(
            method: .get,
            path: "/v1/me/slots",
            queryItems: [URLQueryItem(name: "view", value: view.rawValue)]
        ))
        return try validatedServerItems(response.items, context: "account Slot")
    }

    func slot(_ slotID: UUID) async throws -> SlotModel {
        let value: SlotModel = try await client.send(
            APIRequest(method: .get, path: "/v1/slots/\(uuidPath(slotID))")
        )
        guard value.id == slotID else {
            throw APIError.protocolViolation("Server returned a different Slot identity.")
        }
        return try validatedServerValue(value, context: "Slot")
    }

    func acceptedParticipants(_ slotID: UUID) async throws -> [SlotOrganizer] {
        let response: ItemsEnvelope<SlotOrganizer> = try await client.send(
            APIRequest(method: .get, path: "/v1/slots/\(uuidPath(slotID))/accepted")
        )
        return try validatedServerItems(response.items, context: "accepted participant")
    }

    func pendingRequests(_ slotID: UUID) async throws -> [PendingSlotRequest] {
        let response: ItemsEnvelope<PendingSlotRequest> = try await client.send(
            APIRequest(method: .get, path: "/v1/slots/\(uuidPath(slotID))/requests")
        )
        return try validatedServerItems(response.items, context: "pending request")
    }

    func chatMessages(_ slotID: UUID, limit: Int = 100) async throws -> [ChatMessage] {
        guard (1...100).contains(limit) else {
            throw APIError.protocolViolation("Invalid chat message limit.")
        }
        let response: ItemsEnvelope<ChatMessage> = try await client.send(APIRequest(
            method: .get,
            path: "/v1/slots/\(uuidPath(slotID))/chat/messages",
            queryItems: [URLQueryItem(name: "limit", value: String(limit))]
        ))
        let messages = try validatedServerItems(response.items, context: "chat message")
        guard messages.allSatisfy({ $0.slotId == slotID }) else {
            throw APIError.protocolViolation("Server returned chat messages for a different Slot.")
        }
        return messages
    }
}
