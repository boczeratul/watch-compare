import SwiftUI
import Charts

@MainActor
@Observable
final class ListingDetailModel {
    var listing: Listing?
    var similar: [Listing] = []
    var history: [PricePoint] = []
    var error: String?
    var notFound = false
    var loaded = false

    let id: Int64

    init(id: Int64, listing: Listing? = nil) {
        self.id = id
        self.listing = listing
    }

    func load(api: APIClient, currency: String) async {
        if let l = listing { Analytics.trackViewListing(l, currency: currency) }
        async let fresh = api.listing(id: id)
        async let similar = api.similar(id: id)
        async let history = api.priceHistory(id: id)
        do {
            let l = try await fresh
            if listing == nil { Analytics.trackViewListing(l, currency: currency) }
            listing = l
        } catch let e as APIError where e.isNotFound {
            notFound = listing == nil
        } catch {
            if listing == nil { self.error = error.localizedDescription }
        }
        self.similar = (try? await similar) ?? []
        self.history = (try? await history) ?? []
        loaded = true
    }

    /// Percent saved against the next-cheapest offer with a USD price (web `saveVs`).
    var savePercent: Int {
        guard let mine = listing?.priceUsd, let other = similar.first(where: { $0.priceUsd != nil })?.priceUsd, other > mine else { return 0 }
        return Int(((1 - mine / other) * 100).rounded())
    }
}

struct ListingDetailView: View {
    @Environment(\.api) private var api
    @Environment(AppSettings.self) private var settings
    @State private var model: ListingDetailModel

    init(listing: Listing) { _model = State(initialValue: ListingDetailModel(id: listing.id, listing: listing)) }
    init(id: Int64) { _model = State(initialValue: ListingDetailModel(id: id)) }

    var body: some View {
        Group {
            if let l = model.listing {
                detail(l)
            } else if model.notFound {
                ContentUnavailableView(L10n.t("listing.notFound"), systemImage: "questionmark.circle", description: Text(L10n.t("listing.notFoundBody")))
            } else if let err = model.error {
                ErrorView(message: err) { Task { await model.load(api: api, currency: settings.currency) } }
            } else {
                ProgressView()
            }
        }
        .navigationTitle(model.listing?.brandName ?? "")
        .inlineNavigationTitle()
        .toolbar {
            if let url = model.listing?.externalURL {
                ToolbarItem(placement: .primaryAction) { ShareLink(item: url) }
            }
        }
        .task { if !model.loaded { await model.load(api: api, currency: settings.currency) } }
        .appDestinations()
    }

    private func detail(_ l: Listing) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                if !l.isActive {
                    Label(L10n.t("listing.inactive", l.sourceName as NSString), systemImage: "exclamationmark.triangle")
                        .font(.footnote).padding(10)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(Color.orange.opacity(0.15), in: RoundedRectangle(cornerRadius: 8))
                }
                ImageGallery(urls: l.displayImageUrls.compactMap(URL.init(string:)))
                header(l)
                priceBox(l)
                specs(l)
                if !model.similar.isEmpty { similarSection }
                if model.history.count >= 2 { PriceHistorySection(points: model.history) }
                if let d = l.description, !d.isEmpty {
                    VStack(alignment: .leading, spacing: 8) {
                        SectionHeader(title: L10n.t("listing.description"))
                        Text(d).font(.subheadline).foregroundStyle(.secondary)
                    }
                }
            }
            .padding()
        }
    }

    private func header(_ l: Listing) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                SourceBadge(name: l.sourceName)
                Label(sellerLabel(l), systemImage: l.sellerType == "private" ? "person" : "storefront")
                    .font(.caption).foregroundStyle(.secondary)
            }
            if let b = l.brandName { Text(b.uppercased()).font(.caption.weight(.semibold)).foregroundStyle(.secondary) }
            Text(l.title).font(.title3.weight(.bold))
            if let loc = l.location { Label(loc, systemImage: "mappin").font(.subheadline).foregroundStyle(.secondary) }
        }
    }

    private func sellerLabel(_ l: Listing) -> String {
        if let s = l.sellerName, !s.isEmpty, s != l.sourceName { return s }
        return l.sellerType == "private" ? L10n.t("listing.private") : L10n.t("listing.dealer")
    }

    private func priceBox(_ l: Listing) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            PriceText(listing: l, showOriginal: l.priceExclTax == nil, font: .title.weight(.bold))
            if let excl = l.priceExclTax, let incl = l.price, let c = l.currency {
                Divider()
                row(L10n.t("listing.priceExclTax"), settings.format(excl, currency: c), bold: true)
                row(L10n.t("listing.priceInclTax"), settings.format(incl, currency: c))
                Text(L10n.t("listing.taxNote")).font(.caption2).foregroundStyle(.secondary)
            }
            if let ship = l.shippingPrice, let c = l.currency {
                Text("\(L10n.t("listing.shipping")): \(settings.format(ship, currency: c))").font(.caption).foregroundStyle(.secondary)
            }
            if model.savePercent > 0 {
                Text(L10n.t("listing.saveVs", model.savePercent)).font(.subheadline.weight(.medium)).foregroundStyle(.green)
            }
            if let url = l.externalURL {
                Link(destination: url) {
                    Label(L10n.t("listing.viewOnSource", l.sourceName as NSString), systemImage: "arrow.up.right.square")
                        .frame(maxWidth: .infinity).padding(.vertical, 6)
                }
                .buttonStyle(.borderedProminent)
                .accessibilityIdentifier("listing.viewOnSource")
            }
        }
        .padding()
        .background(Color.secondary.opacity(0.08), in: RoundedRectangle(cornerRadius: 12))
    }

    private func row(_ k: String, _ v: String, bold: Bool = false) -> some View {
        HStack { Text(k).foregroundStyle(.secondary); Spacer(); Text(v).fontWeight(bold ? .semibold : .regular) }.font(.subheadline)
    }

    private func specs(_ l: Listing) -> some View {
        let yesNo: (Bool?) -> String = { $0 == nil ? "—" : ($0! ? L10n.t("listing.yes") : L10n.t("listing.no")) }
        let rows: [(String, String, SearchQuery?)] = [
            (L10n.t("listing.brand"), l.brandName ?? "—", l.brand.map(SearchQuery.brand)),
            (L10n.t("listing.model"), l.model.nonEmpty ?? "—", nil),
            (L10n.t("listing.reference"), l.referenceNumber.nonEmpty ?? "—", l.referenceNumber.nonEmpty.map(SearchQuery.reference)),
            (L10n.t("listing.condition"), L10n.condition(l.condition), nil),
            (L10n.t("listing.year"), l.year.map(String.init) ?? "—", nil),
            (L10n.t("listing.diameter"), l.caseDiameterMm.map { "\($0.formatted()) mm" } ?? "—", nil),
            (L10n.t("listing.material"), l.caseMaterial.nonEmpty ?? "—", nil),
            (L10n.t("listing.dialColor"), l.dialColor.nonEmpty.map(L10n.dial) ?? "—", dialQuery(l)),
            (L10n.t("listing.movement"), L10n.movement(l.movement), nil),
            (L10n.t("listing.gender"), L10n.gender(l.gender), nil),
            (L10n.t("listing.boxPapers"), "\(yesNo(l.hasBox)) / \(yesNo(l.hasPapers))", nil),
            (L10n.t("listing.location"), l.location ?? "—", nil),
            (L10n.t("listing.firstSeen"), l.firstSeenAt.formatted(date: .abbreviated, time: .omitted), nil),
            (L10n.t("listing.lastSeen"), l.lastSeenAt.formatted(date: .abbreviated, time: .omitted), nil),
        ]
        return VStack(alignment: .leading, spacing: 8) {
            SectionHeader(title: L10n.t("listing.details"))
            VStack(spacing: 0) {
                ForEach(Array(rows.enumerated()), id: \.offset) { i, r in
                    HStack(alignment: .top) {
                        Text(r.0).foregroundStyle(.secondary).frame(width: 130, alignment: .leading)
                        if let q = r.2 {
                            NavigationLink(value: q) { Text(r.1).underline().frame(maxWidth: .infinity, alignment: .leading) }.buttonStyle(.plain)
                        } else {
                            Text(r.1).frame(maxWidth: .infinity, alignment: .leading)
                        }
                    }
                    .font(.subheadline).padding(.vertical, 6)
                    if i < rows.count - 1 { Divider() }
                }
            }
        }
    }

    private func dialQuery(_ l: Listing) -> SearchQuery? {
        guard let d = l.dialColor.nonEmpty else { return nil }
        var q = SearchQuery(); q.dialColors = [d]
        if let b = l.brand { q.brands = [b] }
        return q
    }

    private var similarSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionHeader(title: L10n.t("listing.compareTitle"))
            let cheapest = model.similar.filter { $0.priceUsd != nil }.min { ($0.priceUsd ?? 0) < ($1.priceUsd ?? 0) }?.id
            VStack(spacing: 10) {
                ForEach(model.similar) { s in
                    NavigationLink(value: s) { ListingRowView(listing: s, highlight: s.id == cheapest) }.buttonStyle(.plain)
                    Divider()
                }
            }
        }
    }
}

struct ImageGallery: View {
    let urls: [URL]
    @State private var index = 0

    var body: some View {
        VStack(spacing: 8) {
            Group {
                if urls.isEmpty {
                    RemoteImage(url: nil)
                } else {
                    #if os(iOS)
                    TabView(selection: $index) {
                        ForEach(Array(urls.enumerated()), id: \.offset) { i, u in
                            RemoteImage(url: u).padding(12).tag(i)
                        }
                    }
                    .tabViewStyle(.page(indexDisplayMode: urls.count > 1 ? .automatic : .never))
                    #else
                    RemoteImage(url: urls[min(index, urls.count - 1)]).padding(12)
                    #endif
                }
            }
            .frame(maxWidth: .infinity)
            .aspectRatio(1, contentMode: .fit)
            .background(Color.secondary.opacity(0.06), in: RoundedRectangle(cornerRadius: 12))
            if urls.count > 1 {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach(Array(urls.enumerated()), id: \.offset) { i, u in
                            Button { index = i } label: {
                                RemoteImage(url: u).frame(width: 56, height: 56).padding(4)
                                    .background(Color.secondary.opacity(0.06), in: RoundedRectangle(cornerRadius: 8))
                                    .overlay(RoundedRectangle(cornerRadius: 8).stroke(i == index ? Color.accentColor : .clear, lineWidth: 2))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
            }
        }
    }
}

struct PriceHistorySection: View {
    @Environment(AppSettings.self) private var settings
    let points: [PricePoint]

    private var ascending: [PricePoint] { points.sorted { $0.observedAt < $1.observedAt } }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionHeader(title: L10n.t("listing.priceHistory"))
            Chart(ascending.filter { $0.priceUsd != nil }) { p in
                let v = Money.convertFromUSD(p.priceUsd ?? 0, to: settings.currency, rates: settings.rates) ?? (p.priceUsd ?? 0)
                LineMark(x: .value("Date", p.observedAt), y: .value("Price", v))
                PointMark(x: .value("Date", p.observedAt), y: .value("Price", v))
            }
            .chartYScale(domain: .automatic(includesZero: false))
            .chartYAxis { AxisMarks(position: .leading) { AxisGridLine(); AxisValueLabel(format: FloatingPointFormatStyle<Double>.number.notation(.compactName)) } }
            .foregroundStyle(.green)
            .frame(height: 140)
            VStack(spacing: 6) {
                ForEach(points) { p in
                    HStack {
                        Text(p.observedAt.formatted(date: .abbreviated, time: .omitted)).foregroundStyle(.secondary)
                        Spacer()
                        PriceText(usd: p.priceUsd, original: p.price, originalCurrency: p.currency, showOriginal: false, font: .subheadline)
                    }
                    .font(.subheadline)
                }
            }
        }
    }
}

extension Optional where Wrapped == String {
    var nonEmpty: String? {
        guard let s = self, !s.isEmpty else { return nil }
        return s
    }
}
