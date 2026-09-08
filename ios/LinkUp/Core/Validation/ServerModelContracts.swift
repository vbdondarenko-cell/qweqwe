import Foundation

protocol ServerShapeValidatable {
    var hasValidServerShape: Bool { get }
}

extension UserProfile: ServerShapeValidatable {
    var hasValidServerShape: Bool {
        InputContracts.validAccountEmail(email) &&
        InputContracts.validAccountUsername(username) &&
        InputContracts.validProfileDisplayName(displayName) &&
        (avatarUrl == nil || InputContracts.validAvatarURLPayload(avatarUrl ?? "")) &&
        (profileVisibility == "PUBLIC" || profileVisibility == "HIDDEN") &&
        (language == "uk" || language == "en")
    }
}

extension BlockedUser: ServerShapeValidatable {
    var hasValidServerShape: Bool {
        InputContracts.validAccountUsername(username) &&
        InputContracts.validProfileDisplayName(displayName) &&
        (avatarUrl == nil || InputContracts.validAvatarURLPayload(avatarUrl ?? ""))
    }
}

extension SlotOrganizer: ServerShapeValidatable {
    var hasValidServerShape: Bool {
        InputContracts.validAccountUsername(username) &&
        InputContracts.validProfileDisplayName(displayName) &&
        (avatarUrl == nil || InputContracts.validAvatarURLPayload(avatarUrl ?? ""))
    }
}

extension SlotModel: ServerShapeValidatable {
    var hasValidServerShape: Bool {
        let activityCount = InputContracts.scalarCount(InputContracts.trimmed(activity))
        let detailsValid = details.map { InputContracts.scalarCount(InputContracts.trimmed($0)) <= InputContracts.slotDetailsMaxScalars } ?? true
        let zoneValid = zoneText.map { InputContracts.scalarCount(InputContracts.trimmed($0)) <= 160 } ?? true
        return organizer.hasValidServerShape &&
            InputContracts.validSlotTitle(title) &&
            (1...64).contains(activityCount) &&
            detailsValid &&
            InputContracts.validSlotPlace(placeText) &&
            zoneValid &&
            (InputContracts.slotCapacityMin...InputContracts.slotCapacityMax).contains(capacity) &&
            acceptedCount >= 0 && acceptedCount <= capacity &&
            version >= 1 &&
            createdAt <= updatedAt
    }
}

extension PendingSlotRequest: ServerShapeValidatable {
    var hasValidServerShape: Bool {
        user.hasValidServerShape
    }
}

extension ChatMessage: ServerShapeValidatable {
    var hasValidServerShape: Bool {
        author.hasValidServerShape && InputContracts.validChatMessage(text)
    }
}

extension PlaceModel: ServerShapeValidatable {
    var hasValidServerShape: Bool {
        let nameCount = InputContracts.scalarCount(name)
        let categoryValid = category.map { InputContracts.scalarCount($0) <= 64 } ?? true
        let localityValid = locality.map { InputContracts.scalarCount($0) <= 120 } ?? true
        let countryValid = countryCode.map { code in
            code.utf8.count == 2 && code.unicodeScalars.allSatisfy { (65...90).contains(Int($0.value)) }
        } ?? true
        return (1...160).contains(nameCount) &&
            categoryValid && localityValid && countryValid &&
            (-90_000_000...90_000_000).contains(latitudeE6) &&
            (-180_000_000...180_000_000).contains(longitudeE6) &&
            (1...10_000).contains(precisionM)
    }
}

extension MapCluster: ServerShapeValidatable {
    var hasValidServerShape: Bool {
        guard !key.isEmpty,
              (-90_000_000...90_000_000).contains(latitudeE6),
              (-180_000_000...180_000_000).contains(longitudeE6),
              placeCount >= 1, slotCount >= 1 else { return false }

        if placeCount == 1 {
            return placeId != nil && placeName.map { (1...160).contains(InputContracts.scalarCount($0)) } == true
        }
        return placeId == nil && placeName == nil
    }
}


extension CityLocality: ServerShapeValidatable {
    var hasValidServerShape: Bool {
        let nameCount = InputContracts.scalarCount(name)
        let timezoneCount = InputContracts.scalarCount(timezone)
        let countryValid = countryCode.utf8.count == 2 && countryCode.unicodeScalars.allSatisfy {
            (65...90).contains(Int($0.value))
        }
        return (1...160).contains(nameCount) &&
            countryValid &&
            (1...80).contains(timezoneCount) &&
            TimeZone(identifier: timezone) != nil &&
            (-90_000_000...90_000_000).contains(centroidLatitudeE6) &&
            (-180_000_000...180_000_000).contains(centroidLongitudeE6)
    }
}

extension CityContextModel: ServerShapeValidatable {
    var hasValidServerShape: Bool {
        locality.hasValidServerShape &&
        (1...10_000).contains(accuracyM) &&
        observedAt < expiresAt
    }
}
