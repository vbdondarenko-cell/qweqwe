import Foundation

actor LinkUpAPI {
    let client: APIClient
    let credentials: KeychainSessionStore

    private let mutationStore: MutationKeyStore
    private var pendingMutationKeys: [String: StoredMutationKey]

    init(client: APIClient, credentials: KeychainSessionStore) {
        self.client = client
        self.credentials = credentials
        let mutationStore = MutationKeyStore()
        self.mutationStore = mutationStore
        self.pendingMutationKeys = mutationStore.load()
    }

    func encodeBody<Value: Encodable>(_ value: Value) throws -> Data {
        try APICoding.encoder().encode(value)
    }

    func uuidPath(_ value: UUID) -> String {
        value.uuidString.lowercased()
    }

    func sendIdempotent<Response: Decodable & Sendable>(
        _ request: APIRequest,
        as type: Response.Type = Response.self,
        validate: @Sendable (Response) throws -> Void = { _ in }
    ) async throws -> Response {
        guard request.method != .get else {
            throw APIError.protocolViolation("Idempotent mutation helper cannot send GET requests.")
        }

        let identity = MutationIdentity.digest(for: request)
        let now = Date()
        let key = pendingMutationKeys[identity]?.key ?? UUID()
        pendingMutationKeys[identity] = StoredMutationKey(key: key, touchedAt: now)
        pendingMutationKeys = mutationStore.pruned(pendingMutationKeys)
        mutationStore.persist(pendingMutationKeys)

        var keyedRequest = request
        keyedRequest.idempotencyKey = key
        do {
            let response: Response = try await client.send(keyedRequest, as: type)
            try validate(response)
            releaseMutationKey(identity)
            return response
        } catch is CancellationError {
            // Cancellation after bytes leave the device is ambiguous: retain the persisted key for retry/process death.
            throw CancellationError()
        } catch let error as APIError {
            if error.isDefinitiveMutationFailure {
                releaseMutationKey(identity)
            }
            throw error
        }
    }

    func saveCredential(from envelope: AuthEnvelope) async throws {
        guard OpaqueTokenContract.canonical32ByteBase64URL(envelope.token) != nil,
              envelope.expiresAt > Date() else {
            throw APIError.protocolViolation("Server returned an invalid session credential.")
        }
        do {
            try await credentials.save(SessionCredential(token: envelope.token, expiresAt: envelope.expiresAt))
        } catch {
            throw APIError.secureStorageUnavailable
        }
    }

    func clearLocalSession() async {
        await credentials.clear()
    }

    private func releaseMutationKey(_ identity: String) {
        pendingMutationKeys.removeValue(forKey: identity)
        mutationStore.persist(pendingMutationKeys)
    }
}
