import CryptoKit
import Foundation

let durableMutationMaxCommands = 100
let durableMutationMaxBodyBytes = 64 * 1024
let durableMutationReplayWindow: TimeInterval = 20 * 60 * 60

enum DurableMutationResponseKind: String, Codable, Sendable {
    case slot
    case chat
}

struct DurableMutationQueryItem: Codable, Equatable, Sendable {
    let name: String
    let value: String?

    init(_ item: URLQueryItem) {
        name = item.name
        value = item.value
    }

    var urlQueryItem: URLQueryItem { URLQueryItem(name: name, value: value) }
}

struct DurableMutationCommand: Codable, Equatable, Identifiable, Sendable {
    let idempotencyKey: UUID
    let ownerFingerprint: String
    let requestIdentity: String
    let method: HTTPMethod
    let path: String
    let queryItems: [DurableMutationQueryItem]
    let body: Data?
    let responseKind: DurableMutationResponseKind
    let expectedSlotID: UUID?
    let createdAt: Date
    let firstAttemptAt: Date?

    var id: UUID { idempotencyKey }

    var hasValidShape: Bool {
        guard method != .get,
              (4...2_048).contains(path.utf8.count),
              path.hasPrefix("/v1/"),
              !path.contains("://"),
              !path.contains("\n"),
              !path.contains("\r"),
              queryItems.count <= 32,
              queryItems.allSatisfy({ item in
                  !item.name.isEmpty &&
                  item.name.utf8.count <= 256 &&
                  (item.value?.utf8.count ?? 0) <= 2_048
              }),
              (body?.count ?? 0) <= durableMutationMaxBodyBytes,
              ownerFingerprint.count == 64,
              requestIdentity.count == 64,
              ownerFingerprint.allSatisfy({ $0.isLowercaseHexDigit }),
              requestIdentity.allSatisfy({ $0.isLowercaseHexDigit }),
              createdAt.timeIntervalSince1970 > 0 else { return false }
        if let firstAttemptAt, firstAttemptAt < createdAt { return false }
        if responseKind == .chat && expectedSlotID == nil { return false }
        return MutationIdentity.digest(for: apiRequest) == requestIdentity
    }

    func canAutoReplay(at now: Date) -> Bool {
        guard hasValidShape else { return false }
        guard let firstAttemptAt else { return true }
        guard now >= firstAttemptAt else { return false }
        return now.timeIntervalSince(firstAttemptAt) <= durableMutationReplayWindow
    }

    func markingAttempt(at now: Date) -> DurableMutationCommand? {
        guard now >= createdAt else { return nil }
        guard firstAttemptAt == nil else { return self }
        return DurableMutationCommand(
            idempotencyKey: idempotencyKey,
            ownerFingerprint: ownerFingerprint,
            requestIdentity: requestIdentity,
            method: method,
            path: path,
            queryItems: queryItems,
            body: body,
            responseKind: responseKind,
            expectedSlotID: expectedSlotID,
            createdAt: createdAt,
            firstAttemptAt: now
        )
    }

    var apiRequest: APIRequest {
        APIRequest(
            method: method,
            path: path,
            queryItems: queryItems.map(\.urlQueryItem),
            body: body,
            authenticated: true,
            idempotencyKey: idempotencyKey
        )
    }
}

struct DurableMutationReplayReport: Equatable, Sendable {
    let acknowledgedKeys: [UUID]
    let definitiveFailureKey: UUID?
    let ambiguousFailureKey: UUID?
    let unsafeAmbiguousCount: Int

    static let empty = DurableMutationReplayReport(
        acknowledgedKeys: [],
        definitiveFailureKey: nil,
        ambiguousFailureKey: nil,
        unsafeAmbiguousCount: 0
    )

    var madeProgress: Bool { !acknowledgedKeys.isEmpty || definitiveFailureKey != nil }
    var needsSafetyIntervention: Bool { unsafeAmbiguousCount > 0 }
}

enum MutationOwnerFingerprint {
    static func make(token: String) -> String? {
        guard !token.isEmpty else { return nil }
        let digest = SHA256.hash(data: Data(token.utf8))
        return digest.map { String(format: "%02x", $0) }.joined()
    }
}

private extension Character {
    var isLowercaseHexDigit: Bool {
        "0123456789abcdef".contains(self)
    }
}
