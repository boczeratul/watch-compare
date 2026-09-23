import Foundation
import AmplitudeSwift

/// Product analytics (Amplitude). Two events, shared with the web app so dashboards line up:
///   search        — a results page was loaded (query, filters, sort, page, result count)
///   view_listing  — a listing detail screen was opened
/// Gated by App Tracking Transparency: `TrackingConsent` calls `setEnabled` with its status, so
/// the SDK is only created once `trackingAuthorizationStatus` is `.authorized`, is opted out if
/// the answer later changes to denied, and re-enabled if the visitor allows it in Settings.
/// Autocapture is limited to sessions.
@MainActor
public enum Analytics {
    public static let apiKey = "0f8cd19c77fc0325b6bdd8807e466e13"
    /// Launch argument / user default that turns analytics off (UI tests, screenshot runs).
    public static let disableKey = "WC_DISABLE_ANALYTICS"

    private static var client: Amplitude?
    /// True while events are being recorded (tracking authorized and not disabled by argument).
    public private(set) static var isEnabled = false

    /// Called by TrackingConsent whenever the authorization status is (re)read.
    public static func setEnabled(_ enabled: Bool) {
        let allowed = enabled && !UserDefaults.standard.bool(forKey: disableKey)
        isEnabled = allowed
        if allowed, client == nil {
            client = Amplitude(configuration: Configuration(apiKey: apiKey, autocapture: [.sessions]))
        }
        // Opting out stops sends and discards queued events without tearing the client down.
        client?.configuration.optOut = !allowed
    }

    public static func track(_ event: String, _ properties: [String: Any] = [:]) {
        var props = properties
        props["platform"] = "ios"
        props["locale"] = Locale.current.identifier
        guard isEnabled else { return }
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
