import XCTest
@testable import LinkUp

final class InputContractsTests: XCTestCase {
    func testSlotTitleUsesUnicodeScalarCountLikeGoRunes() {
        let combining = "e\u{301}"
        XCTAssertEqual(combining.count, 1)
        XCTAssertEqual(InputContracts.scalarCount(combining), 2)
        XCTAssertTrue(InputContracts.validSlotTitle(String(repeating: combining, count: 60)))
        XCTAssertFalse(InputContracts.validSlotTitle(String(repeating: combining, count: 61)))
    }

    func testDetailsAndPlaceMatchBackendRuneLimits() {
        let combining = "e\u{301}"
        XCTAssertTrue(InputContracts.validSlotDetails(String(repeating: combining, count: 1_000)))
        XCTAssertFalse(InputContracts.validSlotDetails(String(repeating: combining, count: 1_001)))
        XCTAssertTrue(InputContracts.validSlotPlace(String(repeating: combining, count: 120)))
        XCTAssertFalse(InputContracts.validSlotPlace(String(repeating: combining, count: 121)))
    }

    func testPlaceSearchMatchesBackendRuneWindow() {
        let combining = "e\u{301}"
        XCTAssertFalse(InputContracts.validPlaceSearchQuery("a"))
        XCTAssertTrue(InputContracts.validPlaceSearchQuery("ab"))
        XCTAssertTrue(InputContracts.validPlaceSearchQuery(String(repeating: combining, count: 40)))
        XCTAssertFalse(InputContracts.validPlaceSearchQuery(String(repeating: combining, count: 41)))
    }

    func testProfileNameAndAvatarByteLimitMatchBackend() {
        let combining = "e\u{301}"
        XCTAssertTrue(InputContracts.validProfileDisplayName(String(repeating: combining, count: 40)))
        XCTAssertFalse(InputContracts.validProfileDisplayName(String(repeating: combining, count: 41)))
        XCTAssertTrue(InputContracts.validAvatarURLPayload(String(repeating: "😀", count: 512)))
        XCTAssertFalse(InputContracts.validAvatarURLPayload(String(repeating: "😀", count: 513)))
    }
    func testChatMessageMatchesBackendRuneLimit() {
        let combining = "e\u{301}"
        XCTAssertFalse(InputContracts.validChatMessage("   "))
        XCTAssertTrue(InputContracts.validChatMessage(String(repeating: combining, count: 1_000)))
        XCTAssertFalse(InputContracts.validChatMessage(String(repeating: combining, count: 1_001)))
    }

    func testAccountEmailMatchesBackendShape() {
        XCTAssertTrue(InputContracts.validAccountEmail(" A@Example.com "))
        XCTAssertTrue(InputContracts.validAccountEmail("a@b.co"))
        XCTAssertFalse(InputContracts.validAccountEmail("a@b"))
        XCTAssertFalse(InputContracts.validAccountEmail("a b@example.com"))
        XCTAssertFalse(InputContracts.validAccountEmail(String(repeating: "a", count: 310) + "@example.com"))
    }

    func testUsernameMatchesBackendASCIIAlphabetAfterLowercasing() {
        XCTAssertTrue(InputContracts.validAccountUsername("Alice_1"))
        XCTAssertTrue(InputContracts.validAccountUsername("a.b"))
        XCTAssertFalse(InputContracts.validAccountUsername("ab"))
        XCTAssertFalse(InputContracts.validAccountUsername("alice-1"))
        XCTAssertFalse(InputContracts.validAccountUsername("аліса"))
        XCTAssertFalse(InputContracts.validAccountUsername(String(repeating: "a", count: 33)))
    }

    func testPasswordLimitUsesUTF8BytesLikeBackend() {
        XCTAssertFalse(InputContracts.validPasswordPayload("1234567"))
        XCTAssertTrue(InputContracts.validPasswordPayload("12345678"))
        XCTAssertTrue(InputContracts.validPasswordPayload("😀😀"))
        XCTAssertTrue(InputContracts.validPasswordPayload(String(repeating: "a", count: 1_024)))
        XCTAssertFalse(InputContracts.validPasswordPayload(String(repeating: "a", count: 1_025)))
    }

}
