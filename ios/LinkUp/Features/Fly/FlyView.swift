import SwiftUI

struct FlyView: View {
    @State private var tab = "Now"

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                header
                HStack(spacing: 8) {
                    ForEach(["Now", "Travel", "Motion"], id: \.self) { item in
                        LinkUpChip(title: item, active: tab == item) { tab = item }
                    }
                    Spacer()
                }
                .padding(.horizontal, 20).padding(.vertical, 12)
                LinkUpEmptyState(
                    title: "No flash drops",
                    message: "No live drops are available right now.",
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
                Text("0 LIVE")
                    .font(LinkUpTypography.mono(10, weight: .bold))
                    .foregroundStyle(LinkUpPalette.red)
                    .padding(.horizontal, 10).padding(.vertical, 5)
                    .background(LinkUpPalette.red.opacity(0.15))
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.compact))
                    .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.compact).stroke(LinkUpPalette.red.opacity(0.3)) }
            }
            Text("Flash drops · right now")
                .font(LinkUpTypography.body(14)).foregroundStyle(LinkUpPalette.textDimmed)
            counters
        }
        .padding(.horizontal, 20).padding(.top, 12).padding(.bottom, 12)
        .linkUpGlass()
    }

    private var counters: some View {
        HStack(spacing: 8) {
            counter("Joined", 0, LinkUpPalette.success)
            counter("Passed", 0, LinkUpPalette.textMuted)
            counter("Available", 0, LinkUpPalette.red)
        }
    }

    private func counter(_ label: String, _ value: Int, _ color: Color) -> some View {
        VStack(spacing: 2) {
            Text("\(value)")
                .font(LinkUpTypography.mono(20, weight: .bold))
                .foregroundStyle(color)
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
}
