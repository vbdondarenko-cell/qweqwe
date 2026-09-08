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
    @Published private(set) var durableMutationBlocked = false
    @Published private(set) var queuedMutationKey: UUID?
    @Published private(set) var lastDurableReplayReport = DurableMutationReplayReport.empty
    @Published private(set) var mutationError: String?
    @Published private(set) var realtimeRevision: UInt64 = 0
    @Published private(set) var discoveryRevision: UInt64 = 0

    var mutationControlsDisabled: Bool { isMutating || durableMutationBlocked }

    private var slotRealtimeRevisions: [UUID: UInt64] = [:]
    private var chatRealtimeRevisions: [UUID: UInt64] = [:]
    private var relationshipRealtimeRevisions: [UUID: UInt64] = [:]
    private var accessLostRealtimeRevisions: [UUID: UInt64] = [:]
    private var stagedRelationshipSnapshots: [UUID: (pending: [PendingSlotRequest], accepted: [SlotOrganizer])] = [:]
    private var stagedChatSnapshots: [UUID: [ChatMessage]] = [:]
    private var activeSlot: SlotModel?
    private var activeChatSlotID: UUID?

    private let api: LinkUpAPI
    private let session: SessionCoordinator
    private var pulseGeneration: UInt64 = 0

    init(api: LinkUpAPI, session: SessionCoordinator) {
        self.api = api
        self.session = session
    }

    @discardableResult
    func loadPulse() async -> Bool {
        pulseGeneration &+= 1
        let generation = pulseGeneration
        pulsePhase = pulseItems.isEmpty ? .loading : .refreshing
        do {
            let items = try await api.pulse()
            guard generation == pulseGeneration else { return false }
            pulseItems = items
            pulsePhase = items.isEmpty ? .empty : .content
            return true
        } catch is CancellationError {
            return false
        } catch let error as APIError {
            guard generation == pulseGeneration else { return false }
            if case .unauthorized = error { await session.clearLocalSession(); return false }
            pulsePhase = pulseItems.isEmpty ? .failed(error.localizedDescription) : .content
            return false
        } catch {
            guard generation == pulseGeneration else { return false }
            pulsePhase = pulseItems.isEmpty ? .failed("Unable to load Pulse.") : .content
            return false
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

    func edit(_ slot: SlotModel, body: EditSlotBody) async throws -> SlotModel? {
        try await performMutation {
            let updated = try await api.editSlot(slot.id, body: body)
            reconcile(updated)
            return updated
        }
    }

    func request(_ slot: SlotModel) async throws -> SlotModel? {
        try await performMutation { let updated = try await api.requestSlot(slot.id); reconcile(updated); return updated }
    }

    func leave(_ slot: SlotModel) async throws -> SlotModel? {
        try await performMutation { let updated = try await api.leaveSlot(slot.id); reconcile(updated); return updated }
    }

    func approve(_ slot: SlotModel, userID: UUID) async throws -> SlotModel? {
        try await performMutation {
            let updated = try await api.approveRequest(slot.id, userID: userID)
            reconcile(updated)
            return updated
        }
    }

    func reject(_ slot: SlotModel, userID: UUID) async throws -> SlotModel? {
        try await performMutation {
            let updated = try await api.rejectRequest(slot.id, userID: userID)
            reconcile(updated)
            return updated
        }
    }

    func removeParticipant(_ slot: SlotModel, userID: UUID) async throws -> SlotModel? {
        try await performMutation {
            let updated = try await api.removeParticipant(slot.id, userID: userID, expectedVersion: slot.version)
            reconcile(updated)
            return updated
        }
    }

    func block(_ slot: SlotModel, userID: UUID) async throws -> SlotModel? {
        try await performMutation {
            try await api.blockUser(userID)
            let updated = try await api.slot(slot.id)
            reconcile(updated)
            return updated
        }
    }

    func blockOrganizerAndRevoke(_ slot: SlotModel) async throws -> Bool? {
        guard case .signedIn(let currentUser) = session.state,
              currentUser.id != slot.organizer.id,
              slot.viewerState != .host else {
            throw APIError.protocolViolation("The current account cannot block this organizer from this Slot.")
        }

        return try await performMutation {
            try await api.blockUser(slot.organizer.id)
            applyLocalAccessLoss(slot.id)
            return true
        }
    }

    func cancel(_ slot: SlotModel) async throws -> SlotModel? {
        try await performMutation {
            let updated = try await api.cancelSlot(slot.id, expectedVersion: slot.version)
            reconcile(updated)
            return updated
        }
    }

    func start(_ slot: SlotModel) async throws -> SlotModel? {
        try await performMutation { let updated = try await api.startSlot(slot.id); reconcile(updated); return updated }
    }

    func complete(_ slot: SlotModel) async throws -> SlotModel? {
        try await performMutation {
            let updated = try await api.completeSlot(slot.id)
            reconcile(updated)
            return updated
        }
    }

    func registerActiveSlot(_ slot: SlotModel) {
        activeSlot = slot
    }

    func updateActiveSlot(_ slot: SlotModel) {
        guard activeSlot?.id == slot.id else { return }
        activeSlot = slot
    }

    func unregisterActiveSlot(_ slotID: UUID) {
        if activeSlot?.id == slotID { activeSlot = nil }
        stagedRelationshipSnapshots.removeValue(forKey: slotID)
    }

    func registerActiveChat(_ slotID: UUID) {
        activeChatSlotID = slotID
    }

    func unregisterActiveChat(_ slotID: UUID) {
        if activeChatSlotID == slotID { activeChatSlotID = nil }
        stagedChatSnapshots.removeValue(forKey: slotID)
    }

    func realtimeTargets() -> (slot: SlotModel?, chatSlotID: UUID?) {
        (activeSlot, activeChatSlotID)
    }

    func stageRealtimeRelationshipSnapshot(
        slotID: UUID,
        pending: [PendingSlotRequest],
        accepted: [SlotOrganizer]
    ) {
        stagedRelationshipSnapshots[slotID] = (pending, accepted)
    }

    func stageRealtimeChatSnapshot(slotID: UUID, messages: [ChatMessage]) {
        stagedChatSnapshots[slotID] = messages
    }

    func applyRealtimeHints(
        _ hints: RealtimeInvalidationHints,
        accessLostSlotIDs: Set<UUID> = [],
        additionalRefreshedSlotIDs: Set<UUID> = []
    ) {
        realtimeRevision &+= 1
        let revision = realtimeRevision
        if hints.refreshPulse || !accessLostSlotIDs.isEmpty { discoveryRevision &+= 1 }
        let refreshedSlotIDs = hints.slotIDs.union(additionalRefreshedSlotIDs)
        for id in refreshedSlotIDs { slotRealtimeRevisions[id] = revision }
        for id in hints.chatSlotIDs { chatRealtimeRevisions[id] = revision }
        if hints.refreshRelationships {
            for id in refreshedSlotIDs { relationshipRealtimeRevisions[id] = revision }
        }
        for id in accessLostSlotIDs {
            accessLostRealtimeRevisions[id] = revision
            stagedRelationshipSnapshots.removeValue(forKey: id)
            stagedChatSnapshots.removeValue(forKey: id)
        }
        pruneRealtimeRevisions()
    }

    func slotRealtimeRevision(for slotID: UUID) -> UInt64 { slotRealtimeRevisions[slotID] ?? 0 }
    func chatRealtimeRevision(for slotID: UUID) -> UInt64 { chatRealtimeRevisions[slotID] ?? 0 }
    func relationshipRealtimeRevision(for slotID: UUID) -> UInt64 { relationshipRealtimeRevisions[slotID] ?? 0 }
    func accessLostRealtimeRevision(for slotID: UUID) -> UInt64 { accessLostRealtimeRevisions[slotID] ?? 0 }
    func activeSlotSnapshot(for slotID: UUID) -> SlotModel? { activeSlot?.id == slotID ? activeSlot : nil }
    func realtimeRelationshipSnapshot(for slotID: UUID) -> (pending: [PendingSlotRequest], accepted: [SlotOrganizer])? {
        stagedRelationshipSnapshots[slotID]
    }
    func realtimeChatSnapshot(for slotID: UUID) -> [ChatMessage]? { stagedChatSnapshots[slotID] }

    func registerQueuedMutation(_ key: UUID) {
        durableMutationBlocked = true
        queuedMutationKey = key
        mutationError = APIError.mutationQueued(key).localizedDescription
    }

    func applyDurableReplayReport(_ report: DurableMutationReplayReport) {
        lastDurableReplayReport = report
        if report.needsSafetyIntervention {
            durableMutationBlocked = true
            mutationError = APIError.mutationSafetyBlocked.localizedDescription
            return
        }
        if let ambiguous = report.ambiguousFailureKey {
            durableMutationBlocked = true
            queuedMutationKey = queuedMutationKey ?? ambiguous
            mutationError = APIError.mutationQueued(ambiguous).localizedDescription
            return
        }

        if let queuedMutationKey,
           report.acknowledgedKeys.contains(queuedMutationKey) || report.definitiveFailureKey == queuedMutationKey {
            self.queuedMutationKey = nil
        }
        durableMutationBlocked = queuedMutationKey != nil
        if report.madeProgress {
            mutationError = report.definitiveFailureKey == nil
                ? nil
                : "A queued action was rejected by the server after reconciliation."
        } else if !durableMutationBlocked {
            mutationError = nil
        }
    }

    func applyDurableReplayFailure(_ error: APIError) {
        switch error {
        case .mutationSafetyBlocked, .mutationJournalUnavailable:
            durableMutationBlocked = true
            mutationError = error.localizedDescription
        default:
            break
        }
    }

    func clearMutationError() {
        if !durableMutationBlocked { mutationError = nil }
    }

    func dispose() {
        pulseGeneration &+= 1
        pulseItems = []
        pulsePhase = .idle
        durableMutationBlocked = false
        queuedMutationKey = nil
        lastDurableReplayReport = .empty
        mutationError = nil
        realtimeRevision = 0
        discoveryRevision = 0
        slotRealtimeRevisions.removeAll(keepingCapacity: false)
        chatRealtimeRevisions.removeAll(keepingCapacity: false)
        relationshipRealtimeRevisions.removeAll(keepingCapacity: false)
        accessLostRealtimeRevisions.removeAll(keepingCapacity: false)
        stagedRelationshipSnapshots.removeAll(keepingCapacity: false)
        stagedChatSnapshots.removeAll(keepingCapacity: false)
        activeSlot = nil
        activeChatSlotID = nil
    }

    private func performMutation<Value>(_ work: () async throws -> Value) async throws -> Value? {
        guard !mutationControlsDisabled else { return nil }
        isMutating = true
        mutationError = nil
        defer { isMutating = false }
        do { return try await work() }
        catch is CancellationError { throw CancellationError() }
        catch let error as APIError {
            if case .unauthorized = error {
                await session.clearLocalSession()
            } else {
                if case .mutationQueued(let key) = error { registerQueuedMutation(key) }
                if case .mutationSafetyBlocked = error { durableMutationBlocked = true }
                if case .mutationJournalUnavailable = error { durableMutationBlocked = true }
                mutationError = error.localizedDescription
            }
            throw error
        } catch {
            mutationError = "The action could not be completed."
            throw error
        }
    }

    private func applyLocalAccessLoss(_ slotID: UUID) {
        pulseGeneration &+= 1
        pulseItems.removeAll { $0.id == slotID }
        pulsePhase = pulseItems.isEmpty ? .empty : .content
        if activeSlot?.id == slotID { activeSlot = nil }
        if activeChatSlotID == slotID { activeChatSlotID = nil }
        stagedRelationshipSnapshots.removeValue(forKey: slotID)
        stagedChatSnapshots.removeValue(forKey: slotID)

        realtimeRevision &+= 1
        discoveryRevision &+= 1
        accessLostRealtimeRevisions[slotID] = realtimeRevision
        pruneRealtimeRevisions()
    }

    private func pruneRealtimeRevisions() {
        guard realtimeRevision > 1_024 else { return }
        let floor = realtimeRevision - 1_024
        slotRealtimeRevisions = slotRealtimeRevisions.filter { $0.value >= floor }
        chatRealtimeRevisions = chatRealtimeRevisions.filter { $0.value >= floor }
        relationshipRealtimeRevisions = relationshipRealtimeRevisions.filter { $0.value >= floor }
        accessLostRealtimeRevisions = accessLostRealtimeRevisions.filter { $0.value >= floor }
        stagedRelationshipSnapshots = stagedRelationshipSnapshots.filter {
            (relationshipRealtimeRevisions[$0.key] ?? 0) >= floor
        }
        stagedChatSnapshots = stagedChatSnapshots.filter {
            (chatRealtimeRevisions[$0.key] ?? 0) >= floor
        }
    }

    private func reconcile(_ slot: SlotModel) {
        if activeSlot?.id == slot.id { activeSlot = slot }
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
