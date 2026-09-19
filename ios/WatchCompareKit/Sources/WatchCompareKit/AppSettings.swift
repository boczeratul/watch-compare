import Foundation
import Observation

/// Display currency and the exchange-rate table it depends on. The choice persists in UserDefaults
/// under the same key the web stores in its cookie.
@MainActor
@Observable
public final class AppSettings {
    public static let currencyKey = "wc_currency"

    public var currency: String {
        didSet { defaults.set(currency, forKey: Self.currencyKey) }
    }
    /// 1 USD expressed in each quote currency.
    public private(set) var rates: [String: Double] = [:]
    public private(set) var supportedCurrencies: [String] = Money.supportedCurrencies
    public private(set) var ratesLoaded = false

    private let defaults: UserDefaults

    public init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        let saved = defaults.string(forKey: Self.currencyKey)
        currency = saved.flatMap { Money.supportedCurrencies.contains($0) ? $0 : nil } ?? Money.defaultCurrency()
    }

    public func loadRates(using api: APIClient) async {
        guard let response = try? await api.rates() else { return }
        rates = response.table
        let supported = response.supported.isEmpty ? Money.supportedCurrencies : response.supported
        // Only offer currencies we can actually convert to (plus USD, the base).
        supportedCurrencies = supported.filter { $0 == "USD" || rates[$0] != nil }
        if !supportedCurrencies.contains(currency) { currency = "USD" }
        ratesLoaded = true
    }

    public func price(usd: Double?, original: Double?, originalCurrency: String?, showOriginal: Bool = true) -> PriceDisplay {
        PriceDisplay.resolve(usd: usd, original: original, originalCurrency: originalCurrency,
                             currency: currency, rates: rates, showOriginal: showOriginal)
    }

    public func format(_ amount: Double, currency: String) -> String {
        Money.format(amount, currency: currency)
    }

    /// A USD amount converted and rounded for use as a placeholder or hint in the visitor's currency.
    public func hint(usd: Double?) -> String? {
        guard let usd, let c = Money.convertFromUSD(usd, to: currency, rates: rates) else { return nil }
        return Int(Money.roundForDisplay(c, currency: currency)).formatted()
    }
}
