"use client";

import { useEffect } from "react";
import { useSyncExternalStore } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import {
  decimalPrecisionVersion,
  hydrateDecimalPrecision,
  subscribeDecimalPrecision,
} from "@/lib/format";
import type { DecimalPrecisionSettings } from "@/lib/types";

export function useDecimalPrecision(): number {
  return useSyncExternalStore(
    subscribeDecimalPrecision,
    decimalPrecisionVersion,
    decimalPrecisionVersion,
  );
}

export async function reloadDecimalPrecision(): Promise<void> {
  try {
    const row = await api.get<DecimalPrecisionSettings>("/settings/precision");
    hydrateDecimalPrecision(row);
  } catch {
    hydrateDecimalPrecision(null);
  }
}

export function DecimalPrecisionProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const { user } = useAuth();

  useEffect(() => {
    if (!user) {
      hydrateDecimalPrecision(null);
      return;
    }
    let cancelled = false;
    api
      .get<DecimalPrecisionSettings>("/settings/precision")
      .then((row) => {
        if (!cancelled) hydrateDecimalPrecision(row);
      })
      .catch(() => {
        if (!cancelled) hydrateDecimalPrecision(null);
      });
    return () => {
      cancelled = true;
    };
  }, [user]);

  return <>{children}</>;
}
