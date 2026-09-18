import Foundation

public enum APIError: Error, LocalizedError, Sendable {
    case http(status: Int, path: String)
    case decoding(String)
    case transport(String)
    case invalidURL

    public var errorDescription: String? {
        switch self {
        case .http(let status, let path): return "HTTP \(status) for \(path)"
        case .decoding(let msg): return "Bad response: \(msg)"
        case .transport(let msg): return msg
        case .invalidURL: return "Invalid URL"
        }
    }
    public var isNotFound: Bool {
        if case .http(let status, _) = self { return status == 404 }
        return false
    }
}

/// Read-only client for the Go API (backend/internal/api). Every call is a GET returning JSON;
/// responses are normalized while decoding (see `EmptyIfNull`) so views never see `null` arrays.
public final class APIClient: Sendable {
    public let baseURL: URL
    private let session: URLSession

    public init(baseURL: URL, session: URLSession = .shared) {
        self.baseURL = baseURL
        self.session = session
    }

    // MARK: Endpoints

    public func searchListings(_ query: SearchQuery, currency: String, page: Int = 1, perPage: Int = 30) async throws -> SearchResult {
        try await get("/api/v1/listings", query: query.queryItems(currency: currency, page: page, perPage: perPage))
    }

    public func listing(id: Int64) async throws -> Listing {
        try await get("/api/v1/listings/\(id)")
    }

    public func similar(id: Int64) async throws -> [Listing] {
        try await get("/api/v1/listings/\(id)/similar", as: ItemsResponse<Listing>.self).items
    }

    public func priceHistory(id: Int64) async throws -> [PricePoint] {
        try await get("/api/v1/listings/\(id)/price-history", as: ItemsResponse<PricePoint>.self).items
    }

    public func brands() async throws -> [Brand] {
        try await get("/api/v1/brands", as: ItemsResponse<Brand>.self).items
    }

    public func sources() async throws -> [Source] {
        try await get("/api/v1/sources", as: ItemsResponse<Source>.self).items.filter(\.enabled)
    }

    public func rates() async throws -> RatesResponse {
        try await get("/api/v1/rates")
    }

    public func stats() async throws -> Stats {
        try await get("/api/v1/stats")
    }

    // MARK: Transport

    private func get<T: Decodable>(_ path: String, query: [URLQueryItem] = [], as type: T.Type = T.self) async throws -> T {
        guard var components = URLComponents(url: baseURL.appending(path: path), resolvingAgainstBaseURL: false) else {
            throw APIError.invalidURL
        }
        if !query.isEmpty { components.queryItems = query }
        guard let url = components.url else { throw APIError.invalidURL }
        var request = URLRequest(url: url)
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.timeoutInterval = 20

        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: request)
        } catch {
            throw APIError.transport(error.localizedDescription)
        }
        if let http = response as? HTTPURLResponse, !(200..<300).contains(http.statusCode) {
            throw APIError.http(status: http.statusCode, path: path)
        }
        do {
            return try JSONDecoder.api.decode(T.self, from: data)
        } catch {
            throw APIError.decoding(String(describing: error))
        }
    }
}

extension JSONDecoder {
    /// Go's `time.Time` marshals as RFC 3339 with optional fractional seconds and either `Z` or a
    /// numeric offset; `Date.ISO8601FormatStyle` needs to be told about the fraction explicitly.
    public static var api: JSONDecoder {
        let d = JSONDecoder()
        d.dateDecodingStrategy = .custom { decoder in
            let raw = try decoder.singleValueContainer().decode(String.self)
            if let date = Date.parseRFC3339(raw) { return date }
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Unparseable date \(raw)"))
        }
        return d
    }
}

extension Date {
    public static func parseRFC3339(_ raw: String) -> Date? {
        let fractional = Date.ISO8601FormatStyle(includingFractionalSeconds: true)
        let plain = Date.ISO8601FormatStyle()
        return (try? fractional.parse(raw)) ?? (try? plain.parse(raw))
    }
}
