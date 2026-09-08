import Foundation

extension LinkUpAPI {
    func approveRequest(_ slotID: UUID, userID: UUID) async throws -> SlotModel {
        try await requestDecision(slotID, userID: userID, decision: "approve")
    }

    func rejectRequest(_ slotID: UUID, userID: UUID) async throws -> SlotModel {
        try await requestDecision(slotID, userID: userID, decision: "reject")
    }

    private func requestDecision(_ slotID: UUID, userID: UUID, decision: String) async throws -> SlotModel {
        try await sendIdempotent(APIRequest(
            method: .post,
            path: "/v1/slots/\(uuidPath(slotID))/requests/\(uuidPath(userID))/\(decision)"
        ))
    }

    func removeParticipant(
        _ slotID: UUID,
        userID: UUID,
        expectedVersion: Int64
    ) async throws -> SlotModel {
        guard expectedVersion > 0 else {
            throw APIError.protocolViolation("expectedVersion must be positive.")
        }
        return try await sendIdempotent(APIRequest(
            method: .post,
            path: "/v1/slots/\(uuidPath(slotID))/members/\(uuidPath(userID))/remove",
            body: try encodeBody(ExpectedVersionBody(expectedVersion: expectedVersion))
        ))
    }

    func sendChatMessage(_ slotID: UUID, text: String) async throws -> ChatMessage {
        let normalized = InputContracts.trimmed(text)
        guard InputContracts.validChatMessage(normalized) else {
            throw APIError.protocolViolation("Chat message must contain 1...2000 Unicode characters.")
        }
        return try await sendIdempotent(APIRequest(
            method: .post,
            path: "/v1/slots/\(uuidPath(slotID))/chat/messages",
            body: try encodeBody(ChatSendBody(text: normalized))
        ))
    }
}
