import Foundation
import SwiftUI

@MainActor
final class AuthRouteCoordinator: ObservableObject {
    @Published private(set) var pendingPasswordResetToken: String?

    @discardableResult
    func accept(_ url: URL) -> Bool {
        guard url.scheme?.lowercased() == "https",
              let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              let items = components.queryItems,
              items.count == 1,
              items[0].name == "token",
              let token = passwordResetToken(from: url.absoluteString) else {
            return false
        }
        pendingPasswordResetToken = token
        return true
    }

    func consumePasswordResetToken() -> String? {
        defer { pendingPasswordResetToken = nil }
        return pendingPasswordResetToken
    }

    func clear() {
        pendingPasswordResetToken = nil
    }
}
