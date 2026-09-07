"use client";

import { useLocale, useTranslations } from "next-intl";
import { useCurrency } from "./providers";
import { convertFromUsd, formatMoney, roundDisplay } from "@/lib/currency";
import { cn } from "@/lib/utils";

interface PriceProps {
  usd?: number;
  original?: number;
  originalCurrency?: string;
  size?: "sm" | "md" | "lg";
  showOriginal?: boolean;
  className?: string;
}

/** Shows a price in the visitor's currency, with the seller's original price underneath. */
export function Price({ usd, original, originalCurrency, size = "md", showOriginal = true, className }: PriceProps) {
  const { currency, rates } = useCurrency();
  const locale = useLocale();
  const t = useTranslations("listing");

  if (original === undefined && usd === undefined) {
    return <span className={cn("text-slate-400", className)}>—</span>;
  }
  const sizeClass = { sm: "text-sm", md: "text-base", lg: "text-2xl" }[size];

  // Prefer the exact original when it is already in the visitor's currency.
  let main: string | null = null;
  if (original !== undefined && originalCurrency === currency) {
    main = formatMoney(original, currency, locale);
  } else if (usd !== undefined) {
    const converted = convertFromUsd(usd, currency, rates);
    if (converted !== null) main = formatMoney(roundDisplay(converted, currency), currency, locale);
  }
  if (main === null && original !== undefined && originalCurrency) {
    main = formatMoney(original, originalCurrency, locale);
  }
  const showOrig = showOriginal && original !== undefined && originalCurrency && originalCurrency !== currency;

  return (
    <span className={cn("inline-flex flex-col leading-tight", className)}>
      <span className={cn("font-semibold text-slate-900", sizeClass)}>{main ?? "—"}</span>
      {showOrig && (
        <span className="text-xs text-slate-500" title={t("converted")}>
          {formatMoney(original, originalCurrency, locale)}
        </span>
      )}
    </span>
  );
}
