import Foundation
import Security

enum KeychainStoreError: Error, Sendable {
    case invalidCredential
    case status(OSStatus)
}

actor KeychainSessionStore {
    private let service = "com.linkup.app.session"
    private let account = "bearer.v1"

    func save(_ credential: SessionCredential) throws {
        guard credential.hasValidTokenShape, !credential.isExpired else {
            throw KeychainStoreError.invalidCredential
        }
        let data = try JSONEncoder().encode(credential)
        let query = baseQuery()
        let values: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        ]
        let updated = SecItemUpdate(query as CFDictionary, values as CFDictionary)
        if updated == errSecSuccess { return }
        guard updated == errSecItemNotFound else { throw KeychainStoreError.status(updated) }
        var insert = query
        values.forEach { insert[$0.key] = $0.value }
        let added = SecItemAdd(insert as CFDictionary, nil)
        guard added == errSecSuccess else { throw KeychainStoreError.status(added) }
    }

    func load() throws -> SessionCredential? {
        var query = baseQuery()
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        if status == errSecItemNotFound { return nil }
        guard status == errSecSuccess, let data = result as? Data else {
            throw KeychainStoreError.status(status)
        }
        guard let credential = try? JSONDecoder().decode(SessionCredential.self, from: data),
              credential.hasValidTokenShape, !credential.isExpired else {
            try clear()
            return nil
        }
        return credential
    }

    func clear() throws {
        let status = SecItemDelete(baseQuery() as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainStoreError.status(status)
        }
    }

    private func baseQuery() -> [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]
    }
}
