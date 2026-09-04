"use client";

import type { MouseEvent, ReactNode } from "react";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import { Card, Empty, Spinner, ErrorBanner } from "./ui";

export type SortDirection = "ASC" | "DESC";

export interface Column<T> {
  header: ReactNode;
  cell: (row: T) => ReactNode;
  className?: string;
  /** API orderBy key — when set, header is clickable for sorting. */
  sortKey?: string;
}

export interface SortState {
  orderBy: string;
  sortDirection: SortDirection;
}

function isInteractiveTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false;
  return Boolean(
    target.closest(
      "button, a, input, select, textarea, label, [data-no-row-click]",
    ),
  );
}

export function DataTable<T>({
  columns,
  rows,
  loading,
  error,
  emptyMessage = "No records found",
  onRowClick,
  sort,
  onSort,
  plain,
}: {
  columns: Column<T>[];
  rows: T[] | null | undefined;
  loading?: boolean;
  error?: string;
  emptyMessage?: string;
  onRowClick?: (row: T) => void;
  sort?: SortState;
  onSort?: (orderBy: string) => void;
  /** Skip the outer card when the table already sits inside a panel. */
  plain?: boolean;
}) {
  if (error) return <ErrorBanner message={error} />;
  if (loading) return <Spinner />;

  const showEmpty = !rows || rows.length === 0;

  function handleRowClick(e: MouseEvent<HTMLTableRowElement>, row: T) {
    if (!onRowClick || isInteractiveTarget(e.target)) return;
    onRowClick(row);
  }

  const table = (
    <div className={plain ? "overflow-x-auto" : undefined}>
      <table className="pp-ledger">
        <thead className="pp-table-head">
          <tr>
            {columns.map((c, i) => {
              const sortable = Boolean(c.sortKey && onSort);
              const active = sortable && sort?.orderBy === c.sortKey;
              return (
                <th key={i} className={c.className}>
                  {sortable ? (
                    <button
                      type="button"
                      className={`inline-flex items-center gap-1.5 font-bold uppercase tracking-wide text-inherit hover:opacity-90 ${
                        c.className?.includes("text-right")
                          ? "w-full justify-end"
                          : ""
                      }`}
                      onClick={() => onSort!(c.sortKey!)}
                    >
                      {c.header}
                      {active ? (
                        sort?.sortDirection === "ASC" ? (
                          <ArrowUp className="h-3.5 w-3.5 opacity-90" />
                        ) : (
                          <ArrowDown className="h-3.5 w-3.5 opacity-90" />
                        )
                      ) : (
                        <ArrowUpDown className="h-3.5 w-3.5 opacity-50" />
                      )}
                    </button>
                  ) : typeof c.header === "string" ? (
                    <span className="font-bold uppercase tracking-wide">
                      {c.header}
                    </span>
                  ) : (
                    c.header
                  )}
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {showEmpty ? (
            <tr>
              <td colSpan={columns.length} className="!whitespace-normal">
                <Empty message={emptyMessage} />
              </td>
            </tr>
          ) : (
            rows!.map((row, ri) => (
              <tr
                key={ri}
                onClick={(e) => handleRowClick(e, row)}
                className={`transition-colors hover:!bg-[var(--pp-row-hover)] ${onRowClick ? "cursor-pointer" : ""}`}
              >
                {columns.map((c, ci) => (
                  <td key={ci} className={c.className}>
                    {c.cell(row)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );

  if (plain) return table;
  return <Card className="overflow-x-auto">{table}</Card>;
}
