"use client";

import { useTranslations } from "next-intl";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Link, usePathname } from "@/i18n/navigation";
import { useSearchParams } from "next/navigation";
import { cn } from "@/lib/utils";

export function Pagination({ page, totalPages }: { page: number; totalPages: number }) {
  const t = useTranslations("search");
  const pathname = usePathname();
  const sp = useSearchParams();
  if (totalPages <= 1) return null;

  const href = (p: number) => {
    const next = new URLSearchParams(sp.toString());
    if (p <= 1) next.delete("page");
    else next.set("page", String(p));
    const qs = next.toString();
    return qs ? `${pathname}?${qs}` : pathname;
  };
  const pages = new Set<number>([1, totalPages, page - 1, page, page + 1].filter((p) => p >= 1 && p <= totalPages));
  const list = [...pages].sort((a, b) => a - b);
  const btn = "inline-flex h-9 min-w-9 items-center justify-center rounded-md border px-2 text-sm";

  return (
    <nav className="mt-8 flex flex-wrap items-center justify-center gap-1" aria-label={t("page", { page, total: totalPages })}>
      <Link href={href(page - 1)} aria-disabled={page <= 1} className={cn(btn, "border-slate-300 bg-white text-slate-700 hover:bg-slate-50", page <= 1 && "pointer-events-none opacity-40")}>
        <ChevronLeft className="h-4 w-4" aria-hidden />
        <span className="sr-only">{t("previous")}</span>
      </Link>
      {list.map((p, i) => (
        <span key={p} className="flex items-center gap-1">
          {i > 0 && list[i - 1] !== p - 1 && <span className="px-1 text-slate-400">…</span>}
          <Link href={href(p)} aria-current={p === page ? "page" : undefined} className={cn(btn, p === page ? "border-slate-900 bg-slate-900 text-white" : "border-slate-300 bg-white text-slate-700 hover:bg-slate-50")}>
            {p}
          </Link>
        </span>
      ))}
      <Link href={href(page + 1)} aria-disabled={page >= totalPages} className={cn(btn, "border-slate-300 bg-white text-slate-700 hover:bg-slate-50", page >= totalPages && "pointer-events-none opacity-40")}>
        <ChevronRight className="h-4 w-4" aria-hidden />
        <span className="sr-only">{t("next")}</span>
      </Link>
    </nav>
  );
}
