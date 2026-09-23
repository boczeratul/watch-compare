import Foundation
import AmplitudeSwift

/// Product analytics (Amplitude). Two events, shared with the web app so dashboards line up:
///   search        — a results page was loaded (query, filters, sort, page, result count)
///   view_listing  — a listing detail screen was opened
/// Autocapture is limited to sessions; the SDK never reads the advertising identifier, so this is
/// first-party measurement rather than cross-app tracking (see TrackingConsent for the latter).
@MainActor
public enum Analytics {
    public static let apiKey = "0f8cd19c77fc0325b6bdd8807e466e13"
    /// Launch argument / user default that turns analytics off (UI tests, screenshot runs).
    public static let disableKey = "WC_DISABLE_ANALYTICS"

    private static var client: Amplitude?

    public static func start() {
        guard client == nil, !UserDefaults.standard.bool(forKey: disableKey) else { return }
        let configuration = Configuration(apiKey: apiKey, autocapture: [.sessions])
        client = Amplitude(configuration: configuration)
    }

    public static func track(_ event: String, _ properties: [String: Any] = [:]) {
        var props = properties
        props["platform"] = "ios"
        props["locale"] = Locale.current.identifier
        client?.track(eventType: event, eventProperties: props)
    }

    public static func trackSearch(_ query: SearchQuery, currency: String, page: Int, resultCount: Int) {
        var p: [String: Any] = ["page": page, "result_count": resultCount, "currency": currency, "sort": query.effectiveSort.rawValue]
        func put(_ key: String, _ value: String) { if !value.isEmpty { p[key] = value } }
        func put(_ key: String, _ set: Set<String>) { put(key, set.sorted().joined(separator: ",")) }
        put("query", query.text.trimmingCharacters(in: .whitespacesAndNewlines))
        put("brand", query.brands)
        put("model", query.model)
        put("ref", query.reference)
        put("source", query.sources)
        put("condition", Set(query.conditions.map(\.rawValue)))
        put("movement", Set(query.movements.map(\.rawValue)))
        put("gender", Set(query.genders.map(\.rawValue)))
        put("country", query.countries)
        put("dial", query.dialColors)
        if let v = query.priceMin { p["price_min"] = v }
        if let v = query.priceMax { p["price_max"] = v }
        if let v = query.yearMin { p["year_min"] = v }
        if let v = query.yearMax { p["year_max"] = v }
        if let v = query.diameterMin { p["diameter_min"] = v }
        if let v = query.diameterMax { p["diameter_max"] = v }
        if query.hasBox { p["box"] = "true" }
        if query.hasPapers { p["papers"] = "true" }
        track("search", p)
    }

    public static func trackViewListing(_ l: Listing, currency: String) {
        var p: [String: Any] = ["listing_id": l.id, "source": l.source, "condition": l.condition.rawValue, "currency": l.currency ?? "", "display_currency": currency]
        if let v = l.brand { p["brand"] = v }
        if let v = l.model, !v.isEmpty { p["model"] = v }
        if let v = l.referenceNumber, !v.isEmpty { p["ref"] = v }
        if let v = l.priceUsd { p["price_usd"] = v }
        track("view_listing", p)
    }
}
