import XCTest
@testable import LinkUp

final class APIEndpointTests: XCTestCase {
    func testHTTPSRootIsAccepted() throws {
        let endpoint = try APIEndpoint(raw: "https://api.example.com/")
        XCTAssertEqual(endpoint.baseURL.absoluteString, "https://api.example.com")
    }

    func testCredentialsAreRejected() {
        XCTAssertThrowsError(try APIEndpoint(raw: "https://user:pass@api.example.com"))
    }

    func testQueryAndFragmentAreRejected() {
        XCTAssertThrowsError(try APIEndpoint(raw: "https://api.example.com?debug=1"))
        XCTAssertThrowsError(try APIEndpoint(raw: "https://api.example.com/#debug"))
    }

    func testPathIsRejected() {
        XCTAssertThrowsError(try APIEndpoint(raw: "https://api.example.com/v1"))
    }

    func testRequestPathAndQueryAreEncodedByURLComponents() throws {
        let endpoint = try APIEndpoint(raw: "https://api.example.com")
        let url = try endpoint.url(
            path: "/v1/me/slots",
            queryItems: [URLQueryItem(name: "view", value: "JOINED")]
        )
        XCTAssertEqual(url.absoluteString, "https://api.example.com/v1/me/slots?view=JOINED")
    }
}
