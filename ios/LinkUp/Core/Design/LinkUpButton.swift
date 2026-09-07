import SwiftUI

enum LinkUpButtonVariant { case primary, secondary, ghost, danger, success }

struct LinkUpButton: View {
    let title: String
    var variant: LinkUpButtonVariant = .primary
    var disabled = false
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(LinkUpTypography.body(14, weight: .bold))
                .frame(maxWidth: .infinity, minHeight: 48)
                .foregroundStyle(foreground)
                .background(background)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card, style: .continuous))
                .overlay { border }
        }
        .buttonStyle(.plain)
        .disabled(disabled)
        .opacity(disabled ? 0.4 : 1)
    }

    private var foreground: Color {
        variant == .danger ? LinkUpPalette.critical : LinkUpPalette.textPrimary
    }

    private var background: Color {
        switch variant {
        case .primary: LinkUpPalette.red
        case .success: LinkUpPalette.success
        case .secondary: LinkUpPalette.elevated
        case .ghost: .clear
        case .danger: LinkUpPalette.critical.opacity(0.12)
        }
    }

    @ViewBuilder private var border: some View {
        if variant != .primary && variant != .success {
            RoundedRectangle(cornerRadius: LinkUpRadius.card)
                .stroke(LinkUpPalette.border, lineWidth: 1)
        }
    }
}
