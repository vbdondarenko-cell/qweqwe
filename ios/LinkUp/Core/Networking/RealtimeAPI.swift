import Foundation

extension LinkUpAPI {
    func pullRealtime(after: Int64, limit: Int = 100) async throws -> RealtimeBatch {
        guard after >= 0, (1...200).contains(limit) else {
            throw APIError.protocolViolation("Invalid realtime request.")
        }
        let batch: RealtimeBatch = try await client.send(APIRequest(
            method: .get,
            path: "/v1/realtime/events",
            queryItems: [
                URLQueryItem(name: "after", value: String(after)),
                URLQueryItem(name: "limit", value: String(limit))
            ]
        ))
        guard batch.isValid(after: after, limit: limit) else {
            throw APIError.protocolViolation("Server returned invalid realtime data.")
        }
        return batch
    }
}
