import SwiftUI

struct BrandsView: View {
    @Environment(\.api) private var api
    @State private var brands: [Brand] = []
    @State private var error: String?
    @State private var loaded = false
    @State private var filter = ""

    var body: some View {
        Group {
            if let error, brands.isEmpty {
                ErrorView(message: error) { Task { await load() } }
            } else if !loaded {
                ProgressView()
            } else {
                List(filtered) { b in
                    NavigationLink(value: SearchQuery.brand(b.slug)) {
                        HStack {
                            Text(b.name)
                            Spacer()
                            Text(L10n.t("brands.listings", b.listingCount.formatted() as NSString))
                                .font(.subheadline).foregroundStyle(.secondary)
                        }
                    }
                }
                .searchable(text: $filter)
                .overlay {
                    if filtered.isEmpty { ContentUnavailableView.search(text: filter) }
                }
            }
        }
        .navigationTitle(L10n.t("brands.title"))
        .task { if !loaded { await load() } }
        .refreshable { await load() }
        .appDestinations()
    }

    private var filtered: [Brand] {
        let f = filter.trimmingCharacters(in: .whitespaces)
        let active = brands.filter { $0.listingCount > 0 }
        return f.isEmpty ? active : active.filter { $0.name.localizedCaseInsensitiveContains(f) }
    }

    private func load() async {
        do {
            brands = try await api.brands()
            error = nil
        } catch {
            self.error = error.localizedDescription
        }
        loaded = true
    }
}
