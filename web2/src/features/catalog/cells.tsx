import { Badge } from "@/shared/ui/badge";
import { fmtDate, fmtDateTime, fmtQty, fmtMoney } from "@/shared/lib/format";
import { statusLabel, statusTone } from "@/shared/lib/status";
import type { ColumnDef } from "./registry";

export function cellValue(row: Record<string, unknown>, col: ColumnDef): unknown {
  const raw = row[col.key];
  if (raw && typeof raw === "object" && !Array.isArray(raw)) {
    const o = raw as { code?: string; name?: string; email?: string; plateNumber?: string };
    return o.name || o.code || o.email || o.plateNumber || raw;
  }
  return raw;
}

export function Cell({ row, col }: { row: Record<string, unknown>; col: ColumnDef }) {
  const raw = row[col.key];
  const kind = col.kind ?? "text";
  if (kind === "status") {
    const s = String(raw ?? "");
    return <Badge variant={statusTone(s)}>{statusLabel(s)}</Badge>;
  }
  if (kind === "bool") {
    const on = raw === true || raw === "true" || raw === 1;
    return <Badge variant={on ? "success" : "muted"}>{on ? "Yes" : "No"}</Badge>;
  }
  if (kind === "date") return <span className="tabular-nums">{fmtDate(raw as string)}</span>;
  if (kind === "datetime") return <span className="tabular-nums">{fmtDateTime(raw as string)}</span>;
  if (kind === "qty") return <span className="tabular-nums">{fmtQty(raw)}</span>;
  if (kind === "money") return <span className="tabular-nums">{fmtMoney(raw)}</span>;
  if (kind === "code") {
    const v = cellValue(row, col);
    return <span className="font-mono text-xs">{v == null || v === "" ? "—" : String(v)}</span>;
  }
  if (kind === "ref") {
    const v = cellValue(row, col);
    if (!v || typeof v === "object") return <span className="text-muted-foreground">—</span>;
    return <span>{String(v)}</span>;
  }
  const v = cellValue(row, col);
  if (v == null || v === "") return <span className="text-muted-foreground">—</span>;
  return <span>{String(v)}</span>;
}

export function pickId(row: Record<string, unknown>): string {
  return String(row.id ?? row.uid ?? row.code ?? "");
}
