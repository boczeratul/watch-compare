import SwiftUI

@MainActor
@Observable
final class HomeModel {
    var stats: Stats?
    var popularBrands: [Brand] = []
    var newest: [Listing] = []
    var recent: [String] = []
    var loaded = false

    private let store = RecentSearchesStore()

    func load(api: APIClient) async {
        recent = store.load()
        async let stats = api.stats()
        async let brands = api.brands()
        async let newest = api.searchListings(SearchQuery(), currency: "USD", page: 1, perPage: 8)
        self.stats = try? await stats
        popularBrands = Array(((try? await brands) ?? []).filter { $0.listingCount > 0 }.prefix(18))
        self.newest = (try? await newest)?.items ?? []
        loaded = true
    }

    func remember(_ query: String) { recent = store.add(query) }
    func forget(_ query: String) { recent = store.remove(query) }
    func clearRecent() { recent = store.clear() }
}

struct HomeView: View {
    @Environment(\.api) private var api
    @Environment(AppSettings.self) private var settings
    @State private var model = HomeModel()
    @State private var text = ""
    @FocusState private var searchFocused: Bool
    @State private var path = NavigationPath()

    var body: some View {
        NavigationStack(path: $path) {
            content
                .navigationTitle("WatchCompare")
                .task { if !model.loaded { await model.load(api: api) } }
                .refreshable { await model.load(api: api) }
                .appDestinations()
        }
    }

    private var content: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                hero
                searchBox
                if !model.recent.isEmpty && !searchFocused { recentSection }
                if !model.popularBrands.isEmpty { brandsSection }
                if !model.newest.isEmpty { newestSection }
                if !model.loaded { ProgressView().frame(maxWidth: .infinity).padding() }
            }
            .padding()
        }
    }

    private var hero: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L10n.t("home.heroTitle")).font(.title.weight(.bold))
            if let s = model.stats {
                Text(L10n.t("home.heroSubtitle", s.activeListings.formatted() as NSString, s.activeSourceCount))
                    .font(.subheadline).foregroundStyle(.secondary)
                HStack(spacing: 16) {
                    stat(s.activeListings.formatted(), L10n.t("home.statListings"))
                    stat(String(s.brands), L10n.t("home.statBrands"))
                    stat(String(s.activeSourceCount), L10n.t("home.statSources"))
                }
                .padding(.top, 4)
                if let d = s.lastCrawlAt {
                    Text(L10n.t("home.lastUpdated", d.formatted(date: .abbreviated, time: .omitted) as NSString))
                        .font(.caption).foregroundStyle(.tertiary)
                }
            }
        }
    }

    private func stat(_ value: String, _ label: String) -> some View {
        HStack(spacing: 4) {
            Text(value).font(.subheadline.weight(.semibold))
            Text(label).font(.subheadline).foregroundStyle(.secondary)
        }
    }

    private var recentSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                SectionHeader(title: L10n.t("home.recentSearches"))
                Button(L10n.t("home.clearRecent")) { model.clearRecent() }.font(.subheadline)
            }
            FlowLayout(spacing: 8) {
                ForEach(model.recent, id: \.self) { q in
                    Button { submit(q) } label: {
                        Label(q, systemImage: "clock").font(.subheadline)
                            .padding(.horizontal, 10).padding(.vertical, 6)
                            .background(Color.secondary.opacity(0.12), in: Capsule())
                    }
                    .buttonStyle(.plain)
                    .contextMenu {
                        Button(role: .destructive) { model.forget(q) } label: {
                            Label(L10n.t("home.removeRecent"), systemImage: "trash")
                        }
                    }
                }
            }
        }
    }

    private var brandsSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionHeader(title: L10n.t("home.popularBrands"))
            FlowLayout(spacing: 8) {
                ForEach(model.popularBrands) { b in
                    NavigationLink(value: SearchQuery.brand(b.slug)) {
                        Text(b.name).font(.subheadline)
                            .padding(.horizontal, 10).padding(.vertical, 6)
                            .background(Color.secondary.opacity(0.12), in: Capsule())
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private var newestSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                SectionHeader(title: L10n.t("home.newest"))
                NavigationLink(L10n.t("home.viewAll"), value: SearchQuery()).font(.subheadline)
            }
            LazyVGrid(columns: cardColumns, spacing: 16) {
                ForEach(model.newest) { l in
                    NavigationLink(value: l) { ListingCardView(listing: l) }.buttonStyle(.plain)
                }
            }
            .accessibilityIdentifier("home.newest")
        }
    }

    private var searchBox: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 8) {
                Image(systemName: "magnifyingglass").foregroundStyle(.secondary)
                TextField(L10n.t("home.searchPlaceholder"), text: $text)
                    .searchFieldStyle()
                    .submitLabel(.search)
                    .focused($searchFocused)
                    .onSubmit { submit(text) }
                    .accessibilityIdentifier("home.searchField")
                if !text.isEmpty {
                    Button { text = "" } label: {
                        Image(systemName: "xmark.circle.fill").foregroundStyle(.secondary)
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 12).padding(.vertical, 11)
            .background(Color.secondary.opacity(0.12), in: RoundedRectangle(cornerRadius: 12))
            if searchFocused {
                let matches = RecentSearchesStore.matches(in: model.recent, partial: text)
                if !matches.isEmpty {
                    VStack(alignment: .leading, spacing: 0) {
                        ForEach(matches, id: \.self) { q in
                            Button { submit(q) } label: {
                                Label(q, systemImage: "clock")
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .padding(.vertical, 9).padding(.horizontal, 12)
                            }
                            .buttonStyle(.plain)
                            Divider()
                        }
                    }
                    .background(Color.secondary.opacity(0.06), in: RoundedRectangle(cornerRadius: 12))
                }
            }
        }
    }

    private func submit(_ raw: String) {
        let q = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        searchFocused = false
        if !q.isEmpty { model.remember(q) }
        text = ""
        path.append(SearchQuery(text: q))
    }
}

/// Wrapping horizontal layout for chips.
struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let width = proposal.width ?? .infinity
        return arrange(subviews: subviews, width: width).size
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let result = arrange(subviews: subviews, width: bounds.width)
        for (index, origin) in result.origins.enumerated() {
            subviews[index].place(at: CGPoint(x: bounds.minX + origin.x, y: bounds.minY + origin.y), proposal: .unspecified)
        }
    }

    private func arrange(subviews: Subviews, width: CGFloat) -> (size: CGSize, origins: [CGPoint]) {
        var origins: [CGPoint] = []
        var x: CGFloat = 0, y: CGFloat = 0, rowHeight: CGFloat = 0, maxX: CGFloat = 0
        for view in subviews {
            let size = view.sizeThatFits(.unspecified)
            if x > 0, x + size.width > width {
                x = 0; y += rowHeight + spacing; rowHeight = 0
            }
            origins.append(CGPoint(x: x, y: y))
            x += size.width + spacing
            rowHeight = max(rowHeight, size.height)
            maxX = max(maxX, x - spacing)
        }
        return (CGSize(width: maxX, height: y + rowHeight), origins)
    }
}
