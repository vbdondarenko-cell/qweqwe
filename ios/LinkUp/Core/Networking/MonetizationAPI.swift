import Foundation

extension LinkUpAPI {
    func monetizationSnapshot() async throws -> MonetizationSnapshot {
        try await client.send(APIRequest(
            method: .get,
            path: "/v1/me/monetization"
        ))
    }

    func bindReferral(code: String) async throws -> MonetizationSnapshot {
        let normalized = code.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        let validCharacters = normalized.unicodeScalars.allSatisfy { scalar in
            (65...90).contains(Int(scalar.value)) || (48...57).contains(Int(scalar.value))
        }
        guard (6...20).contains(normalized.count), validCharacters else {
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
