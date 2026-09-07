import Foundation
import SwiftUI

@MainActor
final class ChatCoordinator: ObservableObject {
    @Published private(set) var messages: [ChatMessage] = []
    @Published private(set) var isLoading = false
    @Published private(set) var isSending = false
    @Published private(set) var errorMessage: String?

    private let slotID: UUID
    private let api: LinkUpAPI
    private let session: SessionCoordinator
    private var generation: UInt64 = 0

    init(slotID: UUID, api: LinkUpAPI, session: SessionCoordinator) {
        self.slotID = slotID
        self.api = api
        self.session = session
    }

    func load(silent: Bool = false) async {
        guard !isSending else { return }
        generation &+= 1
        let requestGeneration = generation
        if !silent && messages.isEmpty { isLoading = true }
        defer {
            if requestGeneration == generation { isLoading = false }
        }

        do {
            let snapshot = try await api.chatMessages(slotID)
            guard requestGeneration == generation else { return }
            messages = snapshot
            errorMessage = nil
        } catch is CancellationError {
            return
        } catch let error as APIError {
            guard requestGeneration == generation else { return }
            if case .unauthorized = error {
                await session.clearLocalSession()
            } else {
                errorMessage = error.localizedDescription
            }
        } catch {
            guard requestGeneration == generation else { return }
            errorMessage = "Unable to load chat."
        }
    }

    func send(_ rawText: String) async -> Bool {
        guard !isSending else { return false }
        let text = rawText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else { return false }

        isSending = true
        errorMessage = nil
        generation &+= 1
        defer { isSending = false }

        do {
            let message = try await api.sendChatMessage(slotID, text: text)
            if !messages.contains(where: { $0.id == message.id }) {
                messages.append(message)
                if messages.count > 100 {
                    messages.removeFirst(messages.count - 100)
                }
            }
            return true
        } catch is CancellationError {
            return false
        } catch let error as APIError {
            if case .unauthorized = error {
                await session.clearLocalSession()
            } else {
                errorMessage = error.localizedDescription
            }
            return false
        } catch {
            errorMessage = "Message was not confirmed by the server."
            return false
        }
    }

    func dispose() {
        generation &+= 1
        messages = []
        errorMessage = nil
    }
}
