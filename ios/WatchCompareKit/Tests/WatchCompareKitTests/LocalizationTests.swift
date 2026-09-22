import XCTest
@testable import WatchCompareKit

/// Guards the string tables: every key referenced from the views exists in English, and every
/// locale defines exactly the same keys as English.
final class LocalizationTests: XCTestCase {
    private var packageRoot: URL {
        URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
    }

    private func keys(in locale: String) throws -> Set<String> {
        let url = packageRoot.appending(path: "Sources/WatchCompareKit/Resources/\(locale).lproj/Localizable.strings")
        let text = try String(contentsOf: url, encoding: .utf8)
        let re = try NSRegularExpression(pattern: #"^"([^"]+)"\s*="#, options: .anchorsMatchLines)
        let ns = text as NSString
        return Set(re.matches(in: text, range: NSRange(location: 0, length: ns.length)).map { ns.substring(with: $0.range(at: 1)) })
    }

    func testEveryLocaleHasTheSameKeys() throws {
        let en = try keys(in: "en")
        XCTAssertFalse(en.isEmpty)
        for loc in ["zh-Hant", "zh-Hans", "ja", "de"] {
            let other = try keys(in: loc)
            XCTAssertEqual(other.subtracting(en), [], "\(loc) has keys missing from en")
            XCTAssertEqual(en.subtracting(other), [], "\(loc) is missing keys")
        }
    }

    func testViewsOnlyReferenceKnownKeys() throws {
        let en = try keys(in: "en")
        let sources = packageRoot.appending(path: "Sources/WatchCompareKit")
        let files = try FileManager.default.subpathsOfDirectory(atPath: sources.path).filter { $0.hasSuffix(".swift") }
        let re = try NSRegularExpression(pattern: #"L10n\.t\("([a-zA-Z]+\.[a-zA-Z_]+)""#)
        var referenced: Set<String> = []
        for f in files {
            let text = try String(contentsOf: sources.appending(path: f), encoding: .utf8)
            let ns = text as NSString
            for m in re.matches(in: text, range: NSRange(location: 0, length: ns.length)) {
                referenced.insert(ns.substring(with: m.range(at: 1)))
            }
        }
        XCTAssertFalse(referenced.isEmpty)
        XCTAssertEqual(referenced.subtracting(en), [], "keys used in views but missing from en.lproj")
        // Enum-driven lookups.
        for c in Condition.allCases { XCTAssertTrue(en.contains("condition.\(c.rawValue)")) }
        for m in Movement.allCases { XCTAssertTrue(en.contains("movement.\(m.rawValue)")) }
        for g in Gender.allCases { XCTAssertTrue(en.contains("gender.\(g.rawValue)")) }
        for s in SortKey.allCases { XCTAssertTrue(en.contains("sort.\(s.rawValue)")) }
    }

    /// Every "%" in a table must be a positional/typed specifier or an escaped "%%": a bare
    /// "%1$d%" once rendered "11% cheaper" as "11heaper".
    func testNoStrayPercentSpecifiers() throws {
        let valid = try NSRegularExpression(pattern: #"%(\d+\$)?[@d]|%%"#)
        for loc in ["en", "zh-Hant", "zh-Hans", "ja", "de"] {
            let url = packageRoot.appending(path: "Sources/WatchCompareKit/Resources/\(loc).lproj/Localizable.strings")
            let text = try String(contentsOf: url, encoding: .utf8)
            for line in text.split(separator: "\n") where line.hasPrefix("\"") {
                var value = String(line)
                let ns = value as NSString
                value = valid.stringByReplacingMatches(in: value, range: NSRange(location: 0, length: ns.length), withTemplate: "")
                XCTAssertFalse(value.contains("%"), "\(loc): stray % in \(line)")
            }
        }
    }

    func testSaveVsRendersPercentSign() {
        XCTAssertEqual(L10n.t("listing.saveVs", 11), "11% cheaper than the next offer")
        XCTAssertEqual(L10n.t("search.price", "TWD" as NSString), "Price (TWD)")
    }

    func testLookupsResolve() {
        XCTAssertEqual(L10n.dial("black"), "Black")
        XCTAssertEqual(L10n.dial("teal_blue"), "Teal Blue")
        XCTAssertEqual(L10n.resultsTitle(count: 0, query: ""), "No watches")
        XCTAssertEqual(L10n.resultsTitle(count: 1, query: "sub"), "1 watch for “sub”")
        XCTAssertEqual(L10n.resultsTitle(count: 1234, query: ""), "1,234 watches")
    }
}
