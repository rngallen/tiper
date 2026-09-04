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
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { formatDate } from "@/lib/format";
import { creatorLabel } from "@/lib/creator";

type Ref = { id: string; documentNumber?: string };
type Comp = {
  id: string;
  documentNumber: string;
  status: string;
  createdAt?: string;
  customerName?: string;
  customerOrderNumber?: string;
  almaFileName?: string;
  horsePlate?: string;
  trailerOnePlate?: string;
  ilo?: Ref;
  createdBy?: { name?: string; email?: string };
};

export default function CompartmentalizationPage() {
  const [tick, setTick] = useState(0);
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(PERMISSIONS.compartmentalizationCreate);
  const canSubmit = hasPermission(PERMISSIONS.compartmentalizationSubmit);

  async function submit(id: string) {
    setMsg("");
    try {
      await api.post(`/orders/compartmentalizations/${id}/submit`, {});
      setMsg("Submitted — after approval DFMS writes the SAP3C file ALMA reads");
      setTick((n) => n + 1);
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "Could not submit");
    }
  }

  const columns: EnterpriseColumn<Comp>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        exportValue: (r) => r.documentNumber,
        cell: (r) => (
          <Link
            href={`/stock/compartmentalization/${r.id}`}
            className="font-medium text-brand-800 hover:underline"
          >
            {r.documentNumber}
          </Link>
        ),
      },
      {
        id: "createdAt",
        header: "Date",
        sortKey: "createdAt",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => formatDate(r.createdAt),
        cell: (r) => formatDate(r.createdAt),
      },
      {
        id: "glo",
        header: "GLO",
        defaultVisible: true,
        exportValue: (r) => r.ilo?.documentNumber,
        cell: (r) => r.ilo?.documentNumber || "—",
      },
      {
        id: "customer",
        header: "Customer",
        sortKey: "customer",
        defaultVisible: true,
        exportValue: (r) => r.customerName,
        cell: (r) => r.customerName || "—",
      },
      {
        id: "horsePlate",
        header: "Horse",
        sortKey: "horsePlate",
        defaultVisible: true,
        exportValue: (r) => r.horsePlate,
        cell: (r) => r.horsePlate || "—",
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
        id: "almaFileName",
        header: "ALMA file",
        sortKey: "almaFileName",
        defaultVisible: true,
        exportValue: (r) => r.almaFileName,
        cell: (r) => r.almaFileName || "—",
      },
      {
        id: "_actions",
        fixed: true,
        header: "",
        className: "text-right",
        cell: (r) => (
          <div className="flex justify-end gap-2">
            <Link
              href={`/stock/compartmentalization/${r.id}`}
              className="text-sm font-semibold text-brand-800"
              onClick={(e) => e.stopPropagation()}
            >
              Open
            </Link>
            {canSubmit && (r.status === "draft" || r.status === "running") ? (
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
      tableId="orders-compartmentalization"
      title="Compartmentalization"
      subtitle="Assign litres to each tank cell. Approval writes the ATLAS NEO order file."
      headerExtra={
        canCreate ? (
          <Link
            href="/stock/compartmentalization/new"
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2.5 text-sm font-semibold text-white"
          >
            New document
          </Link>
        ) : null
      }
      lead={msg ? <p className="mb-3 text-sm text-slate-600">{msg}</p> : null}
      pills={{
        options: DOC_STATUS_PILLS,
        of: (r) => r.status,
        guideTitle: "Compartmentalization status",
        allTitle: "All documents",
      }}
      remote={{ path: "/orders/compartmentalizations", dates: true, reloadKey: tick }}
      emptyMessage="No compartmentalizations"
      columns={columns}
      searchPlaceholder="Document, customer, horse, GLO"
      defaultOrderBy="createdAt"
      defaultSortDirection="DESC"
      exportName="compartmentalizations"
      inquiryTitle={(r) => r.documentNumber}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Stock · Gantry · Compartmentalization · Inquiry"
            title={r.documentNumber}
            pills={<StatusPill tone="navy">{r.status}</StatusPill>}
          />
          <InquirySection title="Dispatch">
            <ErpField label="Date" value={formatDate(r.createdAt)} />
            <ErpField label="GLO" value={r.ilo?.documentNumber || "—"} />
            <ErpField label="Customer" value={r.customerName || "—"} />
            <ErpField label="Customer order" value={r.customerOrderNumber || "—"} />
            <ErpField label="Horse" value={r.horsePlate || "—"} />
            <ErpField label="Trailer" value={r.trailerOnePlate || "—"} />
            <ErpField label="ALMA file" value={r.almaFileName || "—"} />
          </InquirySection>
          <Link
            href={`/stock/compartmentalization/${r.id}`}
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2 text-sm font-semibold text-white"
          >
            Open document
          </Link>
        </div>
      )}
    />
  );
}
