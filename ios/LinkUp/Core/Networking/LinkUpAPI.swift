import Foundation

actor LinkUpAPI {
    let client: APIClient
    let credentials: KeychainSessionStore
    private var pendingMutationKeys: [String: UUID] = [:]

    init(client: APIClient, credentials: KeychainSessionStore) {
        self.client = client
        self.credentials = credentials
    }

    func encodeBody<Value: Encodable>(_ value: Value) throws -> Data {
        try APICoding.encoder().encode(value)
    }

    func uuidPath(_ value: UUID) -> String {
        value.uuidString.lowercased()
    }

    func sendIdempotent<Response: Decodable & Sendable>(
        _ request: APIRequest,
        as type: Response.Type = Response.self
    ) async throws -> Response {
        guard request.method != .get else {
            throw APIError.protocolViolation("Idempotent mutation helper cannot send GET requests.")
        }
        let fingerprint = mutationFingerprint(request)
        let key = pendingMutationKeys[fingerprint] ?? UUID()
        pendingMutationKeys[fingerprint] = key

        var keyedRequest = request
        keyedRequest.idempotencyKey = key
        do {
            let response: Response = try await client.send(keyedRequest, as: type)
            pendingMutationKeys.removeValue(forKey: fingerprint)
            return response
        } catch is CancellationError {
            // Cancellation after bytes leave the device is ambiguous: keep the key for retry.
            throw CancellationError()
        } catch let error as APIError {
            if error.isDefinitiveMutationFailure {
                pendingMutationKeys.removeValue(forKey: fingerprint)
            }
            throw error
        }
    }

    func saveCredential(from envelope: AuthEnvelope) async throws {
        do {
            try await credentials.save(SessionCredential(token: envelope.token, expiresAt: envelope.expiresAt))
        } catch {
            throw APIError.secureStorageUnavailable
        }
    }

    func clearLocalSession() async {
        pendingMutationKeys.removeAll(keepingCapacity: false)
        await credentials.clear()
    }

    private func mutationFingerprint(_ request: APIRequest) -> String {
        let query = request.queryItems
            .map { "\($0.name)=\($0.value ?? "")" }
            .joined(separator: "&")
        let body = request.body?.base64EncodedString() ?? ""
        return [request.method.rawValue, request.path, query, body].joined(separator: "\u{0}")
    }
}
