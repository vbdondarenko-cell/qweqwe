import SwiftUI

struct LinkUpEmptyState: View {
    let title: String
    let message: String
    var systemImage = "bolt"

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: systemImage)
                .font(.system(size: 28))
                .foregroundStyle(LinkUpPalette.textMuted)
                .frame(width: 80, height: 80)
                .background(LinkUpPalette.zone)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.sheet))
            Text(title)
                .font(LinkUpTypography.display(18))
                .foregroundStyle(LinkUpPalette.textPrimary)
            Text(message)
                .font(LinkUpTypography.body(13))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 260)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 32)
    }
}

struct LinkUpErrorState: View {
    let message: String
    let retry: () -> Void

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle")
                .font(.system(size: 28))
                .foregroundStyle(LinkUpPalette.critical)
                .frame(width: 64, height: 64)
                .background(LinkUpPalette.critical.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
            Text("Something went wrong")
                .font(LinkUpTypography.display(18))
                .foregroundStyle(LinkUpPalette.textPrimary)
            Text(message)
                .font(LinkUpTypography.body(13))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .multilineTextAlignment(.center)
            LinkUpButton(title: "Retry", variant: .secondary, action: retry)
                .frame(maxWidth: 220)
        }
        .padding(.vertical, 32)
    }
}
