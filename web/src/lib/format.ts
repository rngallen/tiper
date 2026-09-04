// Formatting helpers shared across pages.

type CurrencySymbolRow = { code: string; symbol?: string };

let symbolByCode: Record<string, string> = {};
let catalogVersion = 0;
const catalogListeners = new Set<() => void>();

function emitCurrencyCatalog() {
  catalogVersion += 1;
  catalogListeners.forEach((fn) => fn());
}

/** Load registered Currency.Symbol values for money display. */
export function hydrateCurrencySymbols(rows: CurrencySymbolRow[]) {
  const next: Record<string, string> = {};
  for (const row of rows || []) {
    const code = String(row.code || "").trim().toUpperCase();
    if (!code) continue;
    const symbol = String(row.symbol || "").trim();
    if (symbol) next[code] = symbol;
  }
  symbolByCode = next;
  emitCurrencyCatalog();
}

export function subscribeCurrencyCatalog(listener: () => void): () => void {
  catalogListeners.add(listener);
  return () => catalogListeners.delete(listener);
}

export function currencyCatalogVersion(): number {
  return catalogVersion;
}

/** Filter dropdown: "TSh TZS" when the symbol differs from the ISO code. */
export function currencyFilterLabel(c: { code: string; symbol?: string }): string {
  const code = (c.code || "").trim().toUpperCase();
  const symbol = (c.symbol || symbolByCode[code] || "").trim();
  if (symbol && symbol.toUpperCase() !== code) return `${symbol} ${code}`;
  return code;
}

/** Form dropdown: "TSh TZS — Tanzanian Shilling". */
export function currencyFullLabel(c: {
  code: string;
  name?: string;
  symbol?: string;
}): string {
  const head = currencyFilterLabel(c);
  const name = (c.name || "").trim();
  return name ? `${head} — ${name}` : head;
}

/** Calendar date as DD/MM/YYYY (no time). Date-only ISO strings are not shifted by timezone. */
export function formatDate(value?: string): string {
  if (!value) return "—";
  const raw = String(value).trim();
  const ymd = raw.length >= 10 && /^\d{4}-\d{2}-\d{2}/.test(raw) ? raw.slice(0, 10) : "";
  if (ymd) {
    const shown = isoToDisplayDate(ymd);
    return shown || "—";
  }
  const d = new Date(raw);
  if (isNaN(d.getTime())) return "—";
  const day = String(d.getDate()).padStart(2, "0");
  const month = String(d.getMonth() + 1).padStart(2, "0");
  return `${day}/${month}/${d.getFullYear()}`;
}

/** Datetime as DD/MM/YYYY, HH:mm (24h). */
export function formatDateTime(value?: string): string {
  if (!value) return "—";
  const d = new Date(value);
  if (isNaN(d.getTime())) return "—";
  const day = String(d.getDate()).padStart(2, "0");
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");
  return `${day}/${month}/${d.getFullYear()}, ${hh}:${mm}`;
}

/** Age of a timestamp for inbox / aging columns (e.g. "14m", "5h", "3d"). */
export function formatWait(value?: string): string {
  if (!value) return "—";
  const d = new Date(value);
  if (isNaN(d.getTime())) return "—";
  const mins = Math.max(0, Math.floor((Date.now() - d.getTime()) / 60_000));
  if (mins < 60) return `${mins}m`;
  const hours = Math.floor(mins / 60);
  if (hours < 48) return `${hours}h`;
  return `${Math.floor(hours / 24)}d`;
}

export function waitOverdue(value?: string): boolean {
  if (!value) return false;
  const d = new Date(value);
  if (isNaN(d.getTime())) return false;
  return Date.now() - d.getTime() >= 3 * 24 * 60 * 60 * 1000;
}

export function titleCase(s: string): string {
  return s
    .replace(/[_-]/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

/** Local calendar date as YYYY-MM-DD (internal state). */
export function localISODate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

/** YYYY-MM-DD → DD/MM/YYYY (picker display and API list filters). */
export function isoToDisplayDate(iso: string): string {
  if (!iso) return "";
  const [y, m, d] = iso.split("-");
  if (!y || !m || !d) return "";
  return `${d}/${m}/${y}`;
}

function isValidYmd(iso: string): boolean {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  if (!m) return false;
  const y = Number(m[1]);
  const mo = Number(m[2]);
  const d = Number(m[3]);
  const dt = new Date(y, mo - 1, d);
  return dt.getFullYear() === y && dt.getMonth() === mo - 1 && dt.getDate() === d;
}

/** Typed calendar date → YYYY-MM-DD. Accepts DD/MM/YYYY, D-M-YYYY, DD.MM.YYYY, DDMMYYYY. */
export function parseTypedDate(raw: string): string {
  const s = (raw || "").trim();
  if (!s) return "";
  const sep = /^(\d{1,2})[/.\\-](\d{1,2})[/.\\-](\d{4})$/.exec(s);
  if (sep) {
    const iso = `${sep[3]}-${sep[2].padStart(2, "0")}-${sep[1].padStart(2, "0")}`;
    return isValidYmd(iso) ? iso : "";
  }
  const digits = s.replace(/\D/g, "");
  if (digits.length === 8) {
    const iso = `${digits.slice(4)}-${digits.slice(2, 4)}-${digits.slice(0, 2)}`;
    return isValidYmd(iso) ? iso : "";
  }
  return "";
}

/** API expects DD/MM/YYYY. */
export function toApiDate(iso: string): string | undefined {
  const display = isoToDisplayDate(iso);
  return display || undefined;
}

/** Operational list lookback. Reports pass an explicit span. */
export const OPS_LOOKBACK_DAYS = 90;

/** Default "from" date for list filters (days back from today). */
export function defaultFromISO(daysBack = OPS_LOOKBACK_DAYS): string {
  const d = new Date();
  d.setDate(d.getDate() - daysBack);
  return localISODate(d);
}

/** camelCase / snake_case / dotted field key → readable label. */
export function formatFieldLabel(key: string): string {
  return key
    .split(".")
    .map((part) =>
      part
        .replace(/([a-z])([A-Z])/g, "$1 $2")
        .replace(/[_-]/g, " ")
        .replace(/\b\w/g, (c) => c.toUpperCase()),
    )
    .join(" · ");
}

function formatAuditValue(value: unknown): string {
  if (value === null || value === undefined || value === "") return "—";
  if (typeof value === "boolean") return value ? "yes" : "no";
  if (typeof value === "object") {
    try {
      return JSON.stringify(value);
    } catch {
      return "—";
    }
  }
  return String(value);
}

/** Digits-only integer with thousand separators for display (MPLW, GCWR, TW). */
export function formatIntegerThousands(raw: string): string {
  const digits = String(raw || "").replace(/[^\d]/g, "");
  if (!digits) return "";
  return Number(digits).toLocaleString("en-US");
}

export function parseIntegerThousands(raw: string): string {
  return String(raw || "").replace(/[^\d]/g, "");
}

/** Plates and driver licences: letters and digits only, uppercase. */
export function normalizePlate(raw: string): string {
  return String(raw || "")
    .toUpperCase()
    .replace(/[^A-Z0-9]/g, "");
}

export type QtyUnit = "l" | "m3" | "mt" | "density" | "price" | "miloss" | "fx";

export type DecimalPrecision = {
  quantityPrecision: number;
  cubicMeterPrecision: number;
  metricTonnePrecision: number;
  densityPrecision: number;
  pricePrecision: number;
  miLossPrecision: number;
  iloExpiryDays: number;
};

export const defaultDecimalPrecision: DecimalPrecision = {
  quantityPrecision: 2,
  cubicMeterPrecision: 3,
  metricTonnePrecision: 3,
  densityPrecision: 5,
  pricePrecision: 4,
  miLossPrecision: 4,
  iloExpiryDays: 14,
};

let decimalPrecision: DecimalPrecision = { ...defaultDecimalPrecision };
let precisionVersion = 0;
const precisionListeners = new Set<() => void>();

function emitPrecision() {
  precisionVersion += 1;
  precisionListeners.forEach((fn) => fn());
}

function clampPlaces(n: number): number {
  if (!Number.isFinite(n)) return 3;
  return Math.min(6, Math.max(0, Math.round(n)));
}

/** Load rounding + ILO days from Settings so L / m³ / MT display is live. */
export function hydrateDecimalPrecision(row?: Partial<DecimalPrecision> | null) {
  const raw = row || {};
  const next = { ...defaultDecimalPrecision, ...raw };
  const cubic =
    raw.cubicMeterPrecision ?? raw.quantityPrecision ?? defaultDecimalPrecision.cubicMeterPrecision;
  decimalPrecision = {
    quantityPrecision: clampPlaces(next.quantityPrecision),
    cubicMeterPrecision: clampPlaces(cubic),
    metricTonnePrecision: clampPlaces(next.metricTonnePrecision),
    densityPrecision: clampPlaces(next.densityPrecision),
    pricePrecision: clampPlaces(next.pricePrecision),
    miLossPrecision: clampPlaces(next.miLossPrecision),
    iloExpiryDays:
      Number.isFinite(next.iloExpiryDays) && next.iloExpiryDays >= 1
        ? Math.min(90, Math.round(next.iloExpiryDays))
        : defaultDecimalPrecision.iloExpiryDays,
  };
  emitPrecision();
}

export function subscribeDecimalPrecision(listener: () => void): () => void {
  precisionListeners.add(listener);
  return () => precisionListeners.delete(listener);
}

export function decimalPrecisionVersion(): number {
  return precisionVersion;
}

export function currentDecimalPrecision(): DecimalPrecision {
  return decimalPrecision;
}

export function placesFor(unit: QtyUnit = "l"): number {
  switch (unit) {
    case "m3":
      return decimalPrecision.cubicMeterPrecision;
    case "mt":
      return decimalPrecision.metricTonnePrecision;
    case "density":
      return decimalPrecision.densityPrecision;
    case "price":
      return decimalPrecision.pricePrecision;
    case "fx":
      return 6;
    case "miloss":
      return decimalPrecision.miLossPrecision;
    default:
      return decimalPrecision.quantityPrecision;
  }
}

/** Litres / m³ / MT / money: uses Settings → Decimal precision. */
export function formatQty(
  v: string | number | undefined | null,
  unit: QtyUnit = "l",
): string {
  if (v === undefined || v === null || v === "") return "—";
  const n = Number(String(v).replace(/,/g, ""));
  if (!Number.isFinite(n)) return "—";
  const d = placesFor(unit);
  return n.toLocaleString("en-US", { minimumFractionDigits: d, maximumFractionDigits: d });
}

/** Strip thousand separators for API payloads. */
export function parseQty(raw: string): string {
  return String(raw || "").replace(/,/g, "").replace(/[^\d.]/g, "");
}

/** Live input: digits and one decimal, integer part grouped by thousands. */
export function formatQtyDraft(raw: string, unit: QtyUnit = "l"): string {
  const s = String(raw || "").replace(/[^\d.]/g, "");
  const dot = s.indexOf(".");
  let intp = dot === -1 ? s : s.slice(0, dot);
  const dec = dot === -1 ? "" : s.slice(dot + 1).replace(/\./g, "").slice(0, placesFor(unit));
  intp = intp.replace(/^0+(?=\d)/, "");
  const grouped = intp.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  if (dot === -1) return grouped;
  return `${grouped}.${dec}`;
}

export function roundQtyNumber(n: number, unit: QtyUnit = "l"): number {
  const d = placesFor(unit);
  const f = 10 ** d;
  return Math.round(n * f) / f;
}
export function formatFieldChange(change: {
  before?: unknown;
  after?: unknown;
}): string {
  return `${formatAuditValue(change?.before)} → ${formatAuditValue(change?.after)}`;
}
