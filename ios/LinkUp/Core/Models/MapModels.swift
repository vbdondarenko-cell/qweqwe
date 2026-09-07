import Foundation
import CoreLocation

struct MapCluster: Codable, Identifiable, Equatable, Sendable {
    let key: String
    let latitudeE6: Int
    let longitudeE6: Int
    let placeCount: Int
    let slotCount: Int
    let placeId: UUID?
    let placeName: String?

    var id: String { key }
    var coordinate: CLLocationCoordinate2D {
        CLLocationCoordinate2D(
            latitude: Double(latitudeE6) / 1_000_000,
            longitude: Double(longitudeE6) / 1_000_000
        )
    }
}

struct MapViewportQuery: Sendable, Equatable {
    let westE6: Int
    let southE6: Int
    let eastE6: Int
    let northE6: Int
    let zoom: Int
    let from: Date
    let to: Date
    let limit: Int

    var isValid: Bool {
        (-180_000_000...180_000_000).contains(westE6) &&
        (-180_000_000...180_000_000).contains(eastE6) &&
        (-90_000_000...90_000_000).contains(southE6) &&
        (-90_000_000...90_000_000).contains(northE6) &&
        southE6 < northE6 && westE6 != eastE6 &&
        (1...20).contains(zoom) && (1...200).contains(limit) &&
        from < to && to.timeIntervalSince(from) <= 7 * 24 * 60 * 60
    }
}
