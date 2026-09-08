import SwiftUI

enum LinkUpTypography {
    static func display(_ size: CGFloat, weight: Font.Weight = .bold) -> Font {
        .custom("Outfit", size: size, relativeTo: displayStyle(for: size)).weight(weight)
    }

    static func body(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        .custom("Inter", size: size, relativeTo: bodyStyle(for: size)).weight(weight)
    }

    static func mono(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        .custom("JetBrains Mono", size: size, relativeTo: monoStyle(for: size)).weight(weight)
    }

    private static func displayStyle(for size: CGFloat) -> Font.TextStyle {
        switch size {
        case 28...: .largeTitle
        case 22..<28: .title2
        case 18..<22: .title3
        case 15..<18: .headline
        default: .subheadline
        }
    }

    private static func bodyStyle(for size: CGFloat) -> Font.TextStyle {
        switch size {
        case 17...: .body
        case 15..<17: .callout
        case 13..<15: .subheadline
        case 11..<13: .caption
        default: .caption2
        }
    }

    private static func monoStyle(for size: CGFloat) -> Font.TextStyle {
        switch size {
        case 15...: .callout
        case 12..<15: .caption
        default: .caption2
        }
    }
}

enum LinkUpRadius {
    static let badge: CGFloat = 6
    static let compact: CGFloat = 8
    static let control: CGFloat = 12
    static let card: CGFloat = 16
    static let sheet: CGFloat = 24
}
