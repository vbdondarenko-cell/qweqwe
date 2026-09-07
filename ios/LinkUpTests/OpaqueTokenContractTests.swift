import XCTest
@testable import LinkUp

final class OpaqueTokenContractTests: XCTestCase {
    private var validToken: String {
        Data((0..<32).map { UInt8($0) })
            .base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }

    func testCanonicalSessionTokenMatchesServerShape() {
        let token = validToken
        XCTAssertEqual(token.utf8.count, 43)
        XCTAssertEqual(OpaqueTokenContract.canonical32ByteBase64URL(token), token)
    }

    func testMalformedOrNonCanonicalTokensAreRejected() {
        let token = validToken
        XCTAssertNil(OpaqueTokenContract.canonical32ByteBase64URL(""))
        XCTAssertNil(OpaqueTokenContract.canonical32ByteBase64URL(String(token.dropLast())))
        XCTAssertNil(OpaqueTokenContract.canonical32ByteBase64URL(token + "="))
        XCTAssertNil(OpaqueTokenContract.canonical32ByteBase64URL(String(repeating: "!", count: 43)))
        XCTAssertNil(OpaqueTokenContract.canonical32ByteBase64URL(
            Data(repeating: 0, count: 31).base64EncodedString()
                .replacingOccurrences(of: "=", with: "")
        ))
    }

    func testSessionCredentialExposesTokenShapeSeparatelyFromExpiry() {
        let future = SessionCredential(token: validToken, expiresAt: Date().addingTimeInterval(60))
        XCTAssertTrue(future.hasValidTokenShape)
        XCTAssertFalse(future.isExpired)

        let malformed = SessionCredential(token: "abc", expiresAt: Date().addingTimeInterval(60))
        XCTAssertFalse(malformed.hasValidTokenShape)
    }

    func testPasswordResetParserUsesSameOpaqueTokenContract() {
        let token = validToken
        XCTAssertEqual(passwordResetToken(from: token), token)
        XCTAssertEqual(passwordResetToken(from: "https://example.com/reset?token=\(token)"), token)
        XCTAssertNil(passwordResetToken(from: "https://example.com/reset?token=abc"))
    }
}
