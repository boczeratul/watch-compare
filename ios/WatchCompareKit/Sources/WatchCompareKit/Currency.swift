import Foundation

/// Pure helpers mirroring frontend/src/lib/currency.ts and the `Price` component.
public enum Money {
    public static let supportedCurrencies = ["USD", "EUR", "GBP", "CHF", "JPY", "TWD", "HKD", "SGD", "CNY", "KRW", "AUD", "CAD"]
    static let zeroDecimal: Set<String> = ["JPY", "KRW", "TWD"]

    public static func convertFromUSD(_ usd: Double, to currency: String, rates: [String: Double]) -> Double? {
        if currency == "USD" { return usd }
        guard let r = rates[currency], r > 0 else { return nil }
        return usd * r
    }

    /// Round a converted price so it reads like a price tag rather than an FX quote.
    public static func roundForDisplay(_ amount: Double, currency: String) -> Double {
        if zeroDecimal.contains(currency) { return (amount / 100).rounded() * 100 }
        return amount >= 10_000 ? (amount / 10).rounded() * 10 : amount.rounded()
    }

    public static func format(_ amount: Double, currency: String, locale: Locale = .current) -> String {
        let zero = zeroDecimal.contains(currency)
        let fraction = zero ? 0 : (amount >= 1000 ? 0 : 2)
        let formatter = NumberFormatter()
        formatter.locale = locale
        formatter.numberStyle = .currency
        formatter.currencyCode = currency
        formatter.minimumFractionDigits = 0
        formatter.maximumFractionDigits = fraction
        formatter.roundingMode = .halfUp // Intl.NumberFormat rounds half away from zero; match the web.
        // Keep dollar-family currencies distinguishable (NT$, HK$, A$…) instead of a bare "$".
        formatter.currencySymbol = symbol(for: currency, locale: locale)
        return formatter.string(from: NSNumber(value: amount)) ?? "\(currency) \(Int(amount.rounded()))"
    }

    static func symbol(for currency: String, locale: Locale) -> String {
        switch currency {
        case "USD": return locale.region?.identifier == "US" ? "$" : "US$"
        case "TWD": return "NT$"
        case "HKD": return "HK$"
        case "SGD": return "S$"
        case "AUD": return "A$"
        case "CAD": return "CA$"
        case "CNY": return "CN¥"
        case "JPY": return "¥"
        case "KRW": return "₩"
        case "EUR": return "€"
        case "GBP": return "£"
        case "CHF": return "CHF "
        default: return currency + " "
        }
    }

    /// Default display currency: the device region's currency when the API supports it, else USD
    /// (the web picks per locale: en→USD, zh-TW→TWD, zh-CN→CNY, ja→JPY, de→EUR).
    public static func defaultCurrency(for locale: Locale = .current) -> String {
        if let code = locale.currency?.identifier, supportedCurrencies.contains(code) { return code }
        return "USD"
    }
}

/// What to print for a price: the visitor's currency first, the seller's original underneath.
public struct PriceDisplay: Equatable, Sendable {
    public var main: String?
    public var original: String?

    public init(main: String?, original: String?) { self.main = main; self.original = original }

    /// Same rules as the web `Price` component: prefer the exact original when it is already in the
    /// display currency, otherwise convert USD with today's rate, otherwise fall back to the original.
    public static func resolve(usd: Double?, original: Double?, originalCurrency: String?,
                               currency: String, rates: [String: Double], showOriginal: Bool = true,
                               locale: Locale = .current) -> PriceDisplay {
        if usd == nil && original == nil { return PriceDisplay(main: nil, original: nil) }
        var main: String?
        if let original, originalCurrency == currency {
            main = Money.format(original, currency: currency, locale: locale)
        } else if let usd, let converted = Money.convertFromUSD(usd, to: currency, rates: rates) {
            main = Money.format(Money.roundForDisplay(converted, currency: currency), currency: currency, locale: locale)
        }
        if main == nil, let original, let originalCurrency {
            main = Money.format(original, currency: originalCurrency, locale: locale)
        }
        var orig: String?
        if showOriginal, let original, let originalCurrency, originalCurrency != currency {
            orig = Money.format(original, currency: originalCurrency, locale: locale)
        }
        return PriceDisplay(main: main, original: orig)
    }
}
