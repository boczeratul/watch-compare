import type { Metadata } from "next";
import { getLocale, getTranslations, setRequestLocale } from "next-intl/server";
import { Link } from "@/i18n/navigation";
import { api, safe } from "@/lib/api";

export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "brands" });
  return { title: t("title") };
}

export default async function BrandsPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  setRequestLocale(locale);
  const [t, currentLocale, brands] = await Promise.all([getTranslations("brands"), getLocale(), safe(api.brands(), { items: [] })]);
  const active = brands.items.filter((b) => b.listingCount > 0).sort((a, b) => a.name.localeCompare(b.name, "en"));
  const groups = new Map<string, typeof active>();
  for (const b of active) {
    const letter = b.name[0].toUpperCase();
    groups.set(letter, [...(groups.get(letter) ?? []), b]);
  }
  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
      <p className="mt-1 text-sm text-slate-500">{t("subtitle", { count: active.length })}</p>
      <div className="mt-8 columns-2 gap-8 sm:columns-3 lg:columns-4">
        {[...groups.entries()].map(([letter, list]) => (
          <section key={letter} className="mb-6 break-inside-avoid">
            <h2 className="mb-2 text-sm font-semibold text-slate-400">{letter}</h2>
            <ul className="space-y-1">
              {list.map((b) => (
                <li key={b.slug}>
                  <Link href={{ pathname: "/search", query: { brand: b.slug } }} className="flex items-baseline justify-between gap-2 text-sm text-slate-800 hover:underline">
                    <span>{b.name}</span>
                    <span className="text-xs text-slate-400">{t("listings", { count: b.listingCount })}</span>
                  </Link>
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
      {active.length === 0 && <p className="mt-8 text-sm text-slate-500">—</p>}
      <p className="sr-only">{currentLocale}</p>
    </div>
  );
}
