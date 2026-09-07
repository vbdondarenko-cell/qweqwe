import SwiftUI

struct LinkUpProgress: View {
    let value: Int
    let maximum: Int
    var tint = LinkUpPalette.red
    var showsLabel = true

    private var fraction: CGFloat {
        guard maximum > 0 else { return 0 }
        return min(1, max(0, CGFloat(value) / CGFloat(maximum)))
    }

    var body: some View {
        VStack(spacing: 6) {
            GeometryReader { proxy in
                ZStack(alignment: .leading) {
                    Capsule().fill(LinkUpPalette.zone)
                    Capsule().fill(tint).frame(width: proxy.size.width * fraction)
                }
            }
            .frame(height: 6)
            if showsLabel {
                Text("\(value)/\(maximum) going")
                    .font(LinkUpTypography.mono(10, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.textMuted)
                    .frame(maxWidth: .infinity, alignment: .trailing)
            }
        }
    }
}
