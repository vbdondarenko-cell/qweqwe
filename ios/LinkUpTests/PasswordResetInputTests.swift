import XCTest
@testable import LinkUp

final class PasswordResetInputTests: XCTestCase {
    private let token = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

    func testAcceptsCanonical32ByteBase64URLToken() {
        XCTAssertEqual(passwordResetToken(from: token), token)
    }

    func testExtractsSingleTokenFromHTTPSURLWithoutFetchingIt() {
        XCTAssertEqual(
            passwordResetToken(from: "https://linkup.example/reset?token=\(token)"),
            token
        )
    }

    func testConfiguredRecoveryRouteMatchesOnlyExactHTTPSOriginAndPath() throws {
        let route = try XCTUnwrap(PasswordResetRoute(configuredURL: "https://app.example:8443/reset-password"))
        let valid = try XCTUnwrap(URL(string: "https://app.example:8443/reset-password?token=\(token)"))
        let wrongHost = try XCTUnwrap(URL(string: "https://other.example:8443/reset-password?token=\(token)"))
        let wrongPath = try XCTUnwrap(URL(string: "https://app.example:8443/other?token=\(token)"))
        let wrongPort = try XCTUnwrap(URL(string: "https://app.example/reset-password?token=\(token)"))

        XCTAssertEqual(route.token(fromIncomingURL: valid), token)
        XCTAssertNil(route.token(fromIncomingURL: wrongHost))
        XCTAssertNil(route.token(fromIncomingURL: wrongPath))
        XCTAssertNil(route.token(fromIncomingURL: wrongPort))
    }

    func testConfiguredRecoveryRouteRejectsUnsafeBaseURLShapes() {
        XCTAssertNil(PasswordResetRoute(configuredURL: ""))
        XCTAssertNil(PasswordResetRoute(configuredURL: "http://app.example/reset-password"))
        XCTAssertNil(PasswordResetRoute(configuredURL: "https://user@app.example/reset-password"))
        XCTAssertNil(PasswordResetRoute(configuredURL: "https://app.example/reset-password?next=x"))
        XCTAssertNil(PasswordResetRoute(configuredURL: "https://app.example/reset-password#fragment"))
    }

    func testRejectsDuplicateTokenParameters() {
        XCTAssertNil(passwordResetToken(from: "https://linkup.example/reset?token=\(token)&token=\(token)"))
    }

    func testRejectsUserInfoFragmentAndNonHTTPS() {
        XCTAssertNil(passwordResetToken(from: "https://user@linkup.example/reset?token=\(token)"))
        XCTAssertNil(passwordResetToken(from: "https://linkup.example/reset?token=\(token)#fragment"))
        XCTAssertNil(passwordResetToken(from: "http://linkup.example/reset?token=\(token)"))
    }

    func testRejectsMalformedOrOversizedInput() {
        XCTAssertNil(passwordResetToken(from: "not-a-reset-token"))
        XCTAssertNil(passwordResetToken(from: String(repeating: "x", count: 4097)))
    }
}
