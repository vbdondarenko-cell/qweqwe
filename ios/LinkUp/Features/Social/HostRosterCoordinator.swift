import Foundation
import SwiftUI

@MainActor
final class HostRosterCoordinator: ObservableObject {
    enum Phase: Equatable {
        case idle, loading, content
        case failed(String)
    }

    @Published private(set) var pending: [PendingSlotRequest] = []
    @Published private(set) var accepted: [SlotOrganizer] = []
    @Published private(set) var phase: Phase = .idle

    private let slotID: UUID
    private let api: LinkUpAPI
    private let session: SessionCoordinator
    private var generation: UInt64 = 0

    init(slotID: UUID, api: LinkUpAPI, session: SessionCoordinator) {
        self.slotID = slotID
        self.api = api
        self.session = session
    }

    func load() async {
        generation &+= 1
        let requestGeneration = generation
        phase = .loading
        do {
            async let pending = api.pendingRequests(slotID)
            async let accepted = api.acceptedParticipants(slotID)
            let result = try await (pending, accepted)
            guard requestGeneration == generation else { return }
            self.pending = result.0
            self.accepted = result.1
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
            phase = .failed("Unable to load participants.")
        }
    }

    func dispose() {
        generation &+= 1
        pending = []
        accepted = []
        phase = .idle
    }
}
