import SwiftUI

enum LinkUpTypography {
    static func display(_ size: CGFloat, weight: Font.Weight = .bold) -> Font {
        .custom("Outfit", size: size).weight(weight)
    }

    static func body(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        .custom("Inter", size: size).weight(weight)
    }

    static func mono(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        .custom("JetBrains Mono", size: size).weight(weight)
    }
}

enum LinkUpRadius {
    static let badge: CGFloat = 6
    static let compact: CGFloat = 8
    static let control: CGFloat = 12
    static let card: CGFloat = 16
    static let sheet: CGFloat = 24
}
