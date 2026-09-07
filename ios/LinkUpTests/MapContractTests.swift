import XCTest
@testable import LinkUp

final class MapContractTests: XCTestCase {
    func testClusterResponseDecodesServerAggregate() throws {
        let data = Data(#"{
          "key":"504:305",
          "latitudeE6":50465300,
          "longitudeE6":30517800,
          "placeCount":1,
          "slotCount":3,
          "placeId":"5f1c1694-f4a7-4ca5-8433-ec70cfb7e6b6",
          "placeName":"Kontraktova Square"
        }"#.utf8)

        let cluster = try APICoding.decoder().decode(MapCluster.self, from: data)
        XCTAssertEqual(cluster.key, "504:305")
        XCTAssertEqual(cluster.slotCount, 3)
        XCTAssertEqual(cluster.placeCount, 1)
        XCTAssertEqual(cluster.coordinate.latitude, 50.4653, accuracy: 0.000001)
        XCTAssertEqual(cluster.coordinate.longitude, 30.5178, accuracy: 0.000001)
        XCTAssertNotNil(cluster.placeId)
    }

    func testAggregateClusterMayOmitPlaceIdentity() throws {
        let data = Data(#"{
          "key":"25:15",
          "latitudeE6":50000000,
          "longitudeE6":30000000,
          "placeCount":4,
          "slotCount":9
        }"#.utf8)

        let cluster = try APICoding.decoder().decode(MapCluster.self, from: data)
        XCTAssertEqual(cluster.placeCount, 4)
        XCTAssertNil(cluster.placeId)
        XCTAssertNil(cluster.placeName)
    }

    func testViewportRejectsBackendInvalidBounds() {
        let now = Date()
        let query = MapViewportQuery(
            westE6: 30_000_000,
            southE6: 51_000_000,
            eastE6: 31_000_000,
            northE6: 50_000_000,
            zoom: 12,
            from: now,
            to: now.addingTimeInterval(3600),
            limit: 100
        )
        XCTAssertFalse(query.isValid)
    }

    func testViewportAcceptsAntimeridianCrossing() {
        let now = Date()
        let query = MapViewportQuery(
            westE6: 170_000_000,
            southE6: -10_000_000,
            eastE6: -170_000_000,
            northE6: 10_000_000,
            zoom: 5,
            from: now,
            to: now.addingTimeInterval(24 * 60 * 60),
            limit: 100
        )
        XCTAssertTrue(query.isValid)
    }

    func testViewportRejectsWindowLongerThanSevenDays() {
        let now = Date()
        let query = MapViewportQuery(
            westE6: 30_000_000,
            southE6: 49_000_000,
            eastE6: 31_000_000,
            northE6: 51_000_000,
            zoom: 12,
            from: now,
            to: now.addingTimeInterval(7 * 24 * 60 * 60 + 1),
            limit: 100
        )
        XCTAssertFalse(query.isValid)
    }
}
