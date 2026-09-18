import SwiftUI

@MainActor
@Observable
final class SearchModel {
    var query: SearchQuery
    var items: [Listing] = []
    var facets: Facets = .empty
    var total = 0
    var page = 0
    var totalPages = 0
    var loading = false
    var error: String?

    private var currency = ""
    private var generation = 0

    init(query: SearchQuery) { self.query = query }

    var canLoadMore: Bool { page < totalPages && !loading }

    /// Restart from page 1 (query, sort or currency changed).
    func reload(api: APIClient, currency: String) async {
        self.currency = currency
        generation += 1
        let gen = generation
        items = []; total = 0; page = 0; totalPages = 0; error = nil
        await fetch(api: api, page: 1, generation: gen)
    }

    func loadMore(api: APIClient) async {
        guard canLoadMore else { return }
        await fetch(api: api, page: page + 1, generation: generation)
    }

    private func fetch(api: APIClient, page: Int, generation gen: Int) async {
        loading = true
        defer { loading = false }
        do {
            let r = try await api.searchListings(query, currency: currency, page: page)
            guard gen == generation else { return }
            if page == 1 { items = r.items } else { items.append(contentsOf: r.items) }
            facets = r.facets; total = r.total; self.page = r.page; totalPages = r.totalPages
        } catch {
            guard gen == generation else { return }
            self.error = error.localizedDescription
        }
    }
}

struct SearchResultsView: View {
    @Environment(\.api) private var api
    @Environment(AppSettings.self) private var settings
    @State private var model: SearchModel
    @State private var showFilters = false

    init(query: SearchQuery) { _model = State(initialValue: SearchModel(query: query)) }

    var body: some View {
        Group {
            if let err = model.error, model.items.isEmpty {
                ErrorView(message: err) { Task { await model.reload(api: api, currency: settings.currency) } }
            } else if model.items.isEmpty && !model.loading && model.page > 0 {
                ContentUnavailableView {
                    Label(L10n.t("search.noResults"), systemImage: "magnifyingglass")
                } description: {
                    Text(L10n.t("search.noResultsHint"))
                } actions: {
                    if model.query.activeFilterCount > 0 {
                        Button(L10n.t("search.clearAll")) { apply(model.query.clearingFilters()) }
                    }
                }
            } else {
                grid
            }
        }
        .navigationTitle(L10n.resultsTitle(count: model.total, query: model.query.text))
        .inlineNavigationTitle()
        .toolbar { toolbar }
        .sheet(isPresented: $showFilters) {
            FiltersSheet(query: model.query, facets: model.facets) { apply($0) }
        }
        .task(id: settings.currency) { await model.reload(api: api, currency: settings.currency) }
        .appDestinations()
    }

    private var grid: some View {
        ScrollView {
            LazyVGrid(columns: cardColumns, spacing: 16) {
                ForEach(model.items) { l in
                    NavigationLink(value: l) { ListingCardView(listing: l) }
                        .buttonStyle(.plain)
                        .onAppear { if l.id == model.items.last?.id { Task { await model.loadMore(api: api) } } }
                }
            }
            .padding()
            if model.loading { ProgressView().padding() }
            else if model.total > 0 {
                Text(L10n.t("search.showing", model.items.count.formatted() as NSString, model.total.formatted() as NSString))
                    .font(.footnote).foregroundStyle(.secondary).padding(.bottom)
            }
        }
    }

    @ToolbarContentBuilder
    private var toolbar: some ToolbarContent {
        ToolbarItemGroup(placement: .primaryAction) {
            Menu {
                Picker(L10n.t("search.sortBy"), selection: Binding(
                    get: { model.query.effectiveSort },
                    set: { var q = model.query; q.sort = $0; apply(q) }
                )) {
                    ForEach(SortKey.allCases) { Text(L10n.sort($0)).tag($0) }
                }
            } label: {
                Label(L10n.t("search.sortBy"), systemImage: "arrow.up.arrow.down")
            }
            Button {
                showFilters = true
            } label: {
                Label(L10n.t("search.filters"), systemImage: "line.3.horizontal.decrease.circle")
                    .symbolVariant(model.query.activeFilterCount > 0 ? .fill : .none)
            }
            .badge(model.query.activeFilterCount)
        }
    }

    private func apply(_ q: SearchQuery) {
        guard q != model.query else { return }
        model.query = q
        Task { await model.reload(api: api, currency: settings.currency) }
    }
}
