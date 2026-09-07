import SwiftUI

@MainActor
final class SessionCoordinator: ObservableObject {
    enum State: Equatable {
        case checking
        case signedOut
        case signedIn(UserProfile)
        case offlineSession(expiresAt: Date)
        case recoverableError(String)
    }

    @Published private(set) var state: State = .checking

    private let api: LinkUpAPI
    private let credentials: KeychainSessionStore

    init(api: LinkUpAPI, credentials: KeychainSessionStore) {
        self.api = api
        self.credentials = credentials
    }

    func bootstrap() async {
        state = .checking
        let local: SessionCredential
        do {
            guard let stored = try await credentials.load() else {
                state = .signedOut
                return
            }
            local = stored
        } catch {
            state = .recoverableError("Secure session storage is unavailable.")
            return
        }

        do {
            state = .signedIn(try await api.me())
        } catch is CancellationError {
            return
        } catch let error as APIError {
            switch error {
            case .unauthorized:
                await api.clearLocalSession()
                state = .signedOut
            case .transport:
                state = .offlineSession(expiresAt: local.expiresAt)
            default:
                state = .recoverableError(error.localizedDescription)
            }
        } catch {
            state = .recoverableError("Unable to verify the current session.")
        }
    }

    func login(identifier: String, password: String, deviceLabel: String) async throws {
        let user = try await api.login(identifier: identifier, password: password, deviceLabel: deviceLabel)
        state = .signedIn(user)
    }

    func register(
        email: String,
        username: String,
        displayName: String,
        password: String,
        language: String,
        deviceLabel: String
    ) async throws {
        let user = try await api.register(
            email: email,
            username: username,
            displayName: displayName,
            password: password,
            language: language,
            deviceLabel: deviceLabel
        )
        state = .signedIn(user)
    }

    func requestPasswordRecovery(email: String) async throws {
        try await api.requestPasswordRecovery(email: email)
    }

    func resetPassword(token: String, newPassword: String) async throws {
        try await api.resetPassword(token: token, newPassword: newPassword)
    }

    func updateProfile(
        displayName: String,
        avatarUrl: String?,
        profileVisibility: String,
        language: String
    ) async throws {
        guard case .signedIn(let current) = state else { throw APIError.unauthorized }
        let updated = try await api.updateMe(
            displayName: displayName,
            avatarUrl: avatarUrl,
            profileVisibility: profileVisibility,
            language: language
        )
        guard updated.id == current.id else {
            throw APIError.protocolViolation("Profile response belongs to a different account.")
        }
        state = .signedIn(updated)
    }

    func logout() async {
        await api.logout()
        state = .signedOut
    }

    func acceptSignedInUser(_ user: UserProfile) {
        state = .signedIn(user)
    }

    func clearLocalSession() async {
        await api.clearLocalSession()
        state = .signedOut
    }
}
