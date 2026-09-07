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

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var hosting: [SlotModel] = []
    @Published private(set) var joined: [SlotModel] = []
    @Published private(set) var requested: [SlotModel] = []
    @Published private(set) var blockedUsers: [BlockedUser] = []
    @Published private(set) var isMutating = false

    private let api: LinkUpAPI
    private let session: SessionCoordinator
    private var generation: UInt64 = 0

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
            phase = .failed("Unable to load account data.")
        }
    }

    func unblock(_ user: BlockedUser) async {
        guard !isMutating else { return }
        isMutating = true
        defer { isMutating = false }
        do {
            try await api.unblockUser(user.id)
            blockedUsers.removeAll { $0.id == user.id }
        } catch let error as APIError {
            if case .unauthorized = error {
                await session.clearLocalSession()
            } else {
                phase = .failed(error.localizedDescription)
            }
        } catch {
            phase = .failed("Unable to unblock this account.")
        }
    }

    func dispose() {
        generation &+= 1
        hosting = []
        joined = []
        requested = []
        blockedUsers = []
        phase = .idle
    }
}
