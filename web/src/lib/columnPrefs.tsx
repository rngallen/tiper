"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { TableColumnPref } from "@/lib/types";
import type { Column } from "@/components/DataTable";

export type ColumnDef<T> = Column<T> & {
  /** Stable id used for prefs (required for customizable columns). */
  id: string;
  /** Always shown; omitted from Edit Columns. */
  fixed?: boolean;
  /** Shown when no preference is stored (default true). */
  defaultVisible?: boolean;
};

function storageKey(tableId: string) {
  return `dfms.cols.${tableId}`;
}

function readLocal(tableId: string): TableColumnPref | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(storageKey(tableId));
    if (!raw) return null;
    return JSON.parse(raw) as TableColumnPref;
  } catch {
    return null;
  }
}

function writeLocal(tableId: string, pref: TableColumnPref) {
  try {
    localStorage.setItem(storageKey(tableId), JSON.stringify(pref));
  } catch {
    /* ignore */
  }
}

function defaultOrder<T>(defs: ColumnDef<T>[]): string[] {
  return defs.filter((d) => !d.fixed).map((d) => d.id);
}

function defaultHidden<T>(defs: ColumnDef<T>[]): string[] {
  return defs
    .filter((d) => !d.fixed && d.defaultVisible === false)
    .map((d) => d.id);
}

/** Build visible columns from catalog + preference. */
function applyColumnPref<T>(
  defs: ColumnDef<T>[],
  pref: TableColumnPref | null | undefined,
): ColumnDef<T>[] {
  const flexible = defs.filter((d) => !d.fixed);
  const byId = new Map(flexible.map((d) => [d.id, d]));

  const order =
    pref?.order && pref.order.length > 0
      ? [
          ...pref.order.filter((id) => byId.has(id)),
          ...flexible.map((d) => d.id).filter((id) => !pref.order!.includes(id)),
        ]
      : defaultOrder(defs);

  const hidden = new Set(
    pref?.hidden ??
      (pref?.order?.length ? [] : defaultHidden(defs)),
  );
  // If prefs exist but hidden omitted, also hide defaultVisible:false only when no order saved
  if (!pref?.order?.length && !pref?.hidden) {
    for (const id of defaultHidden(defs)) hidden.add(id);
  }

  const orderedFlex = order
    .map((id) => byId.get(id))
    .filter((d): d is ColumnDef<T> => Boolean(d))
    .filter((d) => !hidden.has(d.id));

  const leadingFixed: ColumnDef<T>[] = [];
  const trailingFixed: ColumnDef<T>[] = [];
  let seenFlex = false;
  for (const d of defs) {
    if (!d.fixed) {
      seenFlex = true;
      continue;
    }
    if (!seenFlex) leadingFixed.push(d);
    else trailingFixed.push(d);
  }

  return [...leadingFixed, ...orderedFlex, ...trailingFixed];
}

export function useColumnPrefs<T>(tableId: string, catalog: ColumnDef<T>[]) {
  const { user, refreshUser } = useAuth();
  const profilePref =
    user?.profile?.appearanceSettings?.tablePrefs?.[tableId] ?? null;

  const [pref, setPref] = useState<TableColumnPref>(() => {
    return profilePref || readLocal(tableId) || {};
  });

  useEffect(() => {
    if (profilePref) {
      setPref(profilePref);
      writeLocal(tableId, profilePref);
      return;
    }
    const local = readLocal(tableId);
    if (local) setPref(local);
  }, [tableId, JSON.stringify(profilePref)]);

  const columns = useMemo(
    () => applyColumnPref(catalog, pref),
    [catalog, pref],
  );

  const save = useCallback(
    async (next: TableColumnPref) => {
      setPref(next);
      writeLocal(tableId, next);
      try {
        await api.put("/auth/profile/table-prefs", {
          tableId,
          order: next.order || [],
          hidden: next.hidden || [],
        });
        await refreshUser();
      } catch {
        /* local preference still applied */
      }
    },
    [tableId, refreshUser],
  );

  const restoreDefaults = useCallback(async () => {
    await save({
      order: defaultOrder(catalog),
      hidden: defaultHidden(catalog),
    });
  }, [save, catalog]);

  return {
    columns,
    pref,
    save,
    restoreDefaults,
    catalog,
  };
}
