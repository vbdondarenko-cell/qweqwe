import Foundation

extension LinkUpAPI {
    func mapViewport(_ query: MapViewportQuery) async throws -> [MapCluster] {
        guard query.isValid else {
            throw APIError.protocolViolation("Invalid map viewport.")
        }

        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        let response: ItemsEnvelope<MapCluster> = try await client.send(APIRequest(
            method: .get,
            path: "/v1/map",
            queryItems: [
                URLQueryItem(name: "westE6", value: String(query.westE6)),
                URLQueryItem(name: "southE6", value: String(query.southE6)),
                URLQueryItem(name: "eastE6", value: String(query.eastE6)),
                URLQueryItem(name: "northE6", value: String(query.northE6)),
                URLQueryItem(name: "zoom", value: String(query.zoom)),
                URLQueryItem(name: "from", value: formatter.string(from: query.from)),
                URLQueryItem(name: "to", value: formatter.string(from: query.to)),
                URLQueryItem(name: "limit", value: String(query.limit))
            ]
        ))
        return response.items
    }
}
