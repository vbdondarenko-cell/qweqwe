import Foundation
import XCTest
@testable import LinkUp

final class CityTimeScopeTests: XCTestCase {
    func testTomorrowUsesCanonicalCityTimezoneInsteadOfDeviceTimezone() throws {
        let kyiv = try XCTUnwrap(CityTimeScope(identifier: "Europe/Kyiv"))
        let losAngeles = try XCTUnwrap(CityTimeScope(identifier: "America/Los_Angeles"))
        let now = try instant("2026-09-08T20:30:00Z")
        let candidate = try instant("2026-09-08T21:30:00Z")

        XCTAssertTrue(kyiv.isTomorrow(candidate, relativeTo: now))
        XCTAssertFalse(kyiv.isToday(candidate, relativeTo: now))
        XCTAssertTrue(losAngeles.isToday(candidate, relativeTo: now))
    }

    func testTonightUsesCityLocalHour() throws {
        let kyiv = try XCTUnwrap(CityTimeScope(identifier: "Europe/Kyiv"))
        let now = try instant("2026-09-08T12:00:00Z")
        let evening = try instant("2026-09-08T15:30:00Z")

        XCTAssertTrue(kyiv.isToday(evening, relativeTo: now))
        XCTAssertGreaterThanOrEqual(kyiv.localHour(for: evening), 17)
    }

    func testTomorrowUsesCalendarDayAcrossDSTFallback() throws {
        let newYork = try XCTUnwrap(CityTimeScope(identifier: "America/New_York"))
        let now = try instant("2026-10-31T04:30:00Z")
        let nextLocalDay = try instant("2026-11-01T06:30:00Z")

        XCTAssertTrue(newYork.isTomorrow(nextLocalDay, relativeTo: now))
    }

    func testUnknownTimezoneIsRejected() {
        XCTAssertNil(CityTimeScope(identifier: "Mars/Olympus_Mons"))
    }

    private func instant(_ raw: String) throws -> Date {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return try XCTUnwrap(formatter.date(from: raw))
    }
}
