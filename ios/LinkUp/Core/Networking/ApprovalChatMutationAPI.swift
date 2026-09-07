import Foundation

extension LinkUpAPI {
    func approveRequest(_ slotID: UUID, userID: UUID) async throws -> SlotModel {
        try await requestDecision(slotID, userID: userID, decision: "approve")
    }

    func rejectRequest(_ slotID: UUID, userID: UUID) async throws -> SlotModel {
        try await requestDecision(slotID, userID: userID, decision: "reject")
    }

    private func requestDecision(_ slotID: UUID, userID: UUID, decision: String) async throws -> SlotModel {
        try await client.send(APIRequest(
            method: .post,
            path: "/v1/slots/\(uuidPath(slotID))/requests/\(uuidPath(userID))/\(decision)",
            idempotencyKey: UUID()
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
        return try await client.send(APIRequest(
            method: .post,
            path: "/v1/slots/\(uuidPath(slotID))/members/\(uuidPath(userID))/remove",
            body: try encodeBody(ExpectedVersionBody(expectedVersion: expectedVersion)),
            idempotencyKey: UUID()
        ))
    }

    func sendChatMessage(_ slotID: UUID, text: String) async throws -> ChatMessage {
        let normalized = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalized.isEmpty else {
            throw APIError.protocolViolation("Chat message cannot be empty.")
        }
        let fingerprint = slotID.uuidString.lowercased() + "\u{0}" + normalized
        let key = pendingChatKeys[fingerprint] ?? UUID()
        pendingChatKeys[fingerprint] = key
        do {
            let message: ChatMessage = try await client.send(APIRequest(
                method: .post,
                path: "/v1/slots/\(uuidPath(slotID))/chat/messages",
                body: try encodeBody(ChatSendBody(text: normalized)),
                idempotencyKey: key
            ))
            pendingChatKeys.removeValue(forKey: fingerprint)
            return message
        } catch let error as APIError {
            if error.isDefinitiveMutationFailure {
                pendingChatKeys.removeValue(forKey: fingerprint)
            }
            throw error
        }
    }
}

private extension APIError {
    var isDefinitiveMutationFailure: Bool {
        switch self {
        case .unauthorized: true
        case .http(let status, _, _, _):
            (400...499).contains(status) && status != 408 && status != 429
        default: false
        }
    }
}
