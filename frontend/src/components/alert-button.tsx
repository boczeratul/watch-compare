"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Bell, BellRing } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { createAlert } from "@/app/actions";
import { ONESIGNAL_APP_ID, enablePush, loadPush } from "@/lib/push";

type State = "idle" | "naming" | "saving" | "saved" | "denied" | "unsupported" | "limit" | "criteria" | "error";

/** "Notify me of new matches": asks for an optional name, then saves the current search's criteria
 *  as a push alert. */
export function AlertButton({ query }: { query: string }) {
  const t = useTranslations("alerts");
  const [state, setState] = useState<State>("idle");
  const [name, setName] = useState("");

  // Load the SDK ahead of the click so the permission prompt keeps the click's user activation.
  useEffect(() => {
    loadPush()?.catch(() => undefined);
  }, []);

  if (!ONESIGNAL_APP_ID) return null;

  const onClick = () => {
    if (!query) {
      setState("criteria");
      return;
    }
    setName("");
    setState("naming");
  };

  // Runs from the Save click, so the permission prompt still has a user activation.
  const onSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setState("saving");
    try {
      const push = await enablePush();
      if (push.status !== "granted" || !push.subscriberId) {
        setState(push.status === "denied" ? "denied" : "unsupported");
        return;
      }
      const res = await createAlert(push.subscriberId, query, name);
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

  if (state === "naming") {
    return (
      <form onSubmit={onSave} className="flex w-full max-w-xs flex-col items-end gap-1">
        <label className="w-full text-xs font-medium text-slate-700">
          {t("nameTitle")}
          <input
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={80}
            placeholder={t("namePlaceholder")}
            className="mt-1 block h-9 w-full rounded-md border border-slate-300 bg-white px-3 text-sm font-normal text-slate-900 shadow-sm"
          />
        </label>
        <p className="text-right text-xs text-slate-500">{t("nameHint")}</p>
        <div className="flex gap-2">
          <button type="button" onClick={() => setState("idle")} className="h-9 rounded-md px-3 text-sm text-slate-700 hover:bg-slate-50">
            {t("cancel")}
          </button>
          <button type="submit" className="inline-flex h-9 items-center gap-1.5 rounded-md bg-slate-900 px-3 text-sm font-medium text-white hover:bg-slate-800">
            <Bell className="h-4 w-4" aria-hidden />
            {t("save")}
          </button>
        </div>
      </form>
    );
  }

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
