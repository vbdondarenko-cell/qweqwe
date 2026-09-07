import Foundation

enum MySlotsView: String, Codable, Sendable {
    case hosting = "HOSTING"
    case joined = "JOINED"
    case requested = "REQUESTED"
}

enum SlotState: String, Codable, Sendable {
    case draft = "DRAFT"
    case published = "PUBLISHED"
    case filling = "FILLING"
    case full = "FULL"
    case active = "ACTIVE"
    case completed = "COMPLETED"
    case cancelled = "CANCELLED"
    case expired = "EXPIRED"
    case moderated = "MODERATED"

    var isTerminal: Bool {
        switch self {
        case .completed, .cancelled, .expired, .moderated: true
        default: false
        }
    }

    var acceptsNewRequests: Bool {
        self == .filling || self == .full
    }
}

enum SlotAccessMode: String, Codable, Sendable {
    case instant = "INSTANT"
    case approval = "APPROVAL"
    case waitlist = "WAITLIST"
}

enum SlotVisibility: String, Codable, Sendable {
    case publicValue = "PUBLIC"
}

enum SlotViewerState: String, Codable, Sendable {
    case none = "NONE"
    case pending = "PENDING"
    case accepted = "ACCEPTED"
    case host = "HOST"
}
