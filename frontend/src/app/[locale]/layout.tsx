import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { cookies } from "next/headers";
import { NextIntlClientProvider, hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { routing, defaultCurrencyForLocale, type Locale } from "@/i18n/routing";
import { api, safe } from "@/lib/api";
import { CURRENCY_COOKIE, isCurrency, SUPPORTED_CURRENCIES } from "@/lib/settings";
import { CurrencyProvider } from "@/components/providers";
import { Header } from "@/components/header";
import { Footer } from "@/components/footer";
import "../globals.css";

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "meta" });
  const site = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";
  return {
    metadataBase: new URL(site),
    title: { default: `${t("title")} – ${t("tagline")}`, template: `%s · ${t("title")}` },
    description: t("description"),
    alternates: { languages: Object.fromEntries(routing.locales.map((l) => [l, l === routing.defaultLocale ? "/" : `/${l}`])) },
    openGraph: { siteName: t("title"), type: "website" },
  };
}

export default async function LocaleLayout({ children, params }: { children: React.ReactNode; params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) notFound();
  setRequestLocale(locale);

  const jar = await cookies();
  const cookieCurrency = jar.get(CURRENCY_COOKIE)?.value;
  const currency = isCurrency(cookieCurrency) ? cookieCurrency : defaultCurrencyForLocale[locale as Locale];

  const [ratesRes, sourcesRes] = await Promise.all([
    safe(api.rates(), { base: "USD" as const, supported: [...SUPPORTED_CURRENCIES], items: [{ quote: "USD", rate: 1, fetchedAt: "" }] }),
    safe(api.sources(), { items: [] }),
  ]);
  const rates = Object.fromEntries(ratesRes.items.map((r) => [r.quote, r.rate]));
  const supported = SUPPORTED_CURRENCIES.filter((c) => c === "USD" || rates[c]);

  return (
    <html lang={locale}>
      <body className="min-h-screen flex flex-col">
        <NextIntlClientProvider>
          <CurrencyProvider value={{ currency, rates, supported }}>
            <Header />
            <main className="flex-1">{children}</main>
            <Footer sources={sourcesRes.items} />
          </CurrencyProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
