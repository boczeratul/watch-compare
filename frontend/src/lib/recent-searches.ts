// Recent search queries, persisted per browser in localStorage.
// Every access is wrapped in try/catch: storage can be unavailable (private mode,
// blocked site data) or throw on quota, and the search bar must still work without it.

export const RECENT_SEARCHES_KEY = "wc_recent_searches";
export const RECENT_SEARCHES_MAX = 10;

function read(): string[] {
  try {
    const raw = window.localStorage.getItem(RECENT_SEARCHES_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((v): v is string => typeof v === "string" && v.trim().length > 0).slice(0, RECENT_SEARCHES_MAX);
  } catch {
    return [];
  }
}

function write(items: string[]): string[] {
  try {
    if (items.length === 0) window.localStorage.removeItem(RECENT_SEARCHES_KEY);
    else window.localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(items));
  } catch {
    // Storage unavailable or full: keep the in-memory list for this page only.
  }
  window.dispatchEvent(new Event(RECENT_SEARCHES_EVENT));
  return items;
}

export function loadRecentSearches(): string[] {
  if (typeof window === "undefined") return [];
  return read();
}

/** Moves `query` to the front, dropping earlier case-insensitive duplicates. Returns the new list. */
export function addRecentSearch(query: string): string[] {
  const q = query.trim();
  if (!q || typeof window === "undefined") return loadRecentSearches();
  const key = q.toLowerCase();
  const rest = read().filter((v) => v.toLowerCase() !== key);
  return write([q, ...rest].slice(0, RECENT_SEARCHES_MAX));
}

export function removeRecentSearch(query: string): string[] {
  if (typeof window === "undefined") return [];
  const key = query.toLowerCase();
  return write(read().filter((v) => v.toLowerCase() !== key));
}

export function clearRecentSearches(): string[] {
  if (typeof window === "undefined") return [];
  return write([]);
}

/**
 * Suggestions for a partial query: prefix matches first (they can be inline-completed),
 * then substring matches, both in most-recent-first order. An empty query returns everything.
 */
export function matchRecentSearches(recent: string[], partial: string, limit = 8): string[] {
  const p = partial.trim().toLowerCase();
  if (!p) return recent.slice(0, limit);
  const prefix: string[] = [];
  const contains: string[] = [];
  for (const item of recent) {
    const lower = item.toLowerCase();
    if (lower === p) continue; // Nothing to suggest for an exact match.
    if (lower.startsWith(p)) prefix.push(item);
    else if (lower.includes(p)) contains.push(item);
  }
  return [...prefix, ...contains].slice(0, limit);
}

/** The inline completion for `partial`, i.e. the first suggestion that starts with it (case-insensitive). */
export function completionFor(suggestions: string[], partial: string): string | null {
  const p = partial.toLowerCase();
  if (!p || partial !== partial.trimStart()) return null;
  const hit = suggestions.find((s) => s.toLowerCase().startsWith(p) && s.length > partial.length);
  return hit ?? null;
}

/** Fired on `window` after every write, so several search bars on one page stay in sync. */
export const RECENT_SEARCHES_EVENT = "wc:recent-searches";

/** Calls `cb` whenever the list changes in this tab (custom event) or another tab (storage event). */
export function subscribeRecentSearches(cb: () => void): () => void {
  if (typeof window === "undefined") return () => {};
  const onStorage = (e: StorageEvent) => {
    if (e.key === null || e.key === RECENT_SEARCHES_KEY) cb();
  };
  window.addEventListener(RECENT_SEARCHES_EVENT, cb);
  window.addEventListener("storage", onStorage);
  return () => {
    window.removeEventListener(RECENT_SEARCHES_EVENT, cb);
    window.removeEventListener("storage", onStorage);
  };
}
