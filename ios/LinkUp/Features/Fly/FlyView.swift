import SwiftUI

struct FlyView: View {
    private let unavailableTabs = ["Now", "Travel", "Motion"]

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                header
                HStack(spacing: 8) {
                    ForEach(unavailableTabs, id: \.self) { item in
                        unavailableChip(item)
                    }
                    Spacer()
                }
                .padding(.horizontal, 20)
                .padding(.vertical, 12)

                LinkUpEmptyState(
                    title: "Fly feed unavailable",
                    message: "The current server contract does not expose Fly live-drop data. LinkUp will not fabricate drops, live counts, or participation totals.",
                    systemImage: "paperplane.fill"
                )
                .padding(.horizontal, 20)
            }
        }
        .scrollIndicators(.hidden)
        .background(LinkUpPalette.background)
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 8) {
                Image(systemName: "paperplane.fill").foregroundStyle(LinkUpPalette.red)
                Text("Fly Now")
                    .font(LinkUpTypography.display(24, weight: .black))
                    .foregroundStyle(LinkUpPalette.textPrimary)
                Spacer()
                Text("LIVE —")
                    .font(LinkUpTypography.mono(10, weight: .bold))
                    .foregroundStyle(LinkUpPalette.textMuted)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(LinkUpPalette.elevated)
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.compact))
                    .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.compact).stroke(LinkUpPalette.border) }
            }
            Text("Flash drops · server feed unavailable")
                .font(LinkUpTypography.body(14))
                .foregroundStyle(LinkUpPalette.textDimmed)
            counters
        }
        .padding(.horizontal, 20)
        .padding(.top, 12)
        .padding(.bottom, 12)
        .linkUpGlass()
    }

    private var counters: some View {
        HStack(spacing: 8) {
            counter("Joined")
            counter("Passed")
            counter("Available")
        }
    }

    private func counter(_ label: String) -> some View {
        VStack(spacing: 2) {
            Text("—")
                .font(LinkUpTypography.mono(20, weight: .bold))
                .foregroundStyle(LinkUpPalette.textMuted)
            Text(label)
                .font(LinkUpTypography.body(10))
                .foregroundStyle(LinkUpPalette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
        .background(LinkUpPalette.elevated)
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
    }

    private func unavailableChip(_ title: String) -> some View {
        Text(title)
            .font(LinkUpTypography.body(12, weight: .semibold))
            .foregroundStyle(LinkUpPalette.textMuted)
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
            .background(LinkUpPalette.elevated)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control, style: .continuous))
            .overlay {
                RoundedRectangle(cornerRadius: LinkUpRadius.control)
                    .stroke(LinkUpPalette.border, lineWidth: 1)
            }
            .accessibilityLabel("\(title), unavailable")
    }
}
