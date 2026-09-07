import SwiftUI

@MainActor
struct MeView: View {
    let user: UserProfile
    let api: LinkUpAPI
    let social: SocialCoordinator

    @ObservedObject var session: SessionCoordinator
    @StateObject private var coordinator: MeCoordinator

    @State private var tab = "Profile"
    @State private var showingMyLinks = false
    @State private var showingEditProfile = false
    @State private var showingLinkUpPlus = false

    init(user: UserProfile, api: LinkUpAPI, session: SessionCoordinator, social: SocialCoordinator) {
        self.user = user
        self.api = api
        self.social = social
        _session = ObservedObject(wrappedValue: session)
        _coordinator = StateObject(wrappedValue: MeCoordinator(api: api, session: session))
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                header
                Group {
                    switch tab {
                    case "Passport": passport
                    case "Settings": settings
                    default: profile
                    }
                }
                .padding(.horizontal, 20)
                .padding(.top, 16)
            }
        }
        .scrollIndicators(.hidden)
        .refreshable { await refreshAll() }
        .background(LinkUpPalette.background)
        .task {
            if coordinator.phase == .idle || coordinator.monetizationPhase == .idle {
                await refreshAll()
            }
        }
        .onDisappear { coordinator.dispose() }
        .sheet(isPresented: $showingMyLinks, onDismiss: {
            Task { await coordinator.load() }
        }) {
            MySlotsDashboardView(api: api, session: session, social: social)
        }
        .sheet(isPresented: $showingEditProfile) {
            EditProfileView(user: user, session: session)
        }
        .sheet(isPresented: $showingLinkUpPlus) {
            LinkUpPlusView(coordinator: coordinator)
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Me").font(LinkUpTypography.display(24, weight: .black))
            HStack(spacing: 8) {
                ForEach(["Profile", "Passport", "Settings"], id: \.self) { item in
                    LinkUpChip(title: item, active: tab == item) { tab = item }
                }
            }
        }
        .foregroundStyle(LinkUpPalette.textPrimary)
        .padding(.horizontal, 20)
        .padding(.top, 12)
        .padding(.bottom, 12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .linkUpGlass()
    }

    private var profile: some View {
        VStack(spacing: 16) {
            HStack(spacing: 16) {
                LinkUpAvatar(initials: initials, size: .xl)
                VStack(alignment: .leading, spacing: 4) {
                    Text(user.displayName)
                        .font(LinkUpTypography.display(20))
                        .foregroundStyle(LinkUpPalette.textPrimary)
                    Text("@\(user.username)")
                        .font(LinkUpTypography.body(13))
                        .foregroundStyle(LinkUpPalette.textDimmed)
                    Text(user.email)
                        .font(LinkUpTypography.body(11))
                        .foregroundStyle(LinkUpPalette.textMuted)
                }
                Spacer()
                Button {
                    showingEditProfile = true
                } label: {
                    Image(systemName: "pencil")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.red)
                        .frame(width: 38, height: 38)
                        .background(LinkUpPalette.red.opacity(0.1))
                        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.red.opacity(0.25)) }
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Edit profile")
            }

            LinkUpCard {
                VStack(spacing: 12) {
                    HStack {
                        Label("Reliability", systemImage: "chart.line.uptrend.xyaxis")
                            .font(LinkUpTypography.body(14, weight: .semibold))
                        Spacer()
                        Text("—").font(LinkUpTypography.mono(18, weight: .bold))
                    }
                    .foregroundStyle(LinkUpPalette.textPrimary)
                    LinkUpProgress(value: 0, maximum: 100, showsLabel: false)
                    HStack(spacing: 8) {
                        metric("Showed up"); metric("Hosted"); metric("BUMP verified"); metric("No-show")
                    }
                }
            }

            LinkUpCard {
                VStack(alignment: .leading, spacing: 12) {
                    HStack {
                        Label("My LinkUps", systemImage: "bolt.fill")
                            .font(LinkUpTypography.body(14, weight: .semibold))
                        Spacer()
                        if coordinator.phase == .loading { ProgressView().tint(LinkUpPalette.red) }
                    }
                    HStack(spacing: 8) {
                        linkMetric("Hosting", coordinator.hosting.count)
                        linkMetric("Joined", coordinator.joined.count)
                        linkMetric("Requests", coordinator.requested.count)
                    }
                    Button {
                        showingMyLinks = true
                    } label: {
                        HStack {
                            Text("Open dashboard")
                            Spacer()
                            Image(systemName: "chevron.right")
                        }
                        .font(LinkUpTypography.body(12, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.red)
                        .padding(.top, 4)
                    }
                    .buttonStyle(.plain)
                }
                .foregroundStyle(LinkUpPalette.textPrimary)
            }

            linkUpPlusSummary

            LinkUpCard {
                HStack {
                    Label("BUMP Vault", systemImage: "heart.fill")
                    Spacer()
                    Text("Not active").foregroundStyle(LinkUpPalette.textMuted)
                }
                .font(LinkUpTypography.body(14, weight: .semibold))
            }

            if case .failed(let message) = coordinator.phase {
                LinkUpErrorState(message: message) { Task { await coordinator.load() } }
            }
        }
    }

    private var linkUpPlusSummary: some View {
        LinkUpCard {
            Button {
                showingLinkUpPlus = true
            } label: {
                HStack(spacing: 12) {
                    Image(systemName: coordinator.monetization?.status.premiumActive == true ? "checkmark.seal.fill" : "seal")
                        .font(.system(size: 19))
                        .foregroundStyle(coordinator.monetization?.status.premiumActive == true ? LinkUpPalette.success : LinkUpPalette.red)
                        .frame(width: 40, height: 40)
                        .background(LinkUpPalette.zone)
                        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                    VStack(alignment: .leading, spacing: 2) {
                        Text("LinkUp+")
                            .font(LinkUpTypography.body(14, weight: .semibold))
                            .foregroundStyle(LinkUpPalette.textPrimary)
                        Text(linkUpPlusSubtitle)
                            .font(LinkUpTypography.body(10))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }
                    Spacer()
                    if coordinator.monetizationPhase == .loading {
                        ProgressView().tint(LinkUpPalette.red)
                    } else {
                        Image(systemName: "chevron.right")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }
                }
            }
            .buttonStyle(.plain)
        }
    }

    private var linkUpPlusSubtitle: String {
        guard let snapshot = coordinator.monetization else {
            if case .failed = coordinator.monetizationPhase { return "Server status unavailable" }
            return "Loading server entitlement…"
        }
        if snapshot.status.premiumActive {
            if let until = snapshot.status.premiumUntil {
                return "Active until \(until.formatted(date: .abbreviated, time: .omitted))"
            }
            return "Active"
        }
        return "Inactive · referral and catalog available"
    }

    private var passport: some View {
        VStack(spacing: 16) {
            HStack(spacing: 8) {
                passportMetric("Meetups"); passportMetric("Cities"); passportMetric("Hosted")
            }
            LinkUpCard {
                VStack(alignment: .leading, spacing: 12) {
                    Label("Cities Explored", systemImage: "globe.europe.africa.fill")
                        .font(LinkUpTypography.body(14, weight: .semibold))
                    Text("No verified city history yet.")
                        .font(LinkUpTypography.body(13)).foregroundStyle(LinkUpPalette.textMuted)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            LinkUpCard {
                VStack(alignment: .leading, spacing: 12) {
                    Text("Interests").font(LinkUpTypography.body(14, weight: .semibold))
                    Text("No server-backed interests yet.")
                        .font(LinkUpTypography.body(13)).foregroundStyle(LinkUpPalette.textMuted)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            LinkUpCard {
                VStack(alignment: .leading, spacing: 12) {
                    Text("Activity · 6 months").font(LinkUpTypography.body(14, weight: .semibold))
                    Rectangle().fill(LinkUpPalette.zone).frame(height: 120)
                        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.compact))
                }
            }
        }
        .foregroundStyle(LinkUpPalette.textPrimary)
    }

    private var settings: some View {
        VStack(spacing: 16) {
            accountSummary
            linkUpPlusSettings
            blockedSection
            settingsSection("Privacy & Safety", ["Privacy Center", "Safety Center", "Guardian", "Ghost Mode"])
            settingsSection("Account", ["Notifications", "Accessibility", "Data & Privacy"])
            settingsSection("App", ["Themes", "Legal", "Version"])
            LinkUpButton(title: "Log out", variant: .danger) {
                Task { await session.logout() }
            }
        }
    }

    private var linkUpPlusSettings: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("LINKUP+")
                .font(LinkUpTypography.mono(10, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textMuted)
                .padding(.horizontal, 4)
            Button {
                showingLinkUpPlus = true
            } label: {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Subscription & referrals")
                            .font(LinkUpTypography.body(14, weight: .medium))
                        Text(linkUpPlusSubtitle)
                            .font(LinkUpTypography.body(10))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }
                    Spacer()
                    Image(systemName: "chevron.right")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.textMuted)
                }
                .foregroundStyle(LinkUpPalette.textPrimary)
                .padding(.horizontal, 16)
                .frame(minHeight: 54)
                .background(LinkUpPalette.elevated)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
            }
            .buttonStyle(.plain)
        }
    }

    private var accountSummary: some View {
        LinkUpCard {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    Text("Language").font(LinkUpTypography.body(13, weight: .semibold))
                    Spacer()
                    Text(user.language.uppercased()).font(LinkUpTypography.mono(11))
                }
                HStack {
                    Text("Profile visibility").font(LinkUpTypography.body(13, weight: .semibold))
                    Spacer()
                    Text(user.profileVisibility).font(LinkUpTypography.mono(11))
                }
                Button {
                    showingEditProfile = true
                } label: {
                    HStack {
                        Text("Edit profile")
                        Spacer()
                        Image(systemName: "chevron.right")
                    }
                    .font(LinkUpTypography.body(12, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.red)
                    .padding(.top, 4)
                }
                .buttonStyle(.plain)
            }
            .foregroundStyle(LinkUpPalette.textPrimary)
        }
    }

    @ViewBuilder private var blockedSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("BLOCKED ACCOUNTS")
                .font(LinkUpTypography.mono(10, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textMuted)
                .padding(.horizontal, 4)
            if coordinator.blockedUsers.isEmpty {
                LinkUpCard {
                    Text("No blocked accounts")
                        .font(LinkUpTypography.body(13))
                        .foregroundStyle(LinkUpPalette.textMuted)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            } else {
                VStack(spacing: 0) {
                    ForEach(coordinator.blockedUsers) { blocked in
                        HStack(spacing: 10) {
                            LinkUpAvatar(initials: initials(blocked.displayName), size: .sm)
                            VStack(alignment: .leading, spacing: 1) {
                                Text(blocked.displayName).font(LinkUpTypography.body(13, weight: .semibold))
                                Text("@\(blocked.username)").font(LinkUpTypography.body(10)).foregroundStyle(LinkUpPalette.textMuted)
                            }
                            Spacer()
                            Button("Unblock") { Task { await coordinator.unblock(blocked) } }
                                .font(LinkUpTypography.body(11, weight: .semibold))
                                .foregroundStyle(LinkUpPalette.red)
                                .disabled(coordinator.isMutating)
                        }
                        .padding(.horizontal, 14).frame(minHeight: 50)
                        if blocked.id != coordinator.blockedUsers.last?.id {
                            Rectangle().fill(LinkUpPalette.border.opacity(0.5)).frame(height: 1)
                        }
                    }
                }
                .background(LinkUpPalette.elevated)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
            }
        }
    }

    private func refreshAll() async {
        async let account: Void = coordinator.load()
        async let plus: Void = coordinator.loadMonetization()
        _ = await (account, plus)
    }

    private func metric(_ label: String) -> some View {
        VStack(spacing: 2) {
            Text("—").font(LinkUpTypography.mono(14, weight: .bold))
            Text(label).font(LinkUpTypography.body(9)).lineLimit(1)
        }
        .foregroundStyle(LinkUpPalette.textMuted)
        .frame(maxWidth: .infinity).padding(.vertical, 8)
        .background(LinkUpPalette.zone.opacity(0.5))
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
    }

    private func linkMetric(_ label: String, _ value: Int) -> some View {
        VStack(spacing: 2) {
            Text("\(value)").font(LinkUpTypography.mono(18, weight: .bold))
            Text(label).font(LinkUpTypography.body(10)).foregroundStyle(LinkUpPalette.textMuted)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 10)
        .background(LinkUpPalette.zone.opacity(0.5))
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
    }

    private func passportMetric(_ label: String) -> some View {
        VStack(spacing: 2) {
            Text("—").font(LinkUpTypography.display(24, weight: .black))
            Text(label).font(LinkUpTypography.body(11)).foregroundStyle(LinkUpPalette.textMuted)
        }
        .foregroundStyle(LinkUpPalette.textPrimary)
        .frame(maxWidth: .infinity).padding(.vertical, 16)
        .background(LinkUpPalette.elevated)
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
    }

    private func settingsSection(_ title: String, _ rows: [String]) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title.uppercased())
                .font(LinkUpTypography.mono(10, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textMuted)
                .padding(.horizontal, 4)
            VStack(spacing: 0) {
                ForEach(rows, id: \.self) { row in
                    HStack {
                        Text(row).font(LinkUpTypography.body(14, weight: .medium))
                        Spacer()
                        Image(systemName: "chevron.right")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }
                    .foregroundStyle(LinkUpPalette.textPrimary)
                    .padding(.horizontal, 16).frame(height: 46)
                    if row != rows.last { Rectangle().fill(LinkUpPalette.border.opacity(0.5)).frame(height: 1) }
                }
            }
            .background(LinkUpPalette.elevated)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
        }
    }

    private var initials: String { initials(user.displayName) }

    private func initials(_ name: String) -> String {
        let value = name.split(separator: " ").prefix(2).compactMap(\.first).map(String.init).joined()
        return value.isEmpty ? "LU" : value.uppercased()
    }
}
