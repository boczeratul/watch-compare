"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Bell, BellRing } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { createAlert } from "@/app/actions";
import { ONESIGNAL_APP_ID, enablePush, loadPush } from "@/lib/push";

type State = "idle" | "saving" | "saved" | "denied" | "unsupported" | "limit" | "criteria" | "error";

/** "Notify me of new matches": saves the current search's criteria as a push alert. */
export function AlertButton({ query }: { query: string }) {
  const t = useTranslations("alerts");
  const [state, setState] = useState<State>("idle");

  // Load the SDK ahead of the click so the permission prompt keeps the click's user activation.
  useEffect(() => {
    loadPush()?.catch(() => undefined);
  }, []);

  if (!ONESIGNAL_APP_ID) return null;

  const onClick = async () => {
    if (!query) {
      setState("criteria");
      return;
    }
    setState("saving");
    try {
      const push = await enablePush();
      if (push.status !== "granted" || !push.subscriberId) {
        setState(push.status === "denied" ? "denied" : "unsupported");
        return;
      }
      const res = await createAlert(push.subscriberId, query);
      setState(res.ok ? "saved" : res.error);
    } catch {
      setState("error");
    }
  };

  const message: Partial<Record<State, string>> = {
    saved: t("created"),
    denied: t("permissionDenied"),
    unsupported: t("unsupported"),
    limit: t("limit"),
    criteria: t("needsCriteria"),
    error: t("error"),
  };

  return (
    <div className="flex flex-col items-end gap-1">
      <button
        type="button"
        onClick={onClick}
        disabled={state === "saving" || state === "saved"}
        className="inline-flex h-9 items-center gap-1.5 rounded-md border border-slate-300 bg-white px-3 text-sm font-medium text-slate-800 shadow-sm hover:bg-slate-50 disabled:opacity-60"
      >
        {state === "saved" ? <BellRing className="h-4 w-4 text-emerald-700" aria-hidden /> : <Bell className="h-4 w-4" aria-hidden />}
        {state === "saving" ? t("saving") : t("create")}
      </button>
      {message[state] && (
        <p role="status" className="max-w-xs text-right text-xs text-slate-600">
          {message[state]}{" "}
          {(state === "saved" || state === "limit") && (
            <Link href="/alerts" className="underline">{t("manage")}</Link>
          )}
        </p>
      )}
    </div>
  );
}
