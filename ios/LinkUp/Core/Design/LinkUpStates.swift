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
            Text(L10n.text(title))
                .font(LinkUpTypography.display(18))
                .foregroundStyle(LinkUpPalette.textPrimary)
            Text(L10n.text(message))
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
            Text(L10n.text(message))
                .font(LinkUpTypography.body(13))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .multilineTextAlignment(.center)
            LinkUpButton(title: "Retry", variant: .secondary, action: retry)
                .frame(maxWidth: 220)
        }
        .padding(.vertical, 32)
    }
}

struct LinkUpInlineError: View {
    let message: String
    var dismiss: (() -> Void)? = nil

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            Image(systemName: "exclamationmark.triangle.fill")
                .foregroundStyle(LinkUpPalette.critical)
            Text(L10n.text(message))
                .font(LinkUpTypography.body(12))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .frame(maxWidth: .infinity, alignment: .leading)
            if let dismiss {
                Button(action: dismiss) {
                    Image(systemName: "xmark")
                        .font(.system(size: 11, weight: .bold))
                        .frame(width: 24, height: 24)
                }
                .buttonStyle(.plain)
                .foregroundStyle(LinkUpPalette.textMuted)
                .accessibilityLabel("Dismiss error")
            }
        }
        .padding(12)
        .background(LinkUpPalette.critical.opacity(0.09))
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.critical.opacity(0.18)) }
    }
}
