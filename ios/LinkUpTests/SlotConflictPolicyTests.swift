import XCTest
@testable import LinkUp

final class SlotConflictPolicyTests: XCTestCase {
    func testOnlyCanonicalSlotVersionConflictMatches() {
        XCTAssertTrue(APIError.http(
            status: 409,
            code: "slot_version_conflict",
            message: "stale",
            requestID: nil
        ).isSlotVersionConflict)

        XCTAssertFalse(APIError.http(
            status: 409,
            code: "slot_full",
            message: "full",
            requestID: nil
        ).isSlotVersionConflict)
        XCTAssertFalse(APIError.http(
            status: 409,
            code: "idempotency_conflict",
            message: "conflict",
            requestID: nil
        ).isSlotVersionConflict)
        XCTAssertFalse(APIError.http(
            status: 400,
            code: "slot_version_conflict",
            message: "bad request",
            requestID: nil
        ).isSlotVersionConflict)
        XCTAssertFalse(APIError.transport("offline").isSlotVersionConflict)
    }
}
