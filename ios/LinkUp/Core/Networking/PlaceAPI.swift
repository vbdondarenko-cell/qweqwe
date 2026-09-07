import Foundation

extension LinkUpAPI {
    func searchPlaces(
        query: String,
        locality: String? = nil,
        limit: Int = 10
    ) async throws -> [PlaceModel] {
        let text = InputContracts.trimmed(query)
        guard InputContracts.validPlaceSearchQuery(text), (1...50).contains(limit) else {
            throw APIError.protocolViolation("Invalid place search query.")
        }

        var queryItems = [
            URLQueryItem(name: "q", value: text),
            URLQueryItem(name: "limit", value: String(limit))
        ]
        if let locality {
            let normalized = InputContracts.trimmed(locality)
            guard InputContracts.scalarCount(normalized) <= InputContracts.placeSearchLocalityMaxScalars else {
                throw APIError.protocolViolation("Invalid place search locality.")
            }
            if !normalized.isEmpty {
                queryItems.append(URLQueryItem(name: "locality", value: normalized))
            }
        }

        let envelope: ItemsEnvelope<PlaceModel> = try await client.send(APIRequest(
            method: .get,
            path: "/v1/places/search",
            queryItems: queryItems
        ))
        return envelope.items
    }
}
