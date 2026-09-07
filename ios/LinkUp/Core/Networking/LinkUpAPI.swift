import Foundation

actor LinkUpAPI {
    let client: APIClient
    let credentials: KeychainSessionStore
    var pendingChatKeys: [String: UUID] = [:]

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

    func saveCredential(from envelope: AuthEnvelope) async throws {
        do {
            try await credentials.save(SessionCredential(token: envelope.token, expiresAt: envelope.expiresAt))
        } catch {
            throw APIError.secureStorageUnavailable
        }
    }

    func clearLocalSession() async {
        pendingChatKeys.removeAll(keepingCapacity: false)
        await credentials.clear()
    }
}
