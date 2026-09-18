import XCTest
@testable import WatchCompareKit

final class RecentSearchesTests: XCTestCase {
    private var defaults: UserDefaults!
    private var store: RecentSearchesStore!

    override func setUp() {
        defaults = UserDefaults(suiteName: "RecentSearchesTests")!
        defaults.removePersistentDomain(forName: "RecentSearchesTests")
        store = RecentSearchesStore(defaults: defaults)
    }

    func testAddDedupesCaseInsensitivelyAndCaps() {
        store.add("Rolex")
        store.add("omega")
        XCTAssertEqual(store.add("ROLEX"), ["ROLEX", "omega"])
        XCTAssertEqual(store.add("   "), ["ROLEX", "omega"])
        for i in 0..<20 { store.add("q\(i)") }
        XCTAssertEqual(store.load().count, RecentSearchesStore.maxCount)
        XCTAssertEqual(store.load().first, "q19")
        XCTAssertEqual(store.remove("Q19").first, "q18")
        XCTAssertEqual(store.clear(), [])
        XCTAssertNil(defaults.object(forKey: RecentSearchesStore.key))
    }

    func testMatchesAndCompletion() {
        let recent = ["Submariner 116610LN", "Omega Speedmaster", "rolex daytona", "Seiko SKX"]
        XCTAssertEqual(RecentSearchesStore.matches(in: recent, partial: ""), recent)
        XCTAssertEqual(RecentSearchesStore.matches(in: recent, partial: "s"), ["Submariner 116610LN", "Seiko SKX", "Omega Speedmaster"])
        XCTAssertEqual(RecentSearchesStore.matches(in: recent, partial: "ROLEX"), ["rolex daytona"])
        XCTAssertEqual(RecentSearchesStore.matches(in: recent, partial: "rolex daytona"), [])
        XCTAssertEqual(RecentSearchesStore.matches(in: recent, partial: "zzz"), [])
        XCTAssertEqual(RecentSearchesStore.completion(in: RecentSearchesStore.matches(in: recent, partial: "sub"), for: "sub"), "Submariner 116610LN")
        XCTAssertNil(RecentSearchesStore.completion(in: RecentSearchesStore.matches(in: recent, partial: "daytona"), for: "daytona"))
        XCTAssertNil(RecentSearchesStore.completion(in: recent, for: " sub"))
        XCTAssertNil(RecentSearchesStore.completion(in: [], for: ""))
    }
}
