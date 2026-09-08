import Foundation
import SwiftUI

struct SlotCardView: View {
    let slot: SlotModel
    let cityTimeScope: CityTimeScope?
    let isMutating: Bool
    let open: () -> Void
    let primaryAction: () -> Void

    var body: some View {
        LinkUpCard {
            VStack(alignment: .leading, spacing: 12) {
                HStack(alignment: .top, spacing: 12) {
                    Image(systemName: symbol)
                        .font(.system(size: 22, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.textPrimary)
                        .frame(width: 48, height: 48)
                        .background(LinkUpPalette.zone)
                        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                    VStack(alignment: .leading, spacing: 5) {
                        HStack(spacing: 8) {
                            LinkUpStatusBadge(status: visualStatus)
                            if let startAt = slot.startAt {
                                Text(startTimeLabel(startAt))
                                    .font(LinkUpTypography.mono(10))
                                    .foregroundStyle(LinkUpPalette.textMuted)
                            }
                        }
                        Text(slot.title)
                            .font(LinkUpTypography.display(16))
                            .foregroundStyle(LinkUpPalette.textPrimary)
                            .lineLimit(2)
                        Label(slot.placeText, systemImage: "mappin")
                            .font(LinkUpTypography.body(12))
                            .foregroundStyle(LinkUpPalette.textDimmed)
                            .lineLimit(1)
                    }
                    Spacer(minLength: 0)
                }

                if let details = slot.details, !details.isEmpty {
                    Text(details)
                        .font(LinkUpTypography.body(13))
                        .foregroundStyle(LinkUpPalette.textDimmed)
                        .lineLimit(2)
                }

                HStack(spacing: 8) {
                    LinkUpAvatar(initials: initials, size: .sm)
                    VStack(alignment: .leading, spacing: 1) {
                        Text(slot.organizer.displayName)
                            .font(LinkUpTypography.body(12, weight: .semibold))
                            .foregroundStyle(LinkUpPalette.textPrimary)
                        Text("@\(slot.organizer.username)")
                            .font(LinkUpTypography.body(10))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }
                    Spacer()
                    Text(slot.viewerState.rawValue)
                        .font(LinkUpTypography.mono(9, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.textMuted)
                }

                LinkUpProgress(value: slot.acceptedCount, maximum: slot.capacity, showsLabel: true)

                HStack(spacing: 8) {
                    Button(action: open) {
                        Text("Details")
                            .font(LinkUpTypography.body(13, weight: .semibold))
                            .frame(maxWidth: .infinity)
                            .frame(height: 42)
                            .background(LinkUpPalette.zone)
                            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                    }
                    .foregroundStyle(LinkUpPalette.textDimmed)
                    .buttonStyle(.plain)

                    Button(action: primaryAction) {
                        Text(actionTitle)
                            .font(LinkUpTypography.body(13, weight: .bold))
                            .frame(maxWidth: .infinity)
                            .frame(height: 42)
                            .background(actionTint.opacity(actionDisabled ? 0.12 : 1))
                            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                    }
                    .foregroundStyle(actionDisabled ? LinkUpPalette.textMuted : .white)
                    .buttonStyle(.plain)
                    .disabled(actionDisabled || isMutating)
                }
            }
        }
        .contentShape(Rectangle())
    }


    private func startTimeLabel(_ date: Date) -> String {
        if let cityTimeScope { return cityTimeScope.shortTimeString(for: date) }
        let formatter = ISO8601DateFormatter()
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.formatOptions = [.withInternetDateTime]
        return "\(formatter.string(from: date)) · UTC"
    }

    private var visualStatus: LinkUpVisualStatus {
        if slot.state == .active { return .live }
        if slot.state == .full { return .full }
        if slot.accessMode == .approval && slot.viewerState == .none { return .approval }
        return .open
    }

    private var actionTitle: String {
        switch slot.viewerState {
        case .host: "Manage"
        case .accepted: "Open"
        case .pending: "Pending"
        case .none:
            slot.state == .full ? "Full" : (slot.accessMode == .approval ? "Request" : "Open")
        }
    }

    private var actionDisabled: Bool {
        isMutating || slot.viewerState == .pending ||
        (slot.viewerState == .none && (slot.state == .full || slot.accessMode != .approval))
    }

    private var actionTint: Color {
        slot.viewerState == .none && slot.accessMode == .approval ? LinkUpPalette.warning : LinkUpPalette.red
    }

    private var initials: String {
        let words = slot.organizer.displayName.split(separator: " ")
        let value = words.prefix(2).compactMap(\.first).map(String.init).joined()
        return value.isEmpty ? "LU" : value.uppercased()
    }

    private var symbol: String {
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
