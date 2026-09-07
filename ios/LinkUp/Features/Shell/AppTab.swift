import Foundation

enum AppTab: String, CaseIterable, Identifiable {
    case pulse = "Pulse"
    case map = "Map"
    case fly = "Fly"
    case me = "Me"

    var id: String { rawValue }
    var symbol: String {
        switch self {
        case .pulse: "bolt.fill"
        case .map: "mappin.and.ellipse"
        case .fly: "paperplane.fill"
        case .me: "person.fill"
        }
    }
}
