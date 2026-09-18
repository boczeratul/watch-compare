import XCTest
@testable import WatchCompareKit

final class SearchQueryTests: XCTestCase {
    func testDefaultSortMatchesWeb() {
        XCTAssertEqual(SearchQuery().effectiveSort, .newest)
        XCTAssertEqual(SearchQuery(text: "sub").effectiveSort, .relevance)
        var q = SearchQuery(text: "sub"); q.sort = .priceAsc
        XCTAssertEqual(q.effectiveSort, .priceAsc)
    }

    func testQueryItems() {
        var q = SearchQuery(text: " Submariner ")
        q.brands = ["tudor", "rolex"]
        q.conditions = [.new, .unworn]
        q.priceMin = 1000; q.priceMax = 2500.5
        q.yearMin = 2010
        q.diameterMax = 41
        q.hasBox = true
        q.dialColors = ["black"]
        let items = q.queryItems(currency: "TWD", page: 2, perPage: 30)
        let dict = Dictionary(items.map { ($0.name, $0.value ?? "") }, uniquingKeysWith: { a, _ in a })
        XCTAssertEqual(dict["currency"], "TWD")
        XCTAssertEqual(dict["q"], "Submariner")
        XCTAssertEqual(dict["brand"], "rolex,tudor")
        XCTAssertEqual(dict["condition"], "new,unworn")
        XCTAssertEqual(dict["price_min"], "1000")
        XCTAssertEqual(dict["price_max"], "2500.5")
        XCTAssertEqual(dict["year_min"], "2010")
        XCTAssertNil(dict["year_max"])
        XCTAssertEqual(dict["diameter_max"], "41")
        XCTAssertEqual(dict["box"], "true")
        XCTAssertNil(dict["papers"])
        XCTAssertEqual(dict["dial"], "black")
        XCTAssertEqual(dict["sort"], "relevance")
        XCTAssertEqual(dict["page"], "2")
        XCTAssertEqual(dict["per_page"], "30")
        XCTAssertEqual(items.first?.name, "currency")
    }

    func testActiveFilterCountAndClear() {
        var q = SearchQuery(text: "x")
        XCTAssertEqual(q.activeFilterCount, 0)
        q.brands = ["rolex"]; q.priceMax = 5; q.hasPapers = true; q.genders = [.men]; q.sort = .yearAsc
        XCTAssertEqual(q.activeFilterCount, 4)
        let cleared = q.clearingFilters()
        XCTAssertEqual(cleared.activeFilterCount, 0)
        XCTAssertEqual(cleared.text, "x")
        XCTAssertEqual(cleared.sort, .yearAsc)
        XCTAssertEqual(SearchQuery.brand("omega").brands, ["omega"])
        XCTAssertEqual(SearchQuery.reference("116610LN").queryItems(currency: "USD", page: 1, perPage: 1).first { $0.name == "ref" }?.value, "116610LN")
    }
}
