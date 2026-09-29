import SwiftUI
import WatchCompareKit

@main
struct WatchCompareApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate

    var body: some Scene {
        WindowGroup {
            RootView(apiBaseURL: AppConfig.apiBaseURL)
        }
    }
}

/// OneSignal must be initialized in `didFinishLaunching` so a notification that launched the app
/// is delivered to the click handler.
final class AppDelegate: NSObject, UIApplicationDelegate {
    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil) -> Bool {
        let options = launchOptions.map { Dictionary(uniqueKeysWithValues: $0.map { (AnyHashable($0.key), $0.value) }) }
        PushCenter.shared.configure(appID: AppConfig.oneSignalAppID, launchOptions: options)
        return true
    }
}

enum AppConfig {
    /// `OneSignalAppID` in Info.plist, from the `ONESIGNAL_APP_ID` build setting. Empty disables push alerts.
    static var oneSignalAppID: String? {
        Bundle.main.object(forInfoDictionaryKey: PushCenter.appIDInfoKey) as? String
    }

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
