/**
 * Server list contract (`response.ServeList`).
 *
 * Every operational list accepts the same query and returns the same page
 * envelope. Master catalogues additionally dump the full active set when
 * `page` is omitted.
 */
import type { QueryParams } from "./http";

export interface Page<T> {
  items: T[];
  page: number;
  pageSize: number;
  totalPages: number;
  itemsCount: number;
}

export type SortDirection = "asc" | "desc";

export interface ListQuery {
  page?: number;
  pageSize?: number;
  search?: string;
  orderBy?: string;
  sortDirection?: SortDirection;
  fromDate?: string;
  toDate?: string;
  allDates?: boolean;
  status?: string;
  export?: "xlsx";
  [key: string]: QueryParams[string];
}

export const DEFAULT_PAGE_SIZE = 25;
export const PAGE_SIZE_OPTIONS = [10, 25, 50, 100] as const;

/** One-shot picker fetch: every active row regardless of date. */
export const PICKER_QUERY: ListQuery = { page: 1, pageSize: 100, allDates: true };

export function emptyPage<T>(): Page<T> {
  return { items: [], page: 1, pageSize: DEFAULT_PAGE_SIZE, totalPages: 0, itemsCount: 0 };
}

/**
 * Normalise whatever the server returned into a Page. Catalogue endpoints
 * may return a raw array when `page` is omitted.
 */
export function toPage<T>(raw: unknown): Page<T> {
  if (Array.isArray(raw)) {
    return { items: raw as T[], page: 1, pageSize: raw.length, totalPages: 1, itemsCount: raw.length };
  }
  if (raw && typeof raw === "object" && Array.isArray((raw as Page<T>).items)) {
    const p = raw as Partial<Page<T>>;
    return {
      items: p.items ?? [],
      page: p.page ?? 1,
      pageSize: p.pageSize ?? p.items?.length ?? 0,
      totalPages: p.totalPages ?? 1,
      itemsCount: p.itemsCount ?? p.items?.length ?? 0,
    };
  }
  return emptyPage<T>();
}
