import SwiftUI

@MainActor
struct AppInfoView: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.openURL) private var openURL

    let configuration: AppInfoConfiguration

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    LinkUpCard {
                        VStack(spacing: 12) {
                            valueRow(title: "Version", value: configuration.version)
                            divider
                            valueRow(title: "Build", value: configuration.build)
                        }
                    }

                    LinkUpCard {
                        VStack(spacing: 0) {
                            legalRow(title: "Privacy Policy", url: configuration.privacyURL)
                            divider
                            legalRow(title: "Terms of Service", url: configuration.termsURL)
                        }
                    }
                }
                .padding(20)
            }
            .background(LinkUpPalette.background)
            .navigationTitle(L10n.text("Legal"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button(L10n.text("Done")) { dismiss() }
                        .foregroundStyle(LinkUpPalette.red)
                }
            }
        }
        .preferredColorScheme(.dark)
    }

    private func valueRow(title: String, value: String) -> some View {
        HStack(spacing: 12) {
            Text(L10n.text(title))
                .font(LinkUpTypography.body(14, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textPrimary)
            Spacer()
            Text(value)
                .font(LinkUpTypography.mono(12, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textMuted)
                .textSelection(.disabled)
        }
        .frame(minHeight: 44)
    }

    @ViewBuilder
    private func legalRow(title: String, url: URL?) -> some View {
        if let url {
            Button {
                openURL(url)
            } label: {
                HStack(spacing: 12) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L10n.text(title))
                            .font(LinkUpTypography.body(14, weight: .semibold))
                        Text(url.host ?? url.absoluteString)
                            .font(LinkUpTypography.body(10))
                            .foregroundStyle(LinkUpPalette.textMuted)
                            .lineLimit(1)
                    }
                    Spacer()
                    Image(systemName: "arrow.up.right")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.red)
                }
                .foregroundStyle(LinkUpPalette.textPrimary)
                .frame(minHeight: 52)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityHint(L10n.text("Opens in browser"))
        } else {
            HStack(spacing: 12) {
                Text(L10n.text(title))
                    .font(LinkUpTypography.body(14, weight: .semibold))
                Spacer()
                Text(L10n.text("Not configured"))
                    .font(LinkUpTypography.body(11))
                    .foregroundStyle(LinkUpPalette.textMuted)
            }
            .foregroundStyle(LinkUpPalette.textPrimary)
            .frame(minHeight: 52)
        }
    }

    private var divider: some View {
        Rectangle()
            .fill(LinkUpPalette.border.opacity(0.6))
            .frame(height: 1)
    }
}
