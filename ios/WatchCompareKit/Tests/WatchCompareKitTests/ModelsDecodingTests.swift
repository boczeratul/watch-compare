import XCTest
@testable import WatchCompareKit

final class ModelsDecodingTests: XCTestCase {
    // Shaped like the Go API: nil slices marshal as null, dates are RFC 3339 with or without fraction.
    let searchJSON = """
    {"items":[{"id":42,"source":"jackroad","sourceName":"Jackroad","externalId":"x1","url":"https://example.com/1",
      "title":"Rolex Submariner 116610LN","brand":"rolex","brandName":"Rolex","referenceNumber":"116610LN",
      "condition":"very_good","year":2015,"caseDiameterMm":40,"dialColor":"black","movement":"automatic","gender":"men",
      "hasBox":true,"hasPapers":false,"price":1650000,"priceExclTax":1500000,"currency":"JPY","priceUsd":10200.5,
      "locationCountry":"JP","locationCity":"Tokyo","imageUrls":null,"isActive":true,
      "firstSeenAt":"2026-09-01T02:00:00.123456Z","lastSeenAt":"2026-09-16T02:00:00+08:00"},
     {"id":43,"source":"ebay","sourceName":"eBay","externalId":"x2","url":"https://example.com/2","title":"Mystery",
      "condition":"mint","movement":"kinetic","gender":"kids","imageUrls":["https://img/1.jpg"],"isActive":false,
      "firstSeenAt":"2026-09-01T02:00:00Z","lastSeenAt":"2026-09-02T02:00:00Z"}],
     "total":2,"page":1,"perPage":30,"totalPages":1,
     "facets":{"brands":[{"key":"rolex","label":"Rolex","count":2}],"sources":null,"conditions":[],"movements":null,
               "countries":null,"dialColors":null,"years":null,"priceMinUsd":100}}
    """

    func testSearchResultDecodesNullArraysAsEmpty() throws {
        let r = try JSONDecoder.api.decode(SearchResult.self, from: Data(searchJSON.utf8))
        XCTAssertEqual(r.items.count, 2)
        XCTAssertEqual(r.items[0].imageUrls, [])
        XCTAssertEqual(r.items[1].imageUrls, ["https://img/1.jpg"])
        XCTAssertEqual(r.facets.brands.first?.label, "Rolex")
        XCTAssertEqual(r.facets.sources, [])
        XCTAssertEqual(r.facets.years, [])
        XCTAssertEqual(r.facets.priceMinUsd, 100)
        XCTAssertNil(r.facets.priceMaxUsd)
    }

    func testUnknownEnumValuesFallBackToUnknown() throws {
        let r = try JSONDecoder.api.decode(SearchResult.self, from: Data(searchJSON.utf8))
        XCTAssertEqual(r.items[0].condition, .veryGood)
        XCTAssertEqual(r.items[1].condition, .unknown)
        XCTAssertEqual(r.items[1].movement, .unknown)
        XCTAssertEqual(r.items[1].gender, .unknown)
    }

    func testDatesWithFractionAndOffset() throws {
        let r = try JSONDecoder.api.decode(SearchResult.self, from: Data(searchJSON.utf8))
        let first = r.items[0].firstSeenAt
        XCTAssertEqual(first.timeIntervalSince1970, Date.parseRFC3339("2026-09-01T02:00:00Z")!.timeIntervalSince1970 + 0.123456, accuracy: 0.001)
        let last = r.items[0].lastSeenAt
        XCTAssertEqual(last, Date.parseRFC3339("2026-09-15T18:00:00Z"))
    }

    func testComparisonPricePrefersTaxFreeAmount() throws {
        let r = try JSONDecoder.api.decode(SearchResult.self, from: Data(searchJSON.utf8))
        XCTAssertEqual(r.items[0].comparisonPrice, 1_500_000)
        XCTAssertEqual(r.items[0].location, "Tokyo, JP")
        XCTAssertNil(r.items[1].location)
    }

    func testEmptyBodyObjectsDecode() throws {
        let r = try JSONDecoder.api.decode(SearchResult.self, from: Data("{}".utf8))
        XCTAssertEqual(r.items, [])
        XCTAssertEqual(r.total, 0)
        let items = try JSONDecoder.api.decode(ItemsResponse<Brand>.self, from: Data(#"{"items":null}"#.utf8))
        XCTAssertEqual(items.items, [])
        let stats = try JSONDecoder.api.decode(Stats.self, from: Data(#"{"activeListings":5,"brands":2,"lastCrawlAt":null,"sources":[{"key":"a","name":"A","count":0},{"key":"b","name":"B","count":3}]}"#.utf8))
        XCTAssertNil(stats.lastCrawlAt)
        XCTAssertEqual(stats.activeSourceCount, 1)
    }
}
