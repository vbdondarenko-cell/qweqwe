import SwiftUI

@MainActor
struct HostManagementView: View {
    private enum ConfirmAction {
        case remove
        case block
    }

    @Environment(\.dismiss) private var dismiss
    @Binding var slot: SlotModel
    @ObservedObject var social: SocialCoordinator
    @StateObject private var roster: HostRosterCoordinator

    @State private var target: SlotOrganizer?
    @State private var confirmAction: ConfirmAction = .remove
    @State private var showConfirmation = false
    @State private var localError: String?
    @State private var lastRealtimeRevision: UInt64 = 0

    init(
        slot: Binding<SlotModel>,
        social: SocialCoordinator,
        api: LinkUpAPI,
        session: SessionCoordinator
    ) {
        _slot = slot
        self.social = social
        _roster = StateObject(wrappedValue: HostRosterCoordinator(
            slotID: slot.wrappedValue.id,
            api: api,
            session: session
        ))
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    if let localError {
                        errorBanner(localError)
                    }
                    if let mutationError = social.mutationError {
                        errorBanner(mutationError)
                    }

                    switch roster.phase {
                    case .idle, .loading:
                        ProgressView().tint(LinkUpPalette.red).frame(maxWidth: .infinity).padding(.top, 40)
                    case .failed(let message):
                        LinkUpErrorState(message: message) { Task { await roster.load() } }
                    case .content:
                        pendingSection
                        acceptedSection
                    }
                }
                .padding(20)
            }
            .background(LinkUpPalette.background)
            .navigationTitle("Participants")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }.foregroundStyle(LinkUpPalette.red)
                }
            }
            .task {
                await roster.load()
                lastRealtimeRevision = social.realtimeRevision
            }
            .onChange(of: social.realtimeRevision) { _, revision in
                applyRealtimeRevision(revision)
            }
            .refreshable { await roster.load() }
            .onDisappear { roster.dispose() }
            .confirmationDialog(
                confirmAction == .block ? "Block this account?" : "Remove this participant?",
                isPresented: $showConfirmation,
                titleVisibility: .visible
            ) {
                if let target {
                    Button(confirmAction == .block ? "Block" : "Remove", role: .destructive) {
                        runConfirmedAction(target)
                    }
                }
                Button("Cancel", role: .cancel) { }
            } message: {
                if let target {
                    Text(confirmAction == .block
                         ? "Blocking @\(target.username) revokes the current social relationship server-side."
                         : "@\(target.username) will leave this LinkUp without being blocked.")
                }
            }
        }
        .preferredColorScheme(.dark)
    }

    private var pendingSection: some View {
        section(title: "REQUESTS", empty: roster.pending.isEmpty, emptyText: "No pending requests") {
            ForEach(roster.pending, id: \.user.id) { request in
                VStack(spacing: 0) {
                    identityRow(request.user, subtitle: request.requestedAt.formatted(date: .abbreviated, time: .shortened))
                    HStack(spacing: 8) {
                        compactButton("Accept", tint: LinkUpPalette.success) {
                            mutate { try await social.approve(slot, userID: request.user.id) }
                        }
                        compactButton("Decline", tint: LinkUpPalette.warning) {
                            mutate { try await social.reject(slot, userID: request.user.id) }
                        }
                        compactButton("Block", tint: LinkUpPalette.critical) {
                            target = request.user
                            confirmAction = .block
                            showConfirmation = true
                        }
                    }
                    .padding(.horizontal, 12)
                    .padding(.bottom, 12)
                }
            }
        }
    }

    private var acceptedSection: some View {
        section(title: "ACCEPTED", empty: roster.accepted.isEmpty, emptyText: "No accepted participants") {
            ForEach(roster.accepted) { participant in
                VStack(spacing: 0) {
                    identityRow(participant, subtitle: "Accepted participant")
                    HStack(spacing: 8) {
                        compactButton("Remove", tint: LinkUpPalette.warning) {
                            target = participant
                            confirmAction = .remove
                            showConfirmation = true
                        }
                        compactButton("Block", tint: LinkUpPalette.critical) {
                            target = participant
                            confirmAction = .block
                            showConfirmation = true
                        }
                    }
                    .padding(.horizontal, 12)
                    .padding(.bottom, 12)
                }
            }
        }
    }


    private func applyRealtimeRevision(_ revision: UInt64) {
        guard revision > lastRealtimeRevision else { return }
        let previous = lastRealtimeRevision
        lastRealtimeRevision = revision

        if social.accessLostRealtimeRevision(for: slot.id) > previous {
            dismiss()
            return
        }
        if social.slotRealtimeRevision(for: slot.id) > previous,
           let updated = social.activeSlotSnapshot(for: slot.id) {
            slot = updated
        }
        if social.relationshipRealtimeRevision(for: slot.id) > previous,
           let snapshot = social.realtimeRelationshipSnapshot(for: slot.id) {
            roster.applyRealtimeSnapshot(pending: snapshot.pending, accepted: snapshot.accepted)
            localError = nil
        }
    }

    private func section<Content: View>(
        title: String,
        empty: Bool,
        emptyText: String,
        @ViewBuilder content: () -> Content
    ) -> some View {
        VStack(alignment: .leading, spacing: 7) {
            Text(title)
                .font(LinkUpTypography.mono(10, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textMuted)
                .padding(.horizontal, 4)
            if empty {
                LinkUpCard {
                    Text(emptyText)
                        .font(LinkUpTypography.body(13))
                        .foregroundStyle(LinkUpPalette.textMuted)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            } else {
                VStack(spacing: 0) { content() }
                    .background(LinkUpPalette.elevated)
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
                    .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
            }
        }
    }

    private func identityRow(_ user: SlotOrganizer, subtitle: String) -> some View {
        HStack(spacing: 10) {
            LinkUpAvatar(initials: initials(user.displayName), size: .md)
            VStack(alignment: .leading, spacing: 2) {
                Text(user.displayName)
                    .font(LinkUpTypography.body(13, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.textPrimary)
                Text("@\(user.username) · \(subtitle)")
                    .font(LinkUpTypography.body(10))
                    .foregroundStyle(LinkUpPalette.textMuted)
                    .lineLimit(1)
            }
            Spacer()
        }
        .padding(12)
    }

    private func compactButton(_ title: String, tint: Color, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(LinkUpTypography.body(11, weight: .semibold))
                .foregroundStyle(tint)
                .frame(maxWidth: .infinity)
                .frame(height: 34)
                .background(tint.opacity(0.09))
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.compact))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.compact).stroke(tint.opacity(0.3)) }
        }
        .buttonStyle(.plain)
        .disabled(social.isMutating)
    }

    private func mutate(_ operation: @escaping () async throws -> SlotModel?) {
        guard !social.isMutating else { return }
        localError = nil
        Task {
            do {
                if let updated = try await operation(), updated.id == slot.id {
                    slot = updated
                }
                await roster.load()
            } catch is CancellationError {
                return
            } catch {
                localError = error.localizedDescription
            }
        }
    }

    private func runConfirmedAction(_ user: SlotOrganizer) {
        switch confirmAction {
        case .remove:
            mutate { try await social.removeParticipant(slot, userID: user.id) }
        case .block:
            mutate { try await social.block(slot, userID: user.id) }
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

    private func initials(_ name: String) -> String {
        let value = name.split(separator: " ").prefix(2).compactMap(\.first).map(String.init).joined()
        return value.isEmpty ? "LU" : value.uppercased()
    }
}
