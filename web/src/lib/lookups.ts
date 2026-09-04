import { api } from "./api";
import type { BillingCycle } from "./billing";

export type CatalogRow = {
  code: string;
  name: string;
  isActive?: boolean;
};

export type TenderRow = CatalogRow & {
  isSingleReceiving?: boolean;
  supplierPaysUnlessLoading?: boolean;
};

export type RouteRow = CatalogRow;
export type DeliveryRow = CatalogRow & { isGantryLoading?: boolean };
export type PricingRow = CatalogRow & { isPromotional?: boolean };
export type CycleRow = BillingCycle;
export type FeeRow = CatalogRow & { chargeTo?: "customer" | "supplier" | "both" };

export type BillingLookups = {
  billingCycles?: CycleRow[];
  tenders?: TenderRow[];
  deliveryMethods?: DeliveryRow[];
  procurementMethods?: CatalogRow[];
  routes?: RouteRow[];
  contractTypes?: CatalogRow[];
  pricingNatures?: PricingRow[];
  fees?: FeeRow[];
};

export async function fetchLookups(): Promise<BillingLookups> {
  return api.get<BillingLookups>("/master/lookups");
}

export function activeRows<T extends { isActive?: boolean }>(rows?: T[]): T[] {
  return (rows || []).filter((r) => r.isActive !== false);
}

export function catalogLabel(row?: { code?: string; name?: string } | null): string {
  if (!row) return "";
  if (row.name && row.code && row.name !== row.code) return `${row.code} — ${row.name}`;
  return row.name || row.code || "";
}
