import Foundation

enum APIEndpointError: Error, LocalizedError, Sendable {
    case missing
    case invalid
    case insecure

    var errorDescription: String? {
        switch self {
        case .missing: L10n.text("LINKUP_API_BASE_URL is not configured.")
        case .invalid: L10n.text("LINKUP_API_BASE_URL is invalid.")
        case .insecure: L10n.text("The API endpoint must use HTTPS.")
        }
    }
}

struct APIEndpoint: Sendable {
    let baseURL: URL

    init(raw: String) throws {
        let value = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty else { throw APIEndpointError.missing }
        guard var parts = URLComponents(string: value),
              let scheme = parts.scheme?.lowercased(),
              let host = parts.host?.lowercased(),
              parts.user == nil, parts.password == nil,
              parts.query == nil, parts.fragment == nil,
              parts.path.isEmpty || parts.path == "/" else {
            throw APIEndpointError.invalid
        }
        let loopback = host == "localhost" || host == "127.0.0.1" || host == "::1"
#if DEBUG
        guard scheme == "https" || (scheme == "http" && loopback) else { throw APIEndpointError.insecure }
#else
        guard scheme == "https" else { throw APIEndpointError.insecure }
#endif
        parts.path = ""
        guard let normalized = parts.url else { throw APIEndpointError.invalid }
        baseURL = normalized
    }

    func url(path: String, queryItems: [URLQueryItem] = []) throws -> URL {
        guard path.hasPrefix("/"), var parts = URLComponents(url: baseURL, resolvingAgainstBaseURL: false) else {
            throw APIEndpointError.invalid
        }
        parts.path = path
        parts.queryItems = queryItems.isEmpty ? nil : queryItems
        guard let url = parts.url else { throw APIEndpointError.invalid }
        return url
    }
}
