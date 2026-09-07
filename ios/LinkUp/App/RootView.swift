import SwiftUI

@MainActor
struct RootView: View {
    @ObservedObject var runtime: AppRuntime

    var body: some View {
        Group {
            switch runtime.state {
            case .starting:
                startupView
            case .failed(let message):
                configurationFailure(message)
            case .ready(let services):
                SessionRootView(services: services)
            }
        }
        .preferredColorScheme(.dark)
        .background(LinkUpPalette.background.ignoresSafeArea())
    }

    private var startupView: some View {
        VStack(spacing: 14) {
            ProgressView().tint(LinkUpPalette.red)
            Text("Starting LinkUp…")
                .font(LinkUpTypography.body(13))
                .foregroundStyle(LinkUpPalette.textDimmed)
        }
    }

    private func configurationFailure(_ message: String) -> some View {
        VStack(spacing: 16) {
            Image(systemName: "lock.trianglebadge.exclamationmark")
                .font(.system(size: 34))
                .foregroundStyle(LinkUpPalette.critical)
            Text("LinkUp configuration error")
                .font(LinkUpTypography.display(20))
                .foregroundStyle(LinkUpPalette.textPrimary)
            Text(message)
                .font(LinkUpTypography.body(13))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .multilineTextAlignment(.center)
        }
        .padding(24)
    }
}

@MainActor
private struct SessionRootView: View {
    let services: AppServices
    @ObservedObject private var session: SessionCoordinator

    init(services: AppServices) {
        self.services = services
        _session = ObservedObject(wrappedValue: services.session)
    }

    var body: some View {
        switch session.state {
        case .checking:
            ProgressView().tint(LinkUpPalette.red)
        case .signedOut:
            AuthView(session: session)
        case .signedIn(let user):
            MainShellView(services: services, user: user)
                .id(user.id)
        case .offlineSession(let expiresAt):
            sessionProblem(
                title: "You're offline",
                message: "The local session is still valid until \(expiresAt.formatted(date: .abbreviated, time: .shortened)), but the server could not be reached."
            )
        case .recoverableError(let message):
            sessionProblem(title: "Session unavailable", message: message)
        }
    }

    private func sessionProblem(title: String, message: String) -> some View {
        VStack(spacing: 16) {
            Image(systemName: "wifi.exclamationmark")
                .font(.system(size: 34))
                .foregroundStyle(LinkUpPalette.warning)
            Text(title)
                .font(LinkUpTypography.display(20))
                .foregroundStyle(LinkUpPalette.textPrimary)
            Text(message)
                .font(LinkUpTypography.body(13))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .multilineTextAlignment(.center)
            LinkUpButton(title: "Retry") { Task { await session.bootstrap() } }
            LinkUpButton(title: "Sign out", variant: .secondary) { Task { await session.clearLocalSession() } }
        }
        .padding(24)
    }
}

@MainActor
private struct MainShellView: View {
    let services: AppServices
    let user: UserProfile

    @StateObject private var social: SocialCoordinator
    @State private var selection: AppTab = .pulse
    @State private var showingCreate = false

    init(services: AppServices, user: UserProfile) {
        self.services = services
        self.user = user
        _social = StateObject(wrappedValue: SocialCoordinator(api: services.api, session: services.session))
    }

    var body: some View {
        ZStack {
            LinkUpPalette.background.ignoresSafeArea()
            activeScreen
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            FrozenBottomBar(selection: $selection) { showingCreate = true }
        }
        .fullScreenCover(isPresented: $showingCreate) {
            CreateLinkView(coordinator: social) { showingCreate = false }
        }
        .onDisappear { social.dispose() }
    }

    @ViewBuilder private var activeScreen: some View {
        switch selection {
        case .pulse:
            PulseView(coordinator: social, api: services.api, session: services.session)
        case .map:
            MapView()
        case .fly:
            FlyView()
        case .me:
            MeView(user: user, api: services.api, session: services.session)
        }
    }
}
