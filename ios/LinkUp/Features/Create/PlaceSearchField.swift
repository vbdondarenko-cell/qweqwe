import SwiftUI

@MainActor
struct PlaceSearchField: View {
    @ObservedObject var coordinator: SocialCoordinator
    @Binding var text: String
    @Binding var selectedPlace: PlaceModel?

    @State private var suggestions: [PlaceModel] = []
    @State private var isSearching = false
    @State private var errorMessage: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Location")
                .font(LinkUpTypography.body(12, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textDimmed)

            HStack(spacing: 10) {
                Image(systemName: "mappin.and.ellipse")
                    .foregroundStyle(selectedPlace == nil ? LinkUpPalette.textMuted : LinkUpPalette.success)
                TextField("Search a place or enter a location", text: $text)
                    .font(LinkUpTypography.body(14))
                    .textInputAutocapitalization(.words)
                    .autocorrectionDisabled()
                if isSearching {
                    ProgressView().tint(LinkUpPalette.red).controlSize(.small)
                } else if selectedPlace != nil {
                    Image(systemName: "checkmark.seal.fill").foregroundStyle(LinkUpPalette.success)
                }
            }
            .padding(.horizontal, 12)
            .frame(height: 46)
            .background(LinkUpPalette.elevated)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }

            if let selectedPlace {
                HStack(spacing: 8) {
                    Image(systemName: "checkmark.seal.fill").foregroundStyle(LinkUpPalette.success)
                    VStack(alignment: .leading, spacing: 1) {
                        Text(selectedPlace.name)
                            .font(LinkUpTypography.body(12, weight: .semibold))
                            .foregroundStyle(LinkUpPalette.textPrimary)
                        if !selectedPlace.subtitle.isEmpty {
                            Text(selectedPlace.subtitle)
                                .font(LinkUpTypography.body(10))
                                .foregroundStyle(LinkUpPalette.textMuted)
                        }
                    }
                    Spacer()
                    Button("Clear") {
                        self.selectedPlace = nil
                    }
                    .font(LinkUpTypography.body(10, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.red)
                }
                .padding(10)
                .background(LinkUpPalette.success.opacity(0.08))
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.compact))
            } else if !suggestions.isEmpty {
                VStack(spacing: 0) {
                    ForEach(suggestions) { place in
                        Button {
                            selectedPlace = place
                            text = place.name
                            suggestions = []
                            errorMessage = nil
                        } label: {
                            HStack(spacing: 10) {
                                Image(systemName: "mappin.circle.fill")
                                    .foregroundStyle(LinkUpPalette.red)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(place.name)
                                        .font(LinkUpTypography.body(13, weight: .semibold))
                                        .foregroundStyle(LinkUpPalette.textPrimary)
                                    if !place.subtitle.isEmpty {
                                        Text(place.subtitle)
                                            .font(LinkUpTypography.body(10))
                                            .foregroundStyle(LinkUpPalette.textMuted)
                                    }
                                }
                                Spacer()
                                if let category = place.category, !category.isEmpty {
                                    Text(category)
                                        .font(LinkUpTypography.mono(9))
                                        .foregroundStyle(LinkUpPalette.textMuted)
                                }
                            }
                            .padding(.horizontal, 12)
                            .frame(minHeight: 48)
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)

                        if place.id != suggestions.last?.id {
                            Rectangle().fill(LinkUpPalette.border.opacity(0.5)).frame(height: 1)
                        }
                    }
                }
                .background(LinkUpPalette.elevated)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
            }

            if let errorMessage {
                Text(errorMessage)
                    .font(LinkUpTypography.body(10))
                    .foregroundStyle(LinkUpPalette.warning)
            }

            if selectedPlace == nil {
                Text("You can publish with plain place text; selecting a result also attaches the server's canonical place identity.")
                    .font(LinkUpTypography.body(10))
                    .foregroundStyle(LinkUpPalette.textMuted)
            }
        }
        .task(id: text) {
            await searchIfNeeded()
        }
    }

    private func searchIfNeeded() async {
        let query = InputContracts.trimmed(text)

        if let selectedPlace, query == selectedPlace.name {
            suggestions = []
            errorMessage = nil
            return
        }
        selectedPlace = nil
        let scalarCount = InputContracts.scalarCount(query)
        guard scalarCount >= InputContracts.placeSearchMinScalars else {
            suggestions = []
            errorMessage = nil
            return
        }
        guard scalarCount <= InputContracts.placeSearchMaxScalars else {
            suggestions = []
            errorMessage = scalarCount > InputContracts.slotPlaceMaxScalars
                ? "Location is too long. Maximum 240 Unicode characters."
                : "Place search supports up to 80 Unicode characters. You can still use this as a manual location."
            return
        }

        do {
            try await Task.sleep(for: .milliseconds(300))
            guard !Task.isCancelled else { return }
            isSearching = true
            defer { isSearching = false }
            suggestions = try await coordinator.searchPlaces(query)
            errorMessage = nil
        } catch is CancellationError {
            return
        } catch {
            guard !Task.isCancelled else { return }
            suggestions = []
            errorMessage = "Place search is unavailable. You can still enter a location manually."
        }
    }
}
