import Foundation
import XCTest
@testable import LinkUp

final class MutationRetryContractTests: XCTestCase {
    func testDefinitiveClientFailuresReleasePendingMutationKey() {
        XCTAssertTrue(APIError.unauthorized.isDefinitiveMutationFailure)
        XCTAssertTrue(APIError.http(status: 400, code: "invalid", message: "invalid", requestID: nil).isDefinitiveMutationFailure)
        XCTAssertTrue(APIError.http(status: 409, code: "conflict", message: "conflict", requestID: nil).isDefinitiveMutationFailure)
        XCTAssertTrue(APIError.http(status: 422, code: "invalid", message: "invalid", requestID: nil).isDefinitiveMutationFailure)
    }

    func testAmbiguousFailuresRetainPendingMutationKey() {
        XCTAssertFalse(APIError.transport("timeout").isDefinitiveMutationFailure)
        XCTAssertFalse(APIError.responseTooLarge.isDefinitiveMutationFailure)
        XCTAssertFalse(APIError.protocolViolation("invalid response").isDefinitiveMutationFailure)
        XCTAssertFalse(APIError.http(status: 408, code: "timeout", message: "timeout", requestID: nil).isDefinitiveMutationFailure)
        XCTAssertFalse(APIError.http(status: 429, code: "rate_limited", message: "later", requestID: nil).isDefinitiveMutationFailure)
        XCTAssertFalse(APIError.http(status: 500, code: "internal", message: "failed", requestID: nil).isDefinitiveMutationFailure)
        XCTAssertFalse(APIError.http(status: 503, code: "not_ready", message: "failed", requestID: nil).isDefinitiveMutationFailure)
        XCTAssertFalse(APIError.mutationQueued(UUID()).isDefinitiveMutationFailure)
        XCTAssertFalse(APIError.mutationSafetyBlocked.isDefinitiveMutationFailure)
        XCTAssertFalse(APIError.mutationJournalUnavailable.isDefinitiveMutationFailure)
    }
}
