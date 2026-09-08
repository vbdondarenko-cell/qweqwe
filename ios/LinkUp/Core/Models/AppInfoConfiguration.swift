import Foundation

struct AppInfoConfiguration: Equatable, Sendable {
    let version: String
    let build: String
    let privacyURL: URL?
    let termsURL: URL?

    init(bundle: Bundle = .main) {
        version = Self.nonEmptyString(bundle.object(forInfoDictionaryKey: "CFBundleShortVersionString")) ?? "—"
        build = Self.nonEmptyString(bundle.object(forInfoDictionaryKey: "CFBundleVersion")) ?? "—"
        privacyURL = Self.trustedHTTPSURL(bundle.object(forInfoDictionaryKey: "LINKUP_PRIVACY_URL"))
        termsURL = Self.trustedHTTPSURL(bundle.object(forInfoDictionaryKey: "LINKUP_TERMS_URL"))
    }

    init(version: String, build: String, privacyURL: URL?, termsURL: URL?) {
        self.version = version
        self.build = build
        self.privacyURL = privacyURL
        self.termsURL = termsURL
    }

    static func trustedHTTPSURL(_ raw: Any?) -> URL? {
        guard let value = nonEmptyString(raw), value.utf8.count <= 2_048,
              let components = URLComponents(string: value),
              components.scheme?.lowercased() == "https",
              let host = components.host, !host.isEmpty,
              components.user == nil, components.password == nil,
              components.fragment == nil,
              let url = components.url else { return nil }
        return url
    }

    private static func nonEmptyString(_ raw: Any?) -> String? {
        guard let raw = raw as? String else { return nil }
        let value = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        return value.isEmpty ? nil : value
    }
}
