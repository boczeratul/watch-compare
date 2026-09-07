"use client";

import { createContext, useContext, type ReactNode } from "react";
import type { RateTable } from "@/lib/currency";

interface CurrencyContextValue {
  currency: string;
  rates: RateTable;
  supported: string[];
}

const CurrencyContext = createContext<CurrencyContextValue>({ currency: "USD", rates: { USD: 1 }, supported: ["USD"] });

export function CurrencyProvider({ value, children }: { value: CurrencyContextValue; children: ReactNode }) {
  return <CurrencyContext.Provider value={value}>{children}</CurrencyContext.Provider>;
}

export function useCurrency() {
  return useContext(CurrencyContext);
}
