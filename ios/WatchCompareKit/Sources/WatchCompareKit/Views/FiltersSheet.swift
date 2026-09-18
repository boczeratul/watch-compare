import SwiftUI

/// Editable copy of a `SearchQuery`; the caller receives the result on Apply (web `FiltersSidebar`).
struct FiltersSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(AppSettings.self) private var settings
    @State private var draft: SearchQuery
    let facets: Facets
    let onApply: (SearchQuery) -> Void

    init(query: SearchQuery, facets: Facets, onApply: @escaping (SearchQuery) -> Void) {
        _draft = State(initialValue: query)
        self.facets = facets
        self.onApply = onApply
    }

    var body: some View {
        NavigationStack {
            Form {
                Section(L10n.t("search.price", settings.currency as NSString)) {
                    rangeRow(min: $draft.priceMin, max: $draft.priceMax,
                             minHint: settings.hint(usd: facets.priceMinUsd), maxHint: settings.hint(usd: facets.priceMaxUsd))
                }
                multi(L10n.t("search.brand"), values: withSelected(facets.brands, draft.brands), selection: $draft.brands)
                multi(L10n.t("search.dialColor"), values: withSelected(facets.dialColors, draft.dialColors), selection: $draft.dialColors, label: L10n.dial)
                multi(L10n.t("search.source"), values: withSelected(facets.sources, draft.sources), selection: $draft.sources)
                enumMulti(L10n.t("search.condition"), facet: facets.conditions, selection: $draft.conditions, label: L10n.condition)
                enumMulti(L10n.t("search.movement"), facet: facets.movements, selection: $draft.movements, label: L10n.movement)
                Section(L10n.t("search.gender")) {
                    ForEach([Gender.men, .women, .unisex], id: \.self) { g in
                        Toggle(L10n.gender(g), isOn: binding(for: g, in: $draft.genders))
                    }
                }
                if facets.countries.count > 1 {
                    multi(L10n.t("search.country"), values: withSelected(facets.countries, draft.countries), selection: $draft.countries)
                }
                Section(L10n.t("search.year")) {
                    if !facets.years.isEmpty {
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 6) {
                                ForEach(facets.years) { y in
                                    let year = Int(y.key)
                                    let active = year != nil && draft.yearMin == year && draft.yearMax == year
                                    Button {
                                        draft.yearMin = active ? nil : year
                                        draft.yearMax = active ? nil : year
                                    } label: {
                                        Text("\(y.label) \(y.count)")
                                            .font(.caption)
                                            .padding(.horizontal, 8).padding(.vertical, 4)
                                            .background(active ? Color.accentColor : Color.secondary.opacity(0.12), in: Capsule())
                                            .foregroundStyle(active ? Color.white : Color.primary)
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                        }
                    }
                    rangeRow(min: intBinding($draft.yearMin), max: intBinding($draft.yearMax), minHint: nil, maxHint: nil)
                }
                Section(L10n.t("search.diameter")) {
                    rangeRow(min: $draft.diameterMin, max: $draft.diameterMax, minHint: nil, maxHint: nil)
                }
                Section(L10n.t("search.boxPapers")) {
                    Toggle(L10n.t("search.withBox"), isOn: $draft.hasBox)
                    Toggle(L10n.t("search.withPapers"), isOn: $draft.hasPapers)
                }
            }
            .navigationTitle(L10n.t("search.filters"))
            .inlineNavigationTitle()
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L10n.t("search.clearAll")) { draft = draft.clearingFilters() }
                        .disabled(draft.activeFilterCount == 0)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L10n.t("search.apply")) { onApply(draft); dismiss() }
                }
            }
        }
    }

    // MARK: Rows

    private func rangeRow(min: Binding<Double?>, max: Binding<Double?>, minHint: String?, maxHint: String?) -> some View {
        HStack {
            TextField(minHint ?? L10n.t("search.min"), value: min, format: .number).numericKeyboard()
            Text("–").foregroundStyle(.secondary)
            TextField(maxHint ?? L10n.t("search.max"), value: max, format: .number).numericKeyboard()
        }
    }

    private func intBinding(_ b: Binding<Int?>) -> Binding<Double?> {
        Binding(get: { b.wrappedValue.map(Double.init) }, set: { b.wrappedValue = $0.map { Int($0) } })
    }

    /// Keep selected values visible (count 0) even when the narrowed result no longer contains them.
    private func withSelected(_ values: [FacetValue], _ selected: Set<String>) -> [FacetValue] {
        let known = Set(values.map(\.key))
        let extra = selected.subtracting(known).sorted().map {
            FacetValue(key: $0, label: $0.replacingOccurrences(of: "-", with: " ").capitalized, count: 0)
        }
        return values + extra
    }

    @ViewBuilder
    private func multi(_ title: String, values: [FacetValue], selection: Binding<Set<String>>, label: ((String) -> String)? = nil) -> some View {
        if !values.isEmpty {
            Section {
                NavigationLink {
                    MultiSelectList(title: title, values: values, selection: selection, label: label)
                } label: {
                    HStack {
                        Text(title)
                        Spacer()
                        Text(summary(selection.wrappedValue, values: values, label: label))
                            .foregroundStyle(.secondary).lineLimit(1)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func enumMulti<E: RawRepresentable & Hashable>(_ title: String, facet: [FacetValue], selection: Binding<Set<E>>, label: @escaping (E) -> String) -> some View where E.RawValue == String {
        let values = facet.compactMap { v in E(rawValue: v.key).map { ($0, v.count) } }
        if !values.isEmpty {
            Section(title) {
                ForEach(values, id: \.0) { (e, count) in
                    Toggle(isOn: binding(for: e, in: selection)) {
                        HStack { Text(label(e)); Spacer(); Text(count.formatted()).foregroundStyle(.tertiary).font(.caption) }
                    }
                }
            }
        }
    }

    private func summary(_ selected: Set<String>, values: [FacetValue], label: ((String) -> String)?) -> String {
        if selected.isEmpty { return L10n.t("search.any") }
        let names = values.filter { selected.contains($0.key) }.map { label?($0.key) ?? $0.label }
        return names.count <= 2 ? names.joined(separator: ", ") : "\(names.count)"
    }

    private func binding<E: Hashable>(for value: E, in set: Binding<Set<E>>) -> Binding<Bool> {
        Binding(get: { set.wrappedValue.contains(value) },
                set: { on in if on { set.wrappedValue.insert(value) } else { set.wrappedValue.remove(value) } })
    }
}

struct MultiSelectList: View {
    let title: String
    let values: [FacetValue]
    @Binding var selection: Set<String>
    let label: ((String) -> String)?
    @State private var filter = ""

    var body: some View {
        List {
            ForEach(filtered) { v in
                Button {
                    if selection.contains(v.key) { selection.remove(v.key) } else { selection.insert(v.key) }
                } label: {
                    HStack {
                        Image(systemName: selection.contains(v.key) ? "checkmark.circle.fill" : "circle")
                            .foregroundStyle(selection.contains(v.key) ? Color.accentColor : Color.secondary)
                        Text(label?(v.key) ?? v.label).foregroundStyle(.primary)
                        Spacer()
                        Text(v.count.formatted()).font(.caption).foregroundStyle(.tertiary)
                    }
                }
            }
        }
        .navigationTitle(title)
        .inlineNavigationTitle()
        .searchable(text: $filter, placement: values.count > 12 ? .automatic : .toolbar)
        .toolbar {
            if !selection.isEmpty {
                ToolbarItem(placement: .primaryAction) { Button(L10n.t("search.clearAll")) { selection = [] } }
            }
        }
    }

    private var filtered: [FacetValue] {
        let f = filter.trimmingCharacters(in: .whitespaces)
        if f.isEmpty { return values }
        return values.filter { (label?($0.key) ?? $0.label).localizedCaseInsensitiveContains(f) || $0.key.localizedCaseInsensitiveContains(f) }
    }
}
