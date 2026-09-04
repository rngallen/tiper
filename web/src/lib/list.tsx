"use client";

import { useCallback, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Download } from "lucide-react";
import { Button, Input } from "@/components/ui";
import { api, formatApiError } from "@/lib/api";
import type { SortDirection, SortState } from "@/components/DataTable";

/** Fixed 14rem, always last/right in a filter row. Never flex-grow. */
export function ListSearch({
  value,
  onChange,
  placeholder = "Search…",
  label = "Search",
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  label?: string;
}) {
  return (
    <div className="ml-auto w-56 shrink-0">
      <label className="mb-1 block text-xs font-medium text-slate-600">
        {label}
      </label>
      <Input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        aria-label={label}
      />
    </div>
  );
}

export const PAGE_SIZE_OPTIONS = [10, 20, 30, 50, 100] as const;

/** Keep a filter in the query string (vendors/invoices deep-link pattern). */
export function useUrlFilter(key: string, allowed?: readonly string[]) {
  const params = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const [value, setValueRaw] = useState(() => {
    const s = params.get(key) || "";
    if (allowed && s && !allowed.includes(s)) return "";
    return s;
  });

  const setValue = useCallback(
    (next: string) => {
      setValueRaw(next);
      const usp = new URLSearchParams(params.toString());
      if (next) usp.set(key, next);
      else usp.delete(key);
      const q = usp.toString();
      router.replace(q ? `${pathname}?${q}` : pathname, { scroll: false });
    },
    [key, params, pathname, router],
  );

  return [value, setValue] as const;
}

/** Shared list query state: search, pagination, sort, and Excel export. */
export function useListQuery(defaults?: {
  orderBy?: string;
  sortDirection?: SortDirection;
  pageSize?: number;
}) {
  const [search, setSearchRaw] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSizeRaw] = useState(defaults?.pageSize ?? 10);
  const [orderBy, setOrderBy] = useState(defaults?.orderBy ?? "");
  const [sortDirection, setSortDirection] = useState<SortDirection>(
    defaults?.sortDirection ?? "DESC",
  );

  const setSearch = useCallback((value: string) => {
    setSearchRaw(value);
    setPage(1);
  }, []);

  const setPageSize = useCallback((size: number) => {
    setPageSizeRaw(size);
    setPage(1);
  }, []);

  // Toggle ASC ↔ DESC on the active column; new column starts ASC.
  const onSort = useCallback(
    (key: string) => {
      setPage(1);
      if (key === orderBy) {
        setSortDirection((d) => (d === "ASC" ? "DESC" : "ASC"));
        return;
      }
      setOrderBy(key);
      setSortDirection("ASC");
    },
    [orderBy],
  );

  const sort: SortState = { orderBy, sortDirection };

  const query = {
    search: search || undefined,
    page,
    pageSize,
    orderBy: orderBy || undefined,
    sortDirection,
  };

  return {
    search,
    setSearch,
    page,
    setPage,
    pageSize,
    setPageSize,
    orderBy,
    sortDirection,
    sort,
    onSort,
    query,
  };
}

export function ExportButton({
  path,
  query,
  label = "Export Excel",
}: {
  path: string;
  query?: Record<string, string | number | boolean | undefined>;
  label?: string;
}) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function run() {
    setError("");
    setLoading(true);
    try {
      await api.download(path, {
        ...query,
        export: true,
        page: undefined,
        pageSize: undefined,
      });
    } catch (err) {
      setError(formatApiError(err, "Export failed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col items-end gap-1">
      <Button variant="secondary" onClick={run} loading={loading}>
        <Download className="h-4 w-4" />
        {label}
      </Button>
      {error && <span className="text-xs font-medium text-red-700">{error}</span>}
    </div>
  );
}
