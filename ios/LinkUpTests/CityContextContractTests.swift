import XCTest
@testable import LinkUp

final class CityContextContractTests: XCTestCase {
    func testCityContextDecodesPrivacySafeServerShape() throws {
        let data = Data(#"{
          "locality":{
            "id":"5f1c1694-f4a7-4ca5-8433-ec70cfb7e6b6",
            "name":"Kyiv",
            "countryCode":"UA",
            "timezone":"Europe/Kyiv",
            "centroidLatitudeE6":50450000,
            "centroidLongitudeE6":30523000
          },
          "permissionClass":"APPROXIMATE",
          "accuracyM":1200,
          "observedAt":"2026-09-08T06:40:00Z",
          "expiresAt":"2026-09-08T12:40:00Z",
          "switchPending":false
        }"#.utf8)

        let context = try APICoding.decoder().decode(CityContextModel.self, from: data)
        XCTAssertTrue(context.hasValidServerShape)
        XCTAssertEqual(context.locality.name, "Kyiv")
        XCTAssertEqual(context.locality.latitude, 50.45, accuracy: 0.000001)
        XCTAssertEqual(context.locality.longitude, 30.523, accuracy: 0.000001)
    }

    func testCityContextRejectsInvalidExpiryAndCentroid() {
        let now = Date()
        let locality = CityLocality(
            id: UUID(),
            name: "Kyiv",
            countryCode: "UA",
            timezone: "Europe/Kyiv",
            centroidLatitudeE6: 50_450_000,
            centroidLongitudeE6: 30_523_000
        )
        let invalidExpiry = CityContextModel(
            locality: locality,
            permissionClass: .precise,
            accuracyM: 100,
            observedAt: now,
            expiresAt: now,
            switchPending: false
        )
        XCTAssertFalse(invalidExpiry.hasValidServerShape)

        let invalidLocality = CityLocality(
            id: UUID(),
            name: "Broken",
            countryCode: "UA",
            timezone: "Europe/Kyiv",
            centroidLatitudeE6: 91_000_000,
            centroidLongitudeE6: 30_000_000
        )
        XCTAssertFalse(invalidLocality.hasValidServerShape)
    }

    func testServerExpiryTimestampIsTheOnlyClientFreshnessBoundary() {
        let now = Date(timeIntervalSince1970: 1_800_000_000)
        let context = CityContextModel(
            locality: CityLocality(
                id: UUID(),
                name: "Kyiv",
                countryCode: "UA",
                timezone: "Europe/Kyiv",
                centroidLatitudeE6: 50_450_000,
                centroidLongitudeE6: 30_523_000
            ),
            permissionClass: .approximate,
            accuracyM: 1_000,
            observedAt: now.addingTimeInterval(-60),
            expiresAt: now.addingTimeInterval(10),
            switchPending: false
        )

        XCTAssertTrue(context.isFresh(at: now))
        XCTAssertFalse(context.isFresh(at: context.expiresAt))
        XCTAssertFalse(context.isFresh(at: context.expiresAt.addingTimeInterval(1)))
    }

    func testObservationRejectsMockedAndOutOfBoundsInput() {
        let valid = CityLocationObservation(
            latitudeE6: 50_450_000,
            longitudeE6: 30_523_000,
            accuracyM: 100,
            permissionClass: .precise,
            capturedAt: Date(),
            mocked: false
        )
        XCTAssertTrue(valid.hasValidClientShape)

        let mocked = CityLocationObservation(
            latitudeE6: valid.latitudeE6,
            longitudeE6: valid.longitudeE6,
            accuracyM: valid.accuracyM,
            permissionClass: valid.permissionClass,
            capturedAt: valid.capturedAt,
            mocked: true
        )
        XCTAssertFalse(mocked.hasValidClientShape)

        let invalidCoordinate = CityLocationObservation(
            latitudeE6: 91_000_000,
            longitudeE6: 0,
            accuracyM: 100,
            permissionClass: .precise,
            capturedAt: Date(),
            mocked: false
        )
        XCTAssertFalse(invalidCoordinate.hasValidClientShape)
    }
}
