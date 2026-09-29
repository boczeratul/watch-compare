import Foundation
import Observation
#if canImport(OneSignalFramework)
import OneSignalFramework
#endif

/// What a tapped alert notification should open: the one new listing, or the alert's search.
public struct OpenedNotification: Identifiable, Hashable, Sendable {
    public let id = UUID()
    public var listingID: Int64?
    public var query: String?
}

/// Push alerts through OneSignal. The device gets a random subscriber id (UserDefaults, shared
/// by nothing else), logs in to OneSignal with it as its external_id and saves alerts under the
/// same id; the backend's crawler job pushes new matches to it. Without an app id in Info.plist
/// (`OneSignalAppID`, from the `ONESIGNAL_APP_ID` build setting), and on macOS where the SDK is
/// not linked, push stays off and the alert UI is hidden.
@MainActor
@Observable
public final class PushCenter {
    public static let shared = PushCenter()
    nonisolated public static let subscriberKey = "wc_push_subscriber"
    nonisolated public static let appIDInfoKey = "OneSignalAppID"

    /// True once the SDK has been initialized with an app id.
    public private(set) var isAvailable = false
    /// Set when a notification is tapped; RootView presents it and clears it.
    public var opened: OpenedNotification?

    private let defaults: UserDefaults

    public init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    /// The id alerts were saved under, or nil before the first alert.
    public var existingSubscriberID: String? { defaults.string(forKey: Self.subscriberKey) }

    private func subscriberID() -> String {
        if let id = existingSubscriberID { return id }
        let id = UUID().uuidString.lowercased()
        defaults.set(id, forKey: Self.subscriberKey)
        return id
    }

    /// Call once at launch (from the app delegate) before any notification can be tapped.
    public func configure(appID: String?, launchOptions: [AnyHashable: Any]?) {
        #if canImport(OneSignalFramework)
        guard let appID = appID?.trimmingCharacters(in: .whitespaces), !appID.isEmpty, !isAvailable else { return }
        OneSignal.initialize(appID, withLaunchOptions: launchOptions)
        OneSignal.Notifications.addClickListener(clickHandler)
        if let id = existingSubscriberID { OneSignal.login(id) }
        isAvailable = true
        #endif
    }

    public enum Permission: Sendable, Equatable {
        case granted(subscriberID: String)
        case denied
        case unavailable
    }

    /// Asks for notification permission if needed (offering Settings when it was declined before)
    /// and links this device to its subscriber id.
    public func enable() async -> Permission {
        #if canImport(OneSignalFramework)
        guard isAvailable else { return .unavailable }
        var accepted = OneSignal.Notifications.permission
        if !accepted {
            accepted = await withCheckedContinuation { (continuation: CheckedContinuation<Bool, Never>) in
                // @Sendable: the SDK may call back off the main thread.
                OneSignal.Notifications.requestPermission({ @Sendable granted in continuation.resume(returning: granted) }, fallbackToSettings: true)
            }
        }
        guard accepted else { return .denied }
        let id = subscriberID()
        OneSignal.login(id)
        return .granted(subscriberID: id)
        #else
        return .unavailable
        #endif
    }
}

#if canImport(OneSignalFramework)
@MainActor private let clickHandler = ClickHandler()

/// Routes a tapped notification's data (set by backend/internal/alerts) into `PushCenter.opened`.
private final class ClickHandler: NSObject, OSNotificationClickListener {
    func onClick(event: OSNotificationClickEvent) {
        let data = event.notification.additionalData ?? [:]
        let listingID = (data["listingId"] as? NSNumber)?.int64Value
        let query = data["query"] as? String
        Task { @MainActor in
            PushCenter.shared.opened = OpenedNotification(listingID: listingID, query: query)
        }
    }
}
#endif
