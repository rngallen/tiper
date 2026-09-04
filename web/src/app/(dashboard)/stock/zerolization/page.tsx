"use client";

import Link from "next/link";
import { useMemo } from "react";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import { StatusBadge } from "@/components/ui";
import { DOC_STATUS_PILLS } from "@/components/StatusPills";
import { formatDate, formatQty } from "@/lib/format";
import { creatorLabel } from "@/lib/creator";

type Zerol = {
  id: string;
  documentNumber: string;
  transferDate: string;
  quantity: string;
  status: string;
  createdBy?: { name?: string; email?: string };
};

export default function ZerolizationListPage() {
  const columns: EnterpriseColumn<Zerol>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        cell: (r) => (
          <Link href={`/stock/zerolization/${r.id}`} className="font-medium text-brand-800 hover:underline">
            {r.documentNumber}
          </Link>
        ),
      },
      {
        id: "transferDate",
        header: "Date",
        sortKey: "transferDate",
        defaultVisible: true,
        cell: (r) => formatDate(r.transferDate),
      },
      {
        id: "quantity",
        header: "Quantity",
        sortKey: "quantity",
        defaultVisible: true,
        className: "text-right",
        cell: (r) => <span className="tabular-nums">{formatQty(r.quantity)}</span>,
      },
      {
        id: "status",
        header: "Status",
        sortKey: "status",
        defaultVisible: true,
        cell: (r) => <StatusBadge status={r.status} />,
      },
      {
        id: "createdBy",
        header: "Created by",
        defaultVisible: true,
        cell: (r) => creatorLabel(r.createdBy),
      },
    ],
    [],
  );

  return (
    <EnterpriseTable
      tableId="zerolization"
      title="Zerolization"
      subtitle="Same-customer vessel consolidation"
      pills={{ options: DOC_STATUS_PILLS, of: (r) => r.status, guideTitle: "Zerolization status", allTitle: "All transfers" }}
      remote={{ path: "/ic/zerolization", dates: true }}
      emptyMessage="No zerolization transfers"
      columns={columns}
      searchPlaceholder="Document"
      defaultOrderBy="documentNumber"
      defaultSortDirection="DESC"
      dateOf={(r) => r.transferDate}
      exportName="zerolization"
    />
  );
}
