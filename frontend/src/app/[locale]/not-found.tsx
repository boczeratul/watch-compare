import { getTranslations } from "next-intl/server";
import { Link } from "@/i18n/navigation";

export default async function NotFound() {
  const t = await getTranslations("listing");
  const tc = await getTranslations("common");
  return (
    <div className="mx-auto max-w-2xl px-4 py-24 text-center">
      <h1 className="text-2xl font-bold text-slate-900">{t("notFound")}</h1>
      <p className="mt-2 text-slate-600">{t("notFoundBody")}</p>
      <Link href="/" className="mt-6 inline-block rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white">
        {tc("backHome")}
      </Link>
    </div>
  );
}
