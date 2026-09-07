import Foundation
import MapKit
import SwiftUI

@MainActor
final class MapCoordinator: ObservableObject {
    enum Phase: Equatable {
        case idle, loading, content, empty
        case failed(String)
    }

    enum PlaceSlotsPhase: Equatable {
        case idle, loading, content, empty
        case failed(String)
    }

    @Published private(set) var clusters: [MapCluster] = []
    @Published private(set) var phase: Phase = .idle
    @Published private(set) var placeSlots: [SlotModel] = []
    @Published private(set) var placeSlotsPhase: PlaceSlotsPhase = .idle

    private let api: LinkUpAPI
    private let session: SessionCoordinator
    private var generation: UInt64 = 0
    private var placeSlotsGeneration: UInt64 = 0
    private var lastQuery: MapViewportQuery?
    private let horizon: TimeInterval = 24 * 60 * 60

    init(api: LinkUpAPI, session: SessionCoordinator) {
        self.api = api
        self.session = session
    }

    func load(region: MKCoordinateRegion) async {
        guard let query = makeQuery(region: region) else { return }
        generation &+= 1
        placeSlotsGeneration &+= 1
        let requestGeneration = generation
        lastQuery = query
        clearPlaceSlotsState()
        if clusters.isEmpty { phase = .loading }

        do {
            let items = try await api.mapViewport(query)
            guard requestGeneration == generation else { return }
            clusters = items
            phase = items.isEmpty ? .empty : .content
        } catch is CancellationError {
            return
        } catch let error as APIError {
            guard requestGeneration == generation else { return }
            if case .unauthorized = error {
                await session.clearLocalSession()
                return
            }
            if clusters.isEmpty { phase = .failed(error.localizedDescription) }
        } catch {
            guard requestGeneration == generation else { return }
            if clusters.isEmpty { phase = .failed("Unable to load map data.") }
        }
    }

    func loadPlaceSlots(placeID: UUID) async {
        guard let query = lastQuery else {
            placeSlotsPhase = .failed("Map viewport is not ready.")
            return
        }

        placeSlotsGeneration &+= 1
        let requestGeneration = placeSlotsGeneration
        placeSlots = []
        placeSlotsPhase = .loading

        do {
            let items = try await api.mapPlaceSlots(
                placeID: placeID,
                from: query.from,
                to: query.to,
                limit: 50
            )
            guard requestGeneration == placeSlotsGeneration else { return }
            placeSlots = items
            placeSlotsPhase = items.isEmpty ? .empty : .content
        } catch is CancellationError {
            return
        } catch let error as APIError {
            guard requestGeneration == placeSlotsGeneration else { return }
            if case .unauthorized = error {
                await session.clearLocalSession()
                return
            }
            placeSlotsPhase = .failed(error.localizedDescription)
        } catch {
            guard requestGeneration == placeSlotsGeneration else { return }
            placeSlotsPhase = .failed("Unable to load LinkUps for this place.")
        }
    }

    func refreshPlaceSlots(placeID: UUID) async {
        await loadPlaceSlots(placeID: placeID)
    }

    func clearPlaceSlots() {
        placeSlotsGeneration &+= 1
        clearPlaceSlotsState()
    }

    func dispose() {
        generation &+= 1
        placeSlotsGeneration &+= 1
        clusters = []
        phase = .idle
        lastQuery = nil
        clearPlaceSlotsState()
    }

    private func clearPlaceSlotsState() {
        placeSlots = []
        placeSlotsPhase = .idle
    }

    private func makeQuery(region: MKCoordinateRegion) -> MapViewportQuery? {
        guard region.center.latitude.isFinite,
              region.center.longitude.isFinite,
              region.span.latitudeDelta.isFinite,
              region.span.longitudeDelta.isFinite,
              region.span.latitudeDelta > 0,
              region.span.longitudeDelta > 0 else { return nil }

        let latHalf = min(89.999_999, region.span.latitudeDelta / 2)
        let lonHalf = min(179.999_999, region.span.longitudeDelta / 2)
        let south = max(-90, region.center.latitude - latHalf)
        let north = min(90, region.center.latitude + latHalf)
        let west = normalizeLongitude(region.center.longitude - lonHalf)
        let east = normalizeLongitude(region.center.longitude + lonHalf)
        let longitudeDelta = max(0.000_001, min(359.999_998, region.span.longitudeDelta))
        let zoom = min(20, max(1, Int(log2(360 / longitudeDelta).rounded(.down)) + 1))
        let now = Date()

        let query = MapViewportQuery(
            westE6: toMicrodegrees(west),
            southE6: toMicrodegrees(south),
            eastE6: toMicrodegrees(east),
            northE6: toMicrodegrees(north),
            zoom: zoom,
            from: now,
            to: now.addingTimeInterval(horizon),
            limit: 100
        )
        return query.isValid ? query : nil
    }

    private func toMicrodegrees(_ value: Double) -> Int {
        Int((value * 1_000_000).rounded())
    }

    private func normalizeLongitude(_ value: Double) -> Double {
        var longitude = value.truncatingRemainder(dividingBy: 360)
        if longitude > 180 { longitude -= 360 }
        if longitude < -180 { longitude += 360 }
        return longitude
    }
}
