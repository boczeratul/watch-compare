import { cn } from "@/lib/utils";

const STYLES: Record<string, string> = {
  chrono24: "bg-emerald-50 text-emerald-800 ring-emerald-200",
  ebay: "bg-amber-50 text-amber-800 ring-amber-200",
  watchnian: "bg-sky-50 text-sky-800 ring-sky-200",
  jackroad: "bg-rose-50 text-rose-800 ring-rose-200",
  hourstack: "bg-violet-50 text-violet-800 ring-violet-200",
  rdwatch: "bg-orange-50 text-orange-800 ring-orange-200",
  commitwatch: "bg-teal-50 text-teal-800 ring-teal-200",
};

export function SourceBadge({ source, name, className }: { source: string; name: string; className?: string }) {
  return (
    <span className={cn("inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ring-1 ring-inset", STYLES[source] ?? "bg-slate-50 text-slate-700 ring-slate-200", className)}>
      {name}
    </span>
  );
}
