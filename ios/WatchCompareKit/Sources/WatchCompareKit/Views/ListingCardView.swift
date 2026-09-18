import SwiftUI

struct ListingCardView: View {
    let listing: Listing

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            ZStack(alignment: .topLeading) {
                Color.secondary.opacity(0.06)
                RemoteImage(url: listing.firstImageURL).padding(10)
                SourceBadge(name: listing.sourceName).padding(6)
            }
            .aspectRatio(1, contentMode: .fit)
            .clipShape(RoundedRectangle(cornerRadius: 10))

            Text(listing.brandName?.uppercased() ?? " ")
                .font(.caption2.weight(.semibold)).foregroundStyle(.secondary).lineLimit(1)
            Text(listing.title).font(.subheadline).lineLimit(2).multilineTextAlignment(.leading)
                .frame(maxWidth: .infinity, alignment: .leading)
            Text(metaLine.isEmpty ? " " : metaLine).font(.caption).foregroundStyle(.secondary).lineLimit(1)
            HStack(alignment: .bottom) {
                PriceText(listing: listing, font: .subheadline.weight(.semibold))
                Spacer(minLength: 4)
                Chip(text: L10n.condition(listing.condition))
            }
            .padding(.top, 2)
            if let loc = listing.location {
                Label(loc, systemImage: "mappin").font(.caption2).foregroundStyle(.secondary).lineLimit(1)
            }
        }
        .contentShape(Rectangle())
    }

    private var metaLine: String {
        var parts: [String] = []
        if let r = listing.referenceNumber, !r.isEmpty { parts.append(r) }
        if let y = listing.year { parts.append(String(y)) }
        if let d = listing.caseDiameterMm { parts.append("\(d.formatted()) mm") }
        if let c = listing.dialColor, !c.isEmpty { parts.append(L10n.dial(c)) }
        return parts.joined(separator: " · ")
    }
}

/// Compact horizontal row, used for "similar" lists on the detail page.
struct ListingRowView: View {
    let listing: Listing
    var highlight = false

    var body: some View {
        HStack(spacing: 12) {
            RemoteImage(url: listing.firstImageURL)
                .frame(width: 56, height: 56)
                .background(Color.secondary.opacity(0.06), in: RoundedRectangle(cornerRadius: 8))
            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: 6) {
                    SourceBadge(name: listing.sourceName)
                    if highlight { Chip(text: L10n.t("listing.cheapestBadge")).foregroundStyle(.green) }
                }
                Text(listing.title).font(.subheadline).lineLimit(2)
                HStack(spacing: 6) {
                    Chip(text: L10n.condition(listing.condition))
                    if let loc = listing.location { Text(loc).font(.caption).foregroundStyle(.secondary).lineLimit(1) }
                }
            }
            Spacer(minLength: 0)
            PriceText(listing: listing, font: .subheadline.weight(.semibold))
        }
        .contentShape(Rectangle())
    }
}
