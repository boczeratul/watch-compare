"use server";

import { cookies } from "next/headers";
import { CURRENCY_COOKIE, isCurrency } from "@/lib/settings";

export async function setCurrency(currency: string) {
  if (!isCurrency(currency)) return;
  const jar = await cookies();
  jar.set(CURRENCY_COOKIE, currency, { path: "/", maxAge: 60 * 60 * 24 * 365, sameSite: "lax" });
}
