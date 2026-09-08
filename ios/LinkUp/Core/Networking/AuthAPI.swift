import Foundation

private struct LoginBody: Encodable, Sendable {
    let identifier: String
    let password: String
    let deviceLabel: String
}

private struct RegisterBody: Encodable, Sendable {
    let email: String
    let username: String
    let displayName: String
    let password: String
    let language: String
    let deviceLabel: String
}

private struct RecoveryRequestBody: Encodable, Sendable {
    let email: String
}

private struct PasswordResetBody: Encodable, Sendable {
    let token: String
    let newPassword: String
}

private struct UpdateProfileBody: Encodable, Sendable {
    let displayName: String?
    let avatarUrl: String?
    let profileVisibility: String?
    let language: String?
}

extension LinkUpAPI {
    func login(identifier: String, password: String, deviceLabel: String) async throws -> UserProfile {
        let normalizedIdentifier = InputContracts.trimmed(identifier)
        guard InputContracts.validLoginIdentifierShape(normalizedIdentifier),
              InputContracts.validPasswordPayload(password) else {
            throw APIError.protocolViolation("Invalid login input.")
        }
        let envelope: AuthEnvelope = try await client.send(APIRequest(
            method: .post,
            path: "/v1/auth/login",
            body: try encodeBody(LoginBody(identifier: normalizedIdentifier, password: password, deviceLabel: deviceLabel)),
            authenticated: false
        ))
        let user = try validatedServerValue(envelope.user, context: "authenticated user")
        try await saveCredential(from: envelope)
        return user
    }

    func register(
        email: String,
        username: String,
        displayName: String,
        password: String,
        language: String,
        deviceLabel: String
    ) async throws -> UserProfile {
        let normalizedEmail = InputContracts.trimmed(email).lowercased()
        let normalizedUsername = InputContracts.trimmed(username).lowercased()
        let normalizedDisplayName = InputContracts.trimmed(displayName)
        guard InputContracts.validAccountEmail(normalizedEmail),
              InputContracts.validAccountUsername(normalizedUsername),
              InputContracts.validProfileDisplayName(normalizedDisplayName),
              InputContracts.validPasswordPayload(password),
              language == "uk" || language == "en" else {
            throw APIError.protocolViolation("Invalid registration input.")
        }
        let envelope: AuthEnvelope = try await client.send(APIRequest(
            method: .post,
            path: "/v1/auth/register",
            body: try encodeBody(RegisterBody(
                email: normalizedEmail,
                username: normalizedUsername,
                displayName: normalizedDisplayName,
                password: password,
                language: language,
                deviceLabel: deviceLabel
            )),
            authenticated: false
        ))
        let user = try validatedServerValue(envelope.user, context: "authenticated user")
        try await saveCredential(from: envelope)
        return user
    }

    func requestPasswordRecovery(email: String) async throws {
        let normalized = InputContracts.trimmed(email).lowercased()
        guard InputContracts.validAccountEmail(normalized) else {
            throw APIError.protocolViolation("Invalid recovery email.")
        }
        try await client.sendVoid(APIRequest(
            method: .post,
            path: "/v1/auth/recovery/request",
            body: try encodeBody(RecoveryRequestBody(email: normalized)),
            authenticated: false
        ))
    }

    func resetPassword(token: String, newPassword: String) async throws {
        guard let canonicalToken = OpaqueTokenContract.canonical32ByteBase64URL(InputContracts.trimmed(token)),
              InputContracts.validPasswordPayload(newPassword) else {
            throw APIError.protocolViolation("Invalid password reset input.")
        }
        try await client.sendVoid(APIRequest(
            method: .post,
            path: "/v1/auth/recovery/reset",
            body: try encodeBody(PasswordResetBody(token: canonicalToken, newPassword: newPassword)),
            authenticated: false
        ))
    }

    func me() async throws -> UserProfile {
        let user: UserProfile = try await client.send(APIRequest(method: .get, path: "/v1/me"))
        return try validatedServerValue(user, context: "profile")
    }

    func updateMe(
        displayName: String? = nil,
        avatarUrl: String? = nil,
        profileVisibility: String? = nil,
        language: String? = nil
    ) async throws -> UserProfile {
        try beginAuthenticatedWrite()
        defer { endAuthenticatedWrite() }
        if let displayName, !InputContracts.validProfileDisplayName(displayName) {
            throw APIError.protocolViolation("Invalid display name.")
        }
        if let avatarUrl, !InputContracts.validAvatarURLPayload(avatarUrl) {
            throw APIError.protocolViolation("Invalid avatar URL payload.")
        }
        if let profileVisibility, profileVisibility != "PUBLIC" && profileVisibility != "HIDDEN" {
            throw APIError.protocolViolation("Invalid profile visibility.")
        }
        if let language, language != "uk" && language != "en" {
            throw APIError.protocolViolation("Invalid language.")
        }
        let user: UserProfile = try await client.send(APIRequest(
            method: .patch,
            path: "/v1/me",
            body: try encodeBody(UpdateProfileBody(
                displayName: displayName.map(InputContracts.trimmed),
                avatarUrl: avatarUrl.map(InputContracts.trimmed),
                profileVisibility: profileVisibility,
                language: language
            ))
        ))
        return try validatedServerValue(user, context: "updated profile")
    }

    func logout() async throws {
        try beginExplicitLogout()
        defer { endExplicitLogout() }
        try await assertExplicitLogoutSafe()
        do {
            try await client.sendVoid(APIRequest(method: .post, path: "/v1/auth/logout"))
        } catch {
            // With no pending owner-bound work, local sign-out remains authoritative
            // for this device even if remote revocation is temporarily unavailable.
        }
        await clearLocalSession()
    }

    func blockedUsers() async throws -> [BlockedUser] {
        let envelope: ItemsEnvelope<BlockedUser> = try await client.send(
            APIRequest(method: .get, path: "/v1/me/blocks")
        )
        return try validatedServerItems(envelope.items, context: "blocked user")
    }

    func blockUser(_ userID: UUID) async throws {
        try await sendIdempotentRelationshipWrite(
            APIRequest(method: .put, path: "/v1/me/blocks/\(uuidPath(userID))")
        )
    }

    func unblockUser(_ userID: UUID) async throws {
        try await sendIdempotentRelationshipWrite(
            APIRequest(method: .delete, path: "/v1/me/blocks/\(uuidPath(userID))")
        )
    }

    private func sendIdempotentRelationshipWrite(_ request: APIRequest) async throws {
        try beginAuthenticatedWrite()
        defer { endAuthenticatedWrite() }
        guard request.method == .put || request.method == .delete else {
            throw APIError.protocolViolation("Relationship retry helper requires PUT or DELETE.")
        }

        for attempt in 0..<2 {
            do {
                try await client.sendVoid(request)
                return
            } catch is CancellationError {
                throw CancellationError()
            } catch let error as APIError {
                guard attempt == 0, error.retryableForIdempotentWrite else { throw error }
                try await Task<Never, Never>.sleep(nanoseconds: 250_000_000)
            }
        }
    }
}
