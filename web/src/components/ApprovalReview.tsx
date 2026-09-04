"use client";

import type { ReactNode } from "react";
import { formatDateTime, titleCase } from "@/lib/format";
import type { WorkflowEvent } from "@/lib/types";
import { SubstituteBadge } from "@/components/SubstituteBadge";

/** In-record section tabs (underline + optional count). Page nav uses ModuleTabs. */
export function ReviewTabBar<T extends string>({
  tab,
  onChange,
  ids,
  ariaLabel,
}: {
  tab: T;
  onChange: (t: T) => void;
  ids: { id: T; label: string; count?: number }[];
  ariaLabel?: string;
}) {
  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      className="flex flex-wrap border-b border-slate-200"
    >
      {ids.map((t) => {
        const active = tab === t.id;
        return (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onChange(t.id)}
            className={`-mb-px flex items-center gap-2 border-b-[3px] px-4 py-2.5 text-sm font-semibold transition ${
              active
                ? "border-brand-800 text-brand-800"
                : "border-transparent text-slate-500 hover:text-brand-800"
            }`}
          >
            {t.label}
            {t.count != null ? (
              <span
                className={`rounded-full px-1.5 py-0.5 text-[11px] font-bold tabular-nums ${
                  active
                    ? "bg-brand-800 text-white"
                    : "bg-slate-100 text-slate-600"
                }`}
              >
                {t.count}
              </span>
            ) : null}
          </button>
        );
      })}
    </div>
  );
}

export function ErpField({
  label,
  value,
  boxed = true,
}: {
  label: string;
  value: ReactNode;
  boxed?: boolean;
}) {
  return (
    <div className="min-w-0">
      <div className="text-[13px] font-semibold leading-5 text-slate-800">
        {label}
      </div>
      {boxed ? (
        <div className="pp-readonly mt-1 rounded-md border border-slate-400 px-3.5 py-2.5 text-[15px] font-semibold leading-6 text-slate-950">
          {value}
        </div>
      ) : (
        <div className="mt-1 text-[15px] font-semibold leading-6 text-slate-950">{value}</div>
      )}
    </div>
  );
}

function Pill({
  children,
  tone = "slate",
}: {
  children: ReactNode;
  tone?: "navy" | "slate" | "amber" | "emerald";
}) {
  const cls =
    tone === "navy"
      ? "bg-sky-50 text-brand-800"
      : tone === "amber"
        ? "bg-orange-50 text-orange-800"
        : tone === "emerald"
          ? "bg-emerald-50 text-emerald-800"
          : "bg-slate-100 text-slate-700";
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-semibold ${cls}`}
    >
      {children}
    </span>
  );
}

/** Shared identity strip for inquiry / view modals. Optional children sit in the header (master fields). */
export function InquiryHeader({
  crumb,
  title,
  subtitle,
  pills,
  children,
}: {
  crumb: string;
  title: string;
  subtitle?: ReactNode;
  pills?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white">
      <div className="flex flex-wrap items-start justify-between gap-3 px-4 py-2.5">
        <div className="min-w-0">
          <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">
            {crumb}
          </p>
          <h3 className="mt-0.5 text-base font-bold text-slate-900">{title}</h3>
          {subtitle ? (
            <p className="mt-0.5 text-sm text-slate-600">{subtitle}</p>
          ) : null}
        </div>
        {pills ? (
          <div className="flex flex-wrap items-center justify-end gap-1.5">
            {pills}
          </div>
        ) : null}
      </div>
      {children ? (
        <div className="grid gap-3 border-t border-slate-100 px-4 py-3 sm:grid-cols-3">
          {children}
        </div>
      ) : null}
    </div>
  );
}

/** Boxed field panel used by inquiry and review modals. */
export function InquirySection({
  title,
  children,
  columns = 2,
}: {
  title?: string;
  children: ReactNode;
  columns?: 1 | 2 | 3;
}) {
  const grid =
    columns === 3
      ? "grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2 xl:grid-cols-3"
      : columns === 1
        ? "grid grid-cols-1 gap-3"
        : "grid grid-cols-1 gap-3 sm:grid-cols-2";
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-4">
      {title ? (
        <h4 className="mb-3 text-xs font-bold uppercase tracking-wide text-brand-800">
          {title}
        </h4>
      ) : null}
      <div className={grid}>{children}</div>
    </section>
  );
}

export function StatusPill({
  children,
  tone = "slate",
}: {
  children: ReactNode;
  tone?: "navy" | "slate" | "amber" | "emerald";
}) {
  return <Pill tone={tone}>{children}</Pill>;
}

/** Compact crumb + status on a document card (avoids repeating the page title). */
function historyStatus(e: WorkflowEvent): { label: string; className: string } {
  const t = (e.actType || "").toLowerCase();
  const name = (e.actName || "").toLowerCase();
  if (t === "initiate") return { label: "Submitted", className: "text-teal-700" };
  if (name === "resubmitted" || name.includes("resubmit")) {
    return { label: "Resubmitted", className: "text-sky-700" };
  }
  if (t === "agree") return { label: "Approved", className: "text-emerald-700" };
  if (t === "reject") {
    return e.totalRejection
      ? { label: "Rejected", className: "text-red-700" }
      : { label: "Returned", className: "text-orange-700" };
  }
  if (t === "transfer") return { label: "Transferred", className: "text-blue-700" };
  if (t === "back") return { label: "Sent back", className: "text-amber-700" };
  if (t === "complete") return { label: "Completed", className: "text-emerald-700" };
  return { label: e.actName || titleCase(e.actType || "—"), className: "text-slate-700" };
}

/** Compact prior comments so the Decision tab always shows the trail. */
export function PriorComments({ events }: { events?: WorkflowEvent[] }) {
  const rows = [...(events || [])]
    .filter((e) => (e.comment || "").trim())
    .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime());
  if (rows.length === 0) {
    return (
      <p className="text-sm text-slate-500">No previous comments. Yours will be the first on the trail.</p>
    );
  }
  return (
    <ol className="space-y-2">
      {rows.map((e, i) => {
        const st = historyStatus(e);
        return (
          <li key={`${e.createdAt}-${i}`} className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2">
            <div className="flex flex-wrap items-baseline justify-between gap-2 text-xs text-slate-500">
              <span className="font-semibold text-slate-800">
                {e.userName || e.userEmail || "System"}
                {e.userTitle ? ` · ${e.userTitle}` : ""}
              </span>
              <span>{formatDateTime(e.createdAt)}</span>
            </div>
            <p className={`mt-0.5 text-xs font-semibold ${st.className}`}>{st.label}</p>
            <p className="mt-1 text-sm text-slate-800">{e.comment}</p>
          </li>
        );
      })}
    </ol>
  );
}

/** Ledger of prior comments and decisions for the current approver. */
export function ApprovalHistory({ events }: { events?: WorkflowEvent[] }) {
  const rows = [...(events || [])]
    .filter((e) => (e.actType || "").toLowerCase() !== "complete")
    .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime());
  return (
    <div>
      <h3 className="mb-2 text-sm font-semibold text-slate-700">Approval trail</h3>
      <p className="mb-3 text-xs text-slate-600">
        Every submit, approve, return, reject, and transfer on this document — including comments.
      </p>
      {rows.length === 0 ? (
        <p className="text-sm text-slate-500">No trail yet. Your comment will be the first entry.</p>
      ) : (
        <div className="max-h-80 overflow-auto rounded-lg border border-slate-200">
          <table className="pp-ledger min-w-full text-left text-sm">
            <thead>
              <tr>
                <th>#</th>
                <th>When</th>
                <th>Approver</th>
                <th>Role</th>
                <th>Action</th>
                <th>Status</th>
                <th>Step</th>
                <th>Comment</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((e, i) => {
                const st = historyStatus(e);
                return (
                  <tr key={`${e.createdAt}-${i}`}>
                    <td>{i + 1}</td>
                    <td className="whitespace-nowrap">{formatDateTime(e.createdAt)}</td>
                    <td className="!whitespace-normal font-medium">
                      <div>{e.userName || e.userEmail || "System"}</div>
                      {e.onBehalfOfName || e.onBehalfOfEmail ? (
                        <SubstituteBadge covering={e.onBehalfOfName || e.onBehalfOfEmail} />
                      ) : null}
                    </td>
                    <td className="!whitespace-normal">
                      {e.userTitle || (e.actType === "initiate" ? "Initiator" : "—")}
                    </td>
                    <td>{e.actName || titleCase(e.actType)}</td>
                    <td className={`font-semibold ${st.className}`}>{st.label}</td>
                    <td className="!whitespace-normal">
                      {[e.oldNode, e.newNode].filter(Boolean).join(" → ") || "—"}
                    </td>
                    <td className="!whitespace-normal max-w-xs">{e.comment || "—"}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

export function DocumentMeta({
  crumb,
  children,
}: {
  crumb: string;
  children?: ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">
        {crumb}
      </p>
      {children ? <div className="flex flex-wrap items-center gap-2">{children}</div> : null}
    </div>
  );
}
