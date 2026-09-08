import Foundation

struct APIErrorPayload: Decodable, Sendable {
    let code: String?
    let message: String?
    let requestId: String?
}

enum APIError: Error, LocalizedError, Sendable {
    case unauthorized
    case secureStorageUnavailable
    case responseTooLarge
    case protocolViolation(String)
    case mutationQueued(UUID)
    case mutationSafetyBlocked
    case mutationJournalUnavailable
    case signOutBlockedByPendingAction
    case sessionTransitionInProgress
    case authenticationInProgress
    case http(status: Int, code: String, message: String, requestID: String?)
    case transport(String)

    var errorDescription: String? {
        switch self {
        case .unauthorized: L10n.text("Authentication required.")
        case .secureStorageUnavailable: L10n.text("Secure session storage is unavailable.")
        case .responseTooLarge: L10n.text("Server response exceeded the client safety limit.")
        case .protocolViolation(let message): message
        case .mutationQueued(_): L10n.text("Action is queued for safe retry and will be reconciled with the server.")
        case .mutationSafetyBlocked: L10n.text("An older unconfirmed action must be reconciled before new changes can be sent.")
        case .mutationJournalUnavailable: L10n.text("Protected pending-action storage is unavailable.")
        case .signOutBlockedByPendingAction: L10n.text("Resolve the pending server action before signing out.")
        case .sessionTransitionInProgress: L10n.text("Sign-out is already in progress.")
        case .authenticationInProgress: L10n.text("Another authentication request is already in progress.")
        case .http(_, _, let message, _): message
        case .transport: L10n.text("Network request failed.")
        }
    }

    var retryableForGET: Bool {
        switch self {
        case .transport: true
        case .http(let status, _, _, _): [408, 429, 502, 503, 504].contains(status)
        default: false
        }
    }

    var retryableForIdempotentWrite: Bool {
        switch self {
        case .transport:
            true
        case .http(let status, _, _, _):
            [408, 502, 503, 504].contains(status)
        default:
            false
        }
    }

    var isDefinitiveMutationFailure: Bool {
        switch self {
        case .unauthorized:
            true
        case .http(let status, _, _, _):
            (400...499).contains(status) && status != 408 && status != 429
        default:
            false
        }
    }
}

enum HTTPMethod: String, Codable, Sendable {
    case get = "GET"
    case post = "POST"
    case patch = "PATCH"
    case put = "PUT"
    case delete = "DELETE"
}

struct APIRequest: Sendable {
    let method: HTTPMethod
    let path: String
    var queryItems: [URLQueryItem] = []
    var body: Data? = nil
    var authenticated = true
    var idempotencyKey: UUID? = nil
}
