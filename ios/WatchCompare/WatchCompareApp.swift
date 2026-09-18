import SwiftUI
import WatchCompareKit

@main
struct WatchCompareApp: App {
    var body: some Scene {
        WindowGroup {
            RootView(apiBaseURL: AppConfig.apiBaseURL)
        }
    }
}

enum AppConfig {
    /// `APIBaseURL` in Info.plist is filled from the `API_BASE_URL` build setting
    /// (Debug: http://localhost:8080, Release: the Cloud Run URL). A `WC_API_BASE_URL`
    /// launch argument / user default overrides it, e.g. to point a device at a LAN backend.
    static var apiBaseURL: URL {
        let override = UserDefaults.standard.string(forKey: "WC_API_BASE_URL")
        let configured = Bundle.main.object(forInfoDictionaryKey: "APIBaseURL") as? String
        for candidate in [override, configured] {
            if let s = candidate?.trimmingCharacters(in: .whitespaces), !s.isEmpty, let url = URL(string: s), url.scheme != nil {
                return url
            }
        }
        return URL(string: "http://localhost:8080")!
    }
}
