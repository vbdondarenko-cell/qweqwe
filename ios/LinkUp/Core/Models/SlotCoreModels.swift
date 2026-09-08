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

    var isPulseDiscoverable: Bool {
        self == .published || self == .filling || self == .full
    }

    var acceptsNewRequests: Bool {
        self == .published || self == .filling
    }

    var allowsHostEdit: Bool {
        self == .draft || self == .published || self == .filling || self == .full
    }

    var allowsHostStart: Bool {
        self == .filling || self == .full
    }

    var allowsChat: Bool {
        self == .filling || self == .full || self == .active
    }

    var allowsHostManagement: Bool {
        self == .published || self == .filling || self == .full || self == .active
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
