import { Suspense } from "react";
import { getTranslations } from "next-intl/server";
import Image from "next/image";
import { Link } from "@/i18n/navigation";
import { SearchBar } from "./search-bar";
import { SettingsMenu } from "./settings-menu";
import { ONESIGNAL_APP_ID } from "@/lib/onesignal";

export async function Header() {
  const t = await getTranslations("nav");
  const meta = await getTranslations("meta");
  return (
    <header className="sticky top-0 z-40 border-b border-slate-200 bg-white/95 backdrop-blur">
      <div className="mx-auto flex max-w-7xl items-center gap-4 px-4 py-3 sm:px-6">
        <Link href="/" className="flex shrink-0 items-center gap-2 text-slate-900" aria-label={meta("title")}>
          <Image src="/logo.png" alt="" width={28} height={28} className="h-7 w-7" priority />
          <span className="text-lg font-bold tracking-tight">WatchCompare</span>
        </Link>
        <nav className="hidden items-center gap-5 text-sm font-medium text-slate-600 md:flex" aria-label="Primary">
          <Link href="/search" className="hover:text-slate-900">{t("search")}</Link>
          <Link href="/brands" className="hover:text-slate-900">{t("brands")}</Link>
          {ONESIGNAL_APP_ID && <Link href="/alerts" className="hover:text-slate-900">{t("alerts")}</Link>}
        </nav>
        <div className="hidden flex-1 md:block">
          <Suspense>
            <SearchBar className="max-w-xl" />
          </Suspense>
        </div>
        <div className="ml-auto">
          <Suspense>
            <SettingsMenu />
          </Suspense>
        </div>
      </div>
      <div className="border-t border-slate-100 px-4 py-2 md:hidden">
        <Suspense>
          <SearchBar />
        </Suspense>
      </div>
    </header>
  );
}
