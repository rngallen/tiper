"use client";

import type { ReactNode } from "react";
import { Button, Card, DateInput, PageHeader } from "@/components/ui";
import {
  StatusGuide,
  StatusPills,
  type StatusPillOption,
} from "@/components/StatusPills";

/** From / To + Clear dates — same control row as invoices in the payment portal. */
export function DateRangeFields({
  from,
  to,
  onFrom,
  onTo,
  onClear,
}: {
  from: string;
  to: string;
  onFrom: (iso: string) => void;
  onTo: (iso: string) => void;
  onClear?: () => void;
}) {
  return (
    <>
      <FilterField label="From" className="w-36">
        <DateInput
          value={from}
          max={to || undefined}
          onChange={onFrom}
          aria-label="From date"
        />
      </FilterField>
      <FilterField label="To" className="w-36">
        <DateInput
          value={to}
          min={from || undefined}
          onChange={onTo}
          aria-label="To date"
        />
      </FilterField>
      {onClear && (from || to) ? (
        <Button
          type="button"
          variant="ghost"
          className="!px-2 !py-1 text-xs"
          onClick={onClear}
        >
          Clear dates
        </Button>
      ) : null}
    </>
  );
}

/** Labeled dropdown used on the filter row (Active, Class, Currency, …). */
export function FilterField({
  label,
  children,
  className = "w-44",
}: {
  label: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={`shrink-0 ${className}`}>
      <label className="mb-1 block text-xs font-medium text-slate-600">
        {label}
      </label>
      {children}
    </div>
  );
}

/**
 * Standard list page chrome (header actions, status pills, filters + search,
 * table card, pagination footer). Matches the enterprise vendor/catalogue layout.
 */
export function ListShell({
  title,
  subtitle,
  actions,
  lead,
  pills,
  filters,
  search,
  children,
  footer,
}: {
  title?: string;
  subtitle?: string;
  actions?: ReactNode;
  /** Optional create form or notice between the header and the filter row. */
  lead?: ReactNode;
  pills?: {
    options: StatusPillOption[];
    value: string;
    onChange: (id: string) => void;
    allTitle?: string;
    guideTitle?: string;
    showAll?: boolean;
  };
  filters?: ReactNode;
  search?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
}) {
  return (
    <div>
      {title ? (
        <PageHeader title={title} subtitle={subtitle} actions={actions} />
      ) : actions ? (
        <div className="mb-3 flex flex-wrap items-center justify-end gap-2">
          {actions}
        </div>
      ) : null}

      {lead}

      {pills ? (
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <StatusPills
            options={pills.options}
            value={pills.value}
            allTitle={pills.allTitle}
            showAll={pills.showAll}
            onChange={pills.onChange}
          />
          {pills.guideTitle ? (
            <StatusGuide title={pills.guideTitle} options={pills.options} />
          ) : null}
        </div>
      ) : null}

      {filters || search ? (
        <div className="mb-4 flex flex-wrap items-end gap-3">
          {filters}
          {search}
        </div>
      ) : null}

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">{children}</div>
        {footer}
      </Card>
    </div>
  );
}
