import SwiftUI

struct MeView: View {
    @State private var tab = "Profile"

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
                .padding(.horizontal, 20).padding(.top, 16)
            }
        }
        .scrollIndicators(.hidden)
        .background(LinkUpPalette.background)
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
        .padding(.horizontal, 20).padding(.top, 12).padding(.bottom, 12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .linkUpGlass()
    }

    private var profile: some View {
        VStack(spacing: 16) {
            HStack(spacing: 16) {
                LinkUpAvatar(initials: "LU", size: .xl)
                VStack(alignment: .leading, spacing: 4) {
                    Text("Profile")
                        .font(LinkUpTypography.display(20))
                        .foregroundStyle(LinkUpPalette.textPrimary)
                    Text("Account data will load from LinkUp API")
                        .font(LinkUpTypography.body(13))
                        .foregroundStyle(LinkUpPalette.textDimmed)
                }
                Spacer()
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
                HStack {
                    Label("BUMP Vault", systemImage: "heart.fill")
                    Spacer()
                    Text("Not active").foregroundStyle(LinkUpPalette.textMuted)
                }
                .font(LinkUpTypography.body(14, weight: .semibold))
            }
            LinkUpCard {
                HStack {
                    Label("My LinkUps", systemImage: "bolt.fill")
                    Spacer()
                    Text("—").font(LinkUpTypography.mono(12))
                }
                .font(LinkUpTypography.body(14, weight: .semibold))
            }
        }
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
            settingsSection("Privacy & Safety", ["Privacy Center", "Safety Center", "Guardian", "Ghost Mode"])
            settingsSection("Account", ["Notifications", "Language", "Accessibility", "Data & Privacy"])
            settingsSection("LinkUp+", ["Upgrade to LinkUp+", "Rewarded Free Day"])
            settingsSection("App", ["Themes", "Legal", "Version", "Log out"])
        }
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
                    .foregroundStyle(row == "Log out" ? LinkUpPalette.critical : LinkUpPalette.textPrimary)
                    .padding(.horizontal, 16).frame(height: 46)
                    if row != rows.last { Rectangle().fill(LinkUpPalette.border.opacity(0.5)).frame(height: 1) }
                }
            }
            .background(LinkUpPalette.elevated)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
        }
    }
}
