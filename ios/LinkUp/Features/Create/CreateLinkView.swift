import SwiftUI

private struct ActivityOption: Identifiable, Equatable {
    let id: String
    let symbol: String
    let label: String
}

@MainActor
struct CreateLinkView: View {
    @ObservedObject var coordinator: SocialCoordinator
    @ObservedObject var cityContext: CityContextCoordinator
    let close: () -> Void

    @State private var step = 1
    @State private var activity: ActivityOption?
    @State private var title = ""
    @State private var details = ""
    @State private var place = ""
    @State private var selectedPlace: PlaceModel?
    @State private var capacity = 6
    @State private var scheduleEnabled = false
    @State private var scheduledAt = Date().addingTimeInterval(3600)
    @State private var publishError: String?
    @State private var recoveryWorkflow: DraftPublishWorkflow?
    @State private var recoverySlot: SlotModel?
    @State private var recoveryError: String?
    @State private var recoveryBusy = false
    @State private var showingDiscardConfirmation = false
    @State private var recoveryChecked = false
    @State private var draftWorkflowRevisionAtOpen: UInt64

    init(coordinator: SocialCoordinator, cityContext: CityContextCoordinator, close: @escaping () -> Void) {
        _coordinator = ObservedObject(wrappedValue: coordinator)
        _cityContext = ObservedObject(wrappedValue: cityContext)
        _draftWorkflowRevisionAtOpen = State(initialValue: coordinator.draftWorkflowRevision)
        self.close = close
    }

    private let columns = Array(repeating: GridItem(.flexible(), spacing: 8), count: 4)

    private var canContinue: Bool {
        if step == 1 { return activity != nil }
        if step == 2 {
            return InputContracts.validSlotTitle(title) &&
                InputContracts.validSlotDetails(details) &&
                InputContracts.validSlotPlace(place)
        }
        return false
    }

    private var activities: [ActivityOption] {
        [
            .init(id: "coffee", symbol: "cup.and.saucer.fill", label: "Coffee"),
            .init(id: "running", symbol: "figure.run", label: "Running"),
            .init(id: "walk", symbol: "figure.walk", label: "Walk"),
            .init(id: "food", symbol: "fork.knife", label: "Food"),
            .init(id: "drinks", symbol: "wineglass", label: "Drinks"),
            .init(id: "sport", symbol: "basketball.fill", label: "Sport"),
            .init(id: "yoga", symbol: "figure.mind.and.body", label: "Yoga"),
            .init(id: "cycling", symbol: "bicycle", label: "Cycling"),
            .init(id: "photography", symbol: "camera.fill", label: "Photography"),
            .init(id: "music", symbol: "music.note", label: "Music"),
            .init(id: "art", symbol: "paintpalette.fill", label: "Art"),
            .init(id: "games", symbol: "gamecontroller.fill", label: "Games"),
            .init(id: "chess", symbol: "checkerboard.rectangle", label: "Chess"),
            .init(id: "cowork", symbol: "laptopcomputer", label: "Co-work"),
            .init(id: "networking", symbol: "person.2.fill", label: "Networking"),
            .init(id: "study", symbol: "book.fill", label: "Study")
        ]
    }

    var body: some View {
        VStack(spacing: 0) {
            header
            if recoveryChecked && recoveryWorkflow == nil { progress }
            ScrollView {
                Group {
                    if !recoveryChecked { recoveryCheckingContent }
                    else if let workflow = recoveryWorkflow { recoveryContent(workflow) }
                    else { stepContent }
                }
                .padding(.horizontal, 20)
                .padding(.vertical, 16)
            }
            .scrollIndicators(.hidden)
            footer
        }
        .background(LinkUpPalette.background.ignoresSafeArea())
        .foregroundStyle(LinkUpPalette.textPrimary)
        .interactiveDismissDisabled(coordinator.isMutating || recoveryBusy)
        .task {
            await refreshRecovery()
            recoveryChecked = true
            if recoveryWorkflow == nil, coordinator.draftWorkflowRevision != draftWorkflowRevisionAtOpen {
                close()
            }
        }
        .onChange(of: coordinator.lastDurableReplayReport) { _, _ in
            Task { await refreshRecovery() }
        }
        .onChange(of: coordinator.draftWorkflowRevision) { _, _ in
            Task { await refreshRecovery(closeWhenResolved: true) }
        }
        .confirmationDialog(
            "Discard saved draft?",
            isPresented: $showingDiscardConfirmation,
            titleVisibility: .visible
        ) {
            Button("Discard draft", role: .destructive) { discardRecovery() }
            Button("Keep draft", role: .cancel) { }
        } message: {
            Text("If a server draft was confirmed, LinkUp will cancel it server-side before removing local recovery state.")
        }
    }

    private var header: some View {
        HStack {
            Button(action: close) {
                Image(systemName: "xmark")
                    .frame(width: 44, height: 44)
                    .background(LinkUpPalette.elevated)
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                    .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
            }
            .foregroundStyle(LinkUpPalette.textDimmed)
            .accessibilityLabel(L10n.text("Close Create LINK"))
            .disabled(coordinator.isMutating || recoveryBusy)
            Spacer()
            VStack(spacing: 2) {
                Text("Create LINK").font(LinkUpTypography.display(14))
                Text(!recoveryChecked ? L10n.text("Checking saved draft") : (recoveryWorkflow == nil ? L10n.format("fmt.step_of_3", step) : L10n.text("Saved draft recovery")))
                    .font(LinkUpTypography.mono(10))
                    .foregroundStyle(LinkUpPalette.textMuted)
            }
            Spacer()
            Color.clear.frame(width: 44, height: 44)
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
        .overlay(alignment: .bottom) { Rectangle().fill(LinkUpPalette.border).frame(height: 1) }
    }

    private var progress: some View {
        HStack(spacing: 6) {
            ForEach(1...3, id: \.self) { item in
                Capsule()
                    .fill(item <= step ? LinkUpPalette.red : LinkUpPalette.zone)
                    .frame(height: 4)
            }
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 8)
    }

    @ViewBuilder private var stepContent: some View {
        if step == 1 { firstStep }
        else if step == 2 { secondStep }
        else { previewStep }
    }

    private var firstStep: some View {
        VStack(alignment: .leading, spacing: 16) {
            heading("What's happening?", "Pick an activity to get started.")
            LazyVGrid(columns: columns, spacing: 8) {
                ForEach(activities) { item in activityButton(item) }
            }
            if activity != nil {
                field("Title", placeholder: "e.g. Morning Coffee at Green Hills", text: $title)
                field("Description", placeholder: "Tell people what to expect...", text: $details)
            }
        }
    }

    private var secondStep: some View {
        VStack(alignment: .leading, spacing: 20) {
            heading("Where & when?", "Set the details for your LinkUp.")
            PlaceSearchField(
                coordinator: coordinator,
                text: $place,
                selectedPlace: $selectedPlace
            )
            capacityControl
            scheduleControl
            optionSection(
                title: "Access level",
                rowTitle: "Approval Required",
                subtitle: "You approve each request manually",
                symbol: "checkmark.circle.fill"
            )
            optionSection(
                title: "Visibility",
                rowTitle: "Public",
                subtitle: "Visible through server-authorized discovery",
                symbol: "globe"
            )
        }
    }

    private var capacityControl: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Capacity")
                .font(LinkUpTypography.body(12, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textDimmed)
            HStack(spacing: 16) {
                capacityButton("minus") { capacity = max(InputContracts.slotCapacityMin, capacity - 1) }
                Text("\(capacity)")
                    .font(LinkUpTypography.mono(30, weight: .bold))
                    .frame(maxWidth: .infinity)
                capacityButton("plus") { capacity = min(InputContracts.slotCapacityMax, capacity + 1) }
            }
        }
    }

    private var scheduleControl: some View {
        VStack(alignment: .leading, spacing: 10) {
            Toggle(isOn: $scheduleEnabled) {
                Text("Schedule")
                    .font(LinkUpTypography.body(12, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.textDimmed)
            }
            .tint(LinkUpPalette.red)
            .disabled(!scheduleEnabled && cityTimeScope == nil)

            if let scope = cityTimeScope {
                Text(L10n.format("fmt.city_time", scope.identifier))
                    .font(LinkUpTypography.mono(10))
                    .foregroundStyle(LinkUpPalette.textMuted)
                if scheduleEnabled {
                    DatePicker(
                        "Start time",
                        selection: $scheduledAt,
                        in: Date()...,
                        displayedComponents: [.date, .hourAndMinute]
                    )
                    .environment(\.timeZone, scope.timeZone)
                    .datePickerStyle(.compact)
                    .font(LinkUpTypography.body(13))
                    .tint(LinkUpPalette.red)
                }
            } else {
                Text("Set City-Lock before scheduling a LinkUp.")
                    .font(LinkUpTypography.body(10))
                    .foregroundStyle(LinkUpPalette.warning)
            }
        }
        .padding(12)
        .background(LinkUpPalette.elevated)
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
    }

    private var previewStep: some View {
        VStack(alignment: .leading, spacing: 16) {
            heading("Preview", "Review before publishing.")
            LinkUpCard {
                VStack(alignment: .leading, spacing: 12) {
                    HStack(alignment: .top, spacing: 12) {
                        Image(systemName: activity?.symbol ?? "bolt.fill")
                            .font(.system(size: 26))
                            .frame(width: 56, height: 56)
                            .background(LinkUpPalette.zone)
                            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
                        VStack(alignment: .leading, spacing: 5) {
                            LinkUpStatusBadge(status: .approval)
                            Text(title.isEmpty ? L10n.text("Untitled LinkUp") : title)
                                .font(LinkUpTypography.display(16))
                            Text(place.isEmpty ? L10n.text("No location set") : place)
                                .font(LinkUpTypography.body(12))
                                .foregroundStyle(LinkUpPalette.textDimmed)
                        }
                    }
                    if let selectedPlace {
                        HStack(spacing: 6) {
                            Image(systemName: "checkmark.seal.fill")
                            Text(L10n.format("fmt.canonical_place", selectedPlace.subtitle.isEmpty ? selectedPlace.name : selectedPlace.subtitle))
                        }
                        .font(LinkUpTypography.body(10, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.success)
                    }
                    if !details.isEmpty {
                        Text(details).font(LinkUpTypography.body(14)).foregroundStyle(LinkUpPalette.textDimmed)
                    }
                    if scheduleEnabled, let scope = cityTimeScope {
                        Text(scope.displayString(for: scheduledAt))
                            .font(LinkUpTypography.mono(11))
                            .foregroundStyle(LinkUpPalette.textDimmed)
                    }
                    Text(L10n.format("fmt.going_approval_public", capacity))
                        .font(LinkUpTypography.mono(11))
                        .foregroundStyle(LinkUpPalette.textMuted)
                }
            }
            if let visiblePublishError {
                HStack(alignment: .top, spacing: 9) {
                    Image(systemName: "exclamationmark.triangle.fill").foregroundStyle(LinkUpPalette.critical)
                    Text(L10n.text(visiblePublishError))
                        .font(LinkUpTypography.body(12))
                        .foregroundStyle(LinkUpPalette.textDimmed)
                }
                .padding(12)
                .background(LinkUpPalette.critical.opacity(0.09))
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            }
        }
    }

    @ViewBuilder private var footer: some View {
        if !recoveryChecked {
            LinkUpButton(title: "Checking saved draft…", variant: .secondary, disabled: true) { }
                .padding(.horizontal, 20)
                .padding(.top, 12)
                .padding(.bottom, 8)
        } else if let workflow = recoveryWorkflow {
            HStack(spacing: 8) {
                if workflow.requiresAttention {
                    if workflow.draftID != nil {
                        LinkUpButton(
                            title: recoveryBusy ? "Working…" : recoveryRetryTitle,
                            disabled: recoveryBusy || coordinator.mutationControlsDisabled
                        ) { retryRecovery() }
                    }
                    if recoveryCanDiscard {
                        LinkUpButton(
                            title: "Discard draft",
                            variant: .danger,
                            disabled: recoveryBusy || coordinator.mutationControlsDisabled
                        ) { showingDiscardConfirmation = true }
                    }
                    if workflow.draftID == nil {
                        LinkUpButton(title: "Close", variant: .secondary, disabled: recoveryBusy) { close() }
                    }
                } else {
                    LinkUpButton(title: "Publishing queued", variant: .secondary, disabled: true) { }
                    LinkUpButton(title: "Close", variant: .secondary, disabled: recoveryBusy) { close() }
                }
            }
            .padding(.horizontal, 20)
            .padding(.top, 12)
            .padding(.bottom, 8)
            .overlay(alignment: .top) { Rectangle().fill(LinkUpPalette.border).frame(height: 1) }
        } else {
            HStack(spacing: 8) {
                if step > 1 {
                    LinkUpButton(title: "Back", variant: .secondary, disabled: coordinator.isMutating) { step -= 1 }
                }
                if step < 3 {
                    LinkUpButton(title: "Continue", disabled: !canContinue || coordinator.isMutating) { step += 1 }
                } else {
                    LinkUpButton(
                        title: coordinator.isMutating ? "Publishing…" : "Publish LinkUp",
                        disabled: coordinator.mutationControlsDisabled
                    ) {
                        publish()
                    }
                }
            }
            .padding(.horizontal, 20)
            .padding(.top, 12)
            .padding(.bottom, 8)
            .overlay(alignment: .top) { Rectangle().fill(LinkUpPalette.border).frame(height: 1) }
        }
    }

    private func publish() {
        guard let activity else { return }
        if scheduleEnabled && cityTimeScope == nil {
            publishError = L10n.text("City-Lock timezone is required for scheduled LinkUps.")
            return
        }
        publishError = nil
        let normalizedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedDetails = details.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedPlace = place.trimmingCharacters(in: .whitespacesAndNewlines)
        let body = CreateSlotBody(
            title: normalizedTitle,
            activity: activity.id,
            details: normalizedDetails.isEmpty ? nil : normalizedDetails,
            placeText: normalizedPlace,
            zoneText: nil,
            canonicalPlaceId: selectedPlace?.id,
            startAt: scheduleEnabled ? scheduledAt : nil,
            capacity: capacity
        )

        Task {
            do {
                if try await coordinator.createDraftAndPublish(body) != nil { close() }
            } catch is CancellationError {
                return
            } catch let error as APIError {
                publishError = error.localizedDescription
                await refreshRecovery()
            } catch {
                publishError = error.localizedDescription
                await refreshRecovery()
            }
        }
    }

    private var recoveryCheckingContent: some View {
        VStack(spacing: 12) {
            ProgressView().tint(LinkUpPalette.red)
            Text("Checking protected draft recovery…")
                .font(LinkUpTypography.body(12))
                .foregroundStyle(LinkUpPalette.textDimmed)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 64)
    }

    private var recoveryCanDiscard: Bool {
        guard let workflow = recoveryWorkflow else { return false }
        guard workflow.draftID != nil else { return false }
        return recoverySlot == nil || recoverySlot?.state == .draft
    }

    private var recoveryServerAdvanced: Bool {
        guard let state = recoverySlot?.state else { return false }
        return state != .draft
    }

    private var recoveryRetryTitle: String {
        if let state = recoverySlot?.state, state != .draft { return "Resolve server state" }
        return "Retry publish"
    }

    private func recoveryContent(_ workflow: DraftPublishWorkflow) -> some View {
        VStack(alignment: .leading, spacing: 16) {
            heading(
                recoveryServerAdvanced
                    ? "Server state confirmed"
                    : (workflow.requiresAttention ? "Draft needs attention" : "Publishing safely"),
                recoveryServerAdvanced
                    ? "The server already moved this LinkUp past DRAFT. Resolve the stale local recovery record against that authoritative state."
                    : (workflow.requiresAttention
                       ? "The draft stays non-discoverable until the server confirms publish."
                       : "LinkUp will resume the exact create/publish workflow without creating a duplicate.")
            )

            LinkUpCard {
                VStack(alignment: .leading, spacing: 10) {
                    Text(recoverySlot?.title ?? recoveryBody?.title ?? L10n.text("Saved LinkUp"))
                        .font(LinkUpTypography.display(16))
                    Text(recoverySlot?.placeText ?? recoveryBody?.placeText ?? L10n.text("Location unavailable"))
                        .font(LinkUpTypography.body(12))
                        .foregroundStyle(LinkUpPalette.textDimmed)
                    HStack(spacing: 8) {
                        Text(recoverySlot?.state.rawValue ?? "RECOVERY")
                        if let version = recoverySlot?.version ?? workflow.draftVersion {
                            Text(L10n.format("fmt.version", version))
                        }
                    }
                    .font(LinkUpTypography.mono(10, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.textMuted)
                    if let startAt = recoverySlot?.startAt ?? recoveryBody?.startAt {
                        Text(recoveryTimeLabel(startAt))
                            .font(LinkUpTypography.mono(10))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }
                }
            }

            if workflow.requiresAttention {
                Text(L10n.text(recoveryServerAdvanced
                     ? "Resolve server state clears only the stale local workflow record. It does not publish or cancel the already-advanced server Slot again."
                     : (workflow.draftID == nil
                        ? "No server draft ID was confirmed before the safe replay window ended or the create key became unsafe. LinkUp will not replay create because server idempotency retention cannot be guaranteed indefinitely; local discard is also blocked without an authoritative resource ID."
                        : "Retry first re-reads the authoritative server draft and its current version. Discard cancels the server draft before local recovery state is removed.")))
                    .font(LinkUpTypography.body(12))
                    .foregroundStyle(LinkUpPalette.warning)
            } else {
                Text("It is safe to close this screen. The queued command keeps the same idempotency key across restart and reconnect.")
                    .font(LinkUpTypography.body(12))
                    .foregroundStyle(LinkUpPalette.textDimmed)
            }

            if let visiblePublishError { errorBanner(visiblePublishError) }
            if let recoveryError, recoveryError != visiblePublishError { errorBanner(recoveryError) }
        }
    }

    private var recoveryBody: CreateSlotBody? {
        guard let data = recoveryWorkflow?.createBody else { return nil }
        return try? APICoding.decoder().decode(CreateSlotBody.self, from: data)
    }

    private func recoveryTimeLabel(_ date: Date) -> String {
        if let scope = cityTimeScope { return scope.displayString(for: date) }
        let formatter = ISO8601DateFormatter()
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.formatOptions = [.withInternetDateTime]
        return "\(formatter.string(from: date)) · UTC"
    }

    private func refreshRecovery(closeWhenResolved: Bool = false) async {
        do {
            guard let workflow = try await coordinator.pendingDraftPublishWorkflow() else {
                let hadRecovery = recoveryWorkflow != nil
                recoveryWorkflow = nil
                recoverySlot = nil
                recoveryError = nil
                if closeWhenResolved && hadRecovery && !recoveryBusy { close() }
                return
            }
            recoveryWorkflow = workflow
            recoverySlot = nil
            recoveryError = nil
            if let draftID = workflow.draftID {
                do {
                    recoverySlot = try await coordinator.loadSlot(draftID)
                } catch is CancellationError {
                    return
                } catch {
                    recoveryError = L10n.text("Saved recovery is intact, but the server draft snapshot is temporarily unavailable.")
                }
            }
        } catch is CancellationError {
            return
        } catch {
            recoveryError = error.localizedDescription
        }
    }

    private func retryRecovery() {
        guard !recoveryBusy else { return }
        recoveryBusy = true
        recoveryError = nil
        publishError = nil
        Task {
            defer { recoveryBusy = false }
            do {
                if try await coordinator.retryDraftPublishWorkflow() != nil { close() }
            } catch is CancellationError {
                return
            } catch {
                let message = error.localizedDescription
                await refreshRecovery()
                recoveryError = message
            }
        }
    }

    private func discardRecovery() {
        guard !recoveryBusy else { return }
        recoveryBusy = true
        recoveryError = nil
        publishError = nil
        Task {
            defer { recoveryBusy = false }
            do {
                if try await coordinator.discardDraftPublishWorkflow() == true { close() }
            } catch is CancellationError {
                return
            } catch {
                let message = error.localizedDescription
                await refreshRecovery()
                recoveryError = message
            }
        }
    }

    private func errorBanner(_ message: String) -> some View {
        HStack(alignment: .top, spacing: 9) {
            Image(systemName: "exclamationmark.triangle.fill")
                .foregroundStyle(LinkUpPalette.critical)
            Text(L10n.text(message))
                .font(LinkUpTypography.body(12))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(12)
        .background(LinkUpPalette.critical.opacity(0.09))
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
    }

    private var cityTimeScope: CityTimeScope? {
        guard let context = cityContext.context, context.isFresh() else { return nil }
        return context.timeScope
    }

    private var visiblePublishError: String? {
        publishError ?? (coordinator.durableMutationBlocked ? coordinator.mutationError : nil)
    }


    private func heading(_ title: String, _ subtitle: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(L10n.text(title)).font(LinkUpTypography.display(20))
            Text(L10n.text(subtitle)).font(LinkUpTypography.body(14)).foregroundStyle(LinkUpPalette.textDimmed)
        }
    }

    private func field(_ label: String, placeholder: String, text: Binding<String>) -> some View {
        let limit = label == "Title" ? InputContracts.slotTitleMaxScalars : InputContracts.slotDetailsMaxScalars
        let count = InputContracts.scalarCount(text.wrappedValue)
        return VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(L10n.text(label)).font(LinkUpTypography.body(12, weight: .semibold))
                Spacer()
                Text("\(count)/\(limit)")
                    .font(LinkUpTypography.mono(9))
                    .foregroundStyle(count > limit ? LinkUpPalette.critical : LinkUpPalette.textMuted)
            }
            .foregroundStyle(LinkUpPalette.textDimmed)
            TextField(L10n.text(placeholder), text: text, axis: label == "Description" ? .vertical : .horizontal)
                .lineLimit(label == "Description" ? 3...5 : 1...1)
                .font(LinkUpTypography.body(14))
                .padding(12)
                .background(LinkUpPalette.elevated)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
        }
    }

    private func activityButton(_ item: ActivityOption) -> some View {
        Button { activity = item } label: {
            VStack(spacing: 6) {
                Image(systemName: item.symbol).font(.system(size: 22))
                Text(L10n.text(item.label)).font(LinkUpTypography.body(9, weight: .semibold)).lineLimit(1)
            }
            .foregroundStyle(activity == item ? LinkUpPalette.red : LinkUpPalette.textDimmed)
            .frame(maxWidth: .infinity).aspectRatio(1, contentMode: .fit)
            .background(activity == item ? LinkUpPalette.red.opacity(0.15) : LinkUpPalette.elevated)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(activity == item ? LinkUpPalette.red : LinkUpPalette.border) }
        }
        .buttonStyle(.plain)
    }

    private func capacityButton(_ symbol: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: symbol)
                .font(.system(size: 20, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .frame(width: 48, height: 48)
                .background(LinkUpPalette.elevated)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L10n.text(symbol == "minus" ? "Decrease capacity" : "Increase capacity"))
    }

    private func optionSection(title: String, rowTitle: String, subtitle: String, symbol: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L10n.text(title)).font(LinkUpTypography.body(12, weight: .semibold)).foregroundStyle(LinkUpPalette.textDimmed)
            HStack(spacing: 12) {
                Image(systemName: symbol)
                    .frame(width: 40, height: 40)
                    .foregroundStyle(LinkUpPalette.red)
                    .background(LinkUpPalette.red.opacity(0.15))
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                VStack(alignment: .leading, spacing: 2) {
                    Text(L10n.text(rowTitle)).font(LinkUpTypography.body(14, weight: .semibold))
                    Text(L10n.text(subtitle)).font(LinkUpTypography.body(12)).foregroundStyle(LinkUpPalette.textMuted)
                }
                Spacer()
                Image(systemName: "checkmark").foregroundStyle(LinkUpPalette.red)
            }
            .padding(12)
            .background(LinkUpPalette.red.opacity(0.08))
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.red.opacity(0.5)) }
        }
    }
}
