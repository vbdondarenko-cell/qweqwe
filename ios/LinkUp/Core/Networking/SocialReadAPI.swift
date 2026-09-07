import Foundation

extension LinkUpAPI {
    func pulse() async throws -> [SlotModel] {
        let response: ItemsEnvelope<SlotModel> = try await client.send(
            APIRequest(method: .get, path: "/v1/pulse")
        )
        return response.items
    }

    func mySlots(_ view: MySlotsView) async throws -> [SlotModel] {
        let response: ItemsEnvelope<SlotModel> = try await client.send(APIRequest(
            method: .get,
            path: "/v1/me/slots",
            queryItems: [URLQueryItem(name: "view", value: view.rawValue)]
        ))
        return response.items
    }

    func slot(_ slotID: UUID) async throws -> SlotModel {
        try await client.send(APIRequest(method: .get, path: "/v1/slots/\(uuidPath(slotID))"))
    }

    func acceptedParticipants(_ slotID: UUID) async throws -> [SlotOrganizer] {
        let response: ItemsEnvelope<SlotOrganizer> = try await client.send(
            APIRequest(method: .get, path: "/v1/slots/\(uuidPath(slotID))/accepted")
        )
        return response.items
    }

    func pendingRequests(_ slotID: UUID) async throws -> [PendingSlotRequest] {
        let response: ItemsEnvelope<PendingSlotRequest> = try await client.send(
            APIRequest(method: .get, path: "/v1/slots/\(uuidPath(slotID))/requests")
        )
        return response.items
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
        return response.items
    }
}
