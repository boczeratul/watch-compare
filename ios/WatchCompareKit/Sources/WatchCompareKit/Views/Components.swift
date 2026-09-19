import SwiftUI

struct RemoteImage: View {
    let url: URL?
    var contentMode: ContentMode = .fit

    var body: some View {
        AsyncImage(url: url, transaction: Transaction(animation: .easeIn(duration: 0.15))) { phase in
            switch phase {
            case .success(let image):
                image.resizable().aspectRatio(contentMode: contentMode).accessibilityIdentifier("image.loaded")
            case .failure:
                placeholder(symbol: "photo")
            case .empty:
                if url == nil { placeholder(symbol: "photo") } else { ProgressView().controlSize(.small) }
            @unknown default:
                placeholder(symbol: "photo")
            }
        }
    }

    private func placeholder(symbol: String) -> some View {
        Image(systemName: symbol).font(.title2).foregroundStyle(.quaternary)
    }
}

struct SourceBadge: View {
    let name: String
    var body: some View {
        Text(name)
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 6).padding(.vertical, 3)
            .background(.thinMaterial, in: Capsule())
            .lineLimit(1)
    }
}

struct Chip: View {
    let text: String
    var body: some View {
        Text(text)
            .font(.caption)
            .padding(.horizontal, 6).padding(.vertical, 2)
            .background(Color.secondary.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
    }
}

/// A price in the visitor's currency with the seller's original underneath (web `Price`).
struct PriceText: View {
    @Environment(AppSettings.self) private var settings
    let usd: Double?
    let original: Double?
    let originalCurrency: String?
    var showOriginal = true
    var font: Font = .headline

    var body: some View {
        let p = settings.price(usd: usd, original: original, originalCurrency: originalCurrency, showOriginal: showOriginal)
        VStack(alignment: .leading, spacing: 1) {
            Text(p.main ?? "—").font(font).foregroundStyle(p.main == nil ? .secondary : .primary)
            if let o = p.original {
                Text(o).font(.caption).foregroundStyle(.secondary)
            }
        }
    }
}

extension PriceText {
    init(listing: Listing, showOriginal: Bool = true, font: Font = .headline) {
        self.init(usd: listing.priceUsd, original: listing.comparisonPrice, originalCurrency: listing.currency,
                  showOriginal: showOriginal, font: font)
    }
}

struct ErrorView: View {
    let message: String
    let retry: () -> Void
    var body: some View {
        ContentUnavailableView {
            Label(L10n.t("common.error"), systemImage: "exclamationmark.triangle")
        } description: {
            Text(message)
        } actions: {
            Button(L10n.t("common.retry"), action: retry).buttonStyle(.borderedProminent)
        }
    }
}

struct SectionHeader: View {
    let title: String
    var body: some View {
        Text(title).font(.title3.weight(.semibold)).frame(maxWidth: .infinity, alignment: .leading)
    }
}

/// Two-column card grid used by the home page and the search results.
let cardColumns = [GridItem(.adaptive(minimum: 160), spacing: 12)]
