import SwiftUI

/// Push alerts saved on this device. Tapping one opens its search; swipe to delete.
struct AlertsView: View {
    @Environment(\.api) private var api
    @Environment(AppSettings.self) private var settings
    @State private var items: [PushAlert]?
    @State private var error: String?

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

/// The toolbar bell on search results: saves the current criteria as an alert.
struct AlertToolbarButton: View {
    let query: SearchQuery
    @Environment(\.api) private var api
    @Environment(AppSettings.self) private var settings
    @State private var saving = false
    @State private var saved = false
    @State private var message: String?

    var body: some View {
        Button {
            Task { await save() }
        } label: {
            Label(L10n.t("alerts.create"), systemImage: saved ? "bell.fill" : "bell")
        }
        .disabled(saving)
        .accessibilityIdentifier("search.alert")
        .alert(message ?? "", isPresented: Binding(get: { message != nil }, set: { if !$0 { message = nil } })) {
            Button(L10n.t("alerts.done"), role: .cancel) {}
        }
        .onChange(of: query) { saved = false }
    }

    private func save() async {
        guard let alertQuery = query.alertQuery(currency: settings.currency) else {
            message = L10n.t("alerts.needsCriteria")
            return
        }
        saving = true
        defer { saving = false }
        switch await PushCenter.shared.enable() {
        case .granted(let subscriberID):
            do {
                _ = try await api.createAlert(subscriberID: subscriberID, query: alertQuery)
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
