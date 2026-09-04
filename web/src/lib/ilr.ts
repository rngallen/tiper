import { searchMaster, type Destination, type Product } from "@/lib/master";
import { api } from "@/lib/api";
import { asRows } from "@/lib/pagination";

export type GantryGrade = "ago" | "pms";

export function partyName(row?: { code?: string; name?: string } | null): string {
  if (!row) return "—";
  const name = (row.name || "").trim();
  const code = (row.code || "").trim();
  if (name && code && name !== code) return `${name} (${code})`;
  return name || code || "—";
}

export function gantryGrade(p?: { code?: string; name?: string } | null): GantryGrade | "" {
  const s = `${p?.code || ""} ${p?.name || ""}`.toUpperCase();
  if (/\bAGO\b|GAS OIL/.test(s)) return "ago";
  if (/\bPMS\b|\bMOGAS\b|MOTOR SPIRIT|MOTOR GAS/.test(s)) return "pms";
  return "";
}

export function qtyForGrade(
  row: {
    product?: { code?: string; name?: string } | null;
    byProduct?: { code?: string; name?: string } | null;
    quantity?: string;
    byProductQuantity?: string;
  },
  grade: GantryGrade,
): string {
  if (gantryGrade(row.product) === grade) return row.quantity || "0";
  if (gantryGrade(row.byProduct) === grade) return row.byProductQuantity || "0";
  return "0";
}

export function truckComboPlate(t: {
  plateNumber?: string;
  trailer?: string;
  trailerTwo?: string;
  displayPlate?: string;
}): string {
  if (t.displayPlate) return t.displayPlate;
  return [t.plateNumber, t.trailer, t.trailerTwo].map((p) => (p || "").trim()).filter(Boolean).join("/");
}

export async function searchILRProducts(q: string, excludeId?: string): Promise<Product[]> {
  const data = await api.get<unknown>("/orders/loading-products", {
    search: q || undefined,
    exclude: excludeId || undefined,
  });
  return asRows<Product>(data);
}

export async function searchCountryDestinations(q: string) {
  return searchMaster<Destination>("/master/destinations", q, { isCountry: true });
}
