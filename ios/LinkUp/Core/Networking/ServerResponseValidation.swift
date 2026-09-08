import Foundation

extension LinkUpAPI {
    func validatedServerValue<Value: ServerShapeValidatable>(
        _ value: Value,
        context: String
    ) throws -> Value {
        guard value.hasValidServerShape else {
            throw APIError.protocolViolation("Server returned invalid \(context) data.")
        }
        return value
    }

    func validatedServerItems<Value: ServerShapeValidatable>(
        _ values: [Value],
        context: String
    ) throws -> [Value] {
        guard values.allSatisfy(\.hasValidServerShape) else {
            throw APIError.protocolViolation("Server returned invalid \(context) data.")
        }
        return values
    }

    func sendValidatedSlotMutation(
        _ request: APIRequest,
        expectedSlotID: UUID? = nil
    ) async throws -> SlotModel {
        try await sendIdempotent(
            request,
            responseKind: .slot,
            expectedSlotID: expectedSlotID
        ) { (value: SlotModel) in
            if let expectedSlotID, value.id != expectedSlotID {
                throw APIError.protocolViolation("Server mutation response belongs to a different Slot.")
            }
            guard value.hasValidServerShape else {
                throw APIError.protocolViolation("Server returned invalid Slot mutation data.")
            }
        }
    }

    func sendValidatedChatMutation(
        _ request: APIRequest,
        expectedSlotID: UUID
    ) async throws -> ChatMessage {
        try await sendIdempotent(
            request,
            responseKind: .chat,
            expectedSlotID: expectedSlotID
        ) { (value: ChatMessage) in
            guard value.slotId == expectedSlotID else {
                throw APIError.protocolViolation("Server chat response belongs to a different Slot.")
            }
            guard value.hasValidServerShape else {
                throw APIError.protocolViolation("Server returned invalid chat mutation data.")
            }
        }
    }
}
