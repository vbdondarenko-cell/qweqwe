import Foundation
import SwiftUI

@MainActor
final class SocialCoordinator: ObservableObject {
    enum PulsePhase: Equatable {
        case idle, loading, refreshing, content, empty
        case failed(String)
    }

    @Published private(set) var pulseItems: [SlotModel] = []
    @Published private(set) var pulsePhase: PulsePhase = .idle
    @Published private(set) var isMutating = false
    @Published private(set) var mutationError: String?

    private let api: LinkUpAPI
    private let session: SessionCoordinator
    private var pulseGeneration: UInt64 = 0

    init(api: LinkUpAPI, session: SessionCoordinator) {
        self.api = api
        self.session = session
    }

    func loadPulse() async {
        pulseGeneration &+= 1
        let generation = pulseGeneration
        pulsePhase = pulseItems.isEmpty ? .loading : .refreshing
        do {
            let items = try await api.pulse()
            guard generation == pulseGeneration else { return }
            pulseItems = items
            pulsePhase = items.isEmpty ? .empty : .content
        } catch is CancellationError {
            return
        } catch let error as APIError {
            guard generation == pulseGeneration else { return }
            if case .unauthorized = error { await session.clearLocalSession(); return }
            pulsePhase = pulseItems.isEmpty ? .failed(error.localizedDescription) : .content
        } catch {
            guard generation == pulseGeneration else { return }
            pulsePhase = pulseItems.isEmpty ? .failed("Unable to load Pulse.") : .content
        }
    }

    func loadSlot(_ slotID: UUID) async throws -> SlotModel {
        do { return try await api.slot(slotID) }
        catch let error as APIError {
            if case .unauthorized = error { await session.clearLocalSession() }
            throw error
        }
    }

    func searchPlaces(_ query: String) async throws -> [PlaceModel] {
        do { return try await api.searchPlaces(query: query) }
        catch let error as APIError {
            if case .unauthorized = error { await session.clearLocalSession() }
            throw error
        }
    }

    func createSlot(_ body: CreateSlotBody) async throws -> SlotModel? {
        try await performMutation {
            let slot = try await api.createSlot(body)
            pulseGeneration &+= 1
            reconcile(slot)
            return slot
        }
    }

    func request(_ slot: SlotModel) async throws -> SlotModel? {
        try await performMutation { let updated = try await api.requestSlot(slot.id); reconcile(updated); return updated }
    }

    func leave(_ slot: SlotModel) async throws -> SlotModel? {
        try await performMutation { let updated = try await api.leaveSlot(slot.id); reconcile(updated); return updated }
    }

    func cancel(_ slot: SlotModel) async throws -> SlotModel? {
        try await performMutation {
            let updated = try await api.cancelSlot(slot.id, expectedVersion: slot.version)
            pulseItems.removeAll { $0.id == slot.id }
            pulsePhase = pulseItems.isEmpty ? .empty : .content
            return updated
        }
    }

    func start(_ slot: SlotModel) async throws -> SlotModel? {
        try await performMutation { let updated = try await api.startSlot(slot.id); reconcile(updated); return updated }
    }

    func complete(_ slot: SlotModel) async throws -> SlotModel? {
        try await performMutation {
            let updated = try await api.completeSlot(slot.id)
            pulseItems.removeAll { $0.id == slot.id }
            pulsePhase = pulseItems.isEmpty ? .empty : .content
            return updated
        }
    }

    func clearMutationError() { mutationError = nil }

    func dispose() {
        pulseGeneration &+= 1
        pulseItems = []
        pulsePhase = .idle
        mutationError = nil
    }

    private func performMutation(_ work: () async throws -> SlotModel) async throws -> SlotModel? {
        guard !isMutating else { return nil }
        isMutating = true
        mutationError = nil
        defer { isMutating = false }
        do { return try await work() }
        catch is CancellationError { throw CancellationError() }
        catch let error as APIError {
            if case .unauthorized = error { await session.clearLocalSession() }
            else { mutationError = error.localizedDescription }
            throw error
        } catch {
            mutationError = "The action could not be completed."
            throw error
        }
    }

    private func reconcile(_ slot: SlotModel) {
        guard !slot.isTerminal else {
            pulseItems.removeAll { $0.id == slot.id }
            pulsePhase = pulseItems.isEmpty ? .empty : .content
            return
        }
        if let index = pulseItems.firstIndex(where: { $0.id == slot.id }) { pulseItems[index] = slot }
        else if slot.state == .filling || slot.state == .full { pulseItems.insert(slot, at: 0) }
        pulsePhase = pulseItems.isEmpty ? .empty : .content
    }
}
