import SwiftUI

struct RootView: View {
    @State private var selection: AppTab = .pulse
    @State private var showingCreate = false

    var body: some View {
        ZStack {
            LinkUpPalette.background.ignoresSafeArea()
            activeScreen
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            FrozenBottomBar(selection: $selection) {
                showingCreate = true
            }
        }
        .fullScreenCover(isPresented: $showingCreate) {
            CreateLinkView { showingCreate = false }
        }
        .preferredColorScheme(.dark)
    }

    @ViewBuilder private var activeScreen: some View {
        switch selection {
        case .pulse: PulseView()
        case .map: MapView()
        case .fly: FlyView()
        case .me: MeView()
        }
    }
}
