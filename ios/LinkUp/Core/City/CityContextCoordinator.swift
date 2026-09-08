import Foundation
import SwiftUI

@MainActor
final class CityContextCoordinator: ObservableObject {
    enum Phase: Equatable {
        case idle, loading, content, empty
        case failed(String)
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var context: CityContextModel?
    @Published private(set) var isRefreshing = false
    @Published private(set) var refreshError: String?

    private let api: LinkUpAPI
    private let session: SessionCoordinator
    private let locations: CityLocationProvider
    private var generation: UInt64 = 0

    init(api: LinkUpAPI, session: SessionCoordinator, locations: CityLocationProvider) {
        self.api = api
        self.session = session
        self.locations = locations
    }

    func loadCurrent() async {
        let request = beginRequest()
        do {
            let value = try await api.currentCityContext()
            guard request == generation else { return }
            context = value
            phase = .content
            isRefreshing = false
            refreshError = nil
        } catch is CancellationError {
            finishCancellation(request)
        } catch let error as APIError {
            guard request == generation else { return }
            if case .unauthorized = error {
                isRefreshing = false
                await session.clearLocalSession()
                return
            }
            if case .http(let status, let code, _, _) = error,
               status == 404, code == "city_context_unavailable" {
                context = nil
                phase = .empty
                refreshError = nil
            } else if context == nil {
                phase = .failed(error.localizedDescription)
            } else {
                phase = .content
                refreshError = error.localizedDescription
            }
            isRefreshing = false
        } catch {
            guard request == generation else { return }
            phase = context == nil ? .failed(error.localizedDescription) : .content
            if context != nil { refreshError = error.localizedDescription }
            isRefreshing = false
        }
    }

    func resolveFromDevice() async {
        let request = beginRequest()
        do {
            let observation = try await locations.currentObservation()
            guard request == generation else { return }
            let value = try await api.resolveCityContext(observation)
            guard request == generation else { return }
            context = value
            phase = .content
            isRefreshing = false
            refreshError = nil
        } catch is CancellationError {
            finishCancellation(request)
        } catch let error as APIError {
            guard request == generation else { return }
            if case .unauthorized = error {
                isRefreshing = false
                await session.clearLocalSession()
                return
            }
            phase = context == nil ? .failed(error.localizedDescription) : .content
            if context != nil { refreshError = error.localizedDescription }
            isRefreshing = false
        } catch {
            guard request == generation else { return }
            phase = context == nil ? .failed(error.localizedDescription) : .content
            if context != nil { refreshError = error.localizedDescription }
            isRefreshing = false
        }
    }

    func clear() {
        generation &+= 1
        context = nil
        phase = .idle
        isRefreshing = false
        refreshError = nil
    }

    private func beginRequest() -> UInt64 {
        generation &+= 1
        let request = generation
        refreshError = nil
        if context == nil { phase = .loading }
        else { isRefreshing = true }
        return request
    }

    private func finishCancellation(_ request: UInt64) {
        guard request == generation else { return }
        isRefreshing = false
        phase = context == nil ? .idle : .content
    }
}
