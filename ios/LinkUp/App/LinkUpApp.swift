import SwiftUI

@main
@MainActor
struct LinkUpApp: App {
    @StateObject private var runtime = AppRuntime()

    var body: some Scene {
        WindowGroup {
            RootView(runtime: runtime)
                .task { await runtime.start() }
        }
    }
}
