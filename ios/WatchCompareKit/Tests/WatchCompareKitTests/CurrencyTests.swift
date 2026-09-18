import XCTest
@testable import WatchCompareKit

final class CurrencyTests: XCTestCase {
    let rates: [String: Double] = ["TWD": 32.0, "JPY": 150.0, "EUR": 0.9]
    let en = Locale(identifier: "en_US")

    func testConvertAndRound() {
        XCTAssertEqual(Money.convertFromUSD(100, to: "USD", rates: [:]), 100)
        XCTAssertEqual(Money.convertFromUSD(100, to: "TWD", rates: rates), 3200)
        XCTAssertNil(Money.convertFromUSD(100, to: "GBP", rates: rates))
        XCTAssertEqual(Money.roundForDisplay(1_234_567, currency: "JPY"), 1_234_600)
        XCTAssertEqual(Money.roundForDisplay(12_345.6, currency: "USD"), 12_350)
        XCTAssertEqual(Money.roundForDisplay(1234.6, currency: "USD"), 1235)
    }

    func testFormat() {
        XCTAssertEqual(Money.format(1500000, currency: "JPY", locale: en), "¥1,500,000")
        XCTAssertEqual(Money.format(3200, currency: "TWD", locale: en), "NT$3,200")
        XCTAssertEqual(Money.format(999.5, currency: "USD", locale: en), "$999.5") // minimumFractionDigits 0, as on the web
        XCTAssertEqual(Money.format(10200.5, currency: "USD", locale: en), "$10,201")
        XCTAssertEqual(Money.format(10200.5, currency: "USD", locale: Locale(identifier: "de_DE")), "10.201\u{00A0}US$")
    }

    func testPriceDisplayPrefersOriginalInSameCurrency() {
        let p = PriceDisplay.resolve(usd: 10200.5, original: 1_500_000, originalCurrency: "JPY", currency: "JPY", rates: rates, locale: en)
        XCTAssertEqual(p.main, "¥1,500,000")
        XCTAssertNil(p.original)
    }

    func testPriceDisplayConvertsUsdAndShowsOriginal() {
        let p = PriceDisplay.resolve(usd: 100, original: 15000, originalCurrency: "JPY", currency: "TWD", rates: rates, locale: en)
        XCTAssertEqual(p.main, "NT$3,200")
        XCTAssertEqual(p.original, "¥15,000")
        let hidden = PriceDisplay.resolve(usd: 100, original: 15000, originalCurrency: "JPY", currency: "TWD", rates: rates, showOriginal: false, locale: en)
        XCTAssertNil(hidden.original)
    }

    func testPriceDisplayFallsBackToOriginalWhenNoRate() {
        let p = PriceDisplay.resolve(usd: 100, original: 90, originalCurrency: "EUR", currency: "GBP", rates: rates, locale: en)
        XCTAssertEqual(p.main, "€90")
        XCTAssertEqual(p.original, "€90")
        let none = PriceDisplay.resolve(usd: nil, original: nil, originalCurrency: nil, currency: "USD", rates: rates, locale: en)
        XCTAssertNil(none.main)
    }

    func testDefaultCurrencyFollowsRegion() {
        XCTAssertEqual(Money.defaultCurrency(for: Locale(identifier: "zh_TW")), "TWD")
        XCTAssertEqual(Money.defaultCurrency(for: Locale(identifier: "ja_JP")), "JPY")
        XCTAssertEqual(Money.defaultCurrency(for: Locale(identifier: "de_DE")), "EUR")
        XCTAssertEqual(Money.defaultCurrency(for: Locale(identifier: "th_TH")), "USD")
    }
}
