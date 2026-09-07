import Foundation

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
