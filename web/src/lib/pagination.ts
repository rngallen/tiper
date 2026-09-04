import type { Pagination } from "@/lib/types";

/** Normalize list API payloads: paginator object or a bare array. */
export function asPagination<T>(data: unknown): Pagination<T> | null {
  if (data == null) return null;
  if (Array.isArray(data)) {
    return {
      items: data as T[],
      page: 1,
      pageSize: data.length || 10,
      totalPages: data.length > 0 ? 1 : 0,
      itemsCount: data.length,
    };
  }
  if (typeof data !== "object") return null;
  const row = data as Partial<Pagination<T>> & { items?: T[] };
  const items = Array.isArray(row.items) ? row.items : [];
  const itemsCount = Number(row.itemsCount ?? items.length) || 0;
  const pageSize = Number(row.pageSize) || 10;
  const page = Number(row.page) || 1;
  const totalPages =
    Number(row.totalPages) ||
    (itemsCount > 0 && pageSize > 0 ? Math.ceil(itemsCount / pageSize) : 0);
  return { items, page, pageSize, totalPages, itemsCount };
}

export function asRows<T>(data: unknown): T[] {
  return asPagination<T>(data)?.items ?? [];
}
