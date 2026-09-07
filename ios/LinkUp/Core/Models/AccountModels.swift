import Foundation

struct UserProfile: Codable, Identifiable, Equatable, Sendable {
    let id: UUID
    let email: String
    let username: String
    let displayName: String
    let avatarUrl: String?
    let profileVisibility: String
    let language: String
}

struct BlockedUser: Codable, Identifiable, Equatable, Sendable {
    let id: UUID
    let username: String
    let displayName: String
    let avatarUrl: String?
}

struct AuthEnvelope: Decodable, Sendable {
    let user: UserProfile
    let token: String
    let expiresAt: Date
}

struct SessionCredential: Codable, Equatable, Sendable {
    let token: String
    let expiresAt: Date

    var isExpired: Bool { expiresAt <= Date() }
}
