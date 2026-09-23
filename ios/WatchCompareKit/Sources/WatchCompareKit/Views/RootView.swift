import SwiftUI

public enum Tab: Hashable { case search, brands, settings }

/// Entry point used by the app target. Owns the API client and settings for the whole app.
public struct RootView: View {
    private let api: APIClient
    @State private var settings: AppSettings
    @State private var tracking = TrackingConsent()
    @State private var tab: Tab = .search
    @Environment(\.scenePhase) private var scenePhase

    public init(apiBaseURL: URL) {
        api = APIClient(baseURL: apiBaseURL)
        _settings = State(initialValue: AppSettings())
    }

    public var body: some View {
        TabView(selection: $tab) {
            HomeView()
                .tabItem { Label(L10n.t("nav.search"), systemImage: "magnifyingglass") }
                .tag(Tab.search)
            NavigationStack { BrandsView() }
                .tabItem { Label(L10n.t("nav.brands"), systemImage: "list.bullet") }
                .tag(Tab.brands)
            NavigationStack { SettingsView() }
                .tabItem { Label(L10n.t("nav.settings"), systemImage: "gearshape") }
                .tag(Tab.settings)
        }
        .environment(\.api, api)
        .environment(settings)
        .environment(tracking)
        .task { await settings.loadRates(using: api) }
        // ATT requires the app to be active when the prompt appears; the short delay lets the
        // first screen render so the request is not the very first thing the visitor sees.
        .onChange(of: scenePhase, initial: true) { _, phase in
            guard phase == .active else { return }
            tracking.refresh()
            Task {
                try? await Task.sleep(for: .seconds(1))
                await tracking.requestIfNeeded()
            }
        }
    }
}

/// Registers the navigation destinations shared by every stack.
struct AppDestinations: ViewModifier {
    func body(content: Content) -> some View {
        content
            .navigationDestination(for: SearchQuery.self) { SearchResultsView(query: $0) }
            .navigationDestination(for: Listing.self) { ListingDetailView(listing: $0) }
            .navigationDestination(for: ListingRef.self) { ListingDetailView(id: $0.id) }
    }
}

/// Navigate to a listing by id when only the id is known (e.g. from a deep link).
public struct ListingRef: Hashable, Sendable {
    public var id: Int64
    public init(id: Int64) { self.id = id }
}

extension View {
    func appDestinations() -> some View { modifier(AppDestinations()) }
}
