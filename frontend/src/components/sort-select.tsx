"use client";

import { useTranslations } from "next-intl";
import { useRouter, usePathname } from "@/i18n/navigation";
import { useSearchParams } from "next/navigation";

const KEYS = ["relevance", "newest", "oldest", "price_asc", "price_desc", "year_desc", "year_asc", "size_asc", "size_desc"] as const;

export function SortSelect() {
  const t = useTranslations("search");
  const ts = useTranslations("sort");
  const router = useRouter();
  const pathname = usePathname();
  const sp = useSearchParams();
  const current = sp.get("sort") ?? (sp.get("q") ? "relevance" : "newest");

  const onChange = (value: string) => {
    const next = new URLSearchParams(sp.toString());
    next.set("sort", value);
    next.delete("page");
    router.push(`${pathname}?${next.toString()}`);
  };

  return (
    <label className="flex items-center gap-2 text-sm text-slate-600">
      <span className="hidden sm:inline">{t("sortBy")}</span>
      <select value={current} onChange={(e) => onChange(e.target.value)} className="h-9 rounded-md border border-slate-300 bg-white px-2 text-sm text-slate-800 shadow-sm focus:border-slate-900 focus:outline-none">
        {KEYS.map((k) => (
          <option key={k} value={k}>
            {ts(k)}
          </option>
        ))}
      </select>
    </label>
  );
}
