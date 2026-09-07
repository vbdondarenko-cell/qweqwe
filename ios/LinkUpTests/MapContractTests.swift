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
}
