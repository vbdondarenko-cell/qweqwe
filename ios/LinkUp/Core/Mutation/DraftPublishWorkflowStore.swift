import Foundation

struct DraftPublishWorkflow: Codable, Equatable, Sendable {
    let ownerFingerprint: String
    let requestIdentity: String
    let createBody: Data
    let createKey: UUID
    let publishKey: UUID
    let cancelKey: UUID
    let draftID: UUID?
    let draftVersion: Int64?
    let createdAt: Date
    let requiresAttention: Bool

    var hasValidShape: Bool {
        let hex = "0123456789abcdef"
        let draftPairValid = (draftID == nil && draftVersion == nil) || (draftID != nil && (draftVersion ?? 0) > 0)
        let request = APIRequest(method: .post, path: "/v1/slots/drafts", body: createBody)
        let keysDistinct = Set([createKey, publishKey, cancelKey]).count == 3
        return ownerFingerprint.count == 64 && ownerFingerprint.allSatisfy { hex.contains($0) } &&
            requestIdentity.count == 64 && requestIdentity.allSatisfy { hex.contains($0) } &&
            !createBody.isEmpty && createBody.count <= durableMutationMaxBodyBytes &&
            keysDistinct && draftPairValid && createdAt.timeIntervalSince1970 > 0 &&
            MutationIdentity.digest(for: request) == requestIdentity
    }

    func canAutoResume(at now: Date) -> Bool {
        guard hasValidShape, now >= createdAt else { return false }
        return now.timeIntervalSince(createdAt) <= durableMutationReplayWindow
    }

    var canManuallyResolve: Bool {
        requiresAttention && draftID != nil && draftVersion != nil
    }

    func recordingDraft(id: UUID, version: Int64) -> DraftPublishWorkflow {
        DraftPublishWorkflow(
            ownerFingerprint: ownerFingerprint, requestIdentity: requestIdentity, createBody: createBody,
            createKey: createKey, publishKey: publishKey, cancelKey: cancelKey, draftID: id,
            draftVersion: version, createdAt: createdAt, requiresAttention: requiresAttention
        )
    }

    func preparingRetry(version: Int64) -> DraftPublishWorkflow? {
        guard draftID != nil, version > 0 else { return nil }
        return DraftPublishWorkflow(
            ownerFingerprint: ownerFingerprint, requestIdentity: requestIdentity, createBody: createBody,
            createKey: createKey, publishKey: publishKey, cancelKey: cancelKey, draftID: draftID,
            draftVersion: version, createdAt: createdAt, requiresAttention: false
        )
    }

    func markingNeedsAttention() -> DraftPublishWorkflow {
        DraftPublishWorkflow(
            ownerFingerprint: ownerFingerprint, requestIdentity: requestIdentity, createBody: createBody,
            createKey: createKey, publishKey: publishKey, cancelKey: cancelKey, draftID: draftID,
            draftVersion: draftVersion, createdAt: createdAt, requiresAttention: true
        )
    }
}

enum DraftPublishWorkflowStoreError: Error, Sendable {
    case corrupt
    case invalid
}

actor DraftPublishWorkflowStore {
    private let fileURL: URL
    private let maxBytes = 128 * 1024

    init(baseDirectory: URL? = nil) {
        let manager = FileManager.default
        let root = baseDirectory
            ?? manager.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
            ?? manager.urls(for: .libraryDirectory, in: .userDomainMask).first
            ?? URL(fileURLWithPath: NSHomeDirectory(), isDirectory: true)
        fileURL = root.appendingPathComponent("LinkUp", isDirectory: true)
            .appendingPathComponent("DraftPublish", isDirectory: true)
            .appendingPathComponent("v1.json", isDirectory: false)
    }

    func load() throws -> DraftPublishWorkflow? {
        let manager = FileManager.default
        guard manager.fileExists(atPath: fileURL.path) else { return nil }
        do {
            let attributes = try manager.attributesOfItem(atPath: fileURL.path)
            if let size = attributes[.size] as? NSNumber, size.intValue > maxBytes {
                throw DraftPublishWorkflowStoreError.corrupt
            }
            let data = try Data(contentsOf: fileURL, options: [.mappedIfSafe])
            let workflow = try JSONDecoder().decode(DraftPublishWorkflow.self, from: data)
            guard workflow.hasValidShape else { throw DraftPublishWorkflowStoreError.corrupt }
            return workflow
        } catch let error as DraftPublishWorkflowStoreError {
            throw error
        } catch {
            throw DraftPublishWorkflowStoreError.corrupt
        }
    }

    func save(_ workflow: DraftPublishWorkflow) throws {
        guard workflow.hasValidShape else { throw DraftPublishWorkflowStoreError.invalid }
        let manager = FileManager.default
        let directory = fileURL.deletingLastPathComponent()
        try manager.createDirectory(at: directory, withIntermediateDirectories: true)
        let data = try JSONEncoder().encode(workflow)
        guard data.count <= maxBytes else { throw DraftPublishWorkflowStoreError.invalid }
        try data.write(to: fileURL, options: [.atomic, .completeFileProtectionUntilFirstUserAuthentication])
        try TransientStateFilePolicy.excludeFromBackup(fileURL)
    }

    func markNeedsAttention() throws {
        guard let workflow = try load() else { return }
        try save(workflow.markingNeedsAttention())
    }

    func clear() throws {
        if FileManager.default.fileExists(atPath: fileURL.path) {
            try FileManager.default.removeItem(at: fileURL)
        }
    }
}
