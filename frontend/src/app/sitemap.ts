import type { MetadataRoute } from "next";
import { routing } from "@/i18n/routing";

export default function sitemap(): MetadataRoute.Sitemap {
  const site = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";
  const paths = ["", "/search", "/brands", "/support", "/privacy"];
  const STATIC = new Set(["/support", "/privacy"]);
  return routing.locales.flatMap((locale) =>
    paths.map((p) => ({
      url: `${site}${locale === routing.defaultLocale ? "" : `/${locale}`}${p}`,
      changeFrequency: STATIC.has(p) ? ("yearly" as const) : ("daily" as const),
      priority: p === "" ? 1 : STATIC.has(p) ? 0.3 : 0.7,
    })),
  );
}
