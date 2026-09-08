import Foundation

enum DurableMutationOutboxError: Error, Sendable {
    case corruptJournal
    case full
    case invalidCommand
}

actor DurableMutationOutbox {
    private let fileURL: URL
    private let maxJournalBytes = 10 * 1024 * 1024

    init(baseDirectory: URL? = nil) {
        let manager = FileManager.default
        let root = baseDirectory
            ?? manager.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
            ?? manager.urls(for: .libraryDirectory, in: .userDomainMask).first
            ?? URL(fileURLWithPath: NSHomeDirectory(), isDirectory: true)
        fileURL = root
            .appendingPathComponent("LinkUp", isDirectory: true)
            .appendingPathComponent("MutationOutbox", isDirectory: true)
            .appendingPathComponent("v1.json", isDirectory: false)
    }

    func enqueue(_ command: DurableMutationCommand) throws {
        guard command.hasValidShape else { throw DurableMutationOutboxError.invalidCommand }
        var commands = try readCommands()
        if commands.contains(where: { $0.idempotencyKey == command.idempotencyKey }) { return }
        guard commands.count < durableMutationMaxCommands else { throw DurableMutationOutboxError.full }
        commands.append(command)
        try writeCommands(commands)
    }

    func replayableEquivalent(
        ownerFingerprint: String,
        requestIdentity: String,
        responseKind: DurableMutationResponseKind,
        expectedSlotID: UUID?,
        now: Date
    ) throws -> DurableMutationCommand? {
        try readCommands()
            .filter { command in
                command.ownerFingerprint == ownerFingerprint &&
                command.requestIdentity == requestIdentity &&
                command.responseKind == responseKind &&
                command.expectedSlotID == expectedSlotID &&
                command.canAutoReplay(at: now)
            }
            .sorted { $0.createdAt < $1.createdAt }
            .first
    }

    func pendingReplayable(ownerFingerprint: String, now: Date) throws -> [DurableMutationCommand] {
        try readCommands()
            .filter { $0.ownerFingerprint == ownerFingerprint && $0.canAutoReplay(at: now) }
            .sorted { $0.createdAt < $1.createdAt }
    }

    func unsafeAmbiguousCount(ownerFingerprint: String, now: Date) throws -> Int {
        try readCommands().count { command in
            command.ownerFingerprint == ownerFingerprint &&
            command.firstAttemptAt != nil &&
            !command.canAutoReplay(at: now)
        }
    }

    func markAttempt(idempotencyKey: UUID, now: Date) throws -> DurableMutationCommand? {
        var commands = try readCommands()
        guard let index = commands.firstIndex(where: { $0.idempotencyKey == idempotencyKey }) else { return nil }
        guard let updated = commands[index].markingAttempt(at: now) else {
            throw DurableMutationOutboxError.invalidCommand
        }
        if updated != commands[index] {
            commands[index] = updated
            try writeCommands(commands)
        }
        return updated
    }

    func remove(idempotencyKey: UUID) throws {
        let commands = try readCommands()
        let retained = commands.filter { $0.idempotencyKey != idempotencyKey }
        if retained.count != commands.count { try writeCommands(retained) }
    }

    func clearOwner(_ ownerFingerprint: String) throws {
        let commands = try readCommands()
        let retained = commands.filter { $0.ownerFingerprint != ownerFingerprint }
        if retained.count != commands.count { try writeCommands(retained) }
    }

    func clearAll() throws {
        if FileManager.default.fileExists(atPath: fileURL.path) {
            try FileManager.default.removeItem(at: fileURL)
        }
    }

    private func readCommands() throws -> [DurableMutationCommand] {
        let manager = FileManager.default
        guard manager.fileExists(atPath: fileURL.path) else { return [] }
        do {
            let attributes = try manager.attributesOfItem(atPath: fileURL.path)
            if let size = attributes[.size] as? NSNumber, size.intValue > maxJournalBytes {
                throw DurableMutationOutboxError.corruptJournal
            }
            let data = try Data(contentsOf: fileURL, options: [.mappedIfSafe])
            let commands = try JSONDecoder().decode([DurableMutationCommand].self, from: data)
            guard commands.count <= durableMutationMaxCommands,
                  commands.allSatisfy(\.hasValidShape),
                  Set(commands.map(\.idempotencyKey)).count == commands.count else {
                throw DurableMutationOutboxError.corruptJournal
            }
            return commands
        } catch let error as DurableMutationOutboxError {
            throw error
        } catch {
            throw DurableMutationOutboxError.corruptJournal
        }
    }

    private func writeCommands(_ commands: [DurableMutationCommand]) throws {
        guard commands.count <= durableMutationMaxCommands,
              commands.allSatisfy(\.hasValidShape) else {
            throw DurableMutationOutboxError.invalidCommand
        }
        guard !commands.isEmpty else {
            try clearAll()
            return
        }
        let manager = FileManager.default
        let directory = fileURL.deletingLastPathComponent()
        try manager.createDirectory(at: directory, withIntermediateDirectories: true)
        let data = try JSONEncoder().encode(commands)
        guard data.count <= maxJournalBytes else { throw DurableMutationOutboxError.full }
        try data.write(
            to: fileURL,
            options: [.atomic, .completeFileProtectionUntilFirstUserAuthentication]
        )
    }
}
