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
        return try validatedServerItems(response.items, context: "map cluster")
    }

    func mapPlaceSlots(
        placeID: UUID,
        from: Date,
        to: Date,
        limit: Int = 50
    ) async throws -> [SlotModel] {
        guard from < to,
              to.timeIntervalSince(from) <= 7 * 24 * 60 * 60,
              (1...100).contains(limit) else {
            throw APIError.protocolViolation("Invalid map place query.")
        }

        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        let response: ItemsEnvelope<SlotModel> = try await client.send(APIRequest(
            method: .get,
            path: "/v1/map/places/\(uuidPath(placeID))/slots",
            queryItems: [
                URLQueryItem(name: "from", value: formatter.string(from: from)),
                URLQueryItem(name: "to", value: formatter.string(from: to)),
                URLQueryItem(name: "limit", value: String(limit))
            ]
        ))
        let slots = try validatedServerItems(response.items, context: "map place Slot")
        guard slots.allSatisfy({ $0.canonicalPlaceId == placeID }) else {
            throw APIError.protocolViolation("Server returned Slots for a different canonical place.")
        }
        return slots
    }
}
