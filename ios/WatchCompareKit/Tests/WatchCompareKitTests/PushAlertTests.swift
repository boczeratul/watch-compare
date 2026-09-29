import XCTest
@testable import WatchCompareKit

final class PushAlertTests: XCTestCase {
    func testAlertQueryDropsSortAndPaging() throws {
        var q = SearchQuery(text: "sub")
        q.brands = ["rolex"]
        q.priceMax = 300_000
        q.sort = .priceAsc
        let s = try XCTUnwrap(q.alertQuery(currency: "TWD"))
        XCTAssertFalse(s.contains("sort="))
        XCTAssertFalse(s.contains("page="))
        XCTAssertTrue(s.contains("currency=TWD"))
        XCTAssertTrue(s.contains("price_max=300000"))
    }

    func testAlertQueryNeedsCriteria() {
        XCTAssertNil(SearchQuery().alertQuery(currency: "USD"))
        var q = SearchQuery(); q.sort = .priceAsc
        XCTAssertNil(q.alertQuery(currency: "USD"))
    }

    func testRoundTripInSameCurrency() throws {
        var q = SearchQuery(text: "GMT Master")
        q.brands = ["rolex", "tudor"]
        q.conditions = [.new]
        q.priceMin = 5000
        q.hasPapers = true
        let s = try XCTUnwrap(q.alertQuery(currency: "USD"))
        let back = SearchQuery(alertQuery: s, displayCurrency: "USD", rates: [:])
        XCTAssertEqual(back.text, "GMT Master")
        XCTAssertEqual(back.brands, ["rolex", "tudor"])
        XCTAssertEqual(back.conditions, [.new])
        XCTAssertEqual(back.priceMin, 5000)
        XCTAssertTrue(back.hasPapers)
        XCTAssertEqual(back.sort, .newest)
    }

    func testPricesConvertToDisplayCurrency() {
        let rates = ["TWD": 32.0, "EUR": 0.9]
        let q = SearchQuery(alertQuery: "brand=rolex&currency=TWD&price_max=320000", displayCurrency: "EUR", rates: rates)
        XCTAssertEqual(q.priceMax, 9000)
        // Unknown rate: drop the bound rather than apply it in the wrong currency.
        let unknown = SearchQuery(alertQuery: "brand=rolex&currency=TWD&price_max=320000", displayCurrency: "EUR", rates: [:])
        XCTAssertNil(unknown.priceMax)
    }

    func testAlertDecodes() throws {
        let json = #"{"id":7,"name":"rolex · ≤ 300,000 TWD","query":"brand=rolex&currency=TWD","createdAt":"2026-09-29T01:05:04.01442Z"}"#
        let a = try JSONDecoder.api.decode(PushAlert.self, from: Data(json.utf8))
        XCTAssertEqual(a.id, 7)
        XCTAssertEqual(a.query, "brand=rolex&currency=TWD")
        XCTAssertNil(a.lastNotifiedAt)
    }
}
