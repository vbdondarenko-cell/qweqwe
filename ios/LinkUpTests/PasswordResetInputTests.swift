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
