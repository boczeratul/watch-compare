import Foundation
import Observation
#if canImport(AppTrackingTransparency)
import AppTrackingTransparency
#endif

/// App Tracking Transparency: asks the visitor once whether the app may track them, and exposes
/// the answer. Nothing in the app reads an advertising identifier or shares data with a third
/// party unless `isAuthorized` is true; wire any analytics or attribution SDK behind that check.
@MainActor
@Observable
public final class TrackingConsent {
    public enum Status: String, Sendable {
        case notDetermined, restricted, denied, authorized
        /// The framework is missing on this platform (macOS unit-test host without ATT).
        case unavailable
    }

    /// Launch argument / user default that suppresses the prompt (screenshot and UI-test runs).
    public static let skipPromptKey = "WC_SKIP_TRACKING_PROMPT"

    public private(set) var status: Status
    private var requested = false

    public init() {
        status = Self.currentStatus()
    }

    public var isAuthorized: Bool { status == .authorized }

    /// The system prompt can only be shown once per install and only while the app is active,
    /// so call this from the scene's `.active` transition. Later calls are no-ops.
    public func requestIfNeeded() async {
        guard !requested, status == .notDetermined, !UserDefaults.standard.bool(forKey: Self.skipPromptKey) else { return }
        requested = true
        #if canImport(AppTrackingTransparency)
        let result = await ATTrackingManager.requestTrackingAuthorization()
        status = Self.map(result)
        #endif
    }

    /// Re-reads the system status, e.g. after the visitor returns from the Settings app.
    public func refresh() {
        status = Self.currentStatus()
    }

    public static func currentStatus() -> Status {
        #if canImport(AppTrackingTransparency)
        return map(ATTrackingManager.trackingAuthorizationStatus)
        #else
        return .unavailable
        #endif
    }

    #if canImport(AppTrackingTransparency)
    private static func map(_ s: ATTrackingManager.AuthorizationStatus) -> Status {
        switch s {
        case .notDetermined: return .notDetermined
        case .restricted: return .restricted
        case .denied: return .denied
        case .authorized: return .authorized
        @unknown default: return .notDetermined
        }
    }
    #endif
}
