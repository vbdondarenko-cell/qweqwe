import XCTest
@testable import LinkUp

final class DesignContractTests: XCTestCase {
    func testFrozenRadiiContract() {
        XCTAssertEqual(LinkUpRadius.badge, 6)
        XCTAssertEqual(LinkUpRadius.compact, 8)
        XCTAssertEqual(LinkUpRadius.control, 12)
        XCTAssertEqual(LinkUpRadius.card, 16)
        XCTAssertEqual(LinkUpRadius.sheet, 24)
    }
}
