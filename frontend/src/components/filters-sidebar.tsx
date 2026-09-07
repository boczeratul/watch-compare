"use client";

import { useState, useTransition, type FormEvent } from "react";
import { useTranslations } from "next-intl";
import { SlidersHorizontal, X } from "lucide-react";
import { useRouter, usePathname } from "@/i18n/navigation";
import { useSearchParams } from "next/navigation";
import type { Facets, FacetValue } from "@/lib/api";
import { useCurrency } from "./providers";
import { convertFromUsd, roundDisplay } from "@/lib/currency";
import { cn } from "@/lib/utils";

const MULTI = ["brand", "source", "condition", "movement", "gender", "country"] as const;
const RANGE = ["price_min", "price_max", "year_min", "year_max", "diameter_min", "diameter_max"] as const;

export function FiltersSidebar({ facets }: { facets: Facets }) {
  const t = useTranslations("search");
  const tc = useTranslations("condition");
  const tm = useTranslations("movement");
  const tg = useTranslations("gender");
  const router = useRouter();
  const pathname = usePathname();
  const sp = useSearchParams();
  const { currency, rates } = useCurrency();
  const [open, setOpen] = useState(false);
  const [pending, startTransition] = useTransition();

  const selected = (key: string) => new Set((sp.get(key) ?? "").split(",").filter(Boolean));
  const navigate = (next: URLSearchParams) => {
    next.delete("page");
    const qs = next.toString();
    startTransition(() => router.push(qs ? `${pathname}?${qs}` : pathname));
  };
  const toggle = (key: string, value: string) => {
    const next = new URLSearchParams(sp.toString());
    const set = selected(key);
    if (set.has(value)) set.delete(value);
    else set.add(value);
    if (set.size) next.set(key, [...set].join(","));
    else next.delete(key);
    navigate(next);
  };
  const toggleBool = (key: "box" | "papers") => {
    const next = new URLSearchParams(sp.toString());
    if (next.get(key) === "true") next.delete(key);
    else next.set(key, "true");
    navigate(next);
  };
  const submitRanges = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const next = new URLSearchParams(sp.toString());
    for (const k of RANGE) {
      const v = String(fd.get(k) ?? "").trim();
      if (v) next.set(k, v);
      else next.delete(k);
    }
    navigate(next);
  };
  const clearAll = () => {
    const next = new URLSearchParams();
    const q = sp.get("q");
    if (q) next.set("q", q);
    navigate(next);
  };
  const activeCount = [...MULTI, ...RANGE, "box", "papers"].filter((k) => sp.get(k)).length;

  const group = (key: (typeof MULTI)[number], title: string, values: FacetValue[], label?: (k: string) => string) => {
    if (!values.length) return null;
    const set = selected(key);
    return (
      <fieldset className="border-t border-slate-200 py-3">
        <legend className="mb-2 text-sm font-semibold text-slate-800">{title}</legend>
        <ul className="max-h-56 space-y-1 overflow-auto pr-1">
          {values.map((v) => (
            <li key={v.key}>
              <label className="flex cursor-pointer items-center gap-2 text-sm text-slate-700">
                <input type="checkbox" className="h-4 w-4 rounded border-slate-300 accent-slate-900" checked={set.has(v.key)} onChange={() => toggle(key, v.key)} />
                <span className="flex-1 truncate">{label ? label(v.key) : v.label}</span>
                <span className="text-xs text-slate-400">{v.count.toLocaleString()}</span>
              </label>
            </li>
          ))}
        </ul>
      </fieldset>
    );
  };

  const hint = (usd?: number) => {
    if (usd === undefined) return "";
    const c = convertFromUsd(usd, currency, rates);
    return c === null ? "" : String(roundDisplay(c, currency));
  };

  const body = (
    <form onSubmit={submitRanges} className={cn("text-slate-800", pending && "opacity-60")} aria-busy={pending}>
      <div className="flex items-center justify-between pb-2">
        <h2 className="text-base font-semibold">{t("filters")}</h2>
        {activeCount > 0 && (
          <button type="button" onClick={clearAll} className="text-xs text-slate-500 underline hover:text-slate-900">
            {t("clearAll")} ({activeCount})
          </button>
        )}
      </div>

      <fieldset className="border-t border-slate-200 py-3">
        <legend className="mb-2 text-sm font-semibold">{t("price", { currency })}</legend>
        <div className="flex items-center gap-2">
          <input name="price_min" type="number" min={0} inputMode="numeric" defaultValue={sp.get("price_min") ?? ""} placeholder={hint(facets.priceMinUsd) || t("min")} className="h-9 w-full rounded-md border border-slate-300 px-2 text-sm" aria-label={t("min")} />
          <span className="text-slate-400">–</span>
          <input name="price_max" type="number" min={0} inputMode="numeric" defaultValue={sp.get("price_max") ?? ""} placeholder={hint(facets.priceMaxUsd) || t("max")} className="h-9 w-full rounded-md border border-slate-300 px-2 text-sm" aria-label={t("max")} />
        </div>
      </fieldset>

      {group("brand", t("brand"), facets.brands)}
      {group("source", t("source"), facets.sources)}
      {group("condition", t("condition"), facets.conditions, (k) => tc(k as never))}
      {group("movement", t("movement"), facets.movements, (k) => tm(k as never))}
      {facets.countries.length > 1 && group("country", t("country"), facets.countries)}

      <fieldset className="border-t border-slate-200 py-3">
        <legend className="mb-2 text-sm font-semibold">{t("gender")}</legend>
        <ul className="space-y-1">
          {(["men", "women", "unisex"] as const).map((g) => (
            <li key={g}>
              <label className="flex cursor-pointer items-center gap-2 text-sm">
                <input type="checkbox" className="h-4 w-4 accent-slate-900" checked={selected("gender").has(g)} onChange={() => toggle("gender", g)} />
                {tg(g)}
              </label>
            </li>
          ))}
        </ul>
      </fieldset>

      <fieldset className="border-t border-slate-200 py-3">
        <legend className="mb-2 text-sm font-semibold">{t("year")}</legend>
        <div className="flex items-center gap-2">
          <input name="year_min" type="number" min={1900} max={2100} defaultValue={sp.get("year_min") ?? ""} placeholder={t("min")} className="h-9 w-full rounded-md border border-slate-300 px-2 text-sm" aria-label={t("min")} />
          <span className="text-slate-400">–</span>
          <input name="year_max" type="number" min={1900} max={2100} defaultValue={sp.get("year_max") ?? ""} placeholder={t("max")} className="h-9 w-full rounded-md border border-slate-300 px-2 text-sm" aria-label={t("max")} />
        </div>
      </fieldset>

      <fieldset className="border-t border-slate-200 py-3">
        <legend className="mb-2 text-sm font-semibold">{t("diameter")}</legend>
        <div className="flex items-center gap-2">
          <input name="diameter_min" type="number" step="0.5" min={15} max={70} defaultValue={sp.get("diameter_min") ?? ""} placeholder={t("min")} className="h-9 w-full rounded-md border border-slate-300 px-2 text-sm" aria-label={t("min")} />
          <span className="text-slate-400">–</span>
          <input name="diameter_max" type="number" step="0.5" min={15} max={70} defaultValue={sp.get("diameter_max") ?? ""} placeholder={t("max")} className="h-9 w-full rounded-md border border-slate-300 px-2 text-sm" aria-label={t("max")} />
        </div>
      </fieldset>

      <fieldset className="border-t border-slate-200 py-3">
        <legend className="mb-2 text-sm font-semibold">{t("boxPapers")}</legend>
        <label className="flex cursor-pointer items-center gap-2 text-sm">
          <input type="checkbox" className="h-4 w-4 accent-slate-900" checked={sp.get("box") === "true"} onChange={() => toggleBool("box")} />
          {t("withBox")}
        </label>
        <label className="mt-1 flex cursor-pointer items-center gap-2 text-sm">
          <input type="checkbox" className="h-4 w-4 accent-slate-900" checked={sp.get("papers") === "true"} onChange={() => toggleBool("papers")} />
          {t("withPapers")}
        </label>
      </fieldset>

      <button type="submit" className="mt-3 h-10 w-full rounded-md bg-slate-900 text-sm font-medium text-white hover:bg-slate-800">
        {t("apply")}
      </button>
    </form>
  );

  return (
    <>
      <button type="button" onClick={() => setOpen(true)} className="inline-flex h-9 items-center gap-2 rounded-md border border-slate-300 bg-white px-3 text-sm lg:hidden">
        <SlidersHorizontal className="h-4 w-4" aria-hidden />
        {t("filters")}
        {activeCount > 0 && <span className="rounded-full bg-slate-900 px-1.5 text-[11px] text-white">{activeCount}</span>}
      </button>
      <aside className="hidden w-64 shrink-0 lg:block">{body}</aside>
      {open && (
        <div className="fixed inset-0 z-50 flex lg:hidden" role="dialog" aria-modal="true">
          <button type="button" className="flex-1 bg-black/40" aria-label="Close" onClick={() => setOpen(false)} />
          <div className="h-full w-80 max-w-full overflow-auto bg-white p-4 shadow-xl">
            <button type="button" onClick={() => setOpen(false)} className="mb-2 ml-auto flex h-8 w-8 items-center justify-center rounded-md hover:bg-slate-100" aria-label="Close">
              <X className="h-4 w-4" />
            </button>
            {body}
          </div>
        </div>
      )}
    </>
  );
}
