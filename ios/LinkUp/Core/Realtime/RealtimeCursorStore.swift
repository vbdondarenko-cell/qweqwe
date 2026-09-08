import Foundation

actor RealtimeCursorStore {
    private let defaults: UserDefaults
    private let keyPrefix = "linkup.realtime.cursor.v1."

    init(suiteName: String? = nil) {
        if let suiteName, let isolated = UserDefaults(suiteName: suiteName) {
            self.defaults = isolated
        } else {
            self.defaults = .standard
        }
    }

    func load(userID: UUID) -> Int64 {
        let value = (defaults.object(forKey: key(userID)) as? NSNumber)?.int64Value ?? 0
        return max(0, value)
    }

    func save(userID: UUID, cursor: Int64) {
        guard cursor >= 0 else { return }
        let storageKey = key(userID)
        let current = (defaults.object(forKey: storageKey) as? NSNumber)?.int64Value ?? 0
        guard cursor > current else { return }
        defaults.set(NSNumber(value: cursor), forKey: storageKey)
    }

    func clear(userID: UUID) {
        defaults.removeObject(forKey: key(userID))
    }

    private func key(_ userID: UUID) -> String {
        keyPrefix + userID.uuidString.lowercased()
    }
}
