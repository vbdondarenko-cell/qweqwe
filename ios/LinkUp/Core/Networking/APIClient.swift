import Foundation

private final class NoRedirectDelegate: NSObject, URLSessionTaskDelegate, @unchecked Sendable {
    func urlSession(
        _ session: URLSession,
        task: URLSessionTask,
        willPerformHTTPRedirection response: HTTPURLResponse,
        newRequest request: URLRequest,
        completionHandler: @escaping (URLRequest?) -> Void
    ) {
        completionHandler(nil)
    }
}

actor APIClient {
    private let endpoint: APIEndpoint
    private let credentials: KeychainSessionStore
    private let delegate: NoRedirectDelegate
    private let session: URLSession
    private let maxResponseBytes = 1_048_576

    init(endpoint: APIEndpoint, credentials: KeychainSessionStore) {
        self.endpoint = endpoint
        self.credentials = credentials
        let delegate = NoRedirectDelegate()
        self.delegate = delegate
        let configuration = URLSessionConfiguration.ephemeral
        configuration.requestCachePolicy = .reloadIgnoringLocalCacheData
        configuration.urlCache = nil
        configuration.httpCookieStorage = nil
        configuration.timeoutIntervalForRequest = 15
        configuration.timeoutIntervalForResource = 20
        configuration.httpMaximumConnectionsPerHost = 4
        self.session = URLSession(configuration: configuration, delegate: delegate, delegateQueue: nil)
    }

    func send<Response: Decodable & Sendable>(
        _ request: APIRequest,
        as type: Response.Type = Response.self
    ) async throws -> Response {
        let data = try await perform(request)
        guard !data.isEmpty else {
            throw APIError.protocolViolation("Server returned an empty response.")
        }
        do {
            return try APICoding.decoder().decode(Response.self, from: data)
        } catch is CancellationError {
            throw CancellationError()
        } catch {
            throw APIError.protocolViolation("Server returned an invalid response.")
        }
    }

    func sendVoid(_ request: APIRequest) async throws {
        _ = try await perform(request)
    }

    private func perform(_ request: APIRequest) async throws -> Data {
        let attempts = request.method == .get ? 2 : 1
        for attempt in 1...attempts {
            do {
                return try await performOnce(request)
            } catch is CancellationError {
                throw CancellationError()
            } catch let error as APIError {
                if Task.isCancelled { throw CancellationError() }
                if attempt < attempts && error.retryableForGET {
                    try await Task<Never, Never>.sleep(nanoseconds: 250_000_000)
                    continue
                }
                throw error
            } catch {
                if Task.isCancelled { throw CancellationError() }
                let mapped = APIError.transport(String(describing: error))
                if attempt < attempts {
                    try await Task<Never, Never>.sleep(nanoseconds: 250_000_000)
                    continue
                }
                throw mapped
            }
        }
        throw APIError.transport("request failed")
    }

    private func performOnce(_ request: APIRequest) async throws -> Data {
        var urlRequest = URLRequest(url: try endpoint.url(path: request.path, queryItems: request.queryItems))
        urlRequest.httpMethod = request.method.rawValue
        urlRequest.setValue("application/json", forHTTPHeaderField: "Accept")
        urlRequest.cachePolicy = .reloadIgnoringLocalCacheData

        if request.authenticated {
            let credential: SessionCredential
            do {
                guard let stored = try await credentials.load() else {
                    throw APIError.unauthorized
                }
                credential = stored
            } catch let error as APIError {
                throw error
            } catch {
                throw APIError.secureStorageUnavailable
            }
            urlRequest.setValue("Bearer \(credential.token)", forHTTPHeaderField: "Authorization")
        }
        if let key = request.idempotencyKey {
            urlRequest.setValue(key.uuidString.lowercased(), forHTTPHeaderField: "Idempotency-Key")
        }
        if let body = request.body {
            urlRequest.httpBody = body
            urlRequest.setValue("application/json; charset=utf-8", forHTTPHeaderField: "Content-Type")
        }

        let bytes: URLSession.AsyncBytes
        let response: URLResponse
        do {
            (bytes, response) = try await session.bytes(for: urlRequest)
        } catch {
            if Task.isCancelled { throw CancellationError() }
            throw APIError.transport(String(describing: error))
        }
        guard let http = response as? HTTPURLResponse else {
            throw APIError.protocolViolation("Server returned a non-HTTP response.")
        }
        let unauthorized = request.authenticated && http.statusCode == 401
        if unauthorized { await credentials.clear() }

        if response.expectedContentLength > Int64(maxResponseBytes) {
            throw APIError.responseTooLarge
        }
        var data = Data()
        if response.expectedContentLength > 0 {
            data.reserveCapacity(Int(response.expectedContentLength))
        }
        do {
            for try await byte in bytes {
                if data.count >= maxResponseBytes { throw APIError.responseTooLarge }
                data.append(byte)
            }
        } catch let error as APIError {
            throw error
        } catch {
            if Task.isCancelled { throw CancellationError() }
            throw APIError.transport(String(describing: error))
        }

        if unauthorized { throw APIError.unauthorized }
        guard (200...299).contains(http.statusCode) else {
            let payload = try? APICoding.decoder().decode(APIErrorPayload.self, from: data)
            let requestID = payload?.requestId ?? http.value(forHTTPHeaderField: "X-Request-ID")
            let code = payload?.code.flatMap { $0.isEmpty ? nil : $0 } ?? "http_error"
            let message = payload?.message.flatMap { $0.isEmpty ? nil : $0 }
                ?? "Request failed (HTTP \(http.statusCode))."
            throw APIError.http(status: http.statusCode, code: code, message: message, requestID: requestID)
        }
        return data
    }
}
