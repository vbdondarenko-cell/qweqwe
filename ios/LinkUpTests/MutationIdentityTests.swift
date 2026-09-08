import XCTest
@testable import LinkUp

final class MutationIdentityTests: XCTestCase {
    func testMutationIdentityIsCanonicalAcrossQueryOrderAndDoesNotExposePayload() {
        let body = Data(#"{"private":"meet me here","value":1}"#.utf8)
        let first = APIRequest(
            method: .post,
            path: "/v1/test",
            queryItems: [URLQueryItem(name: "b", value: "2"), URLQueryItem(name: "a", value: "1")],
            body: body
        )
        let second = APIRequest(
            method: .post,
            path: "/v1/test",
            queryItems: [URLQueryItem(name: "a", value: "1"), URLQueryItem(name: "b", value: "2")],
            body: body
        )

        let firstIdentity = MutationIdentity.digest(for: first)
        let secondIdentity = MutationIdentity.digest(for: second)
        XCTAssertEqual(firstIdentity, secondIdentity)
        XCTAssertEqual(firstIdentity.count, 64)
        XCTAssertTrue(firstIdentity.allSatisfy { $0.isHexDigit })
        XCTAssertFalse(firstIdentity.contains("private"))
        XCTAssertFalse(firstIdentity.contains("meet"))
    }

    func testMutationIdentityChangesWithMethodPathOrBody() {
        let base = APIRequest(method: .post, path: "/v1/test", body: Data("one".utf8))
        let changedMethod = APIRequest(method: .patch, path: "/v1/test", body: Data("one".utf8))
        let changedPath = APIRequest(method: .post, path: "/v1/other", body: Data("one".utf8))
        let changedBody = APIRequest(method: .post, path: "/v1/test", body: Data("two".utf8))

        let identity = MutationIdentity.digest(for: base)
        XCTAssertNotEqual(identity, MutationIdentity.digest(for: changedMethod))
        XCTAssertNotEqual(identity, MutationIdentity.digest(for: changedPath))
        XCTAssertNotEqual(identity, MutationIdentity.digest(for: changedBody))
    }
}
