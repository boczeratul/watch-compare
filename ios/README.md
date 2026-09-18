# WatchCompare for iOS

A native SwiftUI client for the same Go API the web frontend uses (`/api/v1/...`). Search across
every marketplace, filter and sort Chrono24-style, open a listing with its photos, tax breakdown,
price history and the same reference on other marketplaces, and see everything in your currency.

```
ios/
├── WatchCompare.xcodeproj     # thin app target (entry point, Info.plist, assets)
├── WatchCompare/              # @main App + AppConfig (reads APIBaseURL from Info.plist)
├── WatchCompareKit/           # Swift package: models, API client, currency, storage, all views
│   ├── Sources/WatchCompareKit
│   │   ├── Models.swift           # Codable mirrors of backend/internal/model (null arrays → [])
│   │   ├── APIClient.swift        # GET wrapper for /api/v1, RFC 3339 dates
│   │   ├── SearchQuery.swift      # filter state ⇄ query parameters (1:1 with the web)
│   │   ├── Currency.swift         # convert / round / format, same rules as lib/currency.ts
│   │   ├── RecentSearches.swift   # UserDefaults-backed recent queries + suggestions
│   │   ├── AppSettings.swift      # display currency + exchange rates
│   │   ├── Localization.swift     # L10n.t("section.key") over the .strings tables
│   │   ├── Views/                 # Home, results, filters, detail, brands, settings
│   │   └── Resources/*.lproj      # en, zh-Hant, ja, de — keys mirror frontend/messages/*.json
│   └── Tests/WatchCompareKitTests # decoding, currency, query, recent searches, string tables
└── project.yml                # XcodeGen spec, only needed to regenerate the .xcodeproj
```

## Requirements

- Xcode 16 or newer (the project uses folder-synchronized groups), iOS 17+.
- A running API. For the simulator, `cd backend && make run` serves http://localhost:8080 and the
  Debug configuration points there (`API_BASE_URL` build setting → `APIBaseURL` in Info.plist).

## Run

```bash
open ios/WatchCompare.xcodeproj      # pick an iPhone simulator, ⌘R
```

Set your team under *Signing & Capabilities* to run on a device, and either change the Debug
`API_BASE_URL` to your Mac's LAN address or launch with the argument
`-WC_API_BASE_URL http://192.168.1.10:8080` (Scheme → Run → Arguments), which overrides the
configured URL at runtime. Release builds must point at the Cloud Run URL (`https://…a.run.app`);
replace the `CHANGE-ME` placeholder in the target's Release settings.

## Test

The package is platform-neutral, so it builds and tests without a simulator:

```bash
cd ios/WatchCompareKit
swift build          # type-checks models, client and every view for macOS
swift test           # 19 unit tests (needs the Xcode toolchain for XCTest)
```

Or run the WatchCompare scheme's tests in Xcode.

## How it maps to the web app

| Web | iOS |
|-----|-----|
| Header search bar with recent-search autocomplete | Home tab search field with recent suggestions; recent chips on the home page |
| `/search` with filters sidebar, sort select, pagination | Results grid with Filters sheet, Sort menu, infinite scroll |
| `/listing/[id]` | Detail view: paged gallery, price box (excl./incl. tax for JP), details, same-reference offers, price-history chart |
| `/brands` | Brands tab (searchable) |
| Currency cookie + rates | Settings tab currency picker (persisted in UserDefaults); language follows the iOS setting |

Only enabled sources are returned by the API, so a retired dealer disappears from the app without
a release.
