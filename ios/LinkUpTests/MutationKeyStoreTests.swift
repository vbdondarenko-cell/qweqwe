import XCTest
@testable import LinkUp

final class MutationKeyStoreTests: XCTestCase {
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

    func testStoreSurvivesReloadAndPrunesOldestEntries() {
        let suite = "com.linkup.tests.mutation-keys.\(UUID().uuidString)"
        guard let defaults = UserDefaults(suiteName: suite) else {
            XCTFail("Unable to create isolated UserDefaults suite")
            return
        }
        defer { defaults.removePersistentDomain(forName: suite) }

        let store = MutationKeyStore(defaults: defaults, storageKey: "pending", maxEntries: 2)
        let old = StoredMutationKey(key: UUID(), touchedAt: Date(timeIntervalSince1970: 1))
        let middle = StoredMutationKey(key: UUID(), touchedAt: Date(timeIntervalSince1970: 2))
        let newest = StoredMutationKey(key: UUID(), touchedAt: Date(timeIntervalSince1970: 3))

        store.persist(["old": old, "middle": middle, "newest": newest])
        let reloaded = store.load()

        XCTAssertEqual(reloaded.count, 2)
        XCTAssertNil(reloaded["old"])
        XCTAssertEqual(reloaded["middle"], middle)
        XCTAssertEqual(reloaded["newest"], newest)
    }

    func testPersistingEmptyStoreRemovesPreviousJournal() {
        let suite = "com.linkup.tests.mutation-empty.\(UUID().uuidString)"
        guard let defaults = UserDefaults(suiteName: suite) else {
            XCTFail("Unable to create isolated UserDefaults suite")
            return
        }
        defer { defaults.removePersistentDomain(forName: suite) }

        let store = MutationKeyStore(defaults: defaults, storageKey: "pending", maxEntries: 2)
        store.persist(["x": StoredMutationKey(key: UUID(), touchedAt: Date())])
        XCTAssertEqual(store.load().count, 1)
        store.persist([:])
        XCTAssertTrue(store.load().isEmpty)
    }
}
