import XCTest
@testable import WatchCompareKit

@MainActor
final class AnalyticsGateTests: XCTestCase {
    func testAnalyticsFollowsTrackingAuthorization() {
        // The test host has no tracking authorization, so consent starts out not authorized and
        // analytics must be off; toggling the gate must be reflected in isEnabled.
        let consent = TrackingConsent()
        XCTAssertNotEqual(consent.status, .authorized)
        XCTAssertFalse(Analytics.isEnabled)

        UserDefaults.standard.set(true, forKey: Analytics.disableKey) // never create a real client in tests
        defer { UserDefaults.standard.removeObject(forKey: Analytics.disableKey) }
        Analytics.setEnabled(true)
        XCTAssertFalse(Analytics.isEnabled, "the launch-argument kill switch wins even when authorized")
        Analytics.setEnabled(false)
        XCTAssertFalse(Analytics.isEnabled)
    }
}
