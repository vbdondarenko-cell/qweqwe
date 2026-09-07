import CryptoKit
import Foundation

struct StoredMutationKey: Codable, Equatable {
    let key: UUID
    let touchedAt: Date
}

enum MutationIdentity {
    static func digest(for request: APIRequest) -> String {
        var canonical = Data()
        append(request.method.rawValue, to: &canonical)
        append(request.path, to: &canonical)

        let query = request.queryItems
            .map { ($0.name, $0.value ?? "") }
            .sorted { lhs, rhs in
                lhs.0 == rhs.0 ? lhs.1 < rhs.1 : lhs.0 < rhs.0
            }
        for (name, value) in query {
            append(name, to: &canonical)
            append(value, to: &canonical)
        }

        canonical.append(request.body ?? Data())
        let hash = SHA256.hash(data: canonical)
        return hash.map { String(format: "%02x", $0) }.joined()
    }

    private static func append(_ value: String, to data: inout Data) {
        data.append(contentsOf: value.utf8)
        data.append(0)
    }
}

struct MutationKeyStore {
    private let defaults: UserDefaults
    private let storageKey: String
    private let maxEntries: Int

    init(
        defaults: UserDefaults = .standard,
        storageKey: String = "com.linkup.app.pending-mutations.v1",
        maxEntries: Int = 256
    ) {
        precondition(maxEntries > 0)
        self.defaults = defaults
        self.storageKey = storageKey
        self.maxEntries = maxEntries
    }

    func load() -> [String: StoredMutationKey] {
        guard let data = defaults.data(forKey: storageKey),
              let decoded = try? JSONDecoder().decode([String: StoredMutationKey].self, from: data) else {
            return [:]
        }
        return pruned(decoded)
    }

    func persist(_ entries: [String: StoredMutationKey]) {
        let bounded = pruned(entries)
        guard !bounded.isEmpty else {
            defaults.removeObject(forKey: storageKey)
            return
        }
        guard let data = try? JSONEncoder().encode(bounded) else { return }
        defaults.set(data, forKey: storageKey)
    }

    func pruned(_ entries: [String: StoredMutationKey]) -> [String: StoredMutationKey] {
        guard entries.count > maxEntries else { return entries }
        return Dictionary(uniqueKeysWithValues: entries
            .sorted { $0.value.touchedAt > $1.value.touchedAt }
            .prefix(maxEntries)
            .map { ($0.key, $0.value) })
    }
}
