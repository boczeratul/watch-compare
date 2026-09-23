import XCTest

/// Drives the app through its main screens against the live API and saves full-resolution PNGs.
/// Run through ios/scripts/screenshots.sh, which sets the language, region, currency and output
/// directory per locale via TEST_RUNNER_ environment variables.
@MainActor
final class ScreenshotTests: XCTestCase {
    private var app: XCUIApplication!
    private var outputDir: URL!

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    private func launch() throws {
        let env = ProcessInfo.processInfo.environment
        let locale = env["SCREENSHOT_LOCALE"] ?? "en"
        let language = env["SCREENSHOT_LANGUAGE"] ?? "en"
        let region = env["SCREENSHOT_REGION"] ?? "US"
        let currency = env["SCREENSHOT_CURRENCY"] ?? "USD"
        let base = env["SCREENSHOT_DIR"] ?? NSTemporaryDirectory()
        outputDir = URL(fileURLWithPath: base).appending(path: locale)
        try FileManager.default.createDirectory(at: outputDir, withIntermediateDirectories: true)

        app = XCUIApplication()
        app.launchArguments += [
            "-AppleLanguages", "(\(language))",
            "-AppleLocale", "\(language)_\(region)",
            "-wc_currency", currency,          // AppSettings reads this UserDefaults key
            "-WC_SKIP_TRACKING_PROMPT", "YES", // no App Tracking Transparency alert during captures
            "-WC_DISABLE_ANALYTICS", "YES",    // screenshot runs must not show up in Amplitude
            "-wc_recent_searches", "(\"Submariner 116610LN\", \"Omega Speedmaster\", \"Daytona\")",
        ]
        app.launch()
    }

    func testCaptureScreens() throws {
        try launch()

        // 1. Home: wait for live content (the "newest" grid) and its first photo before shooting.
        XCTAssertTrue(app.descendants(matching: .any)["home.newest"].waitForExistence(timeout: 30), "home grid did not load")
        waitForImages()
        try snap("01-home")

        // 2. Search results.
        let field = app.textFields["home.searchField"]
        XCTAssertTrue(field.waitForExistence(timeout: 10), "home search field missing")
        field.tap()
        field.typeText("Submariner\n")
        let results = app.descendants(matching: .any)["search.results"]
        XCTAssertTrue(results.waitForExistence(timeout: 30), "results grid did not load")
        XCTAssertTrue(results.buttons.firstMatch.waitForExistence(timeout: 30), "no result cards")
        waitForImages()
        try snap("02-search")

        // 3. Filters sheet.
        let filters = app.buttons["search.filters"]
        XCTAssertTrue(filters.waitForExistence(timeout: 10), "filters button missing")
        filters.tap()
        let apply = app.buttons["search.apply"]
        XCTAssertTrue(apply.waitForExistence(timeout: 10), "filters sheet did not open")
        waitForNetwork(seconds: 1)
        try snap("03-filters")
        apply.tap()

        // 4. Listing detail.
        XCTAssertTrue(results.buttons.firstMatch.waitForExistence(timeout: 30), "results gone after filters")
        results.buttons.firstMatch.tap()
        let cta = app.descendants(matching: .any)["listing.viewOnSource"]
        XCTAssertTrue(cta.waitForExistence(timeout: 30), "detail did not load")
        waitForImages()
        try snap("04-listing")

        // 5. Brands tab. iPhone exposes a TabBar; the iPad floating tab bar only exposes plain
        // buttons whose identifier is the tab's SF Symbol name.
        let tabBarItem = app.tabBars.buttons.element(boundBy: 1)
        let brandsTab = tabBarItem.waitForExistence(timeout: 3) ? tabBarItem : app.buttons["list.bullet"].firstMatch
        XCTAssertTrue(brandsTab.waitForExistence(timeout: 10), "brands tab missing")
        brandsTab.tap()
        XCTAssertTrue(app.descendants(matching: .any)["brands.list"].waitForExistence(timeout: 30))
        XCTAssertTrue(app.cells.firstMatch.waitForExistence(timeout: 30), "brands did not load")
        waitForNetwork(seconds: 1)
        try snap("05-brands")
    }

    // MARK: Helpers

    /// Photos arrive after the data: wait for the first decoded image, then a beat for the rest.
    private func waitForImages() {
        _ = app.images["image.loaded"].firstMatch.waitForExistence(timeout: 20)
        waitForNetwork(seconds: 3)
    }

    private func waitForNetwork(seconds: TimeInterval) {
        _ = XCTWaiter.wait(for: [expectation(description: "settle")], timeout: seconds)
    }

    private func snap(_ name: String) throws {
        let shot = XCUIScreen.main.screenshot()
        let url = outputDir.appending(path: "\(name).png")
        try shot.pngRepresentation.write(to: url)
        let attachment = XCTAttachment(screenshot: shot)
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
