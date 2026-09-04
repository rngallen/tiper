"use client";

import Link from "next/link";
import { ArrowLeft } from "lucide-react";

/** Report page chrome: back to catalogue + title/actions. */
export function ReportPageHeader({
  title,
  subtitle,
  actions,
}: {
  title: string;
  subtitle?: string;
  actions?: React.ReactNode;
}) {
  return (
    <div className="mb-6">
      <Link
        href="/reports"
        className="mb-3 inline-flex items-center gap-1.5 text-sm font-medium text-[#033860] hover:underline"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        All reports
      </Link>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">{title}</h1>
          {subtitle ? (
            <p className="mt-1 text-sm font-normal text-slate-700">{subtitle}</p>
          ) : null}
        </div>
        {actions ? <div className="flex gap-2">{actions}</div> : null}
      </div>
    </div>
  );
}
