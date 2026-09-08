import SwiftUI

@MainActor
struct EditSlotView: View {
    @Environment(\.dismiss) private var dismiss
    @Binding var slot: SlotModel
    @ObservedObject var social: SocialCoordinator
    @ObservedObject var cityContext: CityContextCoordinator

    @State private var title: String
    @State private var details: String
    @State private var place: String
    @State private var selectedPlace: PlaceModel?
    @State private var capacity: Int
    @State private var scheduleEnabled: Bool
    @State private var scheduledAt: Date
    @State private var errorMessage: String?
    @State private var pendingSaveKey: UUID?

    private let originalPlace: String
    private let originalCanonicalPlaceID: UUID?
    private let originalStartAt: Date?

    init(slot: Binding<SlotModel>, social: SocialCoordinator, cityContext: CityContextCoordinator) {
        _slot = slot
        _social = ObservedObject(wrappedValue: social)
        _cityContext = ObservedObject(wrappedValue: cityContext)
        let value = slot.wrappedValue
        _title = State(initialValue: value.title)
        _details = State(initialValue: value.details ?? "")
        _place = State(initialValue: value.placeText)
        _capacity = State(initialValue: value.capacity)
        _scheduleEnabled = State(initialValue: value.startAt != nil)
        _scheduledAt = State(initialValue: value.startAt ?? Date().addingTimeInterval(3600))
        originalPlace = value.placeText
        originalCanonicalPlaceID = value.canonicalPlaceId
        originalStartAt = value.startAt
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    field("Title", text: $title, multiline: false)
                    field("Description", text: $details, multiline: true)
                    PlaceSearchField(coordinator: social, text: $place, selectedPlace: $selectedPlace)
                    capacityControl
                    scheduleControl
                    if let errorMessage { errorBanner(errorMessage) }
                }
                .padding(20)
            }
            .background(LinkUpPalette.background)
            .navigationTitle("Edit LINK")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") { dismiss() }
                        .foregroundStyle(LinkUpPalette.textDimmed)
                        .disabled(social.isMutating)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button(social.isMutating ? "Saving…" : (pendingSaveKey == nil ? "Save" : "Queued")) { save() }
                        .foregroundStyle(canSave ? LinkUpPalette.red : LinkUpPalette.textMuted)
                        .disabled(!canSave || social.mutationControlsDisabled)
                }
            }
        }
        .preferredColorScheme(.dark)
        .interactiveDismissDisabled(social.isMutating)
        .onChange(of: social.lastDurableReplayReport) { _, report in
            handleDurableReplay(report)
        }
    }

    private var canSave: Bool {
        InputContracts.validSlotTitle(title) &&
        InputContracts.validSlotDetails(details) &&
        InputContracts.validSlotPlace(place) &&
        capacity >= max(InputContracts.slotCapacityMin, slot.acceptedCount) &&
        capacity <= InputContracts.slotCapacityMax &&
        scheduleSelectionValid
    }

    private var capacityControl: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("Capacity")
                    .font(LinkUpTypography.body(12, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.textDimmed)
                Spacer()
                Text(L10n.format("fmt.minimum_capacity", max(InputContracts.slotCapacityMin, slot.acceptedCount)))
                    .font(LinkUpTypography.mono(9))
                    .foregroundStyle(LinkUpPalette.textMuted)
            }
            HStack(spacing: 16) {
                capacityButton("minus") { capacity = max(max(InputContracts.slotCapacityMin, slot.acceptedCount), capacity - 1) }
                Text("\(capacity)")
                    .font(LinkUpTypography.mono(28, weight: .bold))
                    .frame(maxWidth: .infinity)
                capacityButton("plus") { capacity = min(InputContracts.slotCapacityMax, capacity + 1) }
            }
        }
    }

    private var scheduleControl: some View {
        VStack(alignment: .leading, spacing: 10) {
            Toggle("Scheduled start", isOn: $scheduleEnabled)
                .font(LinkUpTypography.body(12, weight: .semibold))
                .tint(LinkUpPalette.red)
                .disabled(!scheduleEnabled && cityTimeScope == nil)
            if scheduleEnabled {
                if let scope = cityTimeScope {
                    Text(L10n.format("fmt.city_time", scope.identifier))
                        .font(LinkUpTypography.mono(10))
                        .foregroundStyle(LinkUpPalette.textMuted)
                    DatePicker(
                        "Start time",
                        selection: $scheduledAt,
                        displayedComponents: [.date, .hourAndMinute]
                    )
                    .environment(\.timeZone, scope.timeZone)
                    .font(LinkUpTypography.body(13))
                    .tint(LinkUpPalette.red)
                } else {
                    Text("City-Lock timezone is unavailable. The existing start instant will be preserved unless scheduling is turned off.")
                        .font(LinkUpTypography.body(10))
                        .foregroundStyle(LinkUpPalette.warning)
                }
            }
        }
        .foregroundStyle(LinkUpPalette.textDimmed)
        .padding(12)
        .background(LinkUpPalette.elevated)
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
    }

    private var cityTimeScope: CityTimeScope? {
        guard let context = cityContext.context, context.isFresh() else { return nil }
        return context.timeScope
    }

    private var scheduleSelectionValid: Bool {
        guard scheduleEnabled else { return true }
        if cityTimeScope != nil { return true }
        return originalStartAt != nil && scheduledAt == originalStartAt
    }

    private func field(_ label: String, text: Binding<String>, multiline: Bool) -> some View {
        let limit = label == "Title" ? InputContracts.slotTitleMaxScalars : InputContracts.slotDetailsMaxScalars
        let count = InputContracts.scalarCount(text.wrappedValue)
        return VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(L10n.text(label))
                    .font(LinkUpTypography.body(12, weight: .semibold))
                Spacer()
                Text("\(count)/\(limit)")
                    .font(LinkUpTypography.mono(9))
                    .foregroundStyle(count > limit ? LinkUpPalette.critical : LinkUpPalette.textMuted)
            }
            .foregroundStyle(LinkUpPalette.textDimmed)
            TextField(L10n.text(label), text: text, axis: multiline ? .vertical : .horizontal)
                .lineLimit(multiline ? 3...6 : 1...1)
                .font(LinkUpTypography.body(14))
                .padding(12)
                .background(LinkUpPalette.elevated)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
        }
    }

    private func capacityButton(_ symbol: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: symbol)
                .font(.system(size: 18, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .frame(width: 46, height: 46)
                .background(LinkUpPalette.elevated)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
        }
        .buttonStyle(.plain)
    }

    private func save() {
        guard canSave, !social.isMutating else { return }
        errorMessage = nil
        let normalizedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedDetails = details.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedPlace = place.trimmingCharacters(in: .whitespacesAndNewlines)
        let placeChanged = normalizedPlace != originalPlace
        let clearCanonicalPlace = selectedPlace == nil && originalCanonicalPlaceID != nil && placeChanged
        let clearStartAt = !scheduleEnabled && originalStartAt != nil

        let body = EditSlotBody(
            expectedVersion: slot.version,
            title: normalizedTitle,
            details: normalizedDetails,
            placeText: normalizedPlace,
            zoneText: nil,
            canonicalPlaceId: selectedPlace?.id,
            clearCanonicalPlaceId: clearCanonicalPlace,
            startAt: scheduleEnabled ? scheduledAt : nil,
            clearStartAt: clearStartAt,
            capacity: capacity
        )

        Task {
            do {
                if let updated = try await social.edit(slot, body: body), updated.id == slot.id {
                    slot = updated
                    dismiss()
                }
            } catch is CancellationError {
                return
            } catch let error as APIError {
                if case .mutationQueued(let key) = error { pendingSaveKey = key }
                errorMessage = error.localizedDescription
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    private func handleDurableReplay(_ report: DurableMutationReplayReport) {
        guard let pendingSaveKey else { return }
        if report.acknowledgedKeys.contains(pendingSaveKey) {
            self.pendingSaveKey = nil
            errorMessage = nil
            dismiss()
        } else if report.definitiveFailureKey == pendingSaveKey {
            self.pendingSaveKey = nil
            errorMessage = L10n.text("Queued edit was rejected by the server after reconciliation.")
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
}
