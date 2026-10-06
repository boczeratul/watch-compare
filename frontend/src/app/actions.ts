"use server";

import { cookies } from "next/headers";
import { CURRENCY_COOKIE, isCurrency } from "@/lib/settings";
import { alertsApi, type Alert, type CreateAlertResult } from "@/lib/api";

export async function setCurrency(currency: string) {
  if (!isCurrency(currency)) return;
  const jar = await cookies();
  jar.set(CURRENCY_COOKIE, currency, { path: "/", maxAge: 60 * 60 * 24 * 365, sameSite: "lax" });
}

// Push alerts. The subscriber id is the browser's random OneSignal external_id (see lib/push.ts);
// the API validates it and scopes every alert to it.

export async function listAlerts(subscriberId: string): Promise<Alert[] | null> {
  try {
    return await alertsApi.list(subscriberId);
  } catch (err) {
    console.error("[alerts]", err instanceof Error ? err.message : err);
    return null;
  }
}

export async function createAlert(subscriberId: string, query: string, name = ""): Promise<CreateAlertResult> {
  try {
    return await alertsApi.create(subscriberId, query, name.trim());
  } catch (err) {
    console.error("[alerts]", err instanceof Error ? err.message : err);
    return { ok: false, error: "error" };
  }
}

export async function renameAlert(subscriberId: string, id: number, name: string): Promise<Alert | null> {
  try {
    return await alertsApi.rename(subscriberId, id, name.trim());
  } catch (err) {
    console.error("[alerts]", err instanceof Error ? err.message : err);
    return null;
  }
}

export async function deleteAlert(subscriberId: string, id: number): Promise<boolean> {
  try {
    return await alertsApi.remove(subscriberId, id);
  } catch (err) {
    console.error("[alerts]", err instanceof Error ? err.message : err);
    return false;
  }
}
