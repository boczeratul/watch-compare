"use client";

import { useState, type FormEvent } from "react";
import { Search } from "lucide-react";
import { useTranslations } from "next-intl";
import { useRouter } from "@/i18n/navigation";
import { cn } from "@/lib/utils";

export function SearchBar({ initial = "", size = "md", className }: { initial?: string; size?: "md" | "lg"; className?: string }) {
  const t = useTranslations("home");
  const router = useRouter();
  const [q, setQ] = useState(initial);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const query = q.trim();
    router.push({ pathname: "/search", query: query ? { q: query } : {} });
  };

  return (
    <form onSubmit={submit} role="search" className={cn("flex w-full items-stretch overflow-hidden rounded-lg border border-slate-300 bg-white shadow-sm focus-within:border-slate-900 focus-within:ring-1 focus-within:ring-slate-900", className)}>
      <Search className={cn("ml-3 shrink-0 self-center text-slate-400", size === "lg" ? "h-5 w-5" : "h-4 w-4")} aria-hidden />
      <input
        type="search"
        name="q"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder={t("searchPlaceholder")}
        aria-label={t("searchPlaceholder")}
        className={cn("w-full bg-transparent px-3 text-slate-900 placeholder:text-slate-400 focus:outline-none", size === "lg" ? "h-12 text-base" : "h-9 text-sm")}
      />
      <button type="submit" className={cn("shrink-0 bg-slate-900 px-4 font-medium text-white hover:bg-slate-800", size === "lg" ? "text-base" : "text-sm")}>
        {t("searchButton")}
      </button>
    </form>
  );
}
