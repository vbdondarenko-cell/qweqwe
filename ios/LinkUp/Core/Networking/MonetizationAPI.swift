import Foundation

extension LinkUpAPI {
    func monetizationSnapshot() async throws -> MonetizationSnapshot {
        try await client.send(APIRequest(
            method: .get,
            path: "/v1/me/monetization"
        ))
    }

    func bindReferral(code: String) async throws -> MonetizationSnapshot {
        try beginAuthenticatedWrite()
        defer { endAuthenticatedWrite() }
        guard let normalized = ReferralCodeContract.normalize(code) else {
            throw APIError.protocolViolation("Invalid referral code.")
        }

        do {
            return try await client.send(APIRequest(
                method: .put,
                path: "/v1/me/referral",
                body: try encodeBody(BindReferralBody(code: normalized))
            ))
        } catch let error as APIError {
            if case .http(_, let code, _, _) = error, code == "referral_already_bound" {
                let snapshot = try await monetizationSnapshot()
                if snapshot.status.referral.boundReferralCode == normalized {
                    return snapshot
                }
            }
            throw error
        }
    }
}
