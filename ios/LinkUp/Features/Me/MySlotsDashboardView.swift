import SwiftUI

@MainActor
struct MySlotsDashboardView: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var social: SocialCoordinator
    @StateObject private var coordinator: MySlotsDashboardCoordinator

    private let api: LinkUpAPI
    private let session: SessionCoordinator

    @State private var selectedView: MySlotsView = .hosting
    @State private var selectedSlot: SlotModel?

    init(api: LinkUpAPI, session: SessionCoordinator, social: SocialCoordinator) {
        self.api = api
        self.session = session
        self.social = social
        _coordinator = StateObject(wrappedValue: MySlotsDashboardCoordinator(api: api, session: session))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                tabs
                content
            }
            .background(LinkUpPalette.background)
            .navigationTitle("My LinkUps")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }.foregroundStyle(LinkUpPalette.red)
                }
            }
            .task(id: selectedView) {
                await coordinator.load(selectedView)
            }
            .sheet(item: $selectedSlot, onDismiss: {
                Task { await coordinator.load(selectedView) }
            }) { slot in
                SlotDetailView(slot: slot, coordinator: social, api: api, session: session)
                    .presentationDetents([.medium, .large])
            }
        }
        .preferredColorScheme(.dark)
    }

    private var tabs: some View {
        HStack(spacing: 8) {
            tab(.hosting, "Hosting")
            tab(.joined, "Joined")
            tab(.requested, "Requests")
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
        .background(LinkUpPalette.surface)
        .overlay(alignment: .bottom) { Rectangle().fill(LinkUpPalette.border).frame(height: 1) }
    }

    @ViewBuilder private var content: some View {
        switch coordinator.phase {
        case .idle, .loading:
            Spacer()
            ProgressView().tint(LinkUpPalette.red)
            Spacer()
        case .failed(let message):
            Spacer()
            LinkUpErrorState(message: message) { Task { await coordinator.load(selectedView) } }
                .padding(20)
            Spacer()
        case .empty:
            Spacer()
            LinkUpEmptyState(
                title: emptyTitle,
                message: emptyMessage,
                systemImage: "bolt"
            )
            .padding(20)
            Spacer()
        case .content:
            ScrollView {
                LazyVStack(spacing: 10) {
                    ForEach(coordinator.items) { slot in
                        Button { selectedSlot = slot } label: {
                            dashboardRow(slot)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(20)
            }
            .scrollIndicators(.hidden)
            .refreshable { await coordinator.load(selectedView) }
        }
    }

    private func tab(_ view: MySlotsView, _ title: String) -> some View {
        LinkUpChip(title: title, active: selectedView == view) {
            selectedView = view
        }
    }

    private func dashboardRow(_ slot: SlotModel) -> some View {
        LinkUpCard {
            HStack(spacing: 12) {
                Image(systemName: activitySymbol(slot.activity))
                    .font(.system(size: 18, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.red)
                    .frame(width: 42, height: 42)
                    .background(LinkUpPalette.red.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                VStack(alignment: .leading, spacing: 4) {
                    Text(slot.title)
                        .font(LinkUpTypography.body(14, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.textPrimary)
                        .lineLimit(1)
                    Text(slot.placeText)
                        .font(LinkUpTypography.body(11))
                        .foregroundStyle(LinkUpPalette.textMuted)
                        .lineLimit(1)
                    Text("\(slot.acceptedCount)/\(slot.capacity) · \(slot.state.rawValue)")
                        .font(LinkUpTypography.mono(9, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.textDimmed)
                }
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.textMuted)
            }
        }
    }

    private var emptyTitle: String {
        switch selectedView {
        case .hosting: "Nothing hosted"
        case .joined: "Nothing joined"
        case .requested: "No pending requests"
        }
    }

    private var emptyMessage: String {
        switch selectedView {
        case .hosting: "Create a LINK to see it here."
        case .joined: "Accepted LinkUps will appear here, including ACTIVE ones."
        case .requested: "Pending approval requests will appear here."
        }
    }

    private func activitySymbol(_ raw: String) -> String {
        switch raw.lowercased() {
        case "coffee": "cup.and.saucer.fill"
        case "running": "figure.run"
        case "walk": "figure.walk"
        case "food": "fork.knife"
        case "cycling": "bicycle"
        case "photography": "camera.fill"
        case "music": "music.note"
        case "games": "gamecontroller.fill"
        case "study": "book.fill"
        default: "bolt.fill"
        }
    }
}
