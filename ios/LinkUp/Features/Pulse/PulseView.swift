import SwiftUI

struct PulseView: View {
    @State private var search = ""
    @State private var time = "Now"
    @State private var category = "All"

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                header
                filters
                LinkUpEmptyState(
                    title: "Quiet around here",
                    message: "No production Pulse data is available yet.",
                    systemImage: "bolt.fill"
                )
                .padding(.horizontal, 20)
            }
        }
        .scrollIndicators(.hidden)
        .background(LinkUpPalette.background)
    }

    private var header: some View {
        VStack(spacing: 12) {
            HStack {
                VStack(alignment: .leading, spacing: 5) {
                    Label("City context unavailable", systemImage: "mappin")
                        .font(LinkUpTypography.body(14, weight: .semibold))
                    HStack(spacing: 8) {
                        Circle().fill(LinkUpPalette.red).frame(width: 8, height: 8)
                        Text("City BPM —").font(LinkUpTypography.mono(12))
                        Text("· unavailable").font(LinkUpTypography.body(10))
                    }
                    .foregroundStyle(LinkUpPalette.textDimmed)
                }
                Spacer()
                Image(systemName: "bell")
                    .font(.system(size: 18))
                    .foregroundStyle(LinkUpPalette.textDimmed)
                    .frame(width: 40, height: 40)
                    .background(LinkUpPalette.elevated)
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                    .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
            }
            HStack(spacing: 10) {
                Image(systemName: "magnifyingglass").foregroundStyle(LinkUpPalette.textMuted)
                TextField("Search activities, places...", text: $search)
                    .font(LinkUpTypography.body(14)).foregroundStyle(LinkUpPalette.textPrimary)
            }
            .padding(.horizontal, 12).frame(height: 42)
            .background(LinkUpPalette.elevated)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
        }
        .padding(.horizontal, 20).padding(.top, 12).padding(.bottom, 12)
        .linkUpGlass()
    }

    private var filters: some View {
        VStack(alignment: .leading, spacing: 10) {
            chipRow(["Now", "Tonight", "Tomorrow", "All"], selection: $time)
            chipRow(["All", "Social", "Active", "Food"], selection: $category)
        }
        .padding(.horizontal, 20).padding(.vertical, 12)
    }

    private func chipRow(_ values: [String], selection: Binding<String>) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(values, id: \.self) { value in
                    LinkUpChip(title: value, active: selection.wrappedValue == value) { selection.wrappedValue = value }
                }
            }
        }
    }
}
