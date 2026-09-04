"use client";

import Link from "next/link";
import { useMemo } from "react";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import { StatusBadge } from "@/components/ui";
import { DOC_STATUS_PILLS } from "@/components/StatusPills";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { formatDate, formatQty } from "@/lib/format";
import { creatorLabel } from "@/lib/creator";

type Ref = { id: string; documentNumber?: string };
type Amend = {
  id: string;
  documentNumber: string;
  kind: string;
  status: string;
  requestedQty: string;
  destination?: string;
  notes?: string;
  expirationDate?: string;
  truckPlate?: string;
  ilo?: Ref;
  createdBy?: { name?: string; email?: string };
};

export default function AmendmentsPage() {
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(PERMISSIONS.amendmentCreate);
  const columns: EnterpriseColumn<Amend>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        exportValue: (r) => r.documentNumber,
        cell: (r) => (
          <Link href={`/stock/amendments/${r.id}`} className="font-medium text-brand-800 hover:underline">
            {r.documentNumber}
          </Link>
        ),
      },
      {
        id: "kind",
        header: "Kind",
        sortKey: "kind",
        defaultVisible: true,
        exportValue: (r) => r.kind,
        cell: (r) => r.kind,
      },
      {
        id: "glo",
        header: "GLO",
        defaultVisible: true,
        exportValue: (r) => r.ilo?.documentNumber,
        cell: (r) => r.ilo?.documentNumber || "—",
      },
      {
        id: "requestedQty",
        header: "Qty",
        sortKey: "requestedQty",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.requestedQty,
        cell: (r) => <span className="tabular-nums">{formatQty(r.requestedQty)}</span>,
      },
      {
        id: "truck",
        header: "Truck",
        defaultVisible: true,
        exportValue: (r) => r.truckPlate,
        cell: (r) => r.truckPlate || "—",
      },
      {
        id: "destination",
        header: "Destination",
        defaultVisible: false,
        exportValue: (r) => r.destination,
        cell: (r) => r.destination || "—",
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
    ],
    [],
  );

  return (
    <EnterpriseTable
      tableId="orders-amendments"
      title="Loading amendments"
      subtitle="Change quantity, product, truck, or expiry on an open GLO. The old compartmentalization is voided after approval."
      headerExtra={
        canCreate ? (
          <Link
            href="/stock/amendments/new"
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2.5 text-sm font-semibold text-white"
          >
            New amendment
          </Link>
        ) : null
      }
      pills={{
        options: DOC_STATUS_PILLS,
        of: (r) => r.status,
        guideTitle: "Amendment status",
        allTitle: "All amendments",
      }}
      remote={{ path: "/orders/amendments", dates: true }}
      emptyMessage="No amendments"
      columns={columns}
      searchPlaceholder="Document, kind, GLO, truck"
      defaultOrderBy="documentNumber"
      defaultSortDirection="DESC"
      exportName="loading_amendments"
      inquiryTitle={(r) => r.documentNumber}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Stock · Gantry · Amendments · Inquiry"
            title={r.documentNumber}
            pills={<StatusPill tone="navy">{r.status}</StatusPill>}
          />
          <InquirySection title="Amendment">
            <ErpField label="Kind" value={r.kind} />
            <ErpField label="GLO" value={r.ilo?.documentNumber || "—"} />
            <ErpField label="Quantity" value={formatQty(r.requestedQty)} />
            <ErpField label="Expiry" value={formatDate(r.expirationDate)} />
            <ErpField label="Truck" value={r.truckPlate || "—"} />
            <ErpField label="Destination" value={r.destination || "—"} />
            <ErpField label="Notes" value={r.notes || "—"} />
          </InquirySection>
        </div>
      )}
    />
  );
}
