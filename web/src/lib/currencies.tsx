"use client";

import { useEffect } from "react";
import { useSyncExternalStore } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import {
  currencyCatalogVersion,
  hydrateCurrencySymbols,
  subscribeCurrencyCatalog,
} from "@/lib/format";
import type { Currency } from "@/lib/types";

/** Subscribe so money cells re-render when the catalogue hydrates. */
export function useCurrencyCatalog(): number {
  return useSyncExternalStore(
    subscribeCurrencyCatalog,
    currencyCatalogVersion,
    currencyCatalogVersion,
  );
}

export async function reloadCurrencyCatalog(): Promise<void> {
  try {
    const rows = await api.get<Currency[]>("/settings/currencies");
    hydrateCurrencySymbols(rows || []);
  } catch {
    hydrateCurrencySymbols([]);
  }
}

/** Loads Currency.Symbol after login so amounts show TSh, not only TZS. */
export function CurrencyCatalogProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const { user } = useAuth();

  useEffect(() => {
    if (!user) {
      hydrateCurrencySymbols([]);
      return;
    }
    let cancelled = false;
    api
      .get<Currency[]>("/settings/currencies")
      .then((rows) => {
        if (!cancelled) hydrateCurrencySymbols(rows || []);
      })
      .catch(() => {
        if (!cancelled) hydrateCurrencySymbols([]);
      });
    return () => {
      cancelled = true;
    };
  }, [user]);

  return <>{children}</>;
}
