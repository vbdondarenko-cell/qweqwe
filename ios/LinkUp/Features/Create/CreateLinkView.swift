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
    @State private var pendingPublishKey: UUID?

    init(coordinator: SocialCoordinator, cityContext: CityContextCoordinator, close: @escaping () -> Void) {
        _coordinator = ObservedObject(wrappedValue: coordinator)
        _cityContext = ObservedObject(wrappedValue: cityContext)
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
            progress
            ScrollView {
                stepContent
                    .padding(.horizontal, 20)
                    .padding(.vertical, 16)
            }
            .scrollIndicators(.hidden)
            footer
        }
        .background(LinkUpPalette.background.ignoresSafeArea())
        .foregroundStyle(LinkUpPalette.textPrimary)
        .interactiveDismissDisabled(coordinator.isMutating)
        .onChange(of: coordinator.lastDurableReplayReport) { _, report in
            handleDurableReplay(report)
        }
    }

    private var header: some View {
        HStack {
            Button(action: close) {
                Image(systemName: "xmark")
                    .frame(width: 36, height: 36)
                    .background(LinkUpPalette.elevated)
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                    .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
            }
            .foregroundStyle(LinkUpPalette.textDimmed)
            .disabled(coordinator.isMutating)
            Spacer()
            VStack(spacing: 2) {
                Text("Create LINK").font(LinkUpTypography.display(14))
                Text("Step \(step) of 3").font(LinkUpTypography.mono(10)).foregroundStyle(LinkUpPalette.textMuted)
            }
            Spacer()
            Color.clear.frame(width: 36, height: 36)
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
                Text("City time · \(scope.identifier)")
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
                            Text(title.isEmpty ? "Untitled LinkUp" : title)
                                .font(LinkUpTypography.display(16))
                            Text(place.isEmpty ? "No location set" : place)
                                .font(LinkUpTypography.body(12))
                                .foregroundStyle(LinkUpPalette.textDimmed)
                        }
                    }
                    if let selectedPlace {
                        HStack(spacing: 6) {
                            Image(systemName: "checkmark.seal.fill")
                            Text("Canonical place · \(selectedPlace.subtitle.isEmpty ? selectedPlace.name : selectedPlace.subtitle)")
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
                    Text("0/\(capacity) going · Approval required · Public")
                        .font(LinkUpTypography.mono(11))
                        .foregroundStyle(LinkUpPalette.textMuted)
                }
            }
            if let visiblePublishError {
                HStack(alignment: .top, spacing: 9) {
                    Image(systemName: "exclamationmark.triangle.fill").foregroundStyle(LinkUpPalette.critical)
                    Text(visiblePublishError)
                        .font(LinkUpTypography.body(12))
                        .foregroundStyle(LinkUpPalette.textDimmed)
                }
                .padding(12)
                .background(LinkUpPalette.critical.opacity(0.09))
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            }
        }
    }

    private var footer: some View {
        HStack(spacing: 8) {
            if step > 1 {
                LinkUpButton(title: "Back", variant: .secondary, disabled: coordinator.isMutating) { step -= 1 }
            }
            if step < 3 {
                LinkUpButton(title: "Continue", disabled: !canContinue || coordinator.isMutating) { step += 1 }
            } else {
                LinkUpButton(
                    title: coordinator.isMutating ? "Publishing…" : (pendingPublishKey == nil ? "Publish LinkUp" : "Queued for retry"),
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

    private func publish() {
        guard let activity else { return }
        if scheduleEnabled && cityTimeScope == nil {
            publishError = "City-Lock timezone is required for scheduled LinkUps."
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
                if try await coordinator.createSlot(body) != nil { close() }
            } catch is CancellationError {
                return
            } catch let error as APIError {
                if case .mutationQueued(let key) = error { pendingPublishKey = key }
                publishError = error.localizedDescription
            } catch {
                publishError = error.localizedDescription
            }
        }
    }

    private var cityTimeScope: CityTimeScope? {
        guard let context = cityContext.context, context.isFresh() else { return nil }
        return context.timeScope
    }

    private var visiblePublishError: String? {
        publishError ?? (coordinator.durableMutationBlocked ? coordinator.mutationError : nil)
    }

    private func handleDurableReplay(_ report: DurableMutationReplayReport) {
        guard let pendingPublishKey else { return }
        if report.acknowledgedKeys.contains(pendingPublishKey) {
            self.pendingPublishKey = nil
            publishError = nil
            close()
        } else if report.definitiveFailureKey == pendingPublishKey {
            self.pendingPublishKey = nil
            publishError = "Queued publish was rejected by the server after reconciliation."
        }
    }

    private func heading(_ title: String, _ subtitle: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title).font(LinkUpTypography.display(20))
            Text(subtitle).font(LinkUpTypography.body(14)).foregroundStyle(LinkUpPalette.textDimmed)
        }
    }

    private func field(_ label: String, placeholder: String, text: Binding<String>) -> some View {
        let limit = label == "Title" ? InputContracts.slotTitleMaxScalars : InputContracts.slotDetailsMaxScalars
        let count = InputContracts.scalarCount(text.wrappedValue)
        return VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(label).font(LinkUpTypography.body(12, weight: .semibold))
                Spacer()
                Text("\(count)/\(limit)")
                    .font(LinkUpTypography.mono(9))
                    .foregroundStyle(count > limit ? LinkUpPalette.critical : LinkUpPalette.textMuted)
            }
            .foregroundStyle(LinkUpPalette.textDimmed)
            TextField(placeholder, text: text, axis: label == "Description" ? .vertical : .horizontal)
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
                Text(item.label).font(LinkUpTypography.body(9, weight: .semibold)).lineLimit(1)
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
    }

    private func optionSection(title: String, rowTitle: String, subtitle: String, symbol: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title).font(LinkUpTypography.body(12, weight: .semibold)).foregroundStyle(LinkUpPalette.textDimmed)
            HStack(spacing: 12) {
                Image(systemName: symbol)
                    .frame(width: 40, height: 40)
                    .foregroundStyle(LinkUpPalette.red)
                    .background(LinkUpPalette.red.opacity(0.15))
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                VStack(alignment: .leading, spacing: 2) {
                    Text(rowTitle).font(LinkUpTypography.body(14, weight: .semibold))
                    Text(subtitle).font(LinkUpTypography.body(12)).foregroundStyle(LinkUpPalette.textMuted)
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
