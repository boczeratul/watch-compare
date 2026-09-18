import Foundation

/// String-catalog lookups keyed like the web's messages/*.json ("section.key") so the two clients
/// share wording. Tables live in Resources/<lang>.lproj/Localizable.strings.
public enum L10n {
    public static func t(_ key: String) -> String {
        String(localized: String.LocalizationValue(key), bundle: .module)
    }

    public static func t(_ key: String, _ args: CVarArg...) -> String {
        String(format: t(key), locale: .current, arguments: args)
    }

    public static func condition(_ c: Condition) -> String { t("condition.\(c.rawValue)") }
    public static func movement(_ m: Movement) -> String { t("movement.\(m.rawValue)") }
    public static func gender(_ g: Gender) -> String { t("gender.\(g.rawValue)") }
    public static func sort(_ s: SortKey) -> String { t("sort.\(s.rawValue)") }

    /// Dial colours are an open set on the API side; fall back to a prettified key.
    public static func dial(_ key: String) -> String {
        let k = "dial.\(key)"
        let v = t(k)
        return v == k ? key.replacingOccurrences(of: "_", with: " ").capitalized : v
    }

    public static func resultsTitle(count: Int, query: String) -> String {
        let q = query.trimmingCharacters(in: .whitespaces)
        let n: String
        switch count {
        case 0: n = t("search.resultsNone")
        case 1: n = t("search.resultsOne")
        default: n = t("search.resultsMany", count.formatted() as NSString)
        }
        return q.isEmpty ? n : t("search.resultsFor", n as NSString, q as NSString)
    }
}
