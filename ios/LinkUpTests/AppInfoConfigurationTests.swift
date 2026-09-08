import Foundation
import XCTest
@testable import LinkUp

final class AppInfoConfigurationTests: XCTestCase {
    func testTrustedLegalURLRequiresHTTPSAuthority() throws {
        XCTAssertEqual(
            AppInfoConfiguration.trustedHTTPSURL("https://linkup.example/privacy")?.absoluteString,
            "https://linkup.example/privacy"
        )
        XCTAssertNil(AppInfoConfiguration.trustedHTTPSURL("http://linkup.example/privacy"))
        XCTAssertNil(AppInfoConfiguration.trustedHTTPSURL("https://user@linkup.example/privacy"))
        XCTAssertNil(AppInfoConfiguration.trustedHTTPSURL("https:///privacy"))
        XCTAssertNil(AppInfoConfiguration.trustedHTTPSURL("https://linkup.example/privacy#section"))
        XCTAssertNil(AppInfoConfiguration.trustedHTTPSURL(""))
    }

    func testLegalURLAllowsOrdinaryHttpsPathAndQuery() throws {
        let url = try XCTUnwrap(AppInfoConfiguration.trustedHTTPSURL("https://linkup.example/legal/privacy?lang=uk"))
        XCTAssertEqual(url.host, "linkup.example")
        XCTAssertEqual(url.path, "/legal/privacy")
        XCTAssertEqual(URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems?.first?.name, "lang")
    }
}
