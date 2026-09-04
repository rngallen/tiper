"use client";

import { useState } from "react";
import { HelpCircle } from "lucide-react";

export type StatusPillOption = {
  id: string;
  label: string;
  meaning?: string;
  count?: number;
};

/** Active / inactive catalogue filter (users, roles, master data). */
export const ACTIVE_STATUS_PILLS: StatusPillOption[] = [
  { id: "true", label: "Active", meaning: "Currently usable in new work." },
  {
    id: "false",
    label: "Inactive",
    meaning: "Hidden from new work; existing rows remain.",
  },
];

/** Document approval lane — same shape as the enterprise vendor list. */
export const DOC_STATUS_PILLS: StatusPillOption[] = [
  { id: "draft", label: "Unapproved", meaning: "Saved as draft; not submitted." },
  { id: "submitted", label: "In progress", meaning: "Submitted and waiting for approval." },
  { id: "returned", label: "Returned", meaning: "Sent back for amendment." },
  { id: "approved", label: "Approved", meaning: "Approved and in force." },
];

/** Primary list-status tabs (All + lifecycle). Used on operational lists and approvals. */
export function StatusPills({
  options,
  value,
  onChange,
  allTitle = "All statuses",
  showAll = true,
}: {
  options: StatusPillOption[];
  value: string;
  onChange: (id: string) => void;
  allTitle?: string;
  /** When false, there is no All pill (e.g. approval inbox always has a lane). */
  showAll?: boolean;
}) {
  return (
    <div className="flex flex-wrap gap-1">
      {showAll ? (
        <button
          type="button"
          title={allTitle}
          onClick={() => onChange("")}
          className={pillClass(value === "")}
        >
          All
        </button>
      ) : null}
      {options.map((s) => (
        <button
          key={s.id}
          type="button"
          title={s.meaning}
          onClick={() => onChange(s.id)}
          className={pillClass(value === s.id)}
        >
          {s.label}
          {typeof s.count === "number" ? ` (${s.count})` : ""}
        </button>
      ))}
    </div>
  );
}

function pillClass(active: boolean) {
  return `rounded-full px-3 py-1 text-xs font-medium ${
    active
      ? "bg-brand-800 text-white"
      : "bg-white text-slate-600 hover:bg-slate-100"
  }`;
}

export function StatusGuide({
  title,
  options,
}: {
  title: string;
  options: StatusPillOption[];
}) {
  const [open, setOpen] = useState(false);
  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="inline-flex items-center gap-1 rounded-full px-2 py-1 text-xs font-medium text-brand-800 hover:bg-brand-50"
        aria-expanded={open}
      >
        <HelpCircle className="h-3.5 w-3.5" />
        Status guide
      </button>
      {open ? (
        <div className="absolute left-0 z-30 mt-2 w-80 rounded-xl border border-slate-200 bg-[var(--pp-surface)] p-3 shadow-lg">
          <p className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-[var(--pp-muted)]">
            {title}
          </p>
          <ul className="space-y-2">
            {options.map((s) => (
              <li key={s.id}>
                <div className="text-xs font-semibold text-[var(--pp-text)]">
                  {s.label}
                </div>
                {s.meaning ? (
                  <p className="text-xs leading-snug text-[var(--pp-muted)]">
                    {s.meaning}
                  </p>
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
