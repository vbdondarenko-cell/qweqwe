import SwiftUI

struct LinkUpChip: View {
    let title: String
    let active: Bool
    var systemImage: String? = nil
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 6) {
                if let systemImage {
                    Image(systemName: systemImage).font(.system(size: 14))
                }
                Text(title).font(LinkUpTypography.body(12, weight: .semibold))
            }
            .foregroundStyle(active ? LinkUpPalette.textPrimary : LinkUpPalette.textDimmed)
            .padding(.horizontal, 14).padding(.vertical, 8)
            .background(active ? LinkUpPalette.red : LinkUpPalette.elevated)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control, style: .continuous))
            .overlay {
                RoundedRectangle(cornerRadius: LinkUpRadius.control)
                    .stroke(active ? LinkUpPalette.red : LinkUpPalette.border, lineWidth: 1)
            }
        }
        .buttonStyle(.plain)
    }
}
