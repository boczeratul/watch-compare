"use client";

import { useTransition } from "react";
import { useLocale, useTranslations } from "next-intl";
import { Globe, Coins } from "lucide-react";
import { usePathname, useRouter } from "@/i18n/navigation";
import { useSearchParams } from "next/navigation";
import { localeNames, locales, type Locale } from "@/i18n/routing";
import { setCurrency } from "@/app/actions";
import { useCurrency } from "./providers";

export function SettingsMenu() {
  const t = useTranslations("nav");
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { currency, supported } = useCurrency();
  const [pending, startTransition] = useTransition();

  const onLocale = (next: string) => {
    const query = Object.fromEntries(searchParams.entries());
    startTransition(() => {
      router.replace({ pathname, query }, { locale: next as Locale });
    });
  };
  const onCurrency = (next: string) => {
    startTransition(async () => {
      await setCurrency(next);
      router.refresh();
    });
  };

  const selectClass =
    "h-9 rounded-md border border-slate-300 bg-white pl-7 pr-2 text-sm text-slate-800 shadow-sm focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 disabled:opacity-60";

  return (
    <div className="flex items-center gap-2" aria-busy={pending}>
      <label className="relative">
        <span className="sr-only">{t("language")}</span>
        <Globe className="pointer-events-none absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" aria-hidden />
        <select className={selectClass} value={locale} onChange={(e) => onLocale(e.target.value)} disabled={pending}>
          {locales.map((l) => (
            <option key={l} value={l}>
              {localeNames[l]}
            </option>
          ))}
        </select>
      </label>
      <label className="relative">
        <span className="sr-only">{t("currency")}</span>
        <Coins className="pointer-events-none absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" aria-hidden />
        <select className={selectClass} value={currency} onChange={(e) => onCurrency(e.target.value)} disabled={pending}>
          {supported.map((c) => (
            <option key={c} value={c}>
              {c}
            </option>
          ))}
        </select>
      </label>
    </div>
  );
}
