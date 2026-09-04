"use client";

import type { ReactNode } from "react";
import { Modal } from "@/components/Modal";
import { Button, ErrorBanner } from "@/components/ui";

export function CatalogueForm({
  title,
  error,
  saving,
  onClose,
  onSave,
  saveLabel,
  children,
  size = "md",
}: {
  title: string;
  error: string;
  saving: boolean;
  onClose: () => void;
  onSave: () => void;
  saveLabel: string;
  children: ReactNode;
  size?: "md" | "lg" | "xl" | "2xl";
}) {
  return (
    <Modal
      open
      onClose={onClose}
      title={title}
      size={size}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={onSave} loading={saving}>
            {saveLabel}
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {error && <ErrorBanner message={error} />}
        {children}
      </div>
    </Modal>
  );
}

export function RecordPlaceholder({ children }: { children: ReactNode }) {
  return (
    <p className="rounded-lg border border-dashed border-slate-200 bg-slate-50 px-3 py-6 text-center text-sm text-slate-600">
      {children}
    </p>
  );
}

/** ERP-style record: section list on the left, one pane at a time on the right. */
export function PartyWorkspace<T extends string>({
  panes,
  pane,
  onPane,
  hint,
  banner,
  children,
}: {
  panes: { id: T; label: string }[];
  pane: T;
  onPane: (id: T) => void;
  hint?: string;
  banner?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex min-h-[26rem] overflow-hidden rounded-lg border border-slate-200">
      <nav
        className="flex w-40 shrink-0 flex-col gap-0.5 border-r border-slate-200 bg-white p-1.5"
        aria-label="Record sections"
      >
        {panes.map((p) => {
          const active = pane === p.id;
          return (
            <button
              key={p.id}
              type="button"
              onClick={() => onPane(p.id)}
              aria-current={active ? "page" : undefined}
              className={`whitespace-nowrap rounded-md px-3 py-2 text-left text-sm font-semibold ${
                active
                  ? "border-l-2 border-brand-800 bg-brand-50 text-brand-900"
                  : "border-l-2 border-transparent text-slate-700 hover:bg-slate-50"
              }`}
            >
              {p.label}
            </button>
          );
        })}
      </nav>
      <div className="min-w-0 flex-1 overflow-y-auto p-4">
        {banner ? (
          <div className="mb-3 border-b border-slate-200 pb-3 text-[15px] font-semibold text-slate-950">
            {banner}
          </div>
        ) : null}
        {hint ? (
          <p className="mb-3 text-xs leading-5 text-slate-500">{hint}</p>
        ) : null}
        {children}
      </div>
    </div>
  );
}
