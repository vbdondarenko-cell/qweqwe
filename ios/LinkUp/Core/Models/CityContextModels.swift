import Foundation

enum CityPermissionClass: String, Codable, Equatable, Sendable {
    case approximate = "APPROXIMATE"
    case precise = "PRECISE"
}

struct CityLocality: Codable, Identifiable, Equatable, Sendable {
    let id: UUID
    let name: String
    let countryCode: String
    let timezone: String
    let centroidLatitudeE6: Int
    let centroidLongitudeE6: Int

    var latitude: Double { Double(centroidLatitudeE6) / 1_000_000 }
    var longitude: Double { Double(centroidLongitudeE6) / 1_000_000 }
}

struct CityContextModel: Codable, Equatable, Sendable {
    let locality: CityLocality
    let permissionClass: CityPermissionClass
    let accuracyM: Int
    let observedAt: Date
    let expiresAt: Date
    let switchPending: Bool
}

struct CityLocationObservation: Encodable, Equatable, Sendable {
    let latitudeE6: Int
    let longitudeE6: Int
    let accuracyM: Int
    let permissionClass: CityPermissionClass
    let capturedAt: Date
    let mocked: Bool

    var hasValidClientShape: Bool {
        (-90_000_000...90_000_000).contains(latitudeE6) &&
        (-180_000_000...180_000_000).contains(longitudeE6) &&
        (1...10_000).contains(accuracyM) &&
        capturedAt.timeIntervalSince1970 > 0 &&
        !mocked
    }
}
