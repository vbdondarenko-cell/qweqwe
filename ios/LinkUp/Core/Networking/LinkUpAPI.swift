import Foundation

actor LinkUpAPI {
    let client: APIClient
    let credentials: KeychainSessionStore

    private let mutationStore: MutationKeyStore
    private var pendingMutationKeys: [String: StoredMutationKey]
    private let durableOutbox: DurableMutationOutbox

    init(
        client: APIClient,
        credentials: KeychainSessionStore,
        durableOutbox: DurableMutationOutbox = DurableMutationOutbox()
    ) {
        self.client = client
        self.credentials = credentials
        let mutationStore = MutationKeyStore()
        self.mutationStore = mutationStore
        self.pendingMutationKeys = mutationStore.load()
        self.durableOutbox = durableOutbox
    }

    func encodeBody<Value: Encodable>(_ value: Value) throws -> Data {
        try APICoding.encoder().encode(value)
    }

    func uuidPath(_ value: UUID) -> String {
        value.uuidString.lowercased()
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
            if let existing = try await durableOutbox.replayableEquivalent(
                ownerFingerprint: ownerFingerprint,
                requestIdentity: identity,
                responseKind: responseKind,
                expectedSlotID: expectedSlotID,
                now: now
            ) {
                command = existing
            } else {
                let legacy = pendingMutationKeys[identity]
                if let legacy, now < legacy.touchedAt || now.timeIntervalSince(legacy.touchedAt) > durableMutationReplayWindow {
                    throw APIError.mutationSafetyBlocked
                }
                command = DurableMutationCommand(
                    idempotencyKey: legacy?.key ?? UUID(),
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
