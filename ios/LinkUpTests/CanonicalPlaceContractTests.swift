import XCTest
@testable import LinkUp

final class CanonicalPlaceContractTests: XCTestCase {
    func testPlaceSearchResponseDecodesMicrodegrees() throws {
        let data = Data(#"{
          "id":"5f1c1694-f4a7-4ca5-8433-ec70cfb7e6b6",
          "name":"Kontraktova Square",
          "category":"square",
          "locality":"Kyiv",
          "countryCode":"UA",
          "latitudeE6":50465300,
          "longitudeE6":30517800,
          "precisionM":30
        }"#.utf8)

        let place = try APICoding.decoder().decode(PlaceModel.self, from: data)
        XCTAssertEqual(place.name, "Kontraktova Square")
        XCTAssertEqual(place.latitude, 50.4653, accuracy: 0.000001)
        XCTAssertEqual(place.longitude, 30.5178, accuracy: 0.000001)
    }

    func testSlotDecodesOptionalCanonicalPlaceID() throws {
        let data = Data(#"{
          "id":"85a23232-3281-429b-ad42-ad2bc1dd2c41",
          "organizer":{"id":"b7581cc7-511b-4b40-a17d-d8beaa8334e1","username":"host","displayName":"Host"},
          "title":"Coffee",
          "activity":"coffee",
          "placeText":"Kontraktova Square",
          "canonicalPlaceId":"5f1c1694-f4a7-4ca5-8433-ec70cfb7e6b6",
          "capacity":6,
          "acceptedCount":0,
          "state":"FILLING",
          "accessMode":"APPROVAL",
          "visibility":"PUBLIC",
          "viewerState":"HOST",
          "version":1,
          "createdAt":"2026-09-07T23:00:00Z",
          "updatedAt":"2026-09-07T23:00:00Z"
        }"#.utf8)

        let slot = try APICoding.decoder().decode(SlotModel.self, from: data)
        XCTAssertEqual(slot.canonicalPlaceId?.uuidString.lowercased(), "5f1c1694-f4a7-4ca5-8433-ec70cfb7e6b6")
        XCTAssertFalse(slot.isTerminal)
        XCTAssertEqual(slot.remainingCapacity, 6)
    }
}
