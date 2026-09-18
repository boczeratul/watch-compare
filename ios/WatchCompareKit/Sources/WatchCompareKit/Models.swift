import Foundation

// Mirrors backend/internal/model/model.go (JSON tags) and frontend/src/lib/api.ts.

public enum Condition: String, Codable, CaseIterable, Sendable {
    case new, unworn, veryGood = "very_good", good, fair, poor, unknown
    public init(from decoder: Decoder) throws {
        let raw = try decoder.singleValueContainer().decode(String.self)
        self = Condition(rawValue: raw) ?? .unknown
    }
}

public enum Movement: String, Codable, CaseIterable, Sendable {
    case automatic, manual, quartz, unknown
    public init(from decoder: Decoder) throws {
        let raw = try decoder.singleValueContainer().decode(String.self)
        self = Movement(rawValue: raw) ?? .unknown
    }
}

public enum Gender: String, Codable, CaseIterable, Sendable {
    case men, women, unisex, unknown
    public init(from decoder: Decoder) throws {
        let raw = try decoder.singleValueContainer().decode(String.self)
        self = Gender(rawValue: raw) ?? .unknown
    }
}

public enum SortKey: String, Codable, CaseIterable, Sendable, Identifiable {
    case relevance, newest, oldest
    case priceAsc = "price_asc", priceDesc = "price_desc"
    case yearDesc = "year_desc", yearAsc = "year_asc"
    case sizeAsc = "size_asc", sizeDesc = "size_desc"
    public var id: String { rawValue }
}

/// Decodes a JSON array, treating `null` or a missing key as an empty array. Go marshals a nil
/// slice as `null`, and the web client normalizes the same way at its API boundary.
@propertyWrapper
public struct EmptyIfNull<Element: Decodable & Sendable>: Decodable, Sendable {
    public var wrappedValue: [Element]
    public init(wrappedValue: [Element]) { self.wrappedValue = wrappedValue }
    public init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        wrappedValue = container.decodeNil() ? [] : try container.decode([Element].self)
    }
}

extension EmptyIfNull: Equatable where Element: Equatable {}
extension EmptyIfNull: Hashable where Element: Hashable {}

extension KeyedDecodingContainer {
    public func decode<T>(_ type: EmptyIfNull<T>.Type, forKey key: Key) throws -> EmptyIfNull<T> {
        try decodeIfPresent(type, forKey: key) ?? EmptyIfNull(wrappedValue: [])
    }
}

public struct Listing: Decodable, Identifiable, Hashable, Sendable {
    public var id: Int64
    public var source: String
    public var sourceName: String
    public var externalId: String
    public var url: String
    public var title: String
    public var brand: String?
    public var brandName: String?
    public var model: String?
    public var referenceNumber: String?
    public var condition: Condition
    public var year: Int?
    public var caseDiameterMm: Double?
    public var caseMaterial: String?
    public var dialColor: String?
    public var movement: Movement
    public var gender: Gender
    public var hasBox: Bool?
    public var hasPapers: Bool?
    /// As listed (tax-included for Japanese sources).
    public var price: Double?
    /// 税抜 / tax-free amount, Japanese sources only.
    public var priceExclTax: Double?
    public var currency: String?
    public var priceUsd: Double?
    public var shippingPrice: Double?
    public var locationCountry: String?
    public var locationCity: String?
    public var sellerName: String?
    public var sellerType: String?
    @EmptyIfNull public var imageUrls: [String]
    public var description: String?
    public var isActive: Bool
    public var firstSeenAt: Date
    public var lastSeenAt: Date

    public var externalURL: URL? { URL(string: url) }
    public var firstImageURL: URL? { imageUrls.first.flatMap(URL.init(string:)) }
    public var location: String? {
        let parts = [locationCity, locationCountry].compactMap { $0 }.filter { !$0.isEmpty }
        return parts.isEmpty ? nil : parts.joined(separator: ", ")
    }
    /// priceUsd is computed from the tax-free amount, so that is the "original" shown next to it.
    public var comparisonPrice: Double? { priceExclTax ?? price }
}

public struct FacetValue: Decodable, Hashable, Identifiable, Sendable {
    public var key: String
    public var label: String
    public var count: Int
    public var id: String { key }
    public init(key: String, label: String, count: Int) {
        self.key = key; self.label = label; self.count = count
    }
}

public struct Facets: Decodable, Hashable, Sendable {
    @EmptyIfNull public var brands: [FacetValue]
    @EmptyIfNull public var sources: [FacetValue]
    @EmptyIfNull public var conditions: [FacetValue]
    @EmptyIfNull public var movements: [FacetValue]
    @EmptyIfNull public var countries: [FacetValue]
    @EmptyIfNull public var dialColors: [FacetValue]
    @EmptyIfNull public var years: [FacetValue]
    public var priceMinUsd: Double?
    public var priceMaxUsd: Double?

    public static let empty = Facets()
    public init() {
        _brands = .init(wrappedValue: []); _sources = .init(wrappedValue: []); _conditions = .init(wrappedValue: [])
        _movements = .init(wrappedValue: []); _countries = .init(wrappedValue: []); _dialColors = .init(wrappedValue: [])
        _years = .init(wrappedValue: [])
    }
}

public struct SearchResult: Decodable, Sendable {
    @EmptyIfNull public var items: [Listing]
    public var total: Int
    public var page: Int
    public var perPage: Int
    public var totalPages: Int
    public var facets: Facets

    private enum CodingKeys: String, CodingKey { case items, total, page, perPage, totalPages, facets }
    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        _items = try c.decode(EmptyIfNull<Listing>.self, forKey: .items)
        total = try c.decodeIfPresent(Int.self, forKey: .total) ?? 0
        page = try c.decodeIfPresent(Int.self, forKey: .page) ?? 1
        perPage = try c.decodeIfPresent(Int.self, forKey: .perPage) ?? 30
        totalPages = try c.decodeIfPresent(Int.self, forKey: .totalPages) ?? 0
        facets = try c.decodeIfPresent(Facets.self, forKey: .facets) ?? .empty
    }
    public static let empty = SearchResult()
    private init() { _items = .init(wrappedValue: []); total = 0; page = 1; perPage = 30; totalPages = 0; facets = .empty }
}

public struct Brand: Decodable, Hashable, Identifiable, Sendable {
    public var id: Int
    public var slug: String
    public var name: String
    public var listingCount: Int
}

public struct Source: Decodable, Hashable, Identifiable, Sendable {
    public var id: Int
    public var key: String
    public var name: String
    public var baseUrl: String
    public var country: String
    public var currency: String
    public var enabled: Bool
}

public struct ExchangeRate: Decodable, Hashable, Sendable {
    public var quote: String
    public var rate: Double
}

public struct RatesResponse: Decodable, Sendable {
    public var base: String
    @EmptyIfNull public var supported: [String]
    @EmptyIfNull public var items: [ExchangeRate]
    public var table: [String: Double] { Dictionary(items.map { ($0.quote, $0.rate) }, uniquingKeysWith: { a, _ in a }) }
}

public struct PricePoint: Decodable, Hashable, Identifiable, Sendable {
    public var price: Double?
    public var currency: String
    public var priceUsd: Double?
    public var observedAt: Date
    public var id: Date { observedAt }
}

public struct SourceStat: Decodable, Hashable, Identifiable, Sendable {
    public var key: String
    public var name: String
    public var count: Int
    public var id: String { key }
}

public struct Stats: Decodable, Sendable {
    public var activeListings: Int
    public var brands: Int
    public var lastCrawlAt: Date?
    @EmptyIfNull public var sources: [SourceStat]
    public var activeSourceCount: Int {
        let active = sources.filter { $0.count > 0 }.count
        return active > 0 ? active : sources.count
    }
}

public struct ItemsResponse<T: Decodable & Sendable>: Decodable, Sendable {
    @EmptyIfNull public var items: [T]
}
