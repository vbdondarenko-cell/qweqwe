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
                state = .signedOut
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
            state = .recoverableError("Unable to verify the current session.")
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
        generation &+= 1
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
        generation &+= 1
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

    func refreshSignedInProfileSnapshot() async -> Bool {
        guard case .signedIn(let current) = state else { return false }
        generation &+= 1
        let requestGeneration = generation
        do {
            let updated = try await api.me()
            guard requestGeneration == generation else { return false }
            guard updated.id == current.id else {
                await api.clearLocalSession()
                state = .signedOut
                return false
            }
            state = .signedIn(updated)
            return true
        } catch is CancellationError {
            return false
        } catch let error as APIError {
            guard requestGeneration == generation else { return false }
            if case .unauthorized = error {
                await api.clearLocalSession()
                state = .signedOut
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
            await api.clearLocalSession()
            state = .signedOut
            throw APIError.protocolViolation("Profile response belongs to a different account.")
        }
        state = .signedIn(updated)
    }

    func logout() async {
        generation &+= 1
        await api.logout()
        state = .signedOut
    }

    func clearLocalSession() async {
        generation &+= 1
        await api.clearLocalSession()
        state = .signedOut
    }

    private func revalidateSignedIn(_ current: UserProfile) async {
        generation &+= 1
        let requestGeneration = generation

        let local: SessionCredential
        do {
            guard let stored = try await credentials.load() else {
                guard requestGeneration == generation else { return }
                state = .signedOut
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
                await api.clearLocalSession()
                state = .signedOut
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
            state = .recoverableError("Unable to verify the current session.")
        }
    }

    private func applyVerificationFailure(_ error: APIError, local: SessionCredential) async {
        switch error {
        case .unauthorized:
            await api.clearLocalSession()
            state = .signedOut
        case .transport:
            state = .offlineSession(expiresAt: local.expiresAt)
        default:
            state = .recoverableError(error.localizedDescription)
        }
    }
}
