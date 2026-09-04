import { format, isValid, parseISO, differenceInCalendarDays, formatDistanceToNowStrict } from "date-fns";

/** Query-string date format expected by the API (`02/01/2006`). */
export const API_DATE_FORMAT = "dd/MM/yyyy";
/** HTML `<input type="date">` value format. */
export const ISO_DATE = "yyyy-MM-dd";

export function parseDate(value: string | Date | null | undefined): Date | null {
  if (!value) return null;
  if (value instanceof Date) return isValid(value) ? value : null;
  const d = parseISO(value);
  if (isValid(d)) return d;
  // DD/MM/YYYY fallback
  const m = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(value);
  if (m) {
    const d2 = new Date(Number(m[3]), Number(m[2]) - 1, Number(m[1]));
    return isValid(d2) ? d2 : null;
  }
  return null;
}

export function fmtDate(value: string | Date | null | undefined, pattern = "dd MMM yyyy"): string {
  const d = parseDate(value);
  return d ? format(d, pattern) : "—";
}

export function fmtDateTime(value: string | Date | null | undefined): string {
  return fmtDate(value, "dd MMM yyyy HH:mm");
}

export function fmtRelative(value: string | Date | null | undefined): string {
  const d = parseDate(value);
  return d ? formatDistanceToNowStrict(d, { addSuffix: true }) : "—";
}

export function toApiDate(value: string | Date | null | undefined): string | undefined {
  const d = parseDate(value);
  return d ? format(d, API_DATE_FORMAT) : undefined;
}

export function toIsoDate(value: string | Date | null | undefined): string {
  const d = parseDate(value);
  return d ? format(d, ISO_DATE) : "";
}

export function daysUntil(value: string | Date | null | undefined): number | null {
  const d = parseDate(value);
  return d ? differenceInCalendarDays(d, new Date()) : null;
}

/** Decimal values arrive as JSON strings ("12.50"). */
export function toNumber(value: unknown): number | null {
  if (value === null || value === undefined || value === "") return null;
  const n = typeof value === "number" ? value : Number(String(value).replace(/,/g, ""));
  return Number.isFinite(n) ? n : null;
}

const nfCache = new Map<string, Intl.NumberFormat>();
function nf(min: number, max: number): Intl.NumberFormat {
  const key = `${min}:${max}`;
  let f = nfCache.get(key);
  if (!f) {
    f = new Intl.NumberFormat("en-TZ", { minimumFractionDigits: min, maximumFractionDigits: max });
    nfCache.set(key, f);
  }
  return f;
}

export function fmtNumber(value: unknown, digits = 2, empty = "—"): string {
  const n = toNumber(value);
  if (n === null) return empty;
  return nf(digits, digits).format(n);
}

export function fmtQty(value: unknown, digits = 2): string {
  return fmtNumber(value, digits);
}

export function fmtMoney(value: unknown, currency?: string | null, digits = 2): string {
  const n = toNumber(value);
  if (n === null) return "—";
  const s = nf(digits, digits).format(n);
  return currency ? `${currency} ${s}` : s;
}

export function fmtPercent(fraction: unknown, digits = 2): string {
  const n = toNumber(fraction);
  if (n === null) return "—";
  return `${nf(digits, digits).format(n * 100)}%`;
}

export function fmtBytes(bytes: number): string {
  if (!Number.isFinite(bytes)) return "—";
  const units = ["B", "KB", "MB", "GB"];
  let i = 0;
  let v = bytes;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export function initials(first?: string, last?: string): string {
  return `${first?.[0] ?? ""}${last?.[0] ?? ""}`.toUpperCase() || "?";
}

export function fullName(u: { firstName?: string; lastName?: string } | null | undefined): string {
  if (!u) return "";
  return [u.firstName, u.lastName].filter(Boolean).join(" ");
}

export function titleCase(s: string): string {
  return s.replace(/[_-]+/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}
