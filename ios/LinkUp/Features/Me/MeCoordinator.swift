import Foundation
import SwiftUI

@MainActor
final class MeCoordinator: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case content
        case failed(String)
    }

    enum MonetizationPhase: Equatable {
        case idle
        case loading
        case content
        case failed(String)
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var hosting: [SlotModel] = []
    @Published private(set) var joined: [SlotModel] = []
    @Published private(set) var requested: [SlotModel] = []
    @Published private(set) var blockedUsers: [BlockedUser] = []
    @Published private(set) var monetization: MonetizationSnapshot?
    @Published private(set) var monetizationPhase: MonetizationPhase = .idle
    @Published private(set) var isMutating = false
    @Published private(set) var mutationError: String?

    private let api: LinkUpAPI
    private let session: SessionCoordinator
    private var generation: UInt64 = 0
    private var monetizationGeneration: UInt64 = 0

    init(api: LinkUpAPI, session: SessionCoordinator) {
        self.api = api
        self.session = session
    }

    func load() async {
        generation &+= 1
        let requestGeneration = generation
        phase = .loading

        do {
            async let hosting = api.mySlots(.hosting)
            async let joined = api.mySlots(.joined)
            async let requested = api.mySlots(.requested)
            async let blocked = api.blockedUsers()
            let result = try await (hosting, joined, requested, blocked)
            guard requestGeneration == generation else { return }
            self.hosting = result.0
            self.joined = result.1
            self.requested = result.2
            self.blockedUsers = result.3
            phase = .content
        } catch is CancellationError {
            return
        } catch let error as APIError {
            guard requestGeneration == generation else { return }
            if case .unauthorized = error {
                await session.clearLocalSession()
            } else {
                phase = .failed(error.localizedDescription)
            }
        } catch {
            guard requestGeneration == generation else { return }
            phase = .failed(L10n.text("Unable to load account data."))
        }
    }

    func loadMonetization() async {
        monetizationGeneration &+= 1
        let requestGeneration = monetizationGeneration
        monetizationPhase = .loading
        do {
            let snapshot = try await api.monetizationSnapshot()
            guard requestGeneration == monetizationGeneration else { return }
            monetization = snapshot
            monetizationPhase = .content
        } catch is CancellationError {
            return
        } catch let error as APIError {
            guard requestGeneration == monetizationGeneration else { return }
            if case .unauthorized = error {
                await session.clearLocalSession()
            } else {
                monetizationPhase = .failed(error.localizedDescription)
            }
        } catch {
            guard requestGeneration == monetizationGeneration else { return }
            monetizationPhase = .failed(L10n.text("Unable to load LinkUp+ status."))
        }
    }

    func bindReferral(_ code: String) async -> Bool {
        guard !isMutating else { return false }
        isMutating = true
        mutationError = nil
        defer { isMutating = false }
        do {
            let snapshot = try await api.bindReferral(code: code)
            monetizationGeneration &+= 1
            monetization = snapshot
            monetizationPhase = .content
            return true
        } catch is CancellationError {
            return false
        } catch let error as APIError {
            if case .unauthorized = error {
                await session.clearLocalSession()
            } else {
                mutationError = error.localizedDescription
            }
            return false
        } catch {
            mutationError = L10n.text("Unable to bind referral code.")
            return false
        }
    }

    func unblock(_ user: BlockedUser) async -> Bool {
        guard !isMutating else { return false }
        isMutating = true
        mutationError = nil
        defer { isMutating = false }
        do {
            try await api.unblockUser(user.id)
            blockedUsers.removeAll { $0.id == user.id }
            return true
        } catch is CancellationError {
            return false
        } catch let error as APIError {
            if case .unauthorized = error {
                await session.clearLocalSession()
            } else {
                mutationError = error.localizedDescription
            }
            return false
        } catch {
            mutationError = L10n.text("Unable to unblock this account.")
            return false
        }
    }

    func clearMutationError() {
        mutationError = nil
    }

    func dispose() {
        generation &+= 1
        monetizationGeneration &+= 1
        hosting = []
        joined = []
        requested = []
        blockedUsers = []
        monetization = nil
        phase = .idle
        monetizationPhase = .idle
        mutationError = nil
    }
}
