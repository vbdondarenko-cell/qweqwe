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
        let envelope: AuthEnvelope = try await client.send(APIRequest(
            method: .post,
            path: "/v1/auth/login",
            body: try encodeBody(LoginBody(identifier: identifier, password: password, deviceLabel: deviceLabel)),
            authenticated: false
        ))
        try await saveCredential(from: envelope)
        return envelope.user
    }

    func register(
        email: String,
        username: String,
        displayName: String,
        password: String,
        language: String,
        deviceLabel: String
    ) async throws -> UserProfile {
        let envelope: AuthEnvelope = try await client.send(APIRequest(
            method: .post,
            path: "/v1/auth/register",
            body: try encodeBody(RegisterBody(
                email: email,
                username: username,
                displayName: displayName,
                password: password,
                language: language,
                deviceLabel: deviceLabel
            )),
            authenticated: false
        ))
        try await saveCredential(from: envelope)
        return envelope.user
    }

    func requestPasswordRecovery(email: String) async throws {
        try await client.sendVoid(APIRequest(
            method: .post,
            path: "/v1/auth/recovery/request",
            body: try encodeBody(RecoveryRequestBody(email: email)),
            authenticated: false
        ))
    }

    func resetPassword(token: String, newPassword: String) async throws {
        try await client.sendVoid(APIRequest(
            method: .post,
            path: "/v1/auth/recovery/reset",
            body: try encodeBody(PasswordResetBody(token: token, newPassword: newPassword)),
            authenticated: false
        ))
    }

    func me() async throws -> UserProfile {
        try await client.send(APIRequest(method: .get, path: "/v1/me"))
    }

    func updateMe(
        displayName: String? = nil,
        avatarUrl: String? = nil,
        profileVisibility: String? = nil,
        language: String? = nil
    ) async throws -> UserProfile {
        try await client.send(APIRequest(
            method: .patch,
            path: "/v1/me",
            body: try encodeBody(UpdateProfileBody(
                displayName: displayName,
                avatarUrl: avatarUrl,
                profileVisibility: profileVisibility,
                language: language
            ))
        ))
    }

    func logout() async {
        do {
            try await client.sendVoid(APIRequest(method: .post, path: "/v1/auth/logout"))
        } catch {
            // Local sign-out is authoritative for this device even if remote revocation is unavailable.
        }
        await clearLocalSession()
    }

    func blockedUsers() async throws -> [BlockedUser] {
        let envelope: ItemsEnvelope<BlockedUser> = try await client.send(
            APIRequest(method: .get, path: "/v1/me/blocks")
        )
        return envelope.items
    }

    func blockUser(_ userID: UUID) async throws {
        try await client.sendVoid(APIRequest(method: .put, path: "/v1/me/blocks/\(uuidPath(userID))"))
    }

    func unblockUser(_ userID: UUID) async throws {
        try await client.sendVoid(APIRequest(method: .delete, path: "/v1/me/blocks/\(uuidPath(userID))"))
    }
}
