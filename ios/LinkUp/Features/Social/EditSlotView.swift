import SwiftUI

@MainActor
struct EditSlotView: View {
    @Environment(\.dismiss) private var dismiss
    @Binding var slot: SlotModel
    @ObservedObject var social: SocialCoordinator

    @State private var title: String
    @State private var details: String
    @State private var place: String
    @State private var selectedPlace: PlaceModel?
    @State private var capacity: Int
    @State private var scheduleEnabled: Bool
    @State private var scheduledAt: Date
    @State private var errorMessage: String?

    private let originalPlace: String
    private let originalCanonicalPlaceID: UUID?
    private let originalStartAt: Date?

    init(slot: Binding<SlotModel>, social: SocialCoordinator) {
        _slot = slot
        self.social = social
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
                    Button(social.isMutating ? "Saving…" : "Save") { save() }
                        .foregroundStyle(canSave ? LinkUpPalette.red : LinkUpPalette.textMuted)
                        .disabled(!canSave || social.isMutating)
                }
            }
        }
        .preferredColorScheme(.dark)
        .interactiveDismissDisabled(social.isMutating)
    }

    private var canSave: Bool {
        !title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
        !place.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
        capacity >= max(2, slot.acceptedCount)
    }

    private var capacityControl: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("Capacity")
                    .font(LinkUpTypography.body(12, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.textDimmed)
                Spacer()
                Text("Minimum \(max(2, slot.acceptedCount))")
                    .font(LinkUpTypography.mono(9))
                    .foregroundStyle(LinkUpPalette.textMuted)
            }
            HStack(spacing: 16) {
                capacityButton("minus") { capacity = max(max(2, slot.acceptedCount), capacity - 1) }
                Text("\(capacity)")
                    .font(LinkUpTypography.mono(28, weight: .bold))
                    .frame(maxWidth: .infinity)
                capacityButton("plus") { capacity = min(50, capacity + 1) }
            }
        }
    }

    private var scheduleControl: some View {
        VStack(alignment: .leading, spacing: 10) {
            Toggle("Scheduled start", isOn: $scheduleEnabled)
                .font(LinkUpTypography.body(12, weight: .semibold))
                .tint(LinkUpPalette.red)
            if scheduleEnabled {
                DatePicker(
                    "Start time",
                    selection: $scheduledAt,
                    displayedComponents: [.date, .hourAndMinute]
                )
                .font(LinkUpTypography.body(13))
                .tint(LinkUpPalette.red)
            }
        }
        .foregroundStyle(LinkUpPalette.textDimmed)
        .padding(12)
        .background(LinkUpPalette.elevated)
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
    }

    private func field(_ label: String, text: Binding<String>, multiline: Bool) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(label)
                .font(LinkUpTypography.body(12, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textDimmed)
            TextField(label, text: text, axis: multiline ? .vertical : .horizontal)
                .lineLimit(multiline ? 3...6 : 1...1)
                .font(LinkUpTypography.body(14))
                .padding(12)
                .background(LinkUpPalette.elevated)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
                .onChange(of: text.wrappedValue) { _, newValue in
                    if label == "Title" && newValue.count > 60 {
                        text.wrappedValue = String(newValue.prefix(60))
                    }
                }
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
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    private func errorBanner(_ message: String) -> some View {
        HStack(alignment: .top, spacing: 9) {
            Image(systemName: "exclamationmark.triangle.fill")
                .foregroundStyle(LinkUpPalette.critical)
            Text(message)
                .font(LinkUpTypography.body(12))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(12)
        .background(LinkUpPalette.critical.opacity(0.09))
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
    }
}
