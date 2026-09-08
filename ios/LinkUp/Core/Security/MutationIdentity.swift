import CryptoKit
import Foundation

enum MutationIdentity {
    static func digest(for request: APIRequest) -> String {
        var canonical = Data()
        append(request.method.rawValue, to: &canonical)
        append(request.path, to: &canonical)

        let query = request.queryItems
            .map { ($0.name, $0.value ?? "") }
            .sorted { lhs, rhs in
                lhs.0 == rhs.0 ? lhs.1 < rhs.1 : lhs.0 < rhs.0
            }
        for (name, value) in query {
            append(name, to: &canonical)
            append(value, to: &canonical)
        }

        canonical.append(request.body ?? Data())
        let hash = SHA256.hash(data: canonical)
        return hash.map { String(format: "%02x", $0) }.joined()
    }

    private static func append(_ value: String, to data: inout Data) {
        data.append(contentsOf: value.utf8)
        data.append(0)
    }
}
