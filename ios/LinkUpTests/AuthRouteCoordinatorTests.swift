import Foundation
import XCTest
@testable import LinkUp

@MainActor
final class AuthRouteCoordinatorTests: XCTestCase {
    private let token = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

    func testAcceptsHTTPSResetRouteAndConsumesTokenOnce() throws {
        let router = AuthRouteCoordinator()
        let url = try XCTUnwrap(URL(string: "https://app.example/reset-password?token=\(token)"))

        XCTAssertTrue(router.accept(url))
        XCTAssertEqual(router.pendingPasswordResetToken, token)
        XCTAssertEqual(router.consumePasswordResetToken(), token)
        XCTAssertNil(router.pendingPasswordResetToken)
        XCTAssertNil(router.consumePasswordResetToken())
    }

    func testRejectsNonHTTPSAndNonResetQueryShapes() throws {
        let router = AuthRouteCoordinator()
        let http = try XCTUnwrap(URL(string: "http://app.example/reset-password?token=\(token)"))
        let extraQuery = try XCTUnwrap(URL(string: "https://app.example/reset-password?token=\(token)&source=mail"))
        let missing = try XCTUnwrap(URL(string: "https://app.example/reset-password"))
        let fragment = try XCTUnwrap(URL(string: "https://app.example/reset-password?token=\(token)#fragment"))

        XCTAssertFalse(router.accept(http))
        XCTAssertFalse(router.accept(extraQuery))
        XCTAssertFalse(router.accept(missing))
        XCTAssertFalse(router.accept(fragment))
        XCTAssertNil(router.pendingPasswordResetToken)
    }

    func testRejectsMalformedResetToken() throws {
        let router = AuthRouteCoordinator()
        let url = try XCTUnwrap(URL(string: "https://app.example/reset-password?token=abc"))

        XCTAssertFalse(router.accept(url))
        XCTAssertNil(router.pendingPasswordResetToken)
    }

    func testClearDropsPendingSensitiveRouteState() throws {
        let router = AuthRouteCoordinator()
        let url = try XCTUnwrap(URL(string: "https://app.example/reset-password?token=\(token)"))
        XCTAssertTrue(router.accept(url))

        router.clear()
        XCTAssertNil(router.pendingPasswordResetToken)
    }
}
