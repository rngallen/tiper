import { StockStatusCode } from "@/lib/domain";

export type StockStatusRef = {
  id: string;
  code?: string;
  name?: string;
  isTransit?: boolean;
  isLocal?: boolean;
  isMining?: boolean;
  isProration?: boolean;
  parent?: { id: string; name?: string; code?: string } | null;
};

/** Picker label: "Transit — Congo", "Local — Mining", or "Transit". */
export function stockStatusLabel(s: StockStatusRef): string {
  const name = (s.name || s.code || "").trim();
  const parent = (s.parent?.name || "").trim();
  if (parent && parent.toLowerCase() !== name.toLowerCase()) {
    return `${parent} — ${name}`;
  }
  if (s.isTransit && (s.code || "").toUpperCase() !== StockStatusCode.Transit) {
    return `Transit — ${name}`;
  }
  if (s.isMining) {
    return `Local — ${name}`;
  }
  return name;
}
