import Foundation

struct PasswordResetRoute: Equatable, Sendable {
    let host: String
    let path: String

    init?(configuredURL raw: String, associatedDomain rawAssociatedDomain: String) {
        let value = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty, value.utf8.count <= 4_096,
              let components = URLComponents(string: value),
              components.scheme?.lowercased() == "https",
              let host = components.host?.lowercased(), !host.isEmpty,
              components.port == nil,
              components.user == nil, components.password == nil,
              components.query == nil, components.fragment == nil else { return nil }
        guard Self.associatedDomainHost(rawAssociatedDomain) == host else { return nil }
        self.host = host
        self.path = components.path
    }

    static func associatedDomainHost(_ raw: String) -> String? {
        let value = raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard value.hasPrefix("applinks:"), value.utf8.count <= 512 else { return nil }
        let host = String(value.dropFirst("applinks:".count))
        guard !host.isEmpty,
              !host.contains("/"), !host.contains(":"), !host.contains("?"), !host.contains("#"),
              let components = URLComponents(string: "https://\(host)"),
              components.host?.lowercased() == host else { return nil }
        return host
    }

    func token(fromIncomingURL url: URL) -> String? {
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              components.scheme?.lowercased() == "https",
              components.host?.lowercased() == host,
              components.port == nil,
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
