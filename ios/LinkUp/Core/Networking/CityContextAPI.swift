import Foundation

extension LinkUpAPI {
    func currentCityContext() async throws -> CityContextModel {
        let value: CityContextModel = try await client.send(APIRequest(
            method: .get,
            path: "/v1/city-context"
        ))
        return try validatedServerValue(value, context: "city context")
    }

    func resolveCityContext(_ observation: CityLocationObservation) async throws -> CityContextModel {
        guard observation.hasValidClientShape else {
            throw APIError.protocolViolation("Invalid city location observation.")
        }
        let value: CityContextModel = try await client.send(APIRequest(
            method: .post,
            path: "/v1/city-context/resolve",
            body: try encodeBody(observation)
        ))
        return try validatedServerValue(value, context: "city context")
    }
}
