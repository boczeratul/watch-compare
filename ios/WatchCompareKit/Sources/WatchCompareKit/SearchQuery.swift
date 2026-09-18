import Foundation

/// The filter state of a search, 1:1 with the API's query parameters (see `toApiParams` in the
/// web client and `parseQuery` in the Go handler). Value type so it can be a navigation value.
public struct SearchQuery: Hashable, Sendable {
    public var text: String = ""
    public var brands: Set<String> = []
    public var model: String = ""
    public var reference: String = ""
    public var sources: Set<String> = []
    public var conditions: Set<Condition> = []
    public var movements: Set<Movement> = []
    public var genders: Set<Gender> = []
    public var countries: Set<String> = []
    public var dialColors: Set<String> = []
    /// In the visitor's display currency; the API converts to USD using `currency`.
    public var priceMin: Double?
    public var priceMax: Double?
    public var yearMin: Int?
    public var yearMax: Int?
    public var diameterMin: Double?
    public var diameterMax: Double?
    public var hasBox = false
    public var hasPapers = false
    public var sort: SortKey?

    public init(text: String = "") { self.text = text }

    public static func brand(_ slug: String) -> SearchQuery {
        var q = SearchQuery(); q.brands = [slug]; return q
    }
    public static func reference(_ ref: String) -> SearchQuery {
        var q = SearchQuery(); q.reference = ref; return q
    }

    /// Same default as the web sort select: relevance when there is free text, newest otherwise.
    public var effectiveSort: SortKey { sort ?? (text.isEmpty ? .newest : .relevance) }

    public var activeFilterCount: Int {
        var n = 0
        for set in [brands, sources, countries, dialColors] where !set.isEmpty { n += 1 }
        if !conditions.isEmpty { n += 1 }
        if !movements.isEmpty { n += 1 }
        if !genders.isEmpty { n += 1 }
        for v in [priceMin, priceMax, diameterMin, diameterMax] where v != nil { n += 1 }
        for v in [yearMin, yearMax] where v != nil { n += 1 }
        if hasBox { n += 1 }
        if hasPapers { n += 1 }
        return n
    }

    /// Drops every filter but keeps the free-text query (the web "Clear all" behaviour).
    public func clearingFilters() -> SearchQuery {
        var q = SearchQuery(text: text)
        q.sort = sort
        return q
    }

    public func queryItems(currency: String, page: Int, perPage: Int) -> [URLQueryItem] {
        var items: [URLQueryItem] = [URLQueryItem(name: "currency", value: currency)]
        func add(_ name: String, _ value: String?) {
            if let value, !value.isEmpty { items.append(URLQueryItem(name: name, value: value)) }
        }
        func list(_ name: String, _ values: Set<String>) { add(name, values.sorted().joined(separator: ",")) }
        func num(_ v: Double?) -> String? { v.map { $0 == $0.rounded() ? String(Int($0)) : String($0) } }

        add("q", text.trimmingCharacters(in: .whitespacesAndNewlines))
        list("brand", brands)
        add("model", model)
        add("ref", reference)
        list("source", sources)
        list("condition", Set(conditions.map(\.rawValue)))
        list("movement", Set(movements.map(\.rawValue)))
        list("gender", Set(genders.map(\.rawValue)))
        list("country", countries)
        list("dial", dialColors)
        add("price_min", num(priceMin))
        add("price_max", num(priceMax))
        add("year_min", yearMin.map(String.init))
        add("year_max", yearMax.map(String.init))
        add("diameter_min", num(diameterMin))
        add("diameter_max", num(diameterMax))
        if hasBox { add("box", "true") }
        if hasPapers { add("papers", "true") }
        add("sort", effectiveSort.rawValue)
        add("page", String(page))
        add("per_page", String(perPage))
        return items
    }
}
