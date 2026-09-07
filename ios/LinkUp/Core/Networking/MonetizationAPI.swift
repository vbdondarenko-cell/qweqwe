import Foundation

extension LinkUpAPI {
    func monetizationSnapshot() async throws -> MonetizationSnapshot {
        try await client.send(APIRequest(
            method: .get,
            path: "/v1/me/monetization"
        ))
    }

    func bindReferral(code: String) async throws -> MonetizationSnapshot {
        guard let normalized = ReferralCodeContract.normalize(code) else {
            throw APIError.protocolViolation("Invalid referral code.")
        }

        return try await client.send(APIRequest(
            method: .put,
            path: "/v1/me/referral",
            body: try encodeBody(BindReferralBody(code: normalized)),
            idempotencyKey: UUID()
        ))
    }
}
