import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { SUPPORT_EMAIL, ISSUES_URL } from "@/lib/contact";

const SECTIONS = ["scope", "collect", "notCollect", "thirdParties", "choices", "children", "changes"] as const;

export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "privacy" });
  return { title: t("title"), description: t("intro") };
}

/** Privacy policy for the website and the iOS app (also the URL App Store Connect asks for). */
export default async function PrivacyPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("privacy");
  const updated = new Intl.DateTimeFormat(locale, { dateStyle: "long" }).format(new Date(t("updatedOn")));
  return (
    <article className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <h1 className="text-3xl font-bold tracking-tight text-slate-900">{t("title")}</h1>
      <p className="mt-2 text-sm text-slate-500">{t("updated", { date: updated })}</p>
      <p className="mt-6 text-base text-slate-700">{t("intro")}</p>
      {SECTIONS.map((key) => (
        <section key={key} className="mt-8">
          <h2 className="text-xl font-semibold text-slate-900">{t(`${key}Title`)}</h2>
          {t.raw(`${key}Body`).map((paragraph: string) => (
            <p key={paragraph} className="mt-3 text-base leading-relaxed text-slate-700">{paragraph}</p>
          ))}
        </section>
      ))}
      <section className="mt-8">
        <h2 className="text-xl font-semibold text-slate-900">{t("contactTitle")}</h2>
        <p className="mt-3 text-base leading-relaxed text-slate-700">
          {t("contactBody")}{" "}
          <a href={`mailto:${SUPPORT_EMAIL}`} className="underline">{SUPPORT_EMAIL}</a>
          {" · "}
          <a href={ISSUES_URL} target="_blank" rel="noopener noreferrer" className="underline">GitHub</a>
        </p>
      </section>
    </article>
  );
}
