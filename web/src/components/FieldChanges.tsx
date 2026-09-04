"use client";

import { formatFieldLabel } from "@/lib/format";

type FieldChange = {
  before?: unknown;
  after?: unknown;
};

function fmt(v: unknown): string {
  if (v === null || v === undefined || v === "") return "—";
  if (typeof v === "boolean") return v ? "Yes" : "No";
  if (typeof v === "object") {
    try {
      return JSON.stringify(v);
    } catch {
      return String(v);
    }
  }
  return String(v);
}

/** Table of field-level before → after values. */
export function FieldChanges({
  changes,
  emptyMessage = "No field changes recorded for this request.",
}: {
  changes?: Record<string, FieldChange> | null;
  emptyMessage?: string;
}) {
  const rows = Object.entries(changes || {}).sort(([a], [b]) =>
    a.localeCompare(b),
  );
  if (rows.length === 0) {
    return <p className="text-sm text-slate-400">{emptyMessage}</p>;
  }
  return (
    <div className="max-h-[28rem] overflow-auto rounded-lg border border-slate-200">
      <table className="pp-ledger pp-ledger-wrap min-w-full text-left text-sm">
        <thead className="pp-table-head sticky top-0">
          <tr>
            <th className="w-[22%]">Field</th>
            <th className="w-[39%]">From</th>
            <th className="w-[39%]">To</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(([key, change]) => (
            <tr key={key}>
              <td className="font-medium text-slate-700">
                {formatFieldLabel(key)}
              </td>
              <td className="text-slate-500">{fmt(change?.before)}</td>
              <td className="font-semibold text-slate-900">{fmt(change?.after)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
