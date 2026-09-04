import { toApiDate } from "@/lib/format";

export type ReportQuery = Record<string, string | number | boolean | undefined>;

export type ReportParams = {
  from: string;
  to: string;
  year: string;
  month: string;
  date: string;
  id: string;
  route: string;
  customer: string;
  /** all | active | inactive */
  active: string;
  /** Stock category UID */
  category: string;
  /** EWURA license classes (multi-select). */
  classes: string[];
};

export function reportCodeFromHref(href: string): string {
  const parts = href.replace(/\/$/, "").split("/");
  return parts[parts.length - 1] || "";
}

/** API path for JSON / Excel. Dedicated access reports keep /reports/users etc. */
export function reportApiBase(href: string): string {
  if (href.includes("/view/")) {
    return `/reports/${reportCodeFromHref(href)}`;
  }
  return href;
}

export function defaultReportParams(): ReportParams {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  const today = `${y}-${m}-${day}`;
  const ago = new Date(now);
  ago.setMonth(ago.getMonth() - 1);
  const from = `${ago.getFullYear()}-${String(ago.getMonth() + 1).padStart(2, "0")}-${String(ago.getDate()).padStart(2, "0")}`;
  return {
    from,
    to: today,
    year: String(y),
    month: String(now.getMonth() + 1),
    date: today,
    id: "",
    route: "",
    customer: "",
    active: "all",
    category: "",
    classes: [],
  };
}

export function reportParamsReady(filters: string[], p: ReportParams): boolean {
  return !filters.includes("id") || Boolean(p.id.trim());
}

export function buildReportQuery(
  filters: string[],
  p: ReportParams,
  apiBase: string,
): ReportQuery {
  const q: ReportQuery = {};
  if (filters.includes("from")) q.from = p.from;
  if (filters.includes("to")) q.to = p.to;
  if (filters.includes("year")) q.year = p.year;
  if (filters.includes("month")) q.month = p.month;
  if (filters.includes("date")) q.date = p.date;
  if (filters.includes("id")) q.id = p.id.trim() || undefined;
  if (filters.includes("route")) q.route = p.route.trim() || undefined;
  if (filters.includes("customer")) q.customer = p.customer.trim() || undefined;
  if (filters.includes("active") && p.active && p.active !== "all") q.active = p.active;
  if (filters.includes("category") && p.category.trim()) q.category = p.category.trim();
  if (filters.includes("class") && p.classes.length) q.class = p.classes.join(",");
  // Access reports use the paginated search contract (DD/MM/YYYY).
  if (apiBase === "/reports/users" || apiBase === "/reports/audit-activity") {
    q.fromDate = toApiDate(p.from);
    q.toDate = toApiDate(p.to);
  }
  return q;
}
