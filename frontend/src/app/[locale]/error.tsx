"use client";

import { useEffect } from "react";
import { useTranslations } from "next-intl";

export default function ErrorPage({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  const t = useTranslations("common");
  useEffect(() => {
    console.error(error);
  }, [error]);
  return (
    <div className="mx-auto max-w-2xl px-4 py-24 text-center">
      <h1 className="text-xl font-semibold text-slate-900">{t("error")}</h1>
      <button type="button" onClick={reset} className="mt-6 rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white">
        ↻
      </button>
    </div>
  );
}
