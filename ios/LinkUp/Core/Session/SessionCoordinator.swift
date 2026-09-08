import Foundation
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
    private var generation: UInt64 = 0
    private var credentialAuthInProgress = false

    init(api: LinkUpAPI, credentials: KeychainSessionStore) {
        self.api = api
        self.credentials = credentials
    }

    func bootstrap() async {
        generation &+= 1
        let requestGeneration = generation
        state = .checking

        let local: SessionCredential
        do {
            guard let stored = try await credentials.load() else {
                guard requestGeneration == generation else { return }
                _ = try? await transitionToSignedOutStrictly()
                return
            }
            local = stored
        } catch {
            guard requestGeneration == generation else { return }
            state = .recoverableError("Secure session storage is unavailable.")
            return
        }

        do {
            let user = try await api.me()
            guard requestGeneration == generation else { return }
            state = .signedIn(user)
        } catch is CancellationError {
            return
        } catch let error as APIError {
            guard requestGeneration == generation else { return }
            await applyVerificationFailure(error, local: local)
        } catch {
            guard requestGeneration == generation else { return }
            state = .recoverableError(L10n.text("Unable to verify the current session."))
        }
    }

    func revalidateForForeground() async {
        switch state {
        case .checking, .signedOut:
            return
        case .offlineSession, .recoverableError:
            await bootstrap()
        case .signedIn(let current):
            await revalidateSignedIn(current)
        }
    }

    func login(identifier: String, password: String, deviceLabel: String) async throws {
        try beginCredentialAuthOperation()
        defer { endCredentialAuthOperation() }
        generation &+= 1
        let requestGeneration = generation
        let user = try await api.login(identifier: identifier, password: password, deviceLabel: deviceLabel)
        guard requestGeneration == generation else {
            try await transitionToSignedOutStrictly()
            throw CancellationError()
        }
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
        try beginCredentialAuthOperation()
        defer { endCredentialAuthOperation() }
        generation &+= 1
        let requestGeneration = generation
        let user = try await api.register(
            email: email,
            username: username,
            displayName: displayName,
            password: password,
            language: language,
            deviceLabel: deviceLabel
        )
        guard requestGeneration == generation else {
            try await transitionToSignedOutStrictly()
            throw CancellationError()
        }
        state = .signedIn(user)
    }

    func refreshSignedInProfileSnapshot() async -> Bool {
        guard case .signedIn(let current) = state else { return false }
        generation &+= 1
        let requestGeneration = generation
        do {
            let updated = try await api.me()
            guard requestGeneration == generation else { return false }
            guard updated.id == current.id else {
                _ = try? await transitionToSignedOutStrictly()
                return false
            }
            state = .signedIn(updated)
            return true
        } catch is CancellationError {
            return false
        } catch let error as APIError {
            guard requestGeneration == generation else { return false }
            if case .unauthorized = error {
                _ = try? await transitionToSignedOutStrictly()
            }
            return false
        } catch {
            return false
        }
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
        generation &+= 1
        let requestGeneration = generation
        let updated = try await api.updateMe(
            displayName: displayName,
            avatarUrl: avatarUrl,
            profileVisibility: profileVisibility,
            language: language
        )
        guard requestGeneration == generation else { throw CancellationError() }
        guard case .signedIn(let stillCurrent) = state, stillCurrent.id == current.id else {
            throw CancellationError()
        }
        guard updated.id == current.id else {
            try await transitionToSignedOutStrictly()
            throw APIError.protocolViolation("Profile response belongs to a different account.")
        }
        state = .signedIn(updated)
    }

    func logout() async throws {
        generation &+= 1
        try await api.logout()
        state = .signedOut
    }

    func clearLocalSession() async {
        generation &+= 1
        _ = try? await transitionToSignedOutStrictly()
    }

    private func transitionToSignedOutStrictly() async throws {
        do {
            try await api.clearLocalSessionStrict()
            state = .signedOut
        } catch let error as APIError {
            state = .recoverableError(error.localizedDescription)
            throw error
        } catch {
            state = .recoverableError(L10n.text("Secure session storage is unavailable."))
            throw APIError.secureStorageUnavailable
        }
    }

    private func beginCredentialAuthOperation() throws {
        guard !credentialAuthInProgress else { throw APIError.authenticationInProgress }
        credentialAuthInProgress = true
    }

    private func endCredentialAuthOperation() {
        credentialAuthInProgress = false
    }

    private func revalidateSignedIn(_ current: UserProfile) async {
        generation &+= 1
        let requestGeneration = generation

        let local: SessionCredential
        do {
            guard let stored = try await credentials.load() else {
                guard requestGeneration == generation else { return }
                _ = try? await transitionToSignedOutStrictly()
                return
            }
            local = stored
        } catch {
            guard requestGeneration == generation else { return }
            state = .recoverableError("Secure session storage is unavailable.")
            return
        }

        do {
            let updated = try await api.me()
            guard requestGeneration == generation else { return }
            guard updated.id == current.id else {
                _ = try? await transitionToSignedOutStrictly()
                return
            }
            state = .signedIn(updated)
        } catch is CancellationError {
            return
        } catch let error as APIError {
            guard requestGeneration == generation else { return }
            await applyVerificationFailure(error, local: local)
        } catch {
            guard requestGeneration == generation else { return }
            state = .recoverableError(L10n.text("Unable to verify the current session."))
        }
    }

    private func applyVerificationFailure(_ error: APIError, local: SessionCredential) async {
        switch error {
        case .unauthorized:
            _ = try? await transitionToSignedOutStrictly()
        case .transport:
            state = .offlineSession(expiresAt: local.expiresAt)
        default:
            state = .recoverableError(error.localizedDescription)
        }
    }
}
