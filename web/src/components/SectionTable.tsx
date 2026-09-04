"use client";

import { Children, type ReactNode } from "react";

export type SectionColumn = {
  label: ReactNode;
  align?: "left" | "right";
  className?: string;
};

export function SectionPanel({
  title,
  actions,
  children,
}: {
  title?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="overflow-hidden rounded-md border border-slate-400">
      {title || actions ? (
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-300 bg-slate-100 px-4 py-2.5">
          {title ? <h3 className="text-[15px] font-semibold text-slate-950">{title}</h3> : <span />}
          {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
        </div>
      ) : null}
      {children}
    </div>
  );
}

/** ERP document line table — shared by fee batches, receipts, orders, and mappings. */
export function SectionTable({
  title,
  actions,
  columns,
  empty,
  children,
}: {
  title?: ReactNode;
  actions?: ReactNode;
  columns: SectionColumn[];
  empty?: ReactNode;
  children?: ReactNode;
}) {
  const rows = Children.toArray(children).filter(Boolean);
  return (
    <div className="overflow-hidden rounded-md border border-slate-400">
      {title || actions ? (
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-300 bg-slate-100 px-4 py-2.5">
          {title ? <h3 className="text-[15px] font-semibold text-slate-950">{title}</h3> : <span />}
          {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
        </div>
      ) : null}
      <div className="overflow-x-auto">
        <table className="pp-section">
          <thead>
            <tr>
              {columns.map((c, i) => (
                <th
                  key={i}
                  className={[c.align === "right" ? "text-right" : "", c.className || ""].filter(Boolean).join(" ")}
                >
                  {c.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr>
                <td className="pp-section-empty" colSpan={Math.max(columns.length, 1)}>
                  {empty ?? "No lines"}
                </td>
              </tr>
            ) : (
              children
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export function SectionRow({
  children,
  onClick,
  className = "",
}: {
  children: ReactNode;
  onClick?: () => void;
  className?: string;
}) {
  return (
    <tr
      className={[onClick ? "cursor-pointer" : "", className].filter(Boolean).join(" ")}
      onClick={onClick}
    >
      {children}
    </tr>
  );
}

export function SectionGroupRow({
  colSpan,
  children,
  actions,
}: {
  colSpan: number;
  children: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <tr className="pp-section-group">
      <td colSpan={actions ? colSpan - 1 : colSpan}>{children}</td>
      {actions ? <td className="text-right">{actions}</td> : null}
    </tr>
  );
}
