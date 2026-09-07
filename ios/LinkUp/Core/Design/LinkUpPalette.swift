import SwiftUI

enum LinkUpPalette {
    static let background = Color(hex: 0x050506)
    static let surface = Color(hex: 0x0D0E10)
    static let elevated = Color(hex: 0x141518)
    static let zone = Color(hex: 0x1A1C20)
    static let red = Color(hex: 0xFF2D35)
    static let redSignal = Color(hex: 0xFF3B42)
    static let redDeep = Color(hex: 0x9F171E)
    static let critical = Color(hex: 0xEF4444)
    static let success = Color(hex: 0x22C55E)
    static let warning = Color(hex: 0xF59E0B)
    static let info = Color(hex: 0x3B82F6)
    static let textPrimary = Color(hex: 0xF7F8FA)
    static let textDimmed = Color(hex: 0xA5A9B0)
    static let textMuted = Color(hex: 0x747982)
    static let border = Color(hex: 0x24262B)
}

private extension Color {
    init(hex: UInt32) {
        self.init(
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255
        )
    }
}
