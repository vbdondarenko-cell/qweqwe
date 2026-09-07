import Foundation

struct PlaceModel: Codable, Identifiable, Equatable, Sendable {
    let id: UUID
    let name: String
    let category: String?
    let locality: String?
    let countryCode: String?
    let latitudeE6: Int
    let longitudeE6: Int
    let precisionM: Int

    var latitude: Double { Double(latitudeE6) / 1_000_000 }
    var longitude: Double { Double(longitudeE6) / 1_000_000 }

    var subtitle: String {
        [locality, countryCode]
            .compactMap { $0?.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .joined(separator: " · ")
    }
}
