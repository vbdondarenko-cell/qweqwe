import SwiftUI

@MainActor
struct ChatView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var coordinator: ChatCoordinator
    @ObservedObject private var social: SocialCoordinator
    @State private var draft = ""
    @State private var lastRealtimeRevision: UInt64 = 0

    private let slot: SlotModel

    init(slot: SlotModel, api: LinkUpAPI, session: SessionCoordinator, social: SocialCoordinator) {
        self.slot = slot
        _coordinator = StateObject(wrappedValue: ChatCoordinator(slotID: slot.id, api: api, session: session))
        _social = ObservedObject(wrappedValue: social)
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                messageArea
                composer
            }
            .background(LinkUpPalette.background.ignoresSafeArea())
            .navigationTitle(slot.title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }
                        .foregroundStyle(LinkUpPalette.red)
                }
            }
            .task {
                social.registerActiveChat(slot.id)
                await coordinator.load()
                lastRealtimeRevision = social.realtimeRevision
            }
            .onChange(of: social.realtimeRevision) { _, revision in
                applyRealtimeRevision(revision)
            }
            .onDisappear {
                social.unregisterActiveChat(slot.id)
                coordinator.dispose()
            }
        }
        .preferredColorScheme(.dark)
    }

    @ViewBuilder private var messageArea: some View {
        if coordinator.isLoading && coordinator.messages.isEmpty {
            Spacer()
            ProgressView().tint(LinkUpPalette.red)
            Spacer()
        } else if coordinator.messages.isEmpty {
            Spacer()
            LinkUpEmptyState(
                title: "Coordination chat",
                message: "No messages yet. This thread is temporary and closes with the LinkUp.",
                systemImage: "message"
            )
            Spacer()
        } else {
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 10) {
                        ForEach(coordinator.messages) { message in
                            messageRow(message)
                                .id(message.id)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 12)
                }
                .scrollIndicators(.hidden)
                .onChange(of: coordinator.messages.last?.id) { _, id in
                    guard let id else { return }
                    withAnimation(.easeOut(duration: 0.2)) { proxy.scrollTo(id, anchor: .bottom) }
                }
            }
        }
    }

    private var composer: some View {
        let messageCount = InputContracts.scalarCount(InputContracts.trimmed(draft))
        let messageValid = InputContracts.validChatMessage(draft)
        return VStack(spacing: 8) {
            HStack {
                Text("Coordination only")
                    .font(LinkUpTypography.body(10))
                    .foregroundStyle(LinkUpPalette.textMuted)
                Spacer()
                Text("\(messageCount)/\(InputContracts.chatMessageMaxScalars)")
                    .font(LinkUpTypography.mono(9))
                    .foregroundStyle(messageCount > InputContracts.chatMessageMaxScalars ? LinkUpPalette.critical : LinkUpPalette.textMuted)
            }
            if let error = coordinator.errorMessage {
                Text(error)
                    .font(LinkUpTypography.body(11))
                    .foregroundStyle(LinkUpPalette.critical)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
            HStack(alignment: .bottom, spacing: 10) {
                TextField("Message", text: $draft, axis: .vertical)
                    .lineLimit(1...4)
                    .font(LinkUpTypography.body(14))
                    .padding(.horizontal, 12)
                    .padding(.vertical, 10)
                    .background(LinkUpPalette.zone)
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                    .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }

                Button { send() } label: {
                    Image(systemName: "arrow.up")
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(.white)
                        .frame(width: 42, height: 42)
                        .background(LinkUpPalette.red)
                        .clipShape(Circle())
                }
                .buttonStyle(.plain)
                .disabled(coordinator.isSending || !messageValid)
                .opacity(coordinator.isSending ? 0.5 : 1)
            }
        }
        .padding(.horizontal, 14)
        .padding(.top, 10)
        .padding(.bottom, 8)
        .background(LinkUpPalette.surface)
        .overlay(alignment: .top) { Rectangle().fill(LinkUpPalette.border).frame(height: 1) }
    }

    private func messageRow(_ message: ChatMessage) -> some View {
        HStack(alignment: .top, spacing: 10) {
            LinkUpAvatar(initials: initials(message.author.displayName), size: .sm)
            VStack(alignment: .leading, spacing: 4) {
                HStack(spacing: 6) {
                    Text(message.author.displayName)
                        .font(LinkUpTypography.body(12, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.textPrimary)
                    Text("@\(message.author.username)")
                        .font(LinkUpTypography.body(10))
                        .foregroundStyle(LinkUpPalette.textMuted)
                    Spacer()
                    Text(message.createdAt, style: .time)
                        .font(LinkUpTypography.mono(9))
                        .foregroundStyle(LinkUpPalette.textMuted)
                }
                Text(message.text)
                    .font(LinkUpTypography.body(14))
                    .foregroundStyle(LinkUpPalette.textDimmed)
                    .textSelection(.disabled)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .padding(12)
        .background(LinkUpPalette.elevated)
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
    }

    private func send() {
        let text = draft
        Task {
            if await coordinator.send(text) {
                draft = ""
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
        if social.chatRealtimeRevision(for: slot.id) > previous,
           let snapshot = social.realtimeChatSnapshot(for: slot.id) {
            coordinator.applyRealtimeSnapshot(snapshot)
        }
    }

    private func initials(_ name: String) -> String {
        let value = name.split(separator: " ").prefix(2).compactMap(\.first).map(String.init).joined()
        return value.isEmpty ? "LU" : value.uppercased()
    }
}
