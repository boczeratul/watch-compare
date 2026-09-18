import SwiftUI

private struct APIClientKey: EnvironmentKey {
    static let defaultValue = APIClient(baseURL: URL(string: "http://localhost:8080")!)
}

extension EnvironmentValues {
    public var api: APIClient {
        get { self[APIClientKey.self] }
        set { self[APIClientKey.self] = newValue }
    }
}

// Small platform shims so the views compile on macOS too (that is how the package is
// type-checked and unit-tested without Xcode); on iOS they apply the usual modifiers.
extension View {
    func inlineNavigationTitle() -> some View {
        #if os(iOS)
        return self.navigationBarTitleDisplayMode(.inline)
        #else
        return self
        #endif
    }

    func numericKeyboard() -> some View {
        #if os(iOS)
        return self.keyboardType(.decimalPad)
        #else
        return self
        #endif
    }

    func searchFieldStyle() -> some View {
        #if os(iOS)
        return self.textInputAutocapitalization(.never).autocorrectionDisabled()
        #else
        return self.autocorrectionDisabled()
        #endif
    }

    func groupedListStyle() -> some View {
        #if os(iOS)
        return self.listStyle(.insetGrouped)
        #else
        return self.listStyle(.inset)
        #endif
    }
}

/// Opens the system Settings app on iOS so the visitor can change the app language.
enum SystemSettings {
    static var languageURL: URL? {
        #if os(iOS)
        return URL(string: UIApplication.openSettingsURLString)
        #else
        return nil
        #endif
    }
}
