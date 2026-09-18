import Foundation

/// Recent search queries, persisted in UserDefaults. Same rules as the web's localStorage store:
/// newest first, case-insensitive de-duplication, at most `maxCount` entries.
public struct RecentSearchesStore {
    public static let key = "wc_recent_searches"
    public static let maxCount = 10

    private let defaults: UserDefaults

    public init(defaults: UserDefaults = .standard) { self.defaults = defaults }

    public func load() -> [String] {
        let raw = defaults.stringArray(forKey: Self.key) ?? []
        return Array(raw.filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }.prefix(Self.maxCount))
    }

    @discardableResult
    public func add(_ query: String) -> [String] {
        let q = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !q.isEmpty else { return load() }
        let rest = load().filter { $0.caseInsensitiveCompare(q) != .orderedSame }
        return save(Array(([q] + rest).prefix(Self.maxCount)))
    }

    @discardableResult
    public func remove(_ query: String) -> [String] {
        save(load().filter { $0.caseInsensitiveCompare(query) != .orderedSame })
    }

    @discardableResult
    public func clear() -> [String] { save([]) }

    private func save(_ items: [String]) -> [String] {
        if items.isEmpty { defaults.removeObject(forKey: Self.key) } else { defaults.set(items, forKey: Self.key) }
        return items
    }

    /// Suggestions for a partial query: prefix matches first, then substring matches, both
    /// newest first. An empty query returns everything (up to `limit`).
    public static func matches(in recent: [String], partial: String, limit: Int = 8) -> [String] {
        let p = partial.trimmingCharacters(in: .whitespaces).lowercased()
        if p.isEmpty { return Array(recent.prefix(limit)) }
        var prefix: [String] = []
        var contains: [String] = []
        for item in recent {
            let lower = item.lowercased()
            if lower == p { continue }
            if lower.hasPrefix(p) { prefix.append(item) } else if lower.contains(p) { contains.append(item) }
        }
        return Array((prefix + contains).prefix(limit))
    }

    /// The inline completion for `partial`: the first suggestion that starts with it.
    public static func completion(in suggestions: [String], for partial: String) -> String? {
        let p = partial.lowercased()
        guard !p.isEmpty, partial.first?.isWhitespace != true else { return nil }
        return suggestions.first { $0.lowercased().hasPrefix(p) && $0.count > partial.count }
    }
}
