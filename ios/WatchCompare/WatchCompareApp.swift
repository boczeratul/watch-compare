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
    /// `APIBaseURL` in Info.plist is filled from the project-level `API_BASE_URL` build setting:
    /// the production Cloud Run URL for every configuration, so local development runs against
    /// real data. A `WC_API_BASE_URL` launch argument / user default overrides it, e.g.
    /// `-WC_API_BASE_URL http://localhost:8080` while working on the backend.
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
