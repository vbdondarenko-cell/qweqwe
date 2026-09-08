import XCTest
@testable import LinkUp

final class SessionSafetyContractTests: XCTestCase {
    private let owner = String(repeating: "a", count: 64)

    func testExplicitLogoutAllowedOnlyWithNoPendingOwnerWork() {
        XCTAssertTrue(ExplicitLogoutSafety.canProceed(
            activeAuthenticatedWrites: 0,
            legacyPendingCount: 0,
            ownerHasCommands: false,
            workflowOwnerFingerprint: nil,
            currentOwnerFingerprint: owner
        ))

        XCTAssertFalse(ExplicitLogoutSafety.canProceed(
            activeAuthenticatedWrites: 1, legacyPendingCount: 0, ownerHasCommands: false,
            workflowOwnerFingerprint: nil, currentOwnerFingerprint: owner
        ))
        XCTAssertFalse(ExplicitLogoutSafety.canProceed(
            activeAuthenticatedWrites: 0, legacyPendingCount: 1, ownerHasCommands: false,
            workflowOwnerFingerprint: nil, currentOwnerFingerprint: owner
        ))
        XCTAssertFalse(ExplicitLogoutSafety.canProceed(
            activeAuthenticatedWrites: 0, legacyPendingCount: 0, ownerHasCommands: true,
            workflowOwnerFingerprint: nil, currentOwnerFingerprint: owner
        ))
        XCTAssertFalse(ExplicitLogoutSafety.canProceed(
            activeAuthenticatedWrites: 0, legacyPendingCount: 0, ownerHasCommands: false,
            workflowOwnerFingerprint: owner, currentOwnerFingerprint: owner
        ))
        XCTAssertFalse(ExplicitLogoutSafety.canProceed(
            activeAuthenticatedWrites: 0, legacyPendingCount: 0, ownerHasCommands: false,
            workflowOwnerFingerprint: String(repeating: "b", count: 64), currentOwnerFingerprint: owner
        ))
    }

    func testExplicitLogoutRejectsInvalidOwnerFingerprint() {
        XCTAssertFalse(ExplicitLogoutSafety.canProceed(
            activeAuthenticatedWrites: 0,
            legacyPendingCount: 0,
            ownerHasCommands: false,
            workflowOwnerFingerprint: nil,
            currentOwnerFingerprint: "short"
        ))
    }
}
