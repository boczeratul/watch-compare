"use client";

import { useEffect, useState, useTransition } from "react";
import { useTranslations } from "next-intl";
import { Trash2 } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { deleteAlert, listAlerts } from "@/app/actions";
import { existingSubscriberId } from "@/lib/push";
import type { Alert } from "@/lib/api";

/** This browser's alerts, read with the subscriber id stored when the first alert was saved. */
export function AlertsList() {
  const t = useTranslations("alerts");
  const [items, setItems] = useState<Alert[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [pending, startTransition] = useTransition();

  useEffect(() => {
    const id = existingSubscriberId();
    (id ? listAlerts(id) : Promise.resolve([])).then((res) => {
      if (res === null) setFailed(true);
      setItems(res ?? []);
    });
  }, []);

  const onDelete = (alert: Alert) => {
    const id = existingSubscriberId();
    if (!id) return;
    startTransition(async () => {
      if (await deleteAlert(id, alert.id)) setItems((cur) => (cur ?? []).filter((a) => a.id !== alert.id));
      else setFailed(true);
    });
  };

  if (items === null) return <div className="mt-6 h-16 animate-pulse rounded-lg bg-slate-100" />;
  return (
    <div className="mt-6" aria-busy={pending}>
      {failed && <p role="alert" className="mb-4 text-sm text-red-700">{t("error")}</p>}
      {items.length === 0 ? (
        <p className="rounded-lg border border-dashed border-slate-300 p-8 text-center text-sm text-slate-600">{t("empty")}</p>
      ) : (
        <ul className="divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white">
          {items.map((a) => (
            <li key={a.id} className="flex items-center gap-3 px-4 py-3">
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium text-slate-900">{a.name}</p>
                <Link href={`/search?${a.query}&sort=newest`} className="text-sm text-emerald-700 hover:underline">
                  {t("viewResults")}
                </Link>
              </div>
              <button
                type="button"
                onClick={() => onDelete(a)}
                disabled={pending}
                className="inline-flex h-9 items-center gap-1.5 rounded-md border border-slate-300 px-3 text-sm text-slate-700 hover:bg-slate-50 disabled:opacity-60"
              >
                <Trash2 className="h-4 w-4" aria-hidden />
                {t("delete")}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
