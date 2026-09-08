import SwiftUI

struct FrozenBottomBar: View {
    @Binding var selection: AppTab
    let create: () -> Void

    var body: some View {
        HStack(alignment: .bottom, spacing: 0) {
            tab(.pulse)
            tab(.map)
            createButton
            tab(.fly)
            tab(.me)
        }
        .padding(.horizontal, 16)
        .padding(.top, 8)
        .padding(.bottom, 8)
        .linkUpGlass()
        .overlay(alignment: .top) {
            Rectangle().fill(LinkUpPalette.border).frame(height: 1)
        }
    }

    private func tab(_ tab: AppTab) -> some View {
        Button { selection = tab } label: {
            VStack(spacing: 4) {
                Image(systemName: tab.symbol)
                    .font(.system(size: 22, weight: selection == tab ? .semibold : .regular))
                    .frame(width: 34, height: 34)
                    .background(selection == tab ? LinkUpPalette.red.opacity(0.15) : .clear)
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                    .foregroundStyle(selection == tab ? LinkUpPalette.red : LinkUpPalette.textMuted)
                Text(L10n.text(tab.rawValue))
                    .font(LinkUpTypography.body(10, weight: .semibold))
                    .foregroundStyle(selection == tab ? LinkUpPalette.red : LinkUpPalette.textMuted)
                Circle()
                    .fill(selection == tab ? LinkUpPalette.red : .clear)
                    .frame(width: 4, height: 4)
            }
            .frame(maxWidth: .infinity)
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L10n.text(tab.rawValue))
    }

    private var createButton: some View {
        Button(action: create) {
            VStack(spacing: 4) {
                Text("LINK")
                    .font(LinkUpTypography.display(14, weight: .black))
                    .foregroundStyle(Color.white)
                    .frame(width: 56, height: 56)
                    .background(LinkUpPalette.red)
                    .clipShape(Circle())
                    .shadow(color: LinkUpPalette.red.opacity(0.4), radius: 12)
                    .offset(y: -16)
                Text("Create")
                    .font(LinkUpTypography.body(10, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.textMuted)
                    .offset(y: -16)
            }
            .frame(maxWidth: .infinity)
        }
        .buttonStyle(.plain)
        .accessibilityLabel("Create LINK")
    }
}
