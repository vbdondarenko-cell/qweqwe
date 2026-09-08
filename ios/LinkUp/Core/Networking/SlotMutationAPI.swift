import Foundation

extension LinkUpAPI {
    func createSlot(_ body: CreateSlotBody) async throws -> SlotModel {
        try await sendValidatedSlotMutation(APIRequest(
            method: .post,
            path: "/v1/slots",
            body: try encodeBody(body)
        ))
    }

    func editSlot(_ slotID: UUID, body: EditSlotBody) async throws -> SlotModel {
        guard body.expectedVersion > 0 else {
            throw APIError.protocolViolation("expectedVersion must be positive.")
        }
        guard body.canonicalPlaceId == nil || !body.clearCanonicalPlaceId else {
            throw APIError.protocolViolation("canonicalPlaceId cannot be set and cleared together.")
        }
        guard body.startAt == nil || !body.clearStartAt else {
            throw APIError.protocolViolation("startAt cannot be set and cleared together.")
        }
        return try await sendValidatedSlotMutation(APIRequest(
            method: .patch,
            path: "/v1/slots/\(uuidPath(slotID))",
            body: try encodeBody(body)
        ), expectedSlotID: slotID)
    }

    func cancelSlot(_ slotID: UUID, expectedVersion: Int64) async throws -> SlotModel {
        guard expectedVersion > 0 else {
            throw APIError.protocolViolation("expectedVersion must be positive.")
        }
        return try await sendValidatedSlotMutation(APIRequest(
            method: .post,
            path: "/v1/slots/\(uuidPath(slotID))/cancel",
            body: try encodeBody(ExpectedVersionBody(expectedVersion: expectedVersion))
        ), expectedSlotID: slotID)
    }

    func requestSlot(_ slotID: UUID) async throws -> SlotModel {
        try await mutation(slotID, suffix: "request")
    }

    func leaveSlot(_ slotID: UUID) async throws -> SlotModel {
        try await mutation(slotID, suffix: "leave")
    }

    func startSlot(_ slotID: UUID) async throws -> SlotModel {
        try await mutation(slotID, suffix: "start")
    }

    func completeSlot(_ slotID: UUID) async throws -> SlotModel {
        try await mutation(slotID, suffix: "complete")
    }

    private func mutation(_ slotID: UUID, suffix: String) async throws -> SlotModel {
        try await sendValidatedSlotMutation(APIRequest(
            method: .post,
            path: "/v1/slots/\(uuidPath(slotID))/\(suffix)"
        ), expectedSlotID: slotID)
    }
}
