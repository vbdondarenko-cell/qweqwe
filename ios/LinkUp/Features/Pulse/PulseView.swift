import SwiftUI

@MainActor
struct PulseView: View {
    @ObservedObject var coordinator: SocialCoordinator
    @ObservedObject var cityContext: CityContextCoordinator
    let api: LinkUpAPI
    let session: SessionCoordinator

    @State private var search = ""
    @State private var time = "Now"
    @State private var category = "All"
    @State private var selectedSlot: SlotModel?
    @State private var showingNotifications = false

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                header
                filters
                content
                    .padding(.horizontal, 20)
                    .padding(.top, 4)
            }
        }
        .scrollIndicators(.hidden)
        .refreshable { await coordinator.loadPulse() }
        .background(LinkUpPalette.background)
        .task {
            if coordinator.pulsePhase == .idle {
                await coordinator.loadPulse()
            }
        }
        .sheet(item: $selectedSlot) { slot in
            SlotDetailView(
                slot: slot,
                coordinator: coordinator,
                cityContext: cityContext,
                api: api,
                session: session
            )
                .presentationDetents([.medium, .large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $showingNotifications) {
            NotificationsPanelView()
                .presentationDetents([.medium, .large])
                .presentationDragIndicator(.visible)
        }
    }

    private var header: some View {
        VStack(spacing: 12) {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 5) {
                    if let context = cityContext.context {
                        Label("\(context.locality.name), \(context.locality.countryCode)", systemImage: "mappin.circle.fill")
                            .font(LinkUpTypography.body(14, weight: .semibold))
                        HStack(spacing: 8) {
                            Circle().fill(LinkUpPalette.success).frame(width: 8, height: 8)
                            Text("City lock · \(context.permissionClass.rawValue) · \(context.locality.timezone)")
                                .font(LinkUpTypography.mono(10))
                            if context.switchPending {
                                Text("· switch pending")
                                    .font(LinkUpTypography.body(10))
                            }
                        }
                        .foregroundStyle(LinkUpPalette.textDimmed)
                        if let refreshError = cityContext.refreshError {
                            Text(refreshError)
                                .font(LinkUpTypography.body(9))
                                .foregroundStyle(LinkUpPalette.warning)
                                .lineLimit(2)
                        }
                    } else {
                        Label("City context unavailable", systemImage: "mappin.slash")
                            .font(LinkUpTypography.body(14, weight: .semibold))
                        Text(cityContextMessage)
                            .font(LinkUpTypography.body(10))
                            .foregroundStyle(LinkUpPalette.textDimmed)
                    }
                }
                Spacer()
                HStack(spacing: 8) {
                    Button {
                        Task { await cityContext.resolveFromDevice() }
                    } label: {
                        Group {
                            if cityContext.isRefreshing || cityContext.phase == .loading {
                                ProgressView().tint(LinkUpPalette.red).controlSize(.small)
                            } else {
                                Image(systemName: cityContext.context == nil ? "location.fill" : "location.circle.fill")
                                    .font(.system(size: 18))
                            }
                        }
                        .foregroundStyle(LinkUpPalette.textDimmed)
                        .frame(width: 40, height: 40)
                        .background(LinkUpPalette.elevated)
                        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
                    }
                    .buttonStyle(.plain)
                    .disabled(cityContext.isRefreshing || cityContext.phase == .loading)
                    .accessibilityLabel(cityContext.context == nil ? "Set city context" : "Refresh city context")

                    Button { showingNotifications = true } label: {
                        Image(systemName: "bell")
                            .font(.system(size: 18))
                            .foregroundStyle(LinkUpPalette.textDimmed)
                            .frame(width: 40, height: 40)
                            .background(LinkUpPalette.elevated)
                            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Notifications")
                }
            }
            HStack(spacing: 10) {
                Image(systemName: "magnifyingglass").foregroundStyle(LinkUpPalette.textMuted)
                TextField("Search activities, places...", text: $search)
                    .font(LinkUpTypography.body(14))
                    .foregroundStyle(LinkUpPalette.textPrimary)
            }
            .padding(.horizontal, 12)
            .frame(height: 42)
            .background(LinkUpPalette.elevated)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
        }
        .padding(.horizontal, 20)
        .padding(.top, 12)
        .padding(.bottom, 12)
        .linkUpGlass()
    }

    private var cityContextMessage: String {
        switch cityContext.phase {
        case .failed(let message): message
        case .loading: "Loading server city context…"
        default: "Set your city to activate the privacy-safe City-Lock."
        }
    }

    private var filters: some View {
        VStack(alignment: .leading, spacing: 10) {
            chipRow(["Now", "Tonight", "Tomorrow", "All"], selection: $time)
            chipRow(["All", "Social", "Active", "Food"], selection: $category)
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
    }

    @ViewBuilder private var content: some View {
        VStack(spacing: 12) {
            if let mutationError = coordinator.mutationError {
                LinkUpInlineError(message: mutationError) { coordinator.clearMutationError() }
            }
            pulseContent
        }
    }

    @ViewBuilder private var pulseContent: some View {
        switch coordinator.pulsePhase {
        case .idle, .loading:
            loadingCards
        case .failed(let message):
            LinkUpErrorState(message: message) {
                Task { await coordinator.loadPulse() }
            }
        case .empty:
            LinkUpEmptyState(
                title: "Quiet around here",
                message: "No public LinkUps are available right now.",
                systemImage: "bolt.fill"
            )
        case .refreshing, .content:
            if filteredItems.isEmpty {
                LinkUpEmptyState(
                    title: "No matches",
                    message: "Try a different filter or search term.",
                    systemImage: "magnifyingglass"
                )
            } else {
                LazyVStack(spacing: 12) {
                    if coordinator.pulsePhase == .refreshing {
                        ProgressView().tint(LinkUpPalette.red).padding(.vertical, 4)
                    }
                    ForEach(filteredItems) { slot in
                        SlotCardView(
                            slot: slot,
                            cityTimeScope: activeCityTimeScope,
                            isMutating: coordinator.mutationControlsDisabled,
                            open: { selectedSlot = slot },
                            primaryAction: { primaryAction(slot) }
                        )
                    }
                }
            }
        }
    }

    private var loadingCards: some View {
        VStack(spacing: 12) {
            ForEach(0..<4, id: \.self) { _ in
                RoundedRectangle(cornerRadius: LinkUpRadius.card)
                    .fill(LinkUpPalette.elevated)
                    .frame(height: 220)
                    .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
            }
        }
    }

    private var activeCityTimeScope: CityTimeScope? {
        guard let context = cityContext.context, context.isFresh() else { return nil }
        return context.timeScope
    }

    private var filteredItems: [SlotModel] {
        coordinator.pulseItems.filter { slot in
            matchesSearch(slot) && matchesTime(slot) && matchesCategory(slot)
        }
    }

    private func matchesSearch(_ slot: SlotModel) -> Bool {
        let query = search.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !query.isEmpty else { return true }
        return slot.title.lowercased().contains(query) ||
            slot.activity.lowercased().contains(query) ||
            slot.placeText.lowercased().contains(query) ||
            (slot.zoneText?.lowercased().contains(query) ?? false) ||
            (slot.details?.lowercased().contains(query) ?? false)
    }

    private func matchesTime(_ slot: SlotModel) -> Bool {
        guard time != "All" else { return true }
        guard let date = slot.startAt else { return time == "Now" }
        guard let scope = activeCityTimeScope else { return false }
        let now = Date()
        switch time {
        case "Tomorrow":
            return scope.isTomorrow(date, relativeTo: now)
        case "Tonight":
            return scope.isToday(date, relativeTo: now) && scope.localHour(for: date) >= 17
        default:
            return scope.isToday(date, relativeTo: now)
        }
    }

    private func matchesCategory(_ slot: SlotModel) -> Bool {
        guard category != "All" else { return true }
        let value = slot.activity.lowercased()
        switch category {
        case "Food": return ["food", "coffee", "drinks"].contains(value)
        case "Active": return ["running", "walk", "sport", "yoga", "cycling"].contains(value)
        case "Social": return ["coffee", "drinks", "music", "games", "chess", "cowork", "networking", "study", "art", "photography"].contains(value)
        default: return true
        }
    }

    private func primaryAction(_ slot: SlotModel) {
        if slot.canRequestToJoin {
            Task {
                do { _ = try await coordinator.request(slot) }
                catch is CancellationError { return }
                catch { return }
            }
        } else {
            selectedSlot = slot
        }
    }

    private func chipRow(_ values: [String], selection: Binding<String>) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(values, id: \.self) { value in
                    LinkUpChip(title: value, active: selection.wrappedValue == value) {
                        selection.wrappedValue = value
                    }
                }
            }
        }
    }
}
