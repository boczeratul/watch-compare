import { getTranslations } from "next-intl/server";
import type { Source } from "@/lib/api";

export async function Footer({ sources }: { sources: Source[] }) {
  const t = await getTranslations("footer");
  return (
    <footer className="mt-16 border-t border-slate-200 bg-slate-50">
      <div className="mx-auto max-w-7xl px-4 py-10 text-sm text-slate-600 sm:px-6">
        <div className="flex flex-col gap-6 md:flex-row md:justify-between">
          <p className="max-w-2xl">{t("disclaimer")}</p>
          <div>
            <h2 className="mb-2 font-semibold text-slate-800">{t("sources")}</h2>
            <ul className="flex flex-wrap gap-x-4 gap-y-1">
              {sources.map((s) => (
                <li key={s.key}>
                  <a href={s.baseUrl} target="_blank" rel="noopener noreferrer nofollow" className="hover:text-slate-900">
                    {s.name}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        </div>
        <p className="mt-8 text-xs text-slate-400">© {new Date().getFullYear()} WatchCompare</p>
      </div>
    </footer>
  );
}
