import Foundation
import SwiftUI

@MainActor
final class MySlotsDashboardCoordinator: ObservableObject {
    enum Phase: Equatable {
        case idle, loading, content, empty
        case failed(String)
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var items: [SlotModel] = []

    private let api: LinkUpAPI
    private let session: SessionCoordinator
    private var generation: UInt64 = 0

    init(api: LinkUpAPI, session: SessionCoordinator) {
        self.api = api
        self.session = session
    }

    func load(_ view: MySlotsView) async {
        generation &+= 1
        let requestGeneration = generation
        phase = .loading
        do {
            let result = try await api.mySlots(view)
            guard requestGeneration == generation else { return }
            items = result
            phase = result.isEmpty ? .empty : .content
        } catch is CancellationError {
            return
        } catch let error as APIError {
            guard requestGeneration == generation else { return }
            if case .unauthorized = error {
                await session.clearLocalSession()
            } else {
                phase = .failed(error.localizedDescription)
            }
        } catch {
            guard requestGeneration == generation else { return }
            phase = .failed(L10n.text("Unable to load My LinkUps."))
        }
    }

    func invalidate() {
        generation &+= 1
        items = []
        phase = .idle
    }
}
