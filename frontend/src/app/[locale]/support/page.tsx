import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { Link } from "@/i18n/navigation";
import { SUPPORT_EMAIL, ISSUES_URL } from "@/lib/contact";

const FAQ = ["outdated", "price", "currency", "wrongData", "sources", "data"] as const;

export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "support" });
  return { title: t("title"), description: t("intro") };
}

/** Support page for the website and the iOS app (the URL App Store Connect asks for). */
export default async function SupportPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("support");
  return (
    <article className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <h1 className="text-3xl font-bold tracking-tight text-slate-900">{t("title")}</h1>
      <p className="mt-4 text-base text-slate-700">{t("intro")}</p>

      <section className="mt-8 rounded-lg border border-slate-200 bg-slate-50 p-5">
        <h2 className="text-xl font-semibold text-slate-900">{t("contactTitle")}</h2>
        <p className="mt-2 text-base text-slate-700">{t("contactBody")}</p>
        <a href={`mailto:${SUPPORT_EMAIL}`} className="mt-3 inline-flex h-11 items-center rounded-md bg-slate-900 px-5 text-sm font-semibold text-white hover:bg-slate-800">
          {SUPPORT_EMAIL}
        </a>
        <p className="mt-3 text-sm text-slate-500">
          {t("issuesBody")}{" "}
          <a href={ISSUES_URL} target="_blank" rel="noopener noreferrer" className="underline">GitHub</a>
        </p>
        <p className="mt-2 text-sm text-slate-500">{t("responseTime")}</p>
      </section>

      <section className="mt-10">
        <h2 className="text-xl font-semibold text-slate-900">{t("faqTitle")}</h2>
        <dl className="mt-4 divide-y divide-slate-200">
          {FAQ.map((key) => (
            <div key={key} className="py-4">
              <dt className="font-medium text-slate-900">{t(`faq.${key}Q`)}</dt>
              <dd className="mt-2 text-base leading-relaxed text-slate-700">{t(`faq.${key}A`)}</dd>
            </div>
          ))}
        </dl>
      </section>

      <p className="mt-10 text-sm text-slate-500">
        <Link href="/privacy" className="underline">{t("privacyLink")}</Link>
      </p>
    </article>
  );
}
