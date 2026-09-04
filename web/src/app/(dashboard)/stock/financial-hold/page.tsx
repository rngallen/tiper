"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { api } from "@/lib/api";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import { Button, StatusBadge } from "@/components/ui";
import { DOC_STATUS_PILLS } from "@/components/StatusPills";
import { formatDate } from "@/lib/format";
import { creatorLabel } from "@/lib/creator";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";

type HoldRelease = {
  id: string;
  documentNumber: string;
  releaseDate: string;
  description?: string;
  status: string;
  createdBy?: { name?: string; email?: string };
};

export default function FinancialHoldListPage() {
  const [tick, setTick] = useState(0);
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(PERMISSIONS.holdCreate);
  const canSubmit = hasPermission(PERMISSIONS.holdSubmit);

  async function submit(id: string) {
    setMsg("");
    try {
      await api.post(`/ic/hold-releases/${id}/submit`, {});
      setMsg("Submitted for approval");
      setTick((n) => n + 1);
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "Could not submit");
    }
  }

  const columns: EnterpriseColumn<HoldRelease>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        exportValue: (r) => r.documentNumber,
        cell: (r) => (
          <Link href={`/stock/financial-hold/${r.id}`} className="font-medium text-brand-800 hover:underline">
            {r.documentNumber}
          </Link>
        ),
      },
      {
        id: "releaseDate",
        header: "Date",
        sortKey: "releaseDate",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => formatDate(r.releaseDate),
        cell: (r) => formatDate(r.releaseDate),
      },
      {
        id: "description",
        header: "Description",
        defaultVisible: true,
        exportValue: (r) => r.description,
        cell: (r) => r.description || "—",
      },
      {
        id: "status",
        header: "Status",
        sortKey: "status",
        defaultVisible: true,
        exportValue: (r) => r.status,
        cell: (r) => <StatusBadge status={r.status} />,
      },
      {
        id: "createdBy",
        header: "Created by",
        defaultVisible: true,
        exportValue: (r) => creatorLabel(r.createdBy),
        cell: (r) => creatorLabel(r.createdBy),
      },
      {
        id: "_actions",
        fixed: true,
        header: "",
        className: "text-right",
        cell: (r) => (
          <div className="flex justify-end gap-2">
            <Link
              href={`/stock/financial-hold/${r.id}`}
              className="text-sm font-semibold text-brand-800"
              onClick={(e) => e.stopPropagation()}
            >
              Open
            </Link>
            {canSubmit && r.status === "draft" ? (
              <Button
                variant="secondary"
                className="!px-3 !py-1"
                onClick={(e) => {
                  e.stopPropagation();
                  void submit(r.id);
                }}
              >
                Submit
              </Button>
            ) : null}
          </div>
        ),
      },
    ],
    [canSubmit],
  );

  return (
    <EnterpriseTable
      tableId="ic-hold-releases"
      title="Financial hold releases"
      subtitle="Record payment against held parcels — released volume moves from financial hold onto free stock"
      headerExtra={
        canCreate ? (
          <Link
            href="/stock/financial-hold/new"
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2.5 text-sm font-semibold text-white"
          >
            New release
          </Link>
        ) : null
      }
      lead={msg ? <p className="mb-3 text-sm text-slate-600">{msg}</p> : null}
      pills={{
        options: DOC_STATUS_PILLS,
        of: (r) => r.status,
        guideTitle: "Release status",
        allTitle: "All releases",
      }}
      remote={{ path: "/ic/hold-releases", dates: true, reloadKey: tick }}
      emptyMessage="No financial hold releases"
      columns={columns}
      searchPlaceholder="Document or description"
      defaultOrderBy="releaseDate"
      defaultSortDirection="DESC"
      exportName="hold_releases"
      inquiryTitle={(r) => r.documentNumber}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Stock · Terminal · Financial hold · Inquiry"
            title={r.documentNumber}
            pills={<StatusPill tone="navy">{r.status}</StatusPill>}
          />
          <InquirySection title="Release">
            <ErpField label="Date" value={formatDate(r.releaseDate)} />
            <ErpField label="Description" value={r.description || "—"} />
            <ErpField label="Created by" value={creatorLabel(r.createdBy)} />
          </InquirySection>
          <Link
            href={`/stock/financial-hold/${r.id}`}
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2 text-sm font-semibold text-white"
          >
            Open document
          </Link>
        </div>
      )}
    />
  );
}
