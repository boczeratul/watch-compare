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

// MARK: Push alerts

extension SearchQuery {
    /// The search's criteria (no sort or paging) as an alert query string, or nil when there are
    /// none: an alert without criteria would match every new listing, and the API rejects it.
    public func alertQuery(currency: String) -> String? {
        let items = queryItems(currency: currency, page: 1, perPage: 30)
            .filter { !["sort", "page", "per_page"].contains($0.name) }
        guard items.contains(where: { $0.name != "currency" }) else { return nil }
        var components = URLComponents()
        components.queryItems = items
        // URLComponents leaves "+" alone, but the Go side decodes it as a space.
        return components.percentEncodedQuery?.replacingOccurrences(of: "+", with: "%2B")
    }

    /// Rebuilds a search from an alert's query string. Price bounds are converted from the alert's
    /// currency into `displayCurrency` (the unit `SearchQuery` keeps them in) when rates allow;
    /// otherwise they are dropped rather than applied in the wrong currency.
    public init(alertQuery: String, displayCurrency: String, rates: [String: Double]) {
        self.init()
        var components = URLComponents()
        components.percentEncodedQuery = alertQuery
        var values: [String: String] = [:]
        for item in components.queryItems ?? [] { values[item.name] = item.value ?? "" }
        func set(_ name: String) -> Set<String> {
            Set((values[name] ?? "").split(separator: ",").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty })
        }
        text = values["q"] ?? ""
        brands = set("brand")
        model = values["model"] ?? ""
        reference = values["ref"] ?? ""
        sources = set("source")
        conditions = Set(set("condition").compactMap(Condition.init(rawValue:)))
        movements = Set(set("movement").compactMap(Movement.init(rawValue:)))
        genders = Set(set("gender").compactMap(Gender.init(rawValue:)))
        countries = set("country")
        dialColors = set("dial")
        yearMin = values["year_min"].flatMap { Int($0) }
        yearMax = values["year_max"].flatMap { Int($0) }
        diameterMin = values["diameter_min"].flatMap { Double($0) }
        diameterMax = values["diameter_max"].flatMap { Double($0) }
        hasBox = values["box"] == "true"
        hasPapers = values["papers"] == "true"
        sort = .newest

        let from = (values["currency"] ?? "USD").uppercased()
        func convert(_ raw: String?) -> Double? {
            guard let raw, let amount = Double(raw) else { return nil }
            if from == displayCurrency { return amount }
            let fromRate: Double? = from == "USD" ? 1 : rates[from]
            guard let fromRate, fromRate > 0 else { return nil }
            return Money.convertFromUSD(amount / fromRate, to: displayCurrency, rates: rates)
                .map { Money.roundForDisplay($0, currency: displayCurrency) }
        }
        priceMin = convert(values["price_min"])
        priceMax = convert(values["price_max"])
    }
}
