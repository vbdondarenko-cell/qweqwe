import Foundation

actor LinkUpAPI {
    let client: APIClient
    let credentials: KeychainSessionStore

    private let mutationStore: MutationKeyStore
    private var pendingMutationKeys: [String: StoredMutationKey]
    private let durableOutbox: DurableMutationOutbox
    private let draftWorkflowStore: DraftPublishWorkflowStore
    private var explicitLogoutInProgress = false
    private var activeAuthenticatedWrites = 0

    init(
        client: APIClient,
        credentials: KeychainSessionStore,
        durableOutbox: DurableMutationOutbox = DurableMutationOutbox(),
        draftWorkflowStore: DraftPublishWorkflowStore = DraftPublishWorkflowStore()
    ) {
        self.client = client
        self.credentials = credentials
        let mutationStore = MutationKeyStore()
        self.mutationStore = mutationStore
        self.pendingMutationKeys = mutationStore.load()
        self.durableOutbox = durableOutbox
        self.draftWorkflowStore = draftWorkflowStore
    }

    func encodeBody<Value: Encodable>(_ value: Value) throws -> Data {
        try APICoding.encoder().encode(value)
    }

    func uuidPath(_ value: UUID) -> String {
        value.uuidString.lowercased()
    }

    func ensureAuthenticatedWriteAllowed() throws {
        if explicitLogoutInProgress {
            throw APIError.sessionTransitionInProgress
        }
    }

    func beginAuthenticatedWrite() throws {
        try ensureAuthenticatedWriteAllowed()
        activeAuthenticatedWrites += 1
    }

    func endAuthenticatedWrite() {
        activeAuthenticatedWrites = max(0, activeAuthenticatedWrites - 1)
    }

    func beginExplicitLogout() throws {
        guard !explicitLogoutInProgress else { throw APIError.sessionTransitionInProgress }
        explicitLogoutInProgress = true
    }

    func endExplicitLogout() {
        explicitLogoutInProgress = false
    }

    func assertExplicitLogoutSafe() async throws {
        let ownerFingerprint = try await currentMutationOwnerFingerprint()
        do {
            let ownerHasCommands = try await durableOutbox.hasCommands(ownerFingerprint: ownerFingerprint)
            let workflow = try await draftWorkflowStore.load()
            let workflowOwner = workflow?.ownerFingerprint
            guard ExplicitLogoutSafety.canProceed(
                activeAuthenticatedWrites: activeAuthenticatedWrites,
                legacyPendingCount: pendingMutationKeys.count,
                ownerHasCommands: ownerHasCommands,
                workflowOwnerFingerprint: workflowOwner,
                currentOwnerFingerprint: ownerFingerprint
            ) else {
                throw APIError.signOutBlockedByPendingAction
            }
        } catch let error as APIError {
            throw error
        } catch let error as DurableMutationOutboxError {
            throw mapOutboxError(error)
        } catch {
            throw APIError.mutationJournalUnavailable
        }
    }

    func sendIdempotent<Response: Decodable & Sendable>(
        _ request: APIRequest,
        responseKind: DurableMutationResponseKind,
        expectedSlotID: UUID? = nil,
        as type: Response.Type = Response.self,
        validate: @Sendable (Response) throws -> Void = { _ in }
    ) async throws -> Response {
        guard request.method != .get, request.authenticated else {
            throw APIError.protocolViolation("Durable mutation helper requires an authenticated non-GET request.")
        }
        try beginAuthenticatedWrite()
        defer { endAuthenticatedWrite() }
        guard (request.body?.count ?? 0) <= durableMutationMaxBodyBytes else {
            throw APIError.protocolViolation("Mutation body exceeds the durable safety limit.")
        }

        let ownerFingerprint = try await currentMutationOwnerFingerprint()
        let identity = MutationIdentity.digest(for: request)
        let now = Date()
        let command: DurableMutationCommand

        do {
            if try await durableOutbox.unsafeAmbiguousCount(ownerFingerprint: ownerFingerprint, now: now) > 0 {
                throw APIError.mutationSafetyBlocked
            }
            if let requestedKey = request.idempotencyKey,
               let exact = try await durableOutbox.command(idempotencyKey: requestedKey) {
                guard exact.ownerFingerprint == ownerFingerprint,
                      exact.requestIdentity == identity,
                      exact.responseKind == responseKind,
                      exact.expectedSlotID == expectedSlotID,
                      exact.canAutoReplay(at: now) else {
                    throw APIError.mutationSafetyBlocked
                }
                command = exact
            } else if let existing = try await durableOutbox.replayableEquivalent(
                ownerFingerprint: ownerFingerprint,
                requestIdentity: identity,
                responseKind: responseKind,
                expectedSlotID: expectedSlotID,
                now: now
            ) {
                if let requestedKey = request.idempotencyKey, existing.idempotencyKey != requestedKey {
                    throw APIError.mutationSafetyBlocked
                }
                command = existing
            } else {
                let legacy = pendingMutationKeys[identity]
                if let legacy, now < legacy.touchedAt || now.timeIntervalSince(legacy.touchedAt) > durableMutationReplayWindow {
                    throw APIError.mutationSafetyBlocked
                }
                command = DurableMutationCommand(
                    idempotencyKey: try resolveDurableMutationKey(
                        requested: request.idempotencyKey,
                        legacy: legacy
                    ),
                    ownerFingerprint: ownerFingerprint,
                    requestIdentity: identity,
                    method: request.method,
                    path: request.path,
                    queryItems: request.queryItems.map(DurableMutationQueryItem.init),
                    body: request.body,
                    responseKind: responseKind,
                    expectedSlotID: expectedSlotID,
                    createdAt: legacy?.touchedAt ?? now,
                    firstAttemptAt: legacy?.touchedAt
                )
                try await durableOutbox.enqueue(command)
                if legacy != nil { releaseLegacyMutationKey(identity) }
            }
        } catch let error as APIError {
            throw error
        } catch let error as DurableMutationOutboxError {
            throw mapOutboxError(error)
        } catch {
            throw APIError.mutationJournalUnavailable
        }

        let attempted: DurableMutationCommand
        do {
            guard let value = try await durableOutbox.markAttempt(
                idempotencyKey: command.idempotencyKey,
                now: max(now, command.createdAt)
            ) else {
                throw APIError.mutationJournalUnavailable
            }
            attempted = value
        } catch let error as APIError {
            throw error
        } catch let error as DurableMutationOutboxError {
            throw mapOutboxError(error)
        } catch {
            throw APIError.mutationJournalUnavailable
        }

        do {
            let response: Response = try await client.send(attempted.apiRequest, as: type)
            try validate(response)
            try await removeDurableCommand(attempted.idempotencyKey)
            releaseLegacyMutationKey(identity)
            return response
        } catch is CancellationError {
            // Once firstAttemptAt is durably recorded, cancellation is an ambiguous
            // transport outcome: the exact command must remain queued under the same key.
            throw APIError.mutationQueued(attempted.idempotencyKey)
        } catch let error as APIError {
            if error.isDefinitiveMutationFailure {
                try await removeDurableCommand(attempted.idempotencyKey)
                releaseLegacyMutationKey(identity)
                throw error
            }
            throw APIError.mutationQueued(attempted.idempotencyKey)
        } catch {
            throw APIError.mutationQueued(attempted.idempotencyKey)
        }
    }

    func replayPendingMutations() async throws -> DurableMutationReplayReport {
        let ownerFingerprint = try await currentMutationOwnerFingerprint()
        let now = Date()
        let unsafe: Int
        let commands: [DurableMutationCommand]
        do {
            unsafe = try await durableOutbox.unsafeAmbiguousCount(ownerFingerprint: ownerFingerprint, now: now)
            if unsafe > 0 {
                return DurableMutationReplayReport(
                    acknowledgedKeys: [],
                    definitiveFailureKey: nil,
                    ambiguousFailureKey: nil,
                    unsafeAmbiguousCount: unsafe
                )
            }
            commands = try await durableOutbox.pendingReplayable(ownerFingerprint: ownerFingerprint, now: now)
        } catch let error as DurableMutationOutboxError {
            throw mapOutboxError(error)
        } catch {
            throw APIError.mutationJournalUnavailable
        }

        var acknowledged = [UUID]()
        for command in commands {
            let attempted: DurableMutationCommand
            do {
                guard let value = try await durableOutbox.markAttempt(
                    idempotencyKey: command.idempotencyKey,
                    now: max(Date(), command.createdAt)
                ) else { continue }
                attempted = value
            } catch let error as DurableMutationOutboxError {
                throw mapOutboxError(error)
            }

            do {
                try await executeReplay(attempted)
                try await removeDurableCommand(attempted.idempotencyKey)
                acknowledged.append(attempted.idempotencyKey)
            } catch is CancellationError {
                throw CancellationError()
            } catch let error as APIError {
                if error.isDefinitiveMutationFailure {
                    try await removeDurableCommand(attempted.idempotencyKey)
                    return DurableMutationReplayReport(
                        acknowledgedKeys: acknowledged,
                        definitiveFailureKey: attempted.idempotencyKey,
                        ambiguousFailureKey: nil,
                        unsafeAmbiguousCount: 0
                    )
                }
                return DurableMutationReplayReport(
                    acknowledgedKeys: acknowledged,
                    definitiveFailureKey: nil,
                    ambiguousFailureKey: attempted.idempotencyKey,
                    unsafeAmbiguousCount: 0
                )
            } catch {
                return DurableMutationReplayReport(
                    acknowledgedKeys: acknowledged,
                    definitiveFailureKey: nil,
                    ambiguousFailureKey: attempted.idempotencyKey,
                    unsafeAmbiguousCount: 0
                )
            }
        }
        return DurableMutationReplayReport(
            acknowledgedKeys: acknowledged,
            definitiveFailureKey: nil,
            ambiguousFailureKey: nil,
            unsafeAmbiguousCount: 0
        )
    }

    func saveCredential(from envelope: AuthEnvelope) async throws {
        guard OpaqueTokenContract.canonical32ByteBase64URL(envelope.token) != nil,
              envelope.expiresAt > Date() else {
            throw APIError.protocolViolation("Server returned an invalid session credential.")
        }
        do {
            try await durableOutbox.clearAll()
            try await draftWorkflowStore.clear()
        } catch {
            throw APIError.mutationJournalUnavailable
        }
        mutationStore.persist([:])
        pendingMutationKeys.removeAll(keepingCapacity: false)
        do {
            try await credentials.save(SessionCredential(token: envelope.token, expiresAt: envelope.expiresAt))
        } catch {
            throw APIError.secureStorageUnavailable
        }
    }

    func clearLocalSession() async {
        await credentials.clear()
        mutationStore.persist([:])
        pendingMutationKeys.removeAll(keepingCapacity: false)
        try? await durableOutbox.clearAll()
        try? await draftWorkflowStore.clear()
    }

    func beginDraftPublishWorkflow(_ body: CreateSlotBody) async throws -> SlotModel {
        try beginAuthenticatedWrite()
        defer { endAuthenticatedWrite() }
        let ownerFingerprint = try await currentMutationOwnerFingerprint()
        let encoded = try encodeBody(body)
        let identity = MutationIdentity.digest(for: APIRequest(
            method: .post,
            path: "/v1/slots/drafts",
            body: encoded
        ))
        let now = Date()
        var workflow: DraftPublishWorkflow

        do {
            if let existing = try await draftWorkflowStore.load() {
                guard existing.ownerFingerprint == ownerFingerprint,
                      existing.requestIdentity == identity,
                      existing.canAutoResume(at: now),
                      !existing.requiresAttention else {
                    throw APIError.mutationSafetyBlocked
                }
                workflow = existing
            } else {
                workflow = DraftPublishWorkflow(
                    ownerFingerprint: ownerFingerprint,
                    requestIdentity: identity,
                    createBody: encoded,
                    createKey: UUID(),
                    publishKey: UUID(),
                    cancelKey: UUID(),
                    draftID: nil,
                    draftVersion: nil,
                    createdAt: now,
                    requiresAttention: false
                )
                try await saveDraftWorkflow(workflow)
            }
        } catch let error as APIError {
            throw error
        } catch {
            throw APIError.mutationJournalUnavailable
        }

        return try await continueDraftPublishWorkflow(workflow)
    }

    func resumeDraftPublishWorkflow() async throws -> SlotModel? {
        let ownerFingerprint = try await currentMutationOwnerFingerprint()
        let now = Date()
        let workflow: DraftPublishWorkflow
        do {
            guard let existing = try await draftWorkflowStore.load() else { return nil }
            guard existing.ownerFingerprint == ownerFingerprint else {
                throw APIError.mutationSafetyBlocked
            }
            if !existing.canAutoResume(at: now) && !existing.requiresAttention {
                let attention = existing.markingNeedsAttention()
                try await saveDraftWorkflow(attention)
                return nil
            }
            guard !existing.requiresAttention else { return nil }
            workflow = existing
        } catch let error as APIError {
            throw error
        } catch {
            throw APIError.mutationJournalUnavailable
        }
        return try await continueDraftPublishWorkflow(workflow)
    }

    func pendingDraftPublishWorkflow() async throws -> DraftPublishWorkflow? {
        let ownerFingerprint = try await currentMutationOwnerFingerprint()
        do {
            guard let workflow = try await draftWorkflowStore.load() else { return nil }
            guard workflow.ownerFingerprint == ownerFingerprint else {
                throw APIError.mutationSafetyBlocked
            }
            if !workflow.canAutoResume(at: Date()) && !workflow.requiresAttention {
                let attention = workflow.markingNeedsAttention()
                try await saveDraftWorkflow(attention)
                return attention
            }
            return workflow
        } catch let error as APIError {
            throw error
        } catch {
            throw APIError.mutationJournalUnavailable
        }
    }

    func retryDraftPublishWorkflow() async throws -> SlotModel {
        let ownerFingerprint = try await currentMutationOwnerFingerprint()
        let workflow: DraftPublishWorkflow
        do {
            guard let existing = try await draftWorkflowStore.load() else {
                throw APIError.protocolViolation("No saved draft publishing workflow exists.")
            }
            guard existing.ownerFingerprint == ownerFingerprint, existing.canManuallyResolve else {
                throw APIError.mutationSafetyBlocked
            }
            workflow = existing
        } catch let error as APIError {
            throw error
        } catch {
            throw APIError.mutationJournalUnavailable
        }

        guard let draftID = workflow.draftID else {
            throw APIError.mutationSafetyBlocked
        }
        let current = try await slot(draftID)
        try validateDraftCreateReplay(current)
        guard current.state == .draft else {
            try await clearDraftWorkflow()
            return current
        }
        guard let retry = workflow.preparingRetry(version: current.version) else {
            throw APIError.mutationJournalUnavailable
        }
        try await preparePublishCommandForRetry(workflow)
        try await saveDraftWorkflow(retry)
        return try await continueDraftPublishWorkflow(retry)
    }

    func discardDraftPublishWorkflow() async throws -> SlotModel? {
        let ownerFingerprint = try await currentMutationOwnerFingerprint()
        let workflow: DraftPublishWorkflow
        do {
            guard let existing = try await draftWorkflowStore.load() else { return nil }
            guard existing.ownerFingerprint == ownerFingerprint else {
                throw APIError.mutationSafetyBlocked
            }
            workflow = existing
        } catch let error as APIError {
            throw error
        } catch {
            throw APIError.mutationJournalUnavailable
        }

        try await markDraftWorkflowNeedsAttention()
        try await ensureWorkflowCommandsAreSafeToDiscard(workflow)

        guard let draftID = workflow.draftID else {
            throw APIError.mutationSafetyBlocked
        }

        let current = try await slot(draftID)
        try validateDraftCreateReplay(current)
        guard current.state == .draft else {
            try await clearDraftWorkflow()
            return current
        }

        do {
            let cancelled = try await cancelSlot(
                draftID,
                expectedVersion: current.version,
                idempotencyKey: workflow.cancelKey
            )
            guard cancelled.state == .cancelled else {
                throw APIError.protocolViolation("Draft discard did not return a cancelled Slot.")
            }
            try await clearDraftWorkflow()
            return cancelled
        } catch let error as APIError {
            if error.isDefinitiveMutationFailure {
                try? await markDraftWorkflowNeedsAttention()
            }
            throw error
        }
    }

    private func preparePublishCommandForRetry(_ workflow: DraftPublishWorkflow) async throws {
        do {
            guard let command = try await durableOutbox.command(idempotencyKey: workflow.publishKey) else { return }
            guard command.ownerFingerprint == workflow.ownerFingerprint, command.firstAttemptAt == nil else {
                throw APIError.mutationSafetyBlocked
            }
            try await durableOutbox.remove(idempotencyKey: workflow.publishKey)
        } catch let error as APIError {
            throw error
        } catch let error as DurableMutationOutboxError {
            throw mapOutboxError(error)
        } catch {
            throw APIError.mutationJournalUnavailable
        }
    }

    private func ensureWorkflowCommandsAreSafeToDiscard(_ workflow: DraftPublishWorkflow) async throws {
        do {
            for key in [workflow.createKey, workflow.publishKey] {
                guard let command = try await durableOutbox.command(idempotencyKey: key) else { continue }
                guard command.ownerFingerprint == workflow.ownerFingerprint else {
                    throw APIError.mutationSafetyBlocked
                }
                if command.firstAttemptAt != nil {
                    throw APIError.mutationSafetyBlocked
                }
                try await durableOutbox.remove(idempotencyKey: key)
            }
        } catch let error as APIError {
            throw error
        } catch let error as DurableMutationOutboxError {
            throw mapOutboxError(error)
        } catch {
            throw APIError.mutationJournalUnavailable
        }
    }

    private func continueDraftPublishWorkflow(_ initial: DraftPublishWorkflow) async throws -> SlotModel {
        var workflow = initial

        if workflow.draftID == nil {
            do {
                let created = try await createDraftSlot(
                    encodedBody: workflow.createBody,
                    idempotencyKey: workflow.createKey
                )
                if created.state != .draft {
                    try await clearDraftWorkflow()
                    return created
                }
                workflow = workflow.recordingDraft(id: created.id, version: created.version)
                try await saveDraftWorkflow(workflow)
            } catch let error as APIError {
                if error.isDefinitiveMutationFailure {
                    if case .http(let status, _, _, _) = error, status == 409 {
                        try? await markDraftWorkflowNeedsAttention()
                    } else {
                        try? await clearDraftWorkflow()
                    }
                }
                throw error
            }
        }

        guard let draftID = workflow.draftID, let draftVersion = workflow.draftVersion else {
            throw APIError.mutationJournalUnavailable
        }

        do {
            let published = try await publishDraftSlot(
                draftID,
                expectedVersion: draftVersion,
                idempotencyKey: workflow.publishKey
            )
            try await clearDraftWorkflow()
            return published
        } catch let error as APIError {
            if error.isDefinitiveMutationFailure {
                try? await markDraftWorkflowNeedsAttention()
            }
            throw error
        }
    }

    private func saveDraftWorkflow(_ workflow: DraftPublishWorkflow) async throws {
        do { try await draftWorkflowStore.save(workflow) }
        catch { throw APIError.mutationJournalUnavailable }
    }

    private func clearDraftWorkflow() async throws {
        do { try await draftWorkflowStore.clear() }
        catch { throw APIError.mutationJournalUnavailable }
    }

    private func markDraftWorkflowNeedsAttention() async throws {
        do { try await draftWorkflowStore.markNeedsAttention() }
        catch { throw APIError.mutationJournalUnavailable }
    }

    private func currentMutationOwnerFingerprint() async throws -> String {
        do {
            guard let credential = try await credentials.load() else { throw APIError.unauthorized }
            guard let fingerprint = MutationOwnerFingerprint.make(token: credential.token) else {
                throw APIError.protocolViolation("Invalid mutation owner credential.")
            }
            return fingerprint
        } catch let error as APIError {
            throw error
        } catch {
            throw APIError.secureStorageUnavailable
        }
    }

    private func executeReplay(_ command: DurableMutationCommand) async throws {
        switch command.responseKind {
        case .slot:
            let value: SlotModel = try await client.send(command.apiRequest)
            if let expectedSlotID = command.expectedSlotID, value.id != expectedSlotID {
                throw APIError.protocolViolation("Replayed mutation returned a different Slot.")
            }
            guard value.hasValidServerShape else {
                throw APIError.protocolViolation("Replayed mutation returned invalid Slot data.")
            }
        case .chat:
            let value: ChatMessage = try await client.send(command.apiRequest)
            guard let expectedSlotID = command.expectedSlotID,
                  value.slotId == expectedSlotID,
                  value.hasValidServerShape else {
                throw APIError.protocolViolation("Replayed chat mutation returned invalid data.")
            }
        }
    }

    private func removeDurableCommand(_ idempotencyKey: UUID) async throws {
        do {
            try await durableOutbox.remove(idempotencyKey: idempotencyKey)
        } catch let error as DurableMutationOutboxError {
            throw mapOutboxError(error)
        } catch {
            throw APIError.mutationJournalUnavailable
        }
    }

    private func mapOutboxError(_ error: DurableMutationOutboxError) -> APIError {
        switch error {
        case .full:
            .mutationSafetyBlocked
        case .corruptJournal, .invalidCommand:
            .mutationJournalUnavailable
        }
    }

    private func releaseLegacyMutationKey(_ identity: String) {
        pendingMutationKeys.removeValue(forKey: identity)
        mutationStore.persist(pendingMutationKeys)
    }
}


func resolveDurableMutationKey(requested: UUID?, legacy: StoredMutationKey?) throws -> UUID {
    if let requested, let legacy, legacy.key != requested {
        throw APIError.mutationSafetyBlocked
    }
    return requested ?? legacy?.key ?? UUID()
}

enum ExplicitLogoutSafety {
    static func canProceed(
        activeAuthenticatedWrites: Int,
        legacyPendingCount: Int,
        ownerHasCommands: Bool,
        workflowOwnerFingerprint: String?,
        currentOwnerFingerprint: String
    ) -> Bool {
        guard activeAuthenticatedWrites == 0,
              legacyPendingCount == 0,
              !ownerHasCommands,
              workflowOwnerFingerprint == nil else { return false }
        return currentOwnerFingerprint.count == 64
    }
}
