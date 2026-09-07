import XCTest
@testable import LinkUp

final class APICodingTests: XCTestCase {
    private struct TimestampEnvelope: Codable, Equatable {
        let createdAt: Date
    }

    func testFractionalRFC3339RoundTrip() throws {
        let original = TimestampEnvelope(createdAt: Date(timeIntervalSince1970: 1_725_000_000.123))
        let data = try APICoding.encoder().encode(original)
        let decoded = try APICoding.decoder().decode(TimestampEnvelope.self, from: data)
        XCTAssertEqual(decoded.createdAt.timeIntervalSince1970, original.createdAt.timeIntervalSince1970, accuracy: 0.001)
    }

    func testStandardRFC3339IsAccepted() throws {
        let data = Data(#"{"createdAt":"2026-09-08T00:00:00Z"}"#.utf8)
        let decoded = try APICoding.decoder().decode(TimestampEnvelope.self, from: data)
        XCTAssertEqual(decoded.createdAt.timeIntervalSince1970, 1_788_825_600, accuracy: 1)
    }
    func testEncoderUsesCanonicalSortedKeysForMutationFingerprints() throws {
        struct Payload: Encodable {
            let zeta: Int
            let alpha: Int
        }
        let data = try APICoding.encoder().encode(Payload(zeta: 2, alpha: 1))
        XCTAssertEqual(String(decoding: data, as: UTF8.self), #"{"alpha":1,"zeta":2}"#)
    }

}
