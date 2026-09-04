import { api } from "./api";
import { formatDate, localISODate } from "./format";
import { searchMaster, type MasterRef } from "./master";
import { asRows } from "./pagination";
import type { CreatedBy } from "./creator";

export type FeeCode = "FSF" | "VSF" | "KOJ" | "TBS";

export const BILLING_UNITS = [
  { value: "L", label: "L" },
  { value: "MT", label: "MT" },
  { value: "M3", label: "m³" },
] as const;

export type PriceLine = {
  product?: MasterRef | null;
  unit: string;
  sourceCurrencyCode: string;
  sourcePrice: string;
  homePrice?: string;
};

export type PriceBatch = {
  id: string;
  documentNumber: string;
  date: string;
  effectiveFrom?: string;
  description?: string;
  currencyCode?: string;
  exchangeRate?: string;
  fxManual?: boolean;
  status: string;
  createdBy?: CreatedBy;
  fees?: PriceLine[];
};

export type FcfTier = {
  phase: "first" | "nth";
  fromQty: string;
  toQty?: string;
  sourcePrice: string;
  homePrice?: string;
};

export type FcfLine = {
  classOfTrade: string;
  procurementMethod: string;
  dischargeRoute: string;
  collectionMethod?: string;
  isPromotional?: boolean;
  firstDays: number;
  firstChargeTo: string;
  firstRateKind: "flat" | "tier";
  firstUnit: string;
  firstSourceCurrencyCode: string;
  firstSourcePrice: string;
  firstHomePrice?: string;
  nthDays: number;
  nthRateKind: "flat" | "tier";
  nthUnit: string;
  nthSourceCurrencyCode: string;
  nthSourcePrice: string;
  nthHomePrice?: string;
  tiers?: FcfTier[];
};

export type FcfBatch = {
  id: string;
  documentNumber: string;
  date: string;
  effectiveFrom?: string;
  description?: string;
  currencyCode: string;
  exchangeRate: string;
  fxManual?: boolean;
  status: string;
  createdBy?: CreatedBy;
  lines?: FcfLine[];
};

export type VarContract = {
  contractTypeCode: string;
  miLossValue: string;
  feeUsdCm?: string;
  feeUsdMt?: string;
  feeTzsCm?: string;
  feeTzsMt?: string;
};

export type VarProduct = {
  product?: MasterRef | null;
  ewuraPrice: string;
  density: string;
  wholesaleCm?: string;
  wholesaleMt?: string;
  contracts?: VarContract[];
};

export type VariableBatch = {
  id: string;
  documentNumber: string;
  date: string;
  effectiveFrom: string;
  description?: string;
  currencyCode?: string;
  exchangeRate: string;
  fxManual?: boolean;
  status: string;
  createdBy?: CreatedBy;
  miLossBatch?: { id: string; documentNumber: string; date?: string; effectiveFrom?: string; description?: string };
  products?: VarProduct[];
};

export type MiLossLine = {
  id?: string;
  product?: MasterRef | null;
  contractTypeCode: string;
  value: string;
  effectiveFrom?: string;
};

export type MiLossProductRow = {
  id?: string;
  productId?: string;
  product?: MasterRef | null;
  rates?: MiLossLine[];
};

export type MiLossBatch = {
  id: string;
  documentNumber: string;
  date?: string;
  effectiveFrom: string;
  description?: string;
  status: string;
  createdBy?: CreatedBy;
  products?: MiLossProductRow[];
  lines?: MiLossLine[];
};

export type ExchangeRate = {
  id: string;
  effectiveFrom: string;
  fromCurrency: string;
  toCurrency: string;
  rate: string;
  status: string;
  createdBy?: CreatedBy;
};

export type BillingCycle = { days: number; description: string; isActive: boolean };

export function isoDay(v?: string): string {
  if (!v) return localISODate(new Date());
  return v.slice(0, 10);
}

/** Effective from must be the document date or later. */
export function effectiveBeforeDocError(date: string, effective: string): string {
  const d = (date || "").slice(0, 10);
  const e = (effective || "").slice(0, 10);
  if (d && e && e < d) return "Effective from cannot be before the document date.";
  return "";
}

export function clampEffectiveToDoc(date: string, effective: string): string {
  const d = (date || "").slice(0, 10);
  const e = (effective || "").slice(0, 10);
  if (d && e && e < d) return d;
  return e || d;
}

export function feeEditable(status?: string): boolean {
  return status === "draft" || status === "returned";
}

/** Stored MI-loss is a fraction (0.005). The form shows percent (0.5). */
export function miLossPct(stored?: string): string {
  const n = Number(stored);
  if (!Number.isFinite(n) || n === 0) return "";
  if (Math.abs(n) <= 1) return trimNum(n * 100);
  return trimNum(n);
}

export function miLossFraction(pct: string): string {
  const n = Number(pct);
  if (!Number.isFinite(n) || n === 0) return "0";
  return String(n / 100);
}

/** Percent on the form (0.5 = half a percent). Stored fraction must be > 0 and < 1. */
export function miLossPercentError(pct: string): string {
  const n = Number(String(pct || "").replace(/,/g, ""));
  if (!Number.isFinite(n) || n <= 0) return "MI-loss must be greater than 0.";
  if (n >= 100) return "MI-loss must be less than 100% (0.5 = half a percent).";
  if (n / 100 >= 1) return "MI-loss must be less than 1.";
  return "";
}

function trimNum(n: number): string {
  return String(Number(n.toPrecision(8)));
}

export async function productOptions(q: string, excludeIds: string[] = []) {
  const skip = new Set(excludeIds.filter(Boolean));
  const rows = await searchMaster<MasterRef>("/master/products", q);
  return rows
    .filter((p) => !skip.has(p.id))
    .map((p) => ({
      value: p.id,
      label: `${p.code ?? ""} — ${p.name ?? ""}`.replace(/^ — /, ""),
    }));
}

export function productLabel(p?: MasterRef | null): string {
  if (!p) return "";
  return `${p.code ?? ""} — ${p.name ?? ""}`.replace(/^ — /, "");
}

export function miLossBatchLabel(b?: {
  documentNumber?: string;
  date?: string;
  effectiveFrom?: string;
  description?: string;
} | null): string {
  if (!b?.documentNumber) return "";
  const dated = formatDate(b.date || b.effectiveFrom);
  const eff = formatDate(b.effectiveFrom);
  const when =
    b.date && b.effectiveFrom && isoDay(b.date) !== isoDay(b.effectiveFrom)
      ? `${dated} · effective ${eff}`
      : dated;
  const note = b.description ? ` — ${b.description}` : "";
  return `${b.documentNumber} · ${when}${note}`;
}

/** Effective date of an MI-loss card (falls back to document date). */
export function miLossEffectiveDay(b?: { date?: string; effectiveFrom?: string } | null): string {
  return (b?.effectiveFrom || b?.date || "").slice(0, 10);
}

/** Approved MI-loss cards effective on or before the variable storage fee effective date. */
export function miLossOnOrBefore(rows: MiLossBatch[], feeEffective: string): MiLossBatch[] {
  const day = (feeEffective || "").slice(0, 10);
  return [...rows]
    .filter((b) => {
      const start = miLossEffectiveDay(b);
      return Boolean(day && start && start <= day);
    })
    .sort(
      (a, b) =>
        miLossEffectiveDay(b).localeCompare(miLossEffectiveDay(a)) ||
        (b.date || "").slice(0, 10).localeCompare((a.date || "").slice(0, 10)),
    );
}

/** Latest approved MI-loss whose effective date is on or before `asOf`. */
export function latestMiLossAsOf(rows: MiLossBatch[], asOf: string): MiLossBatch | undefined {
  return miLossOnOrBefore(rows, asOf)[0];
}

/** Approved MI-loss batches, ignoring the operational 90-day list window. */
export async function fetchApprovedMiLoss(): Promise<MiLossBatch[]> {
  const data = await api.get("/billing/miloss/batches", {
    status: "approved",
    pageSize: 200,
    allDates: true,
    orderBy: "effectiveFrom",
    sortDirection: "DESC",
  });
  return asRows<MiLossBatch>(data);
}

export async function fetchApprovedFx(asOf: string): Promise<string> {
  try {
    const data = await api.get<{ found?: boolean; rate?: string }>("/billing/fx/approved", {
      asOf,
      from: "USD",
      to: "TZS",
    });
    if (data?.found && data.rate) return String(data.rate);
  } catch {
    /* no approved quote yet */
  }
  return "";
}

export async function fetchHomeCurrency(): Promise<string> {
  try {
    const row = await api.get<{ currencyCode?: string | null }>("/settings/company");
    const ccy = (row?.currencyCode || "").trim().toUpperCase();
    if (ccy.length === 3) return ccy;
  } catch {
    /* company not set yet */
  }
  return "TZS";
}
