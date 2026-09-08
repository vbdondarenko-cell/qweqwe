import Foundation

extension LinkUpAPI {
    func createDraftSlot(_ body: CreateSlotBody, idempotencyKey: UUID) async throws -> SlotModel {
        try await createDraftSlot(encodedBody: encodeBody(body), idempotencyKey: idempotencyKey)
    }

    func createDraftSlot(encodedBody: Data, idempotencyKey: UUID) async throws -> SlotModel {
        guard !encodedBody.isEmpty, encodedBody.count <= durableMutationMaxBodyBytes else {
            throw APIError.protocolViolation("Draft creation body exceeds the durable safety limit.")
        }
        return try await sendValidatedSlotMutation(APIRequest(
            method: .post,
            path: "/v1/slots/drafts",
            body: encodedBody,
            idempotencyKey: idempotencyKey
        ), additionalValidation: { try validateDraftCreateReplay($0) })
    }

    func publishDraftSlot(
        _ slotID: UUID,
        expectedVersion: Int64,
        idempotencyKey: UUID
    ) async throws -> SlotModel {
        guard expectedVersion > 0 else {
            throw APIError.protocolViolation("expectedVersion must be positive.")
        }
        return try await sendValidatedSlotMutation(APIRequest(
            method: .post,
            path: "/v1/slots/\(uuidPath(slotID))/publish",
            body: try encodeBody(ExpectedVersionBody(expectedVersion: expectedVersion)),
            idempotencyKey: idempotencyKey
        ), expectedSlotID: slotID) { value in
            try validateDraftPublishReplay(value, expectedVersion: expectedVersion)
        }
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

    func cancelSlot(
        _ slotID: UUID,
        expectedVersion: Int64,
        idempotencyKey: UUID? = nil
    ) async throws -> SlotModel {
        guard expectedVersion > 0 else {
            throw APIError.protocolViolation("expectedVersion must be positive.")
        }
        return try await sendValidatedSlotMutation(APIRequest(
            method: .post,
            path: "/v1/slots/\(uuidPath(slotID))/cancel",
            body: try encodeBody(ExpectedVersionBody(expectedVersion: expectedVersion)),
            idempotencyKey: idempotencyKey
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


func validateDraftCreateReplay(_ slot: SlotModel) throws {
    guard slot.viewerState == .host,
          slot.accessMode == .approval,
          slot.visibility == .publicValue else {
        throw APIError.protocolViolation("Draft creation replay returned an unauthorized Slot shape.")
    }
    if slot.state == .draft, slot.acceptedCount != 0 {
        throw APIError.protocolViolation("Draft Slot cannot contain accepted participants.")
    }
}

func validateDraftPublishReplay(_ slot: SlotModel, expectedVersion: Int64) throws {
    guard expectedVersion > 0,
          slot.viewerState == .host,
          slot.accessMode == .approval,
          slot.visibility == .publicValue,
          slot.state != .draft,
          slot.version > expectedVersion else {
        throw APIError.protocolViolation("Draft publish replay did not confirm a published Slot transition.")
    }
}
