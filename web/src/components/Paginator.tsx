"use client";

import { Button } from "@/components/ui";
import { PAGE_SIZE_OPTIONS } from "@/lib/list";
import type { Pagination } from "@/lib/types";

/** Server-driven prev/next controls using the API pagination payload. */
export function Paginator<T>({
  data,
  page,
  onPage,
  pageSize,
  onPageSize,
  embedded,
}: {
  data: Pagination<T> | null | undefined;
  page: number;
  onPage: (p: number) => void;
  pageSize?: number;
  onPageSize?: (size: number) => void;
  /** Sit on the table card footer (enterprise list layout). */
  embedded?: boolean;
}) {
  if (!data || Array.isArray(data)) return null;

  const itemsCount = Number(data.itemsCount);
  if (!Number.isFinite(itemsCount)) return null;

  const totalPages = Math.max(Number(data.totalPages) || 0, itemsCount > 0 ? 1 : 0);
  const size = pageSize || Number(data.pageSize) || 10;
  const current = Number(data.page) || page;
  const from = itemsCount === 0 ? 0 : (current - 1) * size + 1;
  const to = Math.min(current * size, itemsCount);

  return (
    <div
      className={
        embedded
          ? "flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 bg-white px-4 py-3 text-sm text-slate-600"
          : "mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-600"
      }
    >
      <div className="flex flex-wrap items-center gap-3">
        <span>
          {itemsCount === 0
            ? "No results"
            : `Showing ${from}–${to} of ${itemsCount}`}
          {totalPages > 0 ? ` · Page ${current} of ${totalPages}` : null}
        </span>
        {onPageSize && (
          <label className="flex items-center gap-2 text-slate-600">
            <span className="whitespace-nowrap">Rows</span>
            <select
              className="pp-control !w-auto !py-1.5 !text-sm"
              value={size}
              onChange={(e) => onPageSize(Number(e.target.value))}
            >
              {PAGE_SIZE_OPTIONS.map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>
          </label>
        )}
      </div>
      <div className="flex gap-2">
        <Button
          variant="secondary"
          disabled={page <= 1 || totalPages <= 1}
          onClick={() => onPage(page - 1)}
        >
          Previous
        </Button>
        <Button
          variant="secondary"
          disabled={page >= totalPages || totalPages <= 1}
          onClick={() => onPage(page + 1)}
        >
          Next
        </Button>
      </div>
    </div>
  );
}
