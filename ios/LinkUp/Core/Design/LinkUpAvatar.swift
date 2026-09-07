import SwiftUI

enum LinkUpAvatarSize { case sm, md, lg, xl
    var value: CGFloat { switch self { case .sm: 24; case .md: 32; case .lg: 48; case .xl: 80 } }
}

struct LinkUpAvatar: View {
    let initials: String
    var tint = LinkUpPalette.red
    var size: LinkUpAvatarSize = .md

    var body: some View {
        Text(initials)
            .font(LinkUpTypography.body(max(10, size.value * 0.28), weight: .bold))
            .foregroundStyle(LinkUpPalette.textPrimary)
            .frame(width: size.value, height: size.value)
            .background(tint.opacity(0.13))
            .clipShape(Circle())
            .overlay { Circle().stroke(tint, lineWidth: 1.5) }
            .accessibilityLabel("Profile avatar")
    }
}
