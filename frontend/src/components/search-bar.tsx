"use client";

import { useEffect, useId, useMemo, useState, type FormEvent, type KeyboardEvent } from "react";
import { Clock, Search, X } from "lucide-react";
import { useTranslations } from "next-intl";
import { useRouter } from "@/i18n/navigation";
import { cn } from "@/lib/utils";
import {
  addRecentSearch,
  clearRecentSearches,
  completionFor,
  loadRecentSearches,
  matchRecentSearches,
  removeRecentSearch,
  subscribeRecentSearches,
} from "@/lib/recent-searches";

export function SearchBar({ initial = "", size = "md", className }: { initial?: string; size?: "md" | "lg"; className?: string }) {
  const t = useTranslations("home");
  const router = useRouter();
  const listId = useId();
  const [q, setQ] = useState(initial);
  const [recent, setRecent] = useState<string[]>([]);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1); // -1 = the typed text, otherwise an index into `suggestions`

  // localStorage is only readable on the client; loading it after mount keeps SSR markup identical.
  useEffect(() => {
    const sync = () => setRecent(loadRecentSearches());
    sync();
    return subscribeRecentSearches(sync);
  }, []);

  const suggestions = useMemo(() => matchRecentSearches(recent, q), [recent, q]);
  const completion = active < 0 ? completionFor(suggestions, q) : null;
  const showList = open && suggestions.length > 0;
  const showCompletion = open && completion !== null;

  const go = (raw: string) => {
    const query = raw.trim();
    setOpen(false);
    setActive(-1);
    if (query) {
      setQ(query);
      setRecent(addRecentSearch(query));
    }
    router.push({ pathname: "/search", query: query ? { q: query } : {} });
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    go(active >= 0 && suggestions[active] ? suggestions[active] : q);
  };

  const acceptCompletion = () => {
    if (!completion) return false;
    setQ(completion);
    setActive(-1);
    return true;
  };

  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    switch (e.key) {
      case "ArrowDown":
        if (suggestions.length === 0) return;
        e.preventDefault();
        setOpen(true);
        setActive((i) => (i + 1 >= suggestions.length ? -1 : i + 1));
        break;
      case "ArrowUp":
        if (suggestions.length === 0) return;
        e.preventDefault();
        setOpen(true);
        setActive((i) => (i - 1 < -1 ? suggestions.length - 1 : i - 1));
        break;
      case "Tab":
        if (showCompletion && acceptCompletion()) e.preventDefault();
        break;
      case "ArrowRight":
      case "End": {
        const el = e.currentTarget;
        const atEnd = el.selectionStart === el.value.length && el.selectionEnd === el.value.length;
        if (showCompletion && atEnd && acceptCompletion()) e.preventDefault();
        break;
      }
      case "Escape":
        if (open) {
          e.preventDefault();
          setOpen(false);
          setActive(-1);
        }
        break;
    }
  };

  const remove = (item: string) => {
    setRecent(removeRecentSearch(item));
    setActive(-1);
  };
  const clearAll = () => {
    setRecent(clearRecentSearches());
    setActive(-1);
    setOpen(false);
  };

  const textSize = size === "lg" ? "h-12 text-base" : "h-9 text-sm";
  const optionId = (i: number) => `${listId}-opt-${i}`;

  return (
    <form
      onSubmit={submit}
      role="search"
      className={cn("relative flex w-full items-stretch rounded-lg border border-slate-300 bg-white shadow-sm focus-within:border-slate-900 focus-within:ring-1 focus-within:ring-slate-900", className)}
    >
      <Search className={cn("ml-3 shrink-0 self-center text-slate-400", size === "lg" ? "h-5 w-5" : "h-4 w-4")} aria-hidden />
      <div className="relative min-w-0 flex-1">
        {showCompletion && (
          <div aria-hidden className={cn("pointer-events-none absolute inset-0 flex items-center overflow-hidden whitespace-pre px-3", textSize)}>
            <span className="invisible">{q}</span>
            <span className="text-slate-400">{completion.slice(q.length)}</span>
          </div>
        )}
        <input
          type="search"
          name="q"
          value={q}
          onChange={(e) => {
            setQ(e.target.value);
            setOpen(true);
            setActive(-1);
          }}
          onFocus={() => setOpen(true)}
          onClick={() => setOpen(true)}
          onBlur={() => {
            setOpen(false);
            setActive(-1);
          }}
          onKeyDown={onKeyDown}
          placeholder={t("searchPlaceholder")}
          aria-label={t("searchPlaceholder")}
          autoComplete="off"
          autoCorrect="off"
          spellCheck={false}
          role="combobox"
          aria-autocomplete="both"
          aria-expanded={showList}
          aria-controls={showList ? listId : undefined}
          aria-activedescendant={showList && active >= 0 ? optionId(active) : undefined}
          className={cn("relative w-full bg-transparent px-3 text-slate-900 placeholder:text-slate-400 focus:outline-none", textSize)}
        />
      </div>
      <button type="submit" className={cn("shrink-0 rounded-r-lg bg-slate-900 px-4 font-medium text-white hover:bg-slate-800", size === "lg" ? "text-base" : "text-sm")}>
        {t("searchButton")}
      </button>

      {showList && (
        <div
          // Keep focus in the input so onBlur does not close the list before a click lands.
          onMouseDown={(e) => e.preventDefault()}
          className="absolute left-0 right-0 top-full z-50 mt-1 overflow-hidden rounded-lg border border-slate-200 bg-white text-left shadow-lg"
        >
          <div className="flex items-center justify-between px-3 pt-2 pb-1 text-xs text-slate-500">
            <span>{t("recentSearches")}</span>
            <button type="button" onClick={clearAll} className="font-medium text-slate-500 hover:text-slate-900">
              {t("clearRecent")}
            </button>
          </div>
          <ul id={listId} role="listbox" aria-label={t("recentSearches")} className="pb-1">
            {suggestions.map((item, i) => (
              <li
                key={item}
                id={optionId(i)}
                role="option"
                aria-selected={i === active}
                onMouseEnter={() => setActive(i)}
                onClick={() => go(item)}
                className={cn("flex cursor-pointer items-center gap-2 px-3 py-1.5 text-sm text-slate-800", i === active ? "bg-slate-100" : "hover:bg-slate-50")}
              >
                <Clock className="h-4 w-4 shrink-0 text-slate-400" aria-hidden />
                <span className="min-w-0 flex-1 truncate">{item}</span>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    remove(item);
                  }}
                  aria-label={t("removeRecent", { query: item })}
                  className="shrink-0 rounded p-0.5 text-slate-400 hover:bg-slate-200 hover:text-slate-700"
                >
                  <X className="h-3.5 w-3.5" aria-hidden />
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </form>
  );
}
