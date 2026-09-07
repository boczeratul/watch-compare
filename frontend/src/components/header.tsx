import { Suspense } from "react";
import { getTranslations } from "next-intl/server";
import { Watch } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { SearchBar } from "./search-bar";
import { SettingsMenu } from "./settings-menu";

export async function Header() {
  const t = await getTranslations("nav");
  const meta = await getTranslations("meta");
  return (
    <header className="sticky top-0 z-40 border-b border-slate-200 bg-white/95 backdrop-blur">
      <div className="mx-auto flex max-w-7xl items-center gap-4 px-4 py-3 sm:px-6">
        <Link href="/" className="flex shrink-0 items-center gap-2 text-slate-900" aria-label={meta("title")}>
          <Watch className="h-6 w-6 text-emerald-700" aria-hidden />
          <span className="text-lg font-bold tracking-tight">WatchCompare</span>
        </Link>
        <nav className="hidden items-center gap-5 text-sm font-medium text-slate-600 md:flex" aria-label="Primary">
          <Link href="/search" className="hover:text-slate-900">{t("search")}</Link>
          <Link href="/brands" className="hover:text-slate-900">{t("brands")}</Link>
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
