import SwiftUI

enum LinkUpVisualStatus: String { case live = "LIVE", open = "OPEN", full = "FULL", approval = "APPROVAL" }

struct LinkUpStatusBadge: View {
    let status: LinkUpVisualStatus

    var body: some View {
        Text(status.rawValue)
            .font(LinkUpTypography.mono(10, weight: .semibold))
            .foregroundStyle(tint)
            .padding(.horizontal, 8).padding(.vertical, 3)
            .background(tint.opacity(0.15))
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.badge))
            .overlay {
                RoundedRectangle(cornerRadius: LinkUpRadius.badge)
                    .stroke(tint.opacity(0.3), lineWidth: 1)
            }
    }

    private var tint: Color {
        switch status {
        case .live: LinkUpPalette.success
        case .open: LinkUpPalette.info
        case .full: LinkUpPalette.textMuted
        case .approval: LinkUpPalette.warning
        }
    }
}
