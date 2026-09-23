import SwiftUI

struct SettingsView: View {
    @Environment(\.api) private var api
    @Environment(AppSettings.self) private var settings
    @Environment(TrackingConsent.self) private var tracking
    @State private var sources: [Source] = []

    var body: some View {
        @Bindable var settings = settings
        Form {
            Section(L10n.t("nav.currency")) {
                Picker(L10n.t("nav.currency"), selection: $settings.currency) {
                    ForEach(settings.supportedCurrencies, id: \.self) { Text($0).tag($0) }
                }
                .pickerStyle(.menu)
                Text(L10n.t("settings.currencyNote")).font(.footnote).foregroundStyle(.secondary)
            }
            Section(L10n.t("nav.language")) {
                if let url = SystemSettings.appSettingsURL {
                    Link(L10n.t("settings.changeLanguage"), destination: url)
                }
                Text(L10n.t("settings.languageNote")).font(.footnote).foregroundStyle(.secondary)
            }
            if tracking.status != .unavailable {
                Section(L10n.t("settings.tracking")) {
                    LabeledContent(L10n.t("settings.trackingStatus"), value: trackingLabel)
                    if let url = SystemSettings.appSettingsURL {
                        Link(L10n.t("settings.trackingChange"), destination: url)
                    }
                    Text(L10n.t("settings.trackingNote")).font(.footnote).foregroundStyle(.secondary)
                }
            }
            if !sources.isEmpty {
                Section(L10n.t("footer.sources")) {
                    ForEach(sources) { s in
                        if let url = URL(string: s.baseUrl) {
                            Link(destination: url) {
                                HStack { Text(s.name); Spacer(); Text(s.country).foregroundStyle(.secondary).font(.caption) }
                            }
                            .foregroundStyle(.primary)
                        }
                    }
                }
            }
            Section {
                Text(L10n.t("footer.disclaimer")).font(.footnote).foregroundStyle(.secondary)
                LabeledContent("API", value: api.baseURL.absoluteString).font(.footnote)
            }
        }
        .groupedListStyle()
        .navigationTitle(L10n.t("nav.settings"))
        .task { sources = (try? await api.sources()) ?? [] }
    }

    private var trackingLabel: String {
        switch tracking.status {
        case .authorized: return L10n.t("settings.trackingAllowed")
        case .denied, .restricted: return L10n.t("settings.trackingDenied")
        case .notDetermined, .unavailable: return L10n.t("settings.trackingNotAsked")
        }
    }
}
