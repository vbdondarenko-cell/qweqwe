import SwiftUI

struct SlotDetailView: View {
    @Environment(\.dismiss) private var dismiss

    let coordinator: SocialCoordinator
    let api: LinkUpAPI
    let session: SessionCoordinator

    @State private var slot: SlotModel
    @State private var loadError: String?
    @State private var showingChat = false

    init(slot: SlotModel, coordinator: SocialCoordinator, api: LinkUpAPI, session: SessionCoordinator) {
        self.coordinator = coordinator
        self.api = api
        self.session = session
        _slot = State(initialValue: slot)
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    identity
                    details
                    organizer
                    capacity
                    if let loadError { errorBanner(loadError) }
                    if let mutationError = coordinator.mutationError { errorBanner(mutationError) }
                    controls
                }
                .padding(20)
            }
            .scrollIndicators(.hidden)
            .background(LinkUpPalette.background)
            .navigationTitle("LINK")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }.foregroundStyle(LinkUpPalette.red)
                }
            }
            .task { await refresh() }
            .sheet(isPresented: $showingChat) {
                ChatView(slot: slot, api: api, session: session)
            }
        }
        .preferredColorScheme(.dark)
    }

    private var identity: some View {
        HStack(alignment: .top, spacing: 14) {
            Image(systemName: activitySymbol)
                .font(.system(size: 26, weight: .semibold))
                .frame(width: 56, height: 56)
                .foregroundStyle(LinkUpPalette.textPrimary)
                .background(LinkUpPalette.zone)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
            VStack(alignment: .leading, spacing: 6) {
                LinkUpStatusBadge(status: visualStatus)
                Text(slot.title)
                    .font(LinkUpTypography.display(20))
                    .foregroundStyle(LinkUpPalette.textPrimary)
                Label(slot.placeText, systemImage: "mappin")
                    .font(LinkUpTypography.body(12))
                    .foregroundStyle(LinkUpPalette.textDimmed)
            }
            Spacer(minLength: 0)
        }
    }

    @ViewBuilder private var details: some View {
        if let details = slot.details, !details.isEmpty {
            Text(details)
                .font(LinkUpTypography.body(14))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        if let startAt = slot.startAt {
            Label {
                Text(startAt.formatted(date: .abbreviated, time: .shortened))
            } icon: {
                Image(systemName: "calendar")
            }
            .font(LinkUpTypography.body(12))
            .foregroundStyle(LinkUpPalette.textDimmed)
        }
        if slot.canonicalPlaceId != nil {
            Label("Verified place identity", systemImage: "checkmark.seal.fill")
                .font(LinkUpTypography.body(11, weight: .semibold))
                .foregroundStyle(LinkUpPalette.success)
        }
    }

    private var organizer: some View {
        LinkUpCard {
            HStack(spacing: 10) {
                LinkUpAvatar(initials: organizerInitials, size: .md)
                VStack(alignment: .leading, spacing: 2) {
                    Text(slot.organizer.displayName)
                        .font(LinkUpTypography.body(13, weight: .semibold))
                    Text("@\(slot.organizer.username)")
                        .font(LinkUpTypography.body(11))
                        .foregroundStyle(LinkUpPalette.textMuted)
                }
                Spacer()
                Text(slot.viewerState.rawValue)
                    .font(LinkUpTypography.mono(9, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.textMuted)
            }
            .foregroundStyle(LinkUpPalette.textPrimary)
        }
    }

    private var capacity: some View {
        LinkUpCard {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    Text("Capacity").font(LinkUpTypography.body(13, weight: .semibold))
                    Spacer()
                    Text("\(slot.acceptedCount)/\(slot.capacity)")
                        .font(LinkUpTypography.mono(12, weight: .bold))
                }
                LinkUpProgress(value: slot.acceptedCount, maximum: slot.capacity, showsLabel: false)
            }
            .foregroundStyle(LinkUpPalette.textPrimary)
        }
    }

    @ViewBuilder private var controls: some View {
        VStack(spacing: 10) {
            switch slot.viewerState {
            case .none:
                if slot.accessMode == .approval && slot.state == .filling {
                    LinkUpButton(title: "Request to join", disabled: coordinator.isMutating) {
                        runMutation { try await coordinator.request(slot) }
                    }
                } else if slot.state == .full {
                    LinkUpButton(title: "Full", variant: .secondary, disabled: true) { }
                }
            case .pending:
                LinkUpButton(title: "Withdraw request", variant: .secondary, disabled: coordinator.isMutating) {
                    runMutation { try await coordinator.leave(slot) }
                }
            case .accepted:
                if !slot.isTerminal {
                    LinkUpButton(title: "Open chat") { showingChat = true }
                    LinkUpButton(title: "Leave LinkUp", variant: .danger, disabled: coordinator.isMutating) {
                        runMutation { try await coordinator.leave(slot) }
                    }
                }
            case .host:
                hostControls
            }
        }
    }

    @ViewBuilder private var hostControls: some View {
        if !slot.isTerminal {
            LinkUpButton(title: "Open chat", variant: .secondary) { showingChat = true }
        }
        if slot.state == .filling || slot.state == .full {
            LinkUpButton(title: "Start LinkUp", variant: .success, disabled: coordinator.isMutating) {
                runMutation { try await coordinator.start(slot) }
            }
        }
        if slot.state == .active {
            LinkUpButton(title: "Complete LinkUp", variant: .success, disabled: coordinator.isMutating) {
                runMutation { try await coordinator.complete(slot) }
            }
        }
        if !slot.isTerminal {
            LinkUpButton(title: "Cancel LinkUp", variant: .danger, disabled: coordinator.isMutating) {
                runMutation { try await coordinator.cancel(slot) }
            }
        }
    }

    private func refresh() async {
        do {
            slot = try await coordinator.loadSlot(slot.id)
            loadError = nil
        } catch is CancellationError {
            return
        } catch {
            loadError = error.localizedDescription
        }
    }

    private func runMutation(_ operation: @escaping () async throws -> SlotModel?) {
        let previousID = slot.id
        Task {
            do {
                if let updated = try await operation(), updated.id == previousID {
                    slot = updated
                    loadError = nil
                    if updated.isTerminal { showingChat = false }
                }
            } catch is CancellationError {
                return
            } catch {
                loadError = error.localizedDescription
            }
        }
    }

    private func errorBanner(_ message: String) -> some View {
        HStack(alignment: .top, spacing: 9) {
            Image(systemName: "exclamationmark.triangle.fill").foregroundStyle(LinkUpPalette.critical)
            Text(message)
                .font(LinkUpTypography.body(12))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(12)
        .background(LinkUpPalette.critical.opacity(0.09))
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
    }

    private var visualStatus: LinkUpVisualStatus {
        if slot.state == .active { return .live }
        if slot.state == .full { return .full }
        if slot.accessMode == .approval && slot.viewerState == .none { return .approval }
        return .open
    }

    private var organizerInitials: String {
        let value = slot.organizer.displayName.split(separator: " ").prefix(2).compactMap(\.first).map(String.init).joined()
        return value.isEmpty ? "LU" : value.uppercased()
    }

    private var activitySymbol: String {
        switch slot.activity.lowercased() {
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
