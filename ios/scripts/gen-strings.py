#!/usr/bin/env python3
"""Generates WatchCompareKit's Localizable.strings tables from the web catalogues.

    python3 ios/scripts/gen-strings.py

Keys are "section.key" as in frontend/messages/<locale>.json. Web placeholders ({count}) become
positional printf specifiers (%1$@); literal percent signs are escaped as %% so "{percent}%" does
not turn into the "% c" specifier. Strings that only exist in the app live in APP_ONLY below.
"""
import json, os, pathlib

ROOT = pathlib.Path(__file__).resolve().parents[2]
WEB = ROOT / "frontend" / "messages"
OUT = ROOT / "ios" / "WatchCompareKit" / "Sources" / "WatchCompareKit" / "Resources"
FOLDER = {"en": "en", "zh-TW": "zh-Hant", "zh-CN": "zh-Hans", "ja": "ja", "de": "de"}

# Keys copied verbatim (no placeholders).
PLAIN = {
    "nav": ["search", "brands", "settings", "currency", "language"],
    "home": ["searchPlaceholder", "heroTitle", "statListings", "statBrands", "statSources", "recentSearches", "clearRecent", "popularBrands", "newest", "viewAll"],
    "search": ["title", "noResults", "noResultsHint", "clearAll", "sortBy", "filters", "brand", "dialColor", "source", "condition", "movement", "gender", "country", "year", "diameter", "boxPapers", "withBox", "withPapers", "min", "max", "apply"],
    "sort": ["relevance", "newest", "oldest", "price_asc", "price_desc", "year_desc", "year_asc", "size_asc", "size_desc"],
    "condition": ["new", "unworn", "very_good", "good", "fair", "poor", "unknown"],
    "movement": ["automatic", "manual", "quartz", "unknown"],
    "gender": ["men", "women", "unisex", "unknown"],
    "dial": ["black", "white", "silver", "blue", "green", "grey", "champagne", "gold", "brown", "red", "pink", "salmon", "purple", "yellow", "orange", "mother_of_pearl", "meteorite", "cream", "bronze"],
    "listing": ["notFound", "notFoundBody", "description", "private", "dealer", "priceExclTax", "priceInclTax", "taxNote", "shipping", "brand", "model", "reference", "condition", "year", "diameter", "material", "dialColor", "movement", "gender", "boxPapers", "yes", "no", "location", "firstSeen", "lastSeen", "details", "compareTitle", "cheapestBadge", "priceHistory"],
    "brands": ["title"],
    "common": ["error"],
    "footer": ["sources", "disclaimer"],
}
# Keys with placeholders: web name -> positional printf, in this order.
FORMAT = {
    "home.heroSubtitle": ["count", "sources"],
    "home.lastUpdated": ["date"],
    "search.price": ["currency"],
    "listing.inactive": ["source"],
    "listing.viewOnSource": ["source"],
    "listing.saveVs": ["percent"],
}
SPEC = {"count": "@", "sources": "d", "date": "@", "currency": "@", "source": "@", "percent": "d"}

APP_ONLY = {
    "en": {
        "home.removeRecent": "Remove from recent searches",
        "search.resultsNone": "No watches", "search.resultsOne": "1 watch", "search.resultsMany": "%@ watches",
        "search.resultsFor": "%1$@ for “%2$@”", "search.showing": "Showing %1$@ of %2$@", "search.any": "Any",
        "brands.listings": "%@ listings", "common.retry": "Retry",
        "settings.currencyNote": "Prices convert with daily exchange rates; the seller's original price is always shown too.",
        "settings.changeLanguage": "Change app language in Settings",
        "settings.languageNote": "WatchCompare follows your iOS language: English, 繁體中文, 简体中文, 日本語 or Deutsch.",
        "settings.tracking": "Tracking", "settings.trackingStatus": "Permission", "settings.trackingAllowed": "Allowed",
        "settings.trackingDenied": "Not allowed", "settings.trackingNotAsked": "Not asked yet",
        "settings.trackingChange": "Change in Settings",
        "settings.trackingNote": "iOS asks once whether WatchCompare may track your activity across other apps and websites. Nothing is tracked unless you allow it; you can change your answer any time in Settings › Privacy & Security › Tracking.",
    },
    "zh-TW": {
        "home.removeRecent": "從最近搜尋中移除",
        "search.resultsNone": "沒有符合的手錶", "search.resultsOne": "1 支手錶", "search.resultsMany": "%@ 支手錶",
        "search.resultsFor": "%1$@：「%2$@」", "search.showing": "已顯示 %1$@ / %2$@ 筆", "search.any": "不限",
        "brands.listings": "%@ 件", "common.retry": "重試",
        "settings.currencyNote": "價格依每日匯率換算，並同時顯示賣家的原始價格。",
        "settings.changeLanguage": "在「設定」中變更 App 語言",
        "settings.languageNote": "WatchCompare 會跟隨 iOS 的語言設定：English、繁體中文、简体中文、日本語 或 Deutsch。",
        "settings.tracking": "追蹤", "settings.trackingStatus": "權限", "settings.trackingAllowed": "已允許",
        "settings.trackingDenied": "不允許", "settings.trackingNotAsked": "尚未詢問",
        "settings.trackingChange": "在「設定」中變更",
        "settings.trackingNote": "iOS 會詢問一次是否允許 WatchCompare 跨其他 App 與網站追蹤您的活動。除非您允許，否則不會進行任何追蹤；您隨時可在「設定 › 隱私權與安全性 › 追蹤」中變更。",
    },
    "zh-CN": {
        "home.removeRecent": "从最近搜索中移除",
        "search.resultsNone": "没有符合的手表", "search.resultsOne": "1 只手表", "search.resultsMany": "%@ 只手表",
        "search.resultsFor": "%1$@：“%2$@”", "search.showing": "已显示 %1$@ / %2$@ 条", "search.any": "不限",
        "brands.listings": "%@ 件", "common.retry": "重试",
        "settings.currencyNote": "价格按每日汇率换算，并同时显示卖家的原始价格。",
        "settings.changeLanguage": "在“设置”中更改应用语言",
        "settings.languageNote": "WatchCompare 会跟随 iOS 的语言设置：English、繁體中文、简体中文、日本語 或 Deutsch。",
        "settings.tracking": "跟踪", "settings.trackingStatus": "权限", "settings.trackingAllowed": "已允许",
        "settings.trackingDenied": "不允许", "settings.trackingNotAsked": "尚未询问",
        "settings.trackingChange": "在“设置”中更改",
        "settings.trackingNote": "iOS 会询问一次是否允许 WatchCompare 跨其他应用与网站跟踪您的活动。除非您允许，否则不会进行任何跟踪；您可随时在“设置 › 隐私与安全性 › 跟踪”中更改。",
    },
    "ja": {
        "home.removeRecent": "最近の検索から削除",
        "search.resultsNone": "該当なし", "search.resultsOne": "1 本", "search.resultsMany": "%@ 本",
        "search.resultsFor": "「%2$@」：%1$@", "search.showing": "%2$@ 件中 %1$@ 件を表示", "search.any": "指定なし",
        "brands.listings": "%@ 件", "common.retry": "再試行",
        "settings.currencyNote": "価格は日次レートで換算され、販売店の元の価格も常に表示されます。",
        "settings.changeLanguage": "「設定」でアプリの言語を変更",
        "settings.languageNote": "WatchCompare は iOS の言語設定に従います：English、繁體中文、简体中文、日本語、Deutsch。",
        "settings.tracking": "トラッキング", "settings.trackingStatus": "許可", "settings.trackingAllowed": "許可済み",
        "settings.trackingDenied": "許可しない", "settings.trackingNotAsked": "未確認",
        "settings.trackingChange": "「設定」で変更",
        "settings.trackingNote": "WatchCompare が他社のアプリや Web サイトを横断してアクティビティを追跡することを許可するか、iOS が一度だけ確認します。許可しない限りトラッキングは行われません。「設定 › プライバシーとセキュリティ › トラッキング」からいつでも変更できます。",
    },
    "de": {
        "home.removeRecent": "Aus den letzten Suchen entfernen",
        "search.resultsNone": "Keine Uhren", "search.resultsOne": "1 Uhr", "search.resultsMany": "%@ Uhren",
        "search.resultsFor": "%1$@ für „%2$@“", "search.showing": "%1$@ von %2$@ angezeigt", "search.any": "Alle",
        "brands.listings": "%@ Angebote", "common.retry": "Erneut versuchen",
        "settings.currencyNote": "Preise werden mit tagesaktuellen Wechselkursen umgerechnet; der Originalpreis des Verkäufers wird immer mit angezeigt.",
        "settings.changeLanguage": "App-Sprache in den Einstellungen ändern",
        "settings.languageNote": "WatchCompare folgt Ihrer iOS-Sprache: English, 繁體中文, 简体中文, 日本語 oder Deutsch.",
        "settings.tracking": "Tracking", "settings.trackingStatus": "Berechtigung", "settings.trackingAllowed": "Erlaubt",
        "settings.trackingDenied": "Nicht erlaubt", "settings.trackingNotAsked": "Noch nicht gefragt",
        "settings.trackingChange": "In den Einstellungen ändern",
        "settings.trackingNote": "iOS fragt einmalig, ob WatchCompare Ihre Aktivitäten über andere Apps und Websites hinweg verfolgen darf. Ohne Ihre Erlaubnis wird nichts erfasst; Sie können die Antwort jederzeit unter Einstellungen › Datenschutz & Sicherheit › Tracking ändern.",
    },
}


def esc(s: str) -> str:
    return s.replace("\\", "\\\\").replace('"', '\\"').replace("\n", "\\n")


def main() -> None:
    for loc, folder in FOLDER.items():
        d = json.load(open(WEB / f"{loc}.json", encoding="utf-8"))
        out = []
        for sec, keys in PLAIN.items():
            for k in keys:
                out.append((f"{sec}.{k}", d[sec][k].replace("%", "%%")))
        for key, names in FORMAT.items():
            sec, k = key.split(".")
            v = d[sec][k].replace("%", "%%")  # literal percent signs first…
            for i, n in enumerate(names, 1):  # …then the placeholders
                v = v.replace("{" + n + "}", f"%{i}${SPEC[n]}")
            assert "{" not in v, (loc, key, v)
            out.append((key, v))
        out.extend(APP_ONLY[loc].items())
        keys = [k for k, _ in out]
        assert len(keys) == len(set(keys)), f"duplicate key in {loc}"
        path = OUT / f"{folder}.lproj" / "Localizable.strings"
        path.parent.mkdir(parents=True, exist_ok=True)
        with open(path, "w", encoding="utf-8") as f:
            f.write(f"/* Generated by ios/scripts/gen-strings.py from frontend/messages/{loc}.json — do not edit by hand. */\n\n")
            for k, v in out:
                f.write(f'"{esc(k)}" = "{esc(v)}";\n')
        print(f"{path.relative_to(ROOT)}: {len(out)} strings")


if __name__ == "__main__":
    main()
