import Foundation
import SwiftUI

@MainActor
final class AuthRouteCoordinator: ObservableObject {
    @Published private(set) var pendingPasswordResetToken: String?
    private let trustedResetRoute: PasswordResetRoute?

    init(trustedResetRoute: PasswordResetRoute? = nil) {
        self.trustedResetRoute = trustedResetRoute
    }

    @discardableResult
    func accept(_ url: URL) -> Bool {
        guard let trustedResetRoute,
              let token = trustedResetRoute.token(fromIncomingURL: url) else {
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
