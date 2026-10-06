import SwiftUI

/// Push alerts saved on this device. Tapping one opens its search; swipe right (or long-press) to
/// rename, swipe left to delete.
struct AlertsView: View {
    @Environment(\.api) private var api
    @Environment(AppSettings.self) private var settings
    @State private var items: [PushAlert]?
    @State private var error: String?
    @State private var renaming: PushAlert?
    @State private var newName = ""

    var body: some View {
        Group {
            if let items, items.isEmpty {
                ContentUnavailableView {
                    Label(L10n.t("alerts.title"), systemImage: "bell")
                } description: {
                    Text(L10n.t("alerts.empty"))
                }
            } else if let items {
                List {
                    Section {
                        ForEach(items) { a in
                            NavigationLink(value: SearchQuery(alertQuery: a.query, displayCurrency: settings.currency, rates: settings.rates)) {
                                Text(a.name).lineLimit(2)
                            }
                            .swipeActions(edge: .leading) {
                                Button(L10n.t("alerts.rename")) { startRename(a) }.tint(.blue)
                            }
                            .contextMenu {
                                Button(L10n.t("alerts.rename"), systemImage: "pencil") { startRename(a) }
                            }
                        }
                        .onDelete { offsets in
                            let doomed = offsets.map { items[$0] }
                            Task { await delete(doomed) }
                        }
                    } footer: {
                        Text(L10n.t("alerts.intro"))
                    }
                }
            } else if let error {
                ErrorView(message: error) { Task { await load() } }
            } else {
                ProgressView()
            }
        }
        .navigationTitle(L10n.t("alerts.title"))
        .alert(L10n.t("alerts.rename"), isPresented: Binding(get: { renaming != nil }, set: { if !$0 { renaming = nil } })) {
            TextField(L10n.t("alerts.namePlaceholder"), text: $newName)
                .accessibilityIdentifier("alerts.name")
            Button(L10n.t("alerts.save")) {
                if let a = renaming { Task { await rename(a, to: newName) } }
            }
            Button(L10n.t("alerts.cancel"), role: .cancel) {}
        } message: {
            Text(L10n.t("alerts.nameHint"))
        }
        .appDestinations()
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        guard let id = PushCenter.shared.existingSubscriberID else {
            items = []
            return
        }
        do {
            items = try await api.alerts(subscriberID: id)
            error = nil
        } catch {
            if items == nil { self.error = L10n.t("alerts.error") }
        }
    }

    private func startRename(_ a: PushAlert) {
        newName = a.name
        renaming = a
    }

    private func rename(_ a: PushAlert, to name: String) async {
        guard let id = PushCenter.shared.existingSubscriberID else { return }
        do {
            let updated = try await api.renameAlert(subscriberID: id, id: a.id, name: name.trimmingCharacters(in: .whitespacesAndNewlines))
            if let i = items?.firstIndex(where: { $0.id == a.id }) { items?[i] = updated }
        } catch {
            self.error = L10n.t("alerts.error")
        }
    }

    private func delete(_ doomed: [PushAlert]) async {
        guard let id = PushCenter.shared.existingSubscriberID else { return }
        for a in doomed {
            do {
                try await api.deleteAlert(subscriberID: id, id: a.id)
                items?.removeAll { $0.id == a.id }
            } catch {
                self.error = L10n.t("alerts.error")
            }
        }
    }
}

/// The toolbar bell on search results: asks for an optional name, then saves the current criteria
/// as an alert.
struct AlertToolbarButton: View {
    let query: SearchQuery
    @Environment(\.api) private var api
    @Environment(AppSettings.self) private var settings
    @State private var saving = false
    @State private var saved = false
    @State private var naming = false
    @State private var name = ""
    @State private var message: String?

    var body: some View {
        Button {
            if query.alertQuery(currency: settings.currency) == nil {
                message = L10n.t("alerts.needsCriteria")
            } else {
                name = ""
                naming = true
            }
        } label: {
            Label(L10n.t("alerts.create"), systemImage: saved ? "bell.fill" : "bell")
        }
        .disabled(saving)
        .accessibilityIdentifier("search.alert")
        .alert(L10n.t("alerts.nameTitle"), isPresented: $naming) {
            TextField(L10n.t("alerts.namePlaceholder"), text: $name)
                .accessibilityIdentifier("alerts.name")
            Button(L10n.t("alerts.save")) { Task { await save(name: name) } }
            Button(L10n.t("alerts.cancel"), role: .cancel) {}
        } message: {
            Text(L10n.t("alerts.nameHint"))
        }
        .alert(message ?? "", isPresented: Binding(get: { message != nil }, set: { if !$0 { message = nil } })) {
            Button(L10n.t("alerts.done"), role: .cancel) {}
        }
        .onChange(of: query) { saved = false }
    }

    private func save(name: String) async {
        guard let alertQuery = query.alertQuery(currency: settings.currency) else {
            message = L10n.t("alerts.needsCriteria")
            return
        }
        saving = true
        defer { saving = false }
        switch await PushCenter.shared.enable() {
        case .granted(let subscriberID):
            do {
                _ = try await api.createAlert(subscriberID: subscriberID, query: alertQuery,
                                              name: name.trimmingCharacters(in: .whitespacesAndNewlines))
                saved = true
                message = L10n.t("alerts.created")
            } catch let APIError.http(status, _) where status == 409 {
                message = L10n.t("alerts.limit")
            } catch let APIError.http(status, _) where status == 422 {
                message = L10n.t("alerts.needsCriteria")
            } catch {
                message = L10n.t("alerts.error")
            }
        case .denied:
            message = L10n.t("alerts.permissionDenied")
        case .unavailable:
            message = L10n.t("alerts.error")
        }
    }
}
