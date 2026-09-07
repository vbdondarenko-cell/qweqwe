import Foundation
import SwiftUI

@MainActor
final class AppServices {
    let api: LinkUpAPI
    let session: SessionCoordinator

    init(api: LinkUpAPI, session: SessionCoordinator) {
        self.api = api
        self.session = session
    }
}

@MainActor
final class AppRuntime: ObservableObject {
    enum State {
        case starting
        case ready(AppServices)
        case failed(String)
    }

    @Published private(set) var state: State = .starting
    private var started = false

    func start() async {
        guard !started else { return }
        started = true

        do {
            guard let raw = Bundle.main.object(forInfoDictionaryKey: "LINKUP_API_BASE_URL") as? String else {
                throw APIEndpointError.missing
            }
            let credentials = KeychainSessionStore()
            let endpoint = try APIEndpoint(raw: raw)
            let client = APIClient(endpoint: endpoint, credentials: credentials)
            let api = LinkUpAPI(client: client, credentials: credentials)
            let session = SessionCoordinator(api: api, credentials: credentials)
            let services = AppServices(api: api, session: session)
            state = .ready(services)
            await session.bootstrap()
        } catch {
            state = .failed(error.localizedDescription)
        }
    }
    func applicationBecameActive() async {
        guard case .ready(let services) = state else { return }
        await services.session.revalidateForForeground()
    }

}
