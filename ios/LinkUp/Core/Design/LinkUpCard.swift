import SwiftUI

struct LinkUpCard<Content: View>: View {
    let content: Content

    init(@ViewBuilder content: () -> Content) {
        self.content = content()
    }

    var body: some View {
        content
            .padding(16)
            .background(LinkUpPalette.elevated)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card, style: .continuous))
            .overlay {
                RoundedRectangle(cornerRadius: LinkUpRadius.card, style: .continuous)
                    .stroke(LinkUpPalette.border, lineWidth: 1)
            }
    }
}

private struct LinkUpGlassModifier: ViewModifier {
    let elevated: Bool

    func body(content: Content) -> some View {
        content
            .background(.ultraThinMaterial)
            .background((elevated ? LinkUpPalette.elevated : LinkUpPalette.surface).opacity(elevated ? 0.78 : 0.72))
    }
}

extension View {
    func linkUpGlass(elevated: Bool = false) -> some View {
        modifier(LinkUpGlassModifier(elevated: elevated))
    }
}
