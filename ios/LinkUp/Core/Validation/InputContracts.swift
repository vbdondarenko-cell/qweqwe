import Foundation

enum InputContracts {
    static let slotCapacityMin = 2
    static let slotCapacityMax = 100
    static let slotTitleMaxScalars = 120
    static let slotDetailsMaxScalars = 2_000
    static let slotPlaceMaxScalars = 240
    static let chatMessageMaxScalars = 2_000
    static let placeSearchMinScalars = 2
    static let placeSearchMaxScalars = 80
    static let placeSearchLocalityMaxScalars = 120
    static let profileDisplayNameMaxScalars = 80
    static let avatarURLMaxUTF8Bytes = 2_048
    static let accountEmailMinUTF8Bytes = 3
    static let accountEmailMaxUTF8Bytes = 320
    static let accountUsernameMinBytes = 3
    static let accountUsernameMaxBytes = 32
    static let passwordMinUTF8Bytes = 8
    static let passwordMaxUTF8Bytes = 1_024

    static func scalarCount(_ value: String) -> Int {
        value.unicodeScalars.count
    }

    static func trimmed(_ value: String) -> String {
        value.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func validSlotTitle(_ value: String) -> Bool {
        let count = scalarCount(trimmed(value))
        return (1...slotTitleMaxScalars).contains(count)
    }

    static func validSlotDetails(_ value: String) -> Bool {
        scalarCount(trimmed(value)) <= slotDetailsMaxScalars
    }

    static func validSlotPlace(_ value: String) -> Bool {
        let count = scalarCount(trimmed(value))
        return (1...slotPlaceMaxScalars).contains(count)
    }

    static func validChatMessage(_ value: String) -> Bool {
        let count = scalarCount(trimmed(value))
        return (1...chatMessageMaxScalars).contains(count)
    }

    static func validPlaceSearchQuery(_ value: String) -> Bool {
        let count = scalarCount(trimmed(value))
        return (placeSearchMinScalars...placeSearchMaxScalars).contains(count)
    }

    static func validProfileDisplayName(_ value: String) -> Bool {
        let count = scalarCount(trimmed(value))
        return (1...profileDisplayNameMaxScalars).contains(count)
    }

    static func validAvatarURLPayload(_ value: String) -> Bool {
        trimmed(value).utf8.count <= avatarURLMaxUTF8Bytes
    }

    static func validAccountEmail(_ raw: String) -> Bool {
        let value = trimmed(raw).lowercased()
        let bytes = Array(value.utf8)
        guard (accountEmailMinUTF8Bytes...accountEmailMaxUTF8Bytes).contains(bytes.count),
              !bytes.contains(32), !bytes.contains(9), !bytes.contains(10), !bytes.contains(13),
              let at = bytes.lastIndex(of: 64), at > 0, at < bytes.count - 3 else {
            return false
        }
        return bytes[(at + 1)...].contains(46)
    }

    static func validAccountUsername(_ raw: String) -> Bool {
        let value = trimmed(raw).lowercased()
        guard (accountUsernameMinBytes...accountUsernameMaxBytes).contains(value.utf8.count) else { return false }
        return value.unicodeScalars.allSatisfy { scalar in
            let code = Int(scalar.value)
            return (97...122).contains(code) || (48...57).contains(code) || code == 95 || code == 46
        }
    }

    static func validLoginIdentifierShape(_ raw: String) -> Bool {
        let value = trimmed(raw)
        return !value.isEmpty && value.utf8.count <= accountEmailMaxUTF8Bytes
    }

    static func validPasswordPayload(_ value: String) -> Bool {
        (passwordMinUTF8Bytes...passwordMaxUTF8Bytes).contains(value.utf8.count)
    }
}
