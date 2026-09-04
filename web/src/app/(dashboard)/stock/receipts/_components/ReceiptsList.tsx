"use client";

import { useMemo } from "react";
import Link from "next/link";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import { StatusBadge } from "@/components/ui";
import { DOC_STATUS_PILLS } from "@/components/StatusPills";
import { formatDate } from "@/lib/format";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { creatorLabel } from "@/lib/creator";
import type { Receipt } from "../_lib/types";

export function ReceiptsList({ kind }: { kind: "internal" | "external" }) {
  const internal = kind === "internal";
  const base = internal ? "/stock/receipts" : "/stock/receipts/external";
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(PERMISSIONS.receiptsCreate);
  const columns: EnterpriseColumn<Receipt>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        exportValue: (r) => r.documentNumber,
        cell: (r) => <div className="font-medium text-slate-800">{r.documentNumber}</div>,
      },
      {
        id: "date",
        header: "Date",
        sortKey: "date",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => formatDate(r.date),
        cell: (r) => formatDate(r.date),
      },
      {
        id: "vessel",
        header: "Vessel",
        sortKey: "vessel",
        defaultVisible: true,
        exportValue: (r) => r.vessel?.code || r.vessel?.name,
        cell: (r) => r.vessel?.code || r.vessel?.name || "—",
      },
      {
        id: "product",
        header: "Product",
        sortKey: "product",
        defaultVisible: true,
        exportValue: (r) => r.product?.code,
        cell: (r) => r.product?.code || "—",
      },
      {
        id: "routeCode",
        header: "Route",
        sortKey: "routeCode",
        defaultVisible: true,
        exportValue: (r) => r.routeCode,
        cell: (r) => r.routeCode || "—",
      },
      ...(internal
        ? [
            {
              id: "isProvision",
              header: "Survey",
              sortKey: "isProvision",
              defaultVisible: true,
              exportValue: (r: Receipt) => (r.isProvision ? "Provision" : "Final"),
              cell: (r: Receipt) => (r.isProvision ? "Provision" : "Final"),
            } satisfies EnterpriseColumn<Receipt>,
          ]
        : []),
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
          <Link
            href={`/stock/receipts/${r.id}/edit`}
            className="text-sm font-semibold text-brand-800"
            onClick={(e) => e.stopPropagation()}
          >
            Open
          </Link>
        ),
      },
    ],
    [internal],
  );

  return (
    <EnterpriseTable
      tableId={internal ? "ic-receipts-internal" : "ic-receipts-external"}
      title={internal ? "Internal vessel receipts" : "External vessel receipts"}
      subtitle={
        internal
          ? "Cargo delivered to TIPER — billed and posted to stock"
          : "Cargo delivered to other depots — report only; KOJ 10-inch pipeline takes a service fee"
      }
      headerExtra={
        canCreate ? (
          <Link
            href={`${base}/new`}
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2.5 text-sm font-semibold text-white"
          >
            New receipt
          </Link>
        ) : null
      }
      pills={{
        options: DOC_STATUS_PILLS,
        of: (r) => r.status,
        guideTitle: "Receipt status",
        allTitle: "All receipts",
      }}
      remote={{ path: "/ic/receipts", dates: true, extraQuery: { receiptType: kind } }}
      emptyMessage={internal ? "No internal receipts" : "No external receipts"}
      columns={columns}
      searchPlaceholder="Document, vessel, product"
      defaultOrderBy="date"
      defaultSortDirection="DESC"
      exportName={internal ? "internal_receipts" : "external_receipts"}
      inquiryTitle={(r) => r.documentNumber}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb={internal ? "Stock · Reception · Internal" : "Stock · Reception · External"}
            title={r.documentNumber}
            pills={
              <>
                <StatusPill tone="navy">{r.status}</StatusPill>
                {internal ? (
                  <StatusPill tone={r.isProvision ? "amber" : "slate"}>
                    {r.isProvision ? "Provision" : "Final"}
                  </StatusPill>
                ) : null}
              </>
            }
          />
          <InquirySection title="Receipt">
            <ErpField label="Date" value={formatDate(r.date)} />
            <ErpField label="Vessel" value={r.vessel?.name || r.vessel?.code || "—"} />
            <ErpField label="Product" value={r.product?.name || r.product?.code || "—"} />
            <ErpField label="Route" value={r.routeCode || "—"} />
          </InquirySection>
          <Link
            href={`/stock/receipts/${r.id}/edit`}
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2 text-sm font-semibold text-white"
          >
            Open document
          </Link>
        </div>
      )}
    />
  );
}
