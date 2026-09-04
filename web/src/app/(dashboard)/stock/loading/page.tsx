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
import { formatDate, formatQty } from "@/lib/format";
import { partyName, qtyForGrade } from "@/lib/ilr";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { stockStatusLabel, type StockStatusRef } from "@/lib/status";
import { creatorLabel } from "@/lib/creator";

type Party = { id: string; code?: string; name?: string };
type Line = { id: string };
type GLR = {
  id: string;
  documentNumber: string;
  orderDate: string;
  description?: string;
  quantity: string;
  byProductQuantity?: string;
  status: string;
  customerOrderNumber?: string;
  validContract?: boolean;
  loadingOrderAvailable?: boolean;
  customer?: Party;
  product?: Party;
  byProduct?: Party | null;
  stockStatus?: StockStatusRef;
  lines?: Line[];
  createdBy?: { name?: string; email?: string };
};

export default function LoadingPage() {
  const [tick, setTick] = useState(0);
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(PERMISSIONS.ilrCreate);
  const canSubmit = hasPermission(PERMISSIONS.ilrSubmit);

  async function submit(id: string) {
    setMsg("");
    try {
      await api.post(`/orders/loading/${id}/submit`, {});
      setMsg("Submitted for approval");
      setTick((n) => n + 1);
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "Could not submit");
    }
  }

  const columns: EnterpriseColumn<GLR>[] = useMemo(
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
        id: "orderDate",
        header: "Date",
        sortKey: "orderDate",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => formatDate(r.orderDate),
        cell: (r) => formatDate(r.orderDate),
      },
      {
        id: "customer",
        header: "Customer",
        sortKey: "customer",
        defaultVisible: true,
        exportValue: (r) => r.customer?.name || r.customer?.code,
        cell: (r) => r.customer?.name || r.customer?.code || "—",
      },
      {
        id: "productStatus",
        header: "Product status",
        sortKey: "productStatus",
        defaultVisible: true,
        exportValue: (r) => (r.stockStatus ? stockStatusLabel(r.stockStatus) : ""),
        cell: (r) => (r.stockStatus ? stockStatusLabel(r.stockStatus) : "—"),
      },
      {
        id: "ago",
        header: "AGO (L)",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => qtyForGrade(r, "ago"),
        cell: (r) => <span className="tabular-nums">{formatQty(qtyForGrade(r, "ago"))}</span>,
      },
      {
        id: "pms",
        header: "PMS (L)",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => qtyForGrade(r, "pms"),
        cell: (r) => <span className="tabular-nums">{formatQty(qtyForGrade(r, "pms"))}</span>,
      },
      {
        id: "trucks",
        header: "Trucks",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.lines?.length ?? 0,
        cell: (r) => String(r.lines?.length ?? 0),
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
              href={`/stock/loading/${r.id}/edit`}
              className="text-sm font-semibold text-brand-800"
              onClick={(e) => e.stopPropagation()}
            >
              {r.status === "draft" ? "Edit" : "Open"}
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
      tableId="orders-loading"
      title="Internal loading requests"
      subtitle="Create and approve the ILR, then complete trucks at Compartmentalization and Gantry completion"
      headerExtra={
        canCreate ? (
          <Link
            href="/stock/loading/new"
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2.5 text-sm font-semibold text-white"
          >
            New ILR
          </Link>
        ) : null
      }
      lead={msg ? <p className="mb-3 text-sm text-slate-600">{msg}</p> : null}
      pills={{
        options: DOC_STATUS_PILLS,
        of: (r) => r.status,
        allTitle: "All requests",
        guideTitle: "Loading status",
      }}
      remote={{ path: "/orders/loading", dates: true, reloadKey: tick }}
      emptyMessage="No internal loading requests"
      columns={columns}
      searchPlaceholder="Document, customer, product"
      defaultOrderBy="orderDate"
      defaultSortDirection="DESC"
      exportName="ilr"
      inquiryTitle={(r) => r.documentNumber}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Stock · Internal loading · Inquiry"
            title={r.documentNumber}
            pills={<StatusPill tone="navy">{r.status}</StatusPill>}
          />
          <InquirySection title="Request">
            <ErpField label="Date" value={formatDate(r.orderDate)} />
            <ErpField label="Description" value={r.description || "—"} />
            <ErpField label="Customer" value={r.customer?.name || r.customer?.code || "—"} />
            <ErpField
              label="Product status"
              value={r.stockStatus ? stockStatusLabel(r.stockStatus) : "—"}
            />
            <ErpField label="Product" value={partyName(r.product)} />
            <ErpField label="By-product" value={partyName(r.byProduct)} />
            <ErpField label="AGO (L)" value={formatQty(qtyForGrade(r, "ago"))} />
            <ErpField label="PMS (L)" value={formatQty(qtyForGrade(r, "pms"))} />
            <ErpField label="Trucks" value={String(r.lines?.length ?? 0)} />
            <ErpField label="Valid contract" value={r.validContract ? "Yes" : "No"} />
            <ErpField label="Loading order" value={r.loadingOrderAvailable ? "Yes" : "No"} />
          </InquirySection>
          <Link
            href={`/stock/loading/${r.id}/edit`}
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2 text-sm font-semibold text-white"
          >
            Open document
          </Link>
        </div>
      )}
    />
  );
}
