import Foundation

func passwordResetToken(from input: String) -> String? {
    let raw = input.trimmingCharacters(in: .whitespacesAndNewlines)
    guard raw.count <= 4096 else { return nil }

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

    guard token.range(of: #"^[A-Za-z0-9_-]{43}$"#, options: .regularExpression) != nil else {
        return nil
    }

    var encoded = token
        .replacingOccurrences(of: "-", with: "+")
        .replacingOccurrences(of: "_", with: "/")
    while encoded.count % 4 != 0 { encoded.append("=") }
    guard let data = Data(base64Encoded: encoded), data.count == 32 else { return nil }

    let canonical = data.base64EncodedString()
        .replacingOccurrences(of: "+", with: "-")
        .replacingOccurrences(of: "/", with: "_")
        .replacingOccurrences(of: "=", with: "")
    return canonical == token ? token : nil
}
