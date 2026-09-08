import SwiftUI

@MainActor
struct RootView: View {
    @ObservedObject var runtime: AppRuntime

    var body: some View {
        Group {
            switch runtime.state {
            case .starting:
                startupView
            case .failed(let message):
                configurationFailure(message)
            case .ready(let services):
                SessionRootView(services: services)
            }
        }
        .preferredColorScheme(.dark)
        .background(LinkUpPalette.background.ignoresSafeArea())
    }

    private var startupView: some View {
        VStack(spacing: 14) {
            ProgressView().tint(LinkUpPalette.red)
            Text("Starting LinkUp…")
                .font(LinkUpTypography.body(13))
                .foregroundStyle(LinkUpPalette.textDimmed)
        }
    }

    private func configurationFailure(_ message: String) -> some View {
        VStack(spacing: 16) {
            Image(systemName: "lock.trianglebadge.exclamationmark")
                .font(.system(size: 34))
                .foregroundStyle(LinkUpPalette.critical)
            Text("LinkUp configuration error")
                .font(LinkUpTypography.display(20))
                .foregroundStyle(LinkUpPalette.textPrimary)
            Text(L10n.text(message))
                .font(LinkUpTypography.body(13))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .multilineTextAlignment(.center)
        }
        .padding(24)
    }
}

@MainActor
private struct SessionRootView: View {
    let services: AppServices
    @ObservedObject private var session: SessionCoordinator

    init(services: AppServices) {
        self.services = services
        _session = ObservedObject(wrappedValue: services.session)
    }

    var body: some View {
        Group {
            switch session.state {
            case .checking:
                ProgressView().tint(LinkUpPalette.red)
            case .signedOut:
                AuthView(session: session, authRoutes: services.authRoutes)
            case .signedIn(let user):
                MainShellView(services: services, user: user)
                    .id(user.id)
            case .offlineSession(let expiresAt):
                sessionProblem(
                    title: "You're offline",
                    message: L10n.format("fmt.offline_session", expiresAt.formatted(date: .abbreviated, time: .shortened))
                )
            case .recoverableError(let message):
                sessionProblem(title: "Session unavailable", message: message)
            }
        }
        .onChange(of: session.state) { _, state in
            switch state {
            case .checking, .signedOut:
                break
            case .signedIn, .offlineSession, .recoverableError:
                services.authRoutes.clear()
            }
        }
    }

    private func sessionProblem(title: String, message: String) -> some View {
        VStack(spacing: 16) {
            Image(systemName: "wifi.exclamationmark")
                .font(.system(size: 34))
                .foregroundStyle(LinkUpPalette.warning)
            Text(L10n.text(title))
                .font(LinkUpTypography.display(20))
                .foregroundStyle(LinkUpPalette.textPrimary)
            Text(L10n.text(message))
                .font(LinkUpTypography.body(13))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .multilineTextAlignment(.center)
            LinkUpButton(title: "Retry") { Task { await session.bootstrap() } }
            LinkUpButton(title: "Sign out", variant: .secondary) { Task { await session.clearLocalSession() } }
        }
        .padding(24)
    }
}

@MainActor
private struct MainShellView: View {
    @Environment(\.scenePhase) private var scenePhase

    let services: AppServices
    let user: UserProfile

    @StateObject private var social: SocialCoordinator
    @State private var selection: AppTab = .pulse
    @State private var showingCreate = false

    init(services: AppServices, user: UserProfile) {
        self.services = services
        self.user = user
        _social = StateObject(wrappedValue: SocialCoordinator(api: services.api, session: services.session))
    }

    var body: some View {
        ZStack {
            LinkUpPalette.background.ignoresSafeArea()
            activeScreen
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            FrozenBottomBar(selection: $selection) { showingCreate = true }
        }
        .fullScreenCover(isPresented: $showingCreate) {
            CreateLinkView(coordinator: social, cityContext: services.cityContext) { showingCreate = false }
        }
        .task(id: scenePhase) {
            guard scenePhase == .active else { return }
            await services.cityContext.loadCurrent()
            guard !Task.isCancelled else { return }
            await runRealtimeLoop()
        }
        .onDisappear {
            social.dispose()
            services.cityContext.clear()
        }
    }


    private enum DraftWorkflowRecovery: Equatable {
        case idle
        case progressed
        case queued
    }

    private func recoverDraftPublishWorkflow() async throws -> DraftWorkflowRecovery {
        do {
            guard let slot = try await services.api.resumeDraftPublishWorkflow() else { return .idle }
            social.applyDraftPublishResult(slot)
            return .progressed
        } catch let error as APIError {
            if case .mutationQueued(let key) = error {
                social.registerQueuedMutation(key)
                return .queued
            }
            throw error
        }
    }

    private func runRealtimeLoop() async {
        var needsAuthoritativeSnapshot = true
        while !Task.isCancelled {
            let delay: Duration
            do {
                if needsAuthoritativeSnapshot {
                    let profileOK = await services.session.refreshSignedInProfileSnapshot()
                    guard profileOK else {
                        delay = .seconds(5)
                        try await Task.sleep(for: delay)
                        continue
                    }
                    let pulseOK = await social.loadPulse()
                    guard pulseOK else {
                        delay = .seconds(5)
                        try await Task.sleep(for: delay)
                        continue
                    }

                    let replay = try await services.api.replayPendingMutations()
                    if replay.needsSafetyIntervention || replay.ambiguousFailureKey != nil {
                        social.applyDurableReplayReport(replay)
                        delay = .seconds(5)
                    } else {
                        if replay.madeProgress {
                            guard await reconcileAfterDurableReplay() else {
                                delay = .seconds(5)
                                try await Task.sleep(for: delay)
                                continue
                            }
                        }
                        social.applyDurableReplayReport(replay)
                        let draftRecovery = try await recoverDraftPublishWorkflow()
                        if draftRecovery == .queued {
                            delay = .seconds(5)
                        } else {
                            needsAuthoritativeSnapshot = false
                            delay = .milliseconds(250)
                        }
                    }
                } else {
                    let replay = try await services.api.replayPendingMutations()
                    if replay.needsSafetyIntervention || replay.ambiguousFailureKey != nil {
                        social.applyDurableReplayReport(replay)
                        delay = .seconds(5)
                    } else {
                        if replay.madeProgress {
                            guard await reconcileAfterDurableReplay() else {
                                delay = .seconds(5)
                                try await Task.sleep(for: delay)
                                continue
                            }
                        }
                        social.applyDurableReplayReport(replay)
                        let draftRecovery = try await recoverDraftPublishWorkflow()
                        if draftRecovery == .queued {
                            delay = .seconds(5)
                        } else if replay.madeProgress || draftRecovery == .progressed {
                            delay = .milliseconds(250)
                        } else {
                            let pull = try await services.realtime.pull(userID: user.id, limit: 100)
                            if pull.nextCursor > pull.fromCursor {
                                let reconciliation = await reconcileRealtime(pull)
                                if reconciliation.success {
                                    try await services.realtime.acknowledge(userID: user.id, cursor: pull.nextCursor)
                                    social.applyRealtimeHints(
                                        pull.hints,
                                        accessLostSlotIDs: reconciliation.accessLostSlotIDs,
                                        additionalRefreshedSlotIDs: reconciliation.refreshedSlotIDs
                                    )
                                    delay = pull.events.count >= 100 ? .milliseconds(250) : .seconds(4)
                                } else {
                                    delay = .seconds(5)
                                }
                            } else {
                                delay = .seconds(4)
                            }
                        }
                    }
                }
            } catch is CancellationError {
                return
            } catch let error as APIError {
                if case .unauthorized = error {
                    await services.session.clearLocalSession()
                    return
                }
                social.applyDurableReplayFailure(error)
                delay = .seconds(5)
            } catch {
                delay = .seconds(5)
            }

            do {
                try await Task.sleep(for: delay)
            } catch {
                return
            }
        }
    }

    private func reconcileAfterDurableReplay() async -> Bool {
        guard await social.loadPulse() else { return false }

        let targets = social.realtimeTargets()
        var refreshedSlotIDs = Set<UUID>()
        var accessLostSlotIDs = Set<UUID>()
        var relationshipSlotIDs = Set<UUID>()
        var chatSlotIDs = Set<UUID>()

        if let activeSlot = targets.slot {
            do {
                let updated = try await social.loadSlot(activeSlot.id)
                social.updateActiveSlot(updated)
                refreshedSlotIDs.insert(updated.id)
            } catch let error as APIError {
                if error.isDefinitiveSlotAccessLoss {
                    accessLostSlotIDs.insert(activeSlot.id)
                } else {
                    return false
                }
            } catch is CancellationError {
                return false
            } catch {
                return false
            }
        }

        if let activeSlot = social.realtimeTargets().slot,
           activeSlot.canManageParticipants,
           !accessLostSlotIDs.contains(activeSlot.id) {
            do {
                async let pendingRequest = services.api.pendingRequests(activeSlot.id)
                async let acceptedRequest = services.api.acceptedParticipants(activeSlot.id)
                let (pending, accepted) = try await (pendingRequest, acceptedRequest)
                social.stageRealtimeRelationshipSnapshot(
                    slotID: activeSlot.id,
                    pending: pending,
                    accepted: accepted
                )
                relationshipSlotIDs.insert(activeSlot.id)
            } catch let error as APIError {
                if case .unauthorized = error { await services.session.clearLocalSession() }
                return false
            } catch is CancellationError {
                return false
            } catch {
                return false
            }
        }

        if let chatSlotID = targets.chatSlotID,
           !accessLostSlotIDs.contains(chatSlotID),
           realtimeChatRefreshAllowed(activeSlot: social.realtimeTargets().slot, chatSlotID: chatSlotID) {
            do {
                let messages = try await services.api.chatMessages(chatSlotID)
                social.stageRealtimeChatSnapshot(slotID: chatSlotID, messages: messages)
                chatSlotIDs.insert(chatSlotID)
            } catch let error as APIError {
                if error.isDefinitiveSlotAccessLoss {
                    accessLostSlotIDs.insert(chatSlotID)
                } else {
                    if case .unauthorized = error { await services.session.clearLocalSession() }
                    return false
                }
            } catch is CancellationError {
                return false
            } catch {
                return false
            }
        }

        let allRefreshedSlots = refreshedSlotIDs.union(relationshipSlotIDs)
        social.applyRealtimeHints(
            RealtimeInvalidationHints(
                refreshPulse: true,
                refreshProfile: false,
                refreshRelationships: !relationshipSlotIDs.isEmpty,
                slotIDs: allRefreshedSlots,
                chatSlotIDs: chatSlotIDs
            ),
            accessLostSlotIDs: accessLostSlotIDs,
            additionalRefreshedSlotIDs: allRefreshedSlots
        )
        return true
    }

    private func reconcileRealtime(_ pull: RealtimePull) async -> (
        success: Bool,
        accessLostSlotIDs: Set<UUID>,
        refreshedSlotIDs: Set<UUID>
    ) {
        let hints = pull.hints
        var accessLost = Set<UUID>()
        var refreshedSlotIDs = Set<UUID>()

        if hints.refreshProfile {
            guard await services.session.refreshSignedInProfileSnapshot() else {
                return (false, accessLost, refreshedSlotIDs)
            }
        }

        if hints.refreshPulse {
            guard await social.loadPulse() else {
                return (false, accessLost, refreshedSlotIDs)
            }
        }

        let targets = social.realtimeTargets()
        if let activeSlot = targets.slot,
           hints.slotIDs.contains(activeSlot.id) || (hints.refreshRelationships && hints.slotIDs.isEmpty) {
            do {
                let updated = try await social.loadSlot(activeSlot.id)
                social.updateActiveSlot(updated)
                refreshedSlotIDs.insert(activeSlot.id)
            } catch let error as APIError {
                if error.isDefinitiveSlotAccessLoss {
                    accessLost.insert(activeSlot.id)
                } else {
                    return (false, accessLost, refreshedSlotIDs)
                }
            } catch is CancellationError {
                return (false, accessLost, refreshedSlotIDs)
            } catch {
                return (false, accessLost, refreshedSlotIDs)
            }
        }

        if hints.refreshRelationships,
           let activeSlot = social.realtimeTargets().slot,
           activeSlot.canManageParticipants,
           (hints.slotIDs.contains(activeSlot.id) || refreshedSlotIDs.contains(activeSlot.id)),
           !accessLost.contains(activeSlot.id) {
            do {
                async let pendingRequest = services.api.pendingRequests(activeSlot.id)
                async let acceptedRequest = services.api.acceptedParticipants(activeSlot.id)
                let (pending, accepted) = try await (pendingRequest, acceptedRequest)
                social.stageRealtimeRelationshipSnapshot(
                    slotID: activeSlot.id,
                    pending: pending,
                    accepted: accepted
                )
            } catch let error as APIError {
                if case .unauthorized = error {
                    await services.session.clearLocalSession()
                }
                return (false, accessLost, refreshedSlotIDs)
            } catch is CancellationError {
                return (false, accessLost, refreshedSlotIDs)
            } catch {
                return (false, accessLost, refreshedSlotIDs)
            }
        }

        if let chatSlotID = targets.chatSlotID,
           hints.chatSlotIDs.contains(chatSlotID),
           realtimeChatRefreshAllowed(activeSlot: social.realtimeTargets().slot, chatSlotID: chatSlotID) {
            do {
                let messages = try await services.api.chatMessages(chatSlotID)
                social.stageRealtimeChatSnapshot(slotID: chatSlotID, messages: messages)
            } catch let error as APIError {
                if case .unauthorized = error {
                    await services.session.clearLocalSession()
                }
                return (false, accessLost, refreshedSlotIDs)
            } catch is CancellationError {
                return (false, accessLost, refreshedSlotIDs)
            } catch {
                return (false, accessLost, refreshedSlotIDs)
            }
        }

        return (true, accessLost, refreshedSlotIDs)
    }

    @ViewBuilder private var activeScreen: some View {
        switch selection {
        case .pulse:
            PulseView(
                coordinator: social,
                cityContext: services.cityContext,
                api: services.api,
                session: services.session
            )
        case .map:
            MapView(
                api: services.api,
                session: services.session,
                social: social,
                cityContext: services.cityContext
            )
        case .fly:
            FlyView()
        case .me:
            MeView(user: user, api: services.api, session: services.session, social: social, cityContext: services.cityContext)
        }
    }
}


private extension APIError {
    var isDefinitiveSlotAccessLoss: Bool {
        switch self {
        case .http(let status, _, _, _):
            status == 403 || status == 404
        default:
            false
        }
    }
}


func realtimeChatRefreshAllowed(activeSlot: SlotModel?, chatSlotID: UUID) -> Bool {
    guard let activeSlot, activeSlot.id == chatSlotID else { return true }
    return activeSlot.canUseChat
}
