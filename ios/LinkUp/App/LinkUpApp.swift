import SwiftUI

@main
@MainActor
struct LinkUpApp: App {
    @Environment(\.scenePhase) private var scenePhase
    @StateObject private var runtime = AppRuntime()

    var body: some Scene {
        WindowGroup {
            RootView(runtime: runtime)
                .task { await runtime.start() }
                .onOpenURL { url in
                    _ = runtime.handleIncomingURL(url)
                }
                .onChange(of: scenePhase) { _, phase in
                    guard phase == .active else { return }
                    Task { await runtime.applicationBecameActive() }
                }
        }
    }
}
