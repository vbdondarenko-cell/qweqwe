import Foundation

struct PasswordResetRoute: Equatable, Sendable {
    let host: String
    let port: Int?
    let path: String

    init?(configuredURL raw: String) {
        let value = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty, value.utf8.count <= 4_096,
              let components = URLComponents(string: value),
              components.scheme?.lowercased() == "https",
              let host = components.host?.lowercased(), !host.isEmpty,
              components.user == nil, components.password == nil,
              components.query == nil, components.fragment == nil else { return nil }
        self.host = host
        self.port = components.port
        self.path = components.path
    }

    func token(fromIncomingURL url: URL) -> String? {
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              components.scheme?.lowercased() == "https",
              components.host?.lowercased() == host,
              components.port == port,
              components.path == path,
              components.user == nil, components.password == nil,
              components.fragment == nil,
              let items = components.queryItems, items.count == 1,
              items[0].name == "token", let value = items[0].value else { return nil }
        return OpaqueTokenContract.canonical32ByteBase64URL(value)
    }
}

func passwordResetToken(from input: String) -> String? {
    let raw = input.trimmingCharacters(in: .whitespacesAndNewlines)
    guard raw.utf8.count <= 4096 else { return nil }

    let token: String
    if raw.hasPrefix("https://") {
        guard let components = URLComponents(string: raw),
              components.scheme?.lowercased() == "https",
              let host = components.host, !host.isEmpty,
              components.user == nil,
              components.password == nil,
              components.fragment == nil else { return nil }
        let values = (components.queryItems ?? [])
            .filter { $0.name == "token" }
            .compactMap(\.value)
        guard values.count == 1 else { return nil }
        token = values[0]
    } else {
        token = raw
    }

    return OpaqueTokenContract.canonical32ByteBase64URL(token)
}
