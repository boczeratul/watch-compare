import type { MetadataRoute } from "next";
import { routing } from "@/i18n/routing";

export default function sitemap(): MetadataRoute.Sitemap {
  const site = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";
  const paths = ["", "/search", "/brands"];
  return routing.locales.flatMap((locale) =>
    paths.map((p) => ({
      url: `${site}${locale === routing.defaultLocale ? "" : `/${locale}`}${p}`,
      changeFrequency: "daily" as const,
      priority: p === "" ? 1 : 0.7,
    })),
  );
}
