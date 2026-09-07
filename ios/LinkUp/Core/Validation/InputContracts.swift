import Foundation

enum InputContracts {
    static let slotTitleMaxScalars = 120
    static let slotDetailsMaxScalars = 2_000
    static let slotPlaceMaxScalars = 240
    static let placeSearchMinScalars = 2
    static let placeSearchMaxScalars = 80
    static let placeSearchLocalityMaxScalars = 120
    static let profileDisplayNameMaxScalars = 80
    static let avatarURLMaxUTF8Bytes = 2_048

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
}
