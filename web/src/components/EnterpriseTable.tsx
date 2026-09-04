"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Download } from "lucide-react";
import { DataTable, type SortDirection } from "@/components/DataTable";
import { EditColumnsButton } from "@/components/EditColumns";
import { DateRangeFields, ListShell } from "@/components/ListShell";
import { Modal } from "@/components/Modal";
import { Paginator } from "@/components/Paginator";
import { Button } from "@/components/ui";
import { ExportButton, ListSearch, useListQuery, useUrlFilter } from "@/lib/list";
import { useColumnPrefs, type ColumnDef } from "@/lib/columnPrefs";
import { downloadExcel } from "@/lib/excel";
import { useAsync } from "@/lib/hooks";
import { api } from "@/lib/api";
import { asPagination, asRows } from "@/lib/pagination";
import { defaultFromISO, localISODate, toApiDate } from "@/lib/format";
import type { Pagination } from "@/lib/types";
import type { StatusPillOption } from "@/components/StatusPills";

export type EnterpriseColumn<T> = ColumnDef<T> & {
  exportValue?: (row: T) => string | number | boolean | null | undefined;
  sortValue?: (row: T) => string | number;
};

export type RemoteList = {
  path: string;
  extraQuery?: Record<string, string | number | boolean | undefined>;
  dates?: boolean;
  /** Map the status-pill lane to extra query params (default: { status: lane }). */
  laneQuery?: (lane: string) => Record<string, string | number | boolean | undefined>;
  reloadKey?: number;
};

function fieldValue(row: unknown, key: string): unknown {
  if (!row || typeof row !== "object") return "";
  const rec = row as Record<string, unknown>;
  if (!(key in rec)) return "";
  const v = rec[key];
  if (v && typeof v === "object" && !Array.isArray(v)) {
    const nested = v as { code?: string; name?: string; documentNumber?: string };
    return nested.documentNumber || nested.code || nested.name || "";
  }
  return v;
}

function compareRows<T>(
  a: T,
  b: T,
  col: EnterpriseColumn<T> | undefined,
  key: string,
  dir: SortDirection,
): number {
  const rawA = col?.sortValue ? col.sortValue(a) : fieldValue(a, key);
  const rawB = col?.sortValue ? col.sortValue(b) : fieldValue(b, key);
  const na = typeof rawA === "number" ? rawA : Number(rawA);
  const nb = typeof rawB === "number" ? rawB : Number(rawB);
  let cmp = 0;
  if (Number.isFinite(na) && Number.isFinite(nb) && String(rawA) !== "" && String(rawB) !== "") {
    cmp = na - nb;
  } else {
    cmp = String(rawA ?? "").toLowerCase().localeCompare(String(rawB ?? "").toLowerCase());
  }
  return dir === "DESC" ? -cmp : cmp;
}

/** Enterprise list: header actions, pills, filters, search, sort, page, Excel, inquiry. */
export function EnterpriseTable<T>({
  tableId,
  title,
  subtitle,
  headerExtra,
  lead,
  rows,
  loading,
  error,
  emptyMessage,
  columns: catalog,
  searchPlaceholder = "Search…",
  rowSearch,
  defaultOrderBy = "",
  defaultSortDirection = "ASC",
  pills,
  filters,
  dateOf,
  inquiryTitle,
  inquiry,
  exportName,
  remote,
}: {
  tableId: string;
  title?: string;
  subtitle?: string;
  headerExtra?: ReactNode;
  lead?: ReactNode;
  rows?: T[] | null | undefined | unknown;
  loading?: boolean;
  error?: string;
  emptyMessage: string;
  columns: EnterpriseColumn<T>[];
  searchPlaceholder?: string;
  rowSearch?: (row: T) => string;
  defaultOrderBy?: string;
  defaultSortDirection?: SortDirection;
  pills?: {
    options: StatusPillOption[];
    of: (row: T) => string;
    guideTitle?: string;
    allTitle?: string;
    /** Query-string key (default status). Use a distinct key when two lists share a page. */
    urlKey?: string;
  };
  filters?: ReactNode;
  /** When set, From/To + Clear dates filter this field (ISO or parseable date). */
  dateOf?: (row: T) => string | undefined;
  inquiryTitle?: (row: T) => string;
  inquiry?: (row: T) => ReactNode;
  exportName: string;
  remote?: RemoteList;
}) {
  const list = useListQuery({
    orderBy: defaultOrderBy,
    sortDirection: defaultSortDirection,
  });
  const pillIds = pills?.options.map((o) => o.id);
  const [lane, setLane] = useUrlFilter(pills?.urlKey || "status", pillIds);
  const useDates = Boolean(remote?.dates || dateOf);
  const [fromDate, setFromDate] = useState(() => (remote?.dates ? defaultFromISO() : ""));
  const [toDate, setToDate] = useState(() => (remote?.dates ? localISODate(new Date()) : ""));
  const [viewing, setViewing] = useState<T | null>(null);
  const [excelErr, setExcelErr] = useState("");

  const { columns, pref, save, restoreDefaults } = useColumnPrefs(tableId, catalog);
  const byId = useMemo(() => new Map(catalog.map((c) => [c.id, c])), [catalog]);

  const datesCleared = !fromDate && !toDate;
  const laneParams = useMemo(() => {
    if (!lane) return {};
    if (remote?.laneQuery) return remote.laneQuery(lane);
    return { [pills?.urlKey === "reportStatus" ? "status" : "status"]: lane };
  }, [lane, remote, pills?.urlKey]);

  const remoteQuery = useMemo(
    () => ({
      ...list.query,
      ...remote?.extraQuery,
      ...laneParams,
      fromDate: remote?.dates && !datesCleared ? toApiDate(fromDate) : undefined,
      toDate: remote?.dates && !datesCleared ? toApiDate(toDate) : undefined,
      allDates: remote?.dates && datesCleared ? true : undefined,
    }),
    [list.query, remote, laneParams, fromDate, toDate, datesCleared],
  );

  const remoteLoad = useAsync(
    () =>
      remote
        ? api.get<Pagination<T>>(remote.path, remoteQuery)
        : Promise.resolve(null),
    remote
      ? [
          remote.path,
          remote.reloadKey,
          list.search,
          list.page,
          list.pageSize,
          list.orderBy,
          list.sortDirection,
          lane,
          fromDate,
          toDate,
          JSON.stringify(remote.extraQuery || {}),
        ]
      : [],
  );

  const serverPage = remote ? asPagination<T>(remoteLoad.data) : null;
  const allRows = useMemo(() => (remote ? [] : asRows<T>(rows)), [remote, rows]);
  const filtered = useMemo(() => {
    if (remote) return [];
    const needle = list.search.trim().toLowerCase();
    const next = allRows.filter((r) => {
      if (lane && pills && pills.of(r) !== lane) return false;
      if (needle && rowSearch && !rowSearch(r).toLowerCase().includes(needle)) return false;
      if (dateOf && (fromDate || toDate)) {
        const raw = dateOf(r) || "";
        const day = raw.slice(0, 10);
        if (fromDate && day && day < fromDate) return false;
        if (toDate && day && day > toDate) return false;
        if ((fromDate || toDate) && !day) return false;
      }
      return true;
    });
    if (list.orderBy) {
      const col = byId.get(list.orderBy);
      next.sort((a, b) => compareRows(a, b, col, list.orderBy, list.sortDirection));
    }
    return next;
  }, [remote, allRows, list.search, list.orderBy, list.sortDirection, rowSearch, byId, lane, pills, dateOf, fromDate, toDate]);

  const pageItems = useMemo(() => {
    if (remote) return serverPage?.items ?? [];
    const start = (list.page - 1) * list.pageSize;
    return filtered.slice(start, start + list.pageSize);
  }, [remote, serverPage, filtered, list.page, list.pageSize]);

  const page: Pagination<T> = remote
    ? serverPage || {
        items: pageItems,
        page: list.page,
        pageSize: list.pageSize,
        itemsCount: 0,
        totalPages: 0,
      }
    : {
        items: pageItems,
        page: list.page,
        pageSize: list.pageSize,
        itemsCount: filtered.length,
        totalPages: filtered.length > 0 ? Math.ceil(filtered.length / list.pageSize) : 0,
      };

  useEffect(() => {
    if (remote) return;
    if (page.totalPages > 0 && list.page > page.totalPages) list.setPage(page.totalPages);
  }, [remote, page.totalPages, list.page, list.setPage]);

  const busy = remote ? remoteLoad.loading : loading;
  const err = remote ? remoteLoad.error : error;

  function exportExcel() {
    setExcelErr("");
    const exportable = catalog.filter((c) => c.exportValue && !c.fixed);
    if (exportable.length === 0) {
      setExcelErr("No export columns configured");
      return;
    }
    try {
      downloadExcel(
        exportName,
        exportable.map((c) => String(c.header)),
        filtered.map((r) => exportable.map((c) => c.exportValue!(r))),
      );
    } catch {
      setExcelErr("Excel export failed");
    }
  }

  const filteredHint = Boolean(list.search || lane || fromDate || toDate);

  return (
    <>
      <ListShell
        title={title}
        subtitle={subtitle}
        lead={lead}
        actions={
          <div className="flex flex-wrap items-center justify-end gap-2">
            <EditColumnsButton
              catalog={catalog}
              pref={pref}
              onApply={save}
              onRestore={restoreDefaults}
            />
            {remote ? (
              <ExportButton path={remote.path} query={remoteQuery} />
            ) : (
              <div className="flex flex-col items-end gap-1">
                <Button variant="secondary" onClick={exportExcel}>
                  <Download className="h-4 w-4" />
                  Export Excel
                </Button>
                {excelErr ? (
                  <span className="text-xs font-medium text-red-700">{excelErr}</span>
                ) : null}
              </div>
            )}
            {headerExtra}
          </div>
        }
        pills={
          pills
            ? {
                options: pills.options,
                value: lane,
                onChange: (id) => {
                  setLane(id);
                  list.setPage(1);
                },
                allTitle: pills.allTitle,
                guideTitle: pills.guideTitle,
              }
            : undefined
        }
        filters={
          useDates || filters ? (
            <>
              {useDates ? (
                <DateRangeFields
                  from={fromDate}
                  to={toDate}
                  onFrom={(iso) => {
                    setFromDate(iso);
                    list.setPage(1);
                  }}
                  onTo={(iso) => {
                    setToDate(iso);
                    list.setPage(1);
                  }}
                  onClear={() => {
                    setFromDate("");
                    setToDate("");
                    list.setPage(1);
                  }}
                />
              ) : null}
              {filters}
            </>
          ) : undefined
        }
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder={searchPlaceholder}
          />
        }
        footer={
          !busy && !err ? (
            <Paginator
              embedded
              data={page}
              page={list.page}
              onPage={list.setPage}
              pageSize={list.pageSize}
              onPageSize={list.setPageSize}
            />
          ) : null
        }
      >
        <DataTable
          plain
          columns={columns}
          rows={pageItems}
          loading={busy}
          error={err}
          emptyMessage={
            filteredHint ? "No records match the current filters" : emptyMessage
          }
          sort={list.sort}
          onSort={list.onSort}
          onRowClick={inquiry ? (r) => setViewing(r) : undefined}
        />
      </ListShell>

      {viewing && inquiry ? (
        <Modal
          open
          size="lg"
          title={inquiryTitle ? inquiryTitle(viewing) : "Inquiry"}
          onClose={() => setViewing(null)}
          footer={
            <Button variant="secondary" onClick={() => setViewing(null)}>
              Close
            </Button>
          }
        >
          {inquiry(viewing)}
        </Modal>
      ) : null}
    </>
  );
}
