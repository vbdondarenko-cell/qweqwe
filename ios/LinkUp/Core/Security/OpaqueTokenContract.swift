import Foundation

enum OpaqueTokenContract {
    static let byteCount = 32
    static let encodedLength = 43

    static func canonical32ByteBase64URL(_ raw: String) -> String? {
        guard raw.utf8.count == encodedLength,
              raw.unicodeScalars.allSatisfy({ scalar in
                  let value = Int(scalar.value)
                  return (65...90).contains(value) ||
                      (97...122).contains(value) ||
                      (48...57).contains(value) ||
                      value == 45 || value == 95
              }) else {
            return nil
        }

        var padded = raw
            .replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")
        while padded.count % 4 != 0 { padded.append("=") }
        guard let data = Data(base64Encoded: padded), data.count == byteCount else { return nil }

        let canonical = data.base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
        return canonical == raw ? raw : nil
    }
}
