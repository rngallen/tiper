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
import { partyName } from "@/lib/ilr";
import { stockStatusLabel } from "@/lib/status";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import type { PumpOverRequest } from "@/lib/pumpOver";
import { creatorLabel } from "@/lib/creator";

export default function PumpOverListPage() {
  const [tick, setTick] = useState(0);
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(PERMISSIONS.pumpOverCreate);
  const canSubmit = hasPermission(PERMISSIONS.pumpOverSubmit);

  async function submit(id: string) {
    setMsg("");
    try {
      await api.post(`/orders/pump-over/${id}/submit`, {});
      setMsg("Submitted for approval");
      setTick((n) => n + 1);
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "Could not submit");
    }
  }

  const columns: EnterpriseColumn<PumpOverRequest>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        exportValue: (r) => r.documentNumber,
        cell: (r) => (
          <Link href={`/stock/pump-over/${r.id}`} className="font-medium text-brand-800 hover:underline">
            {r.documentNumber}
          </Link>
        ),
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
        defaultVisible: true,
        exportValue: (r) => r.customer?.name || r.customer?.code,
        cell: (r) => r.customer?.name || r.customer?.code || "—",
      },
      {
        id: "product",
        header: "Product",
        defaultVisible: true,
        exportValue: (r) => r.product?.code,
        cell: (r) => r.product?.code || "—",
      },
      {
        id: "depot",
        header: "Depot",
        defaultVisible: false,
        exportValue: (r) => r.depot?.code || r.depot?.name,
        cell: (r) => r.depot?.code || r.depot?.name || "—",
      },
      {
        id: "quantity",
        header: "Qty (m³)",
        sortKey: "quantity",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.quantity,
        cell: (r) => <span className="tabular-nums">{formatQty(r.quantity)}</span>,
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
              href={`/stock/pump-over/${r.id}`}
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
      tableId="orders-pump-over"
      title="Pump-over requests"
      subtitle="Pipeline delivery orders from TIPER to another terminal — approve here, post quantities on Pump-over reports"
      headerExtra={
        canCreate ? (
          <Link
            href="/stock/pump-over/new"
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2.5 text-sm font-semibold text-white"
          >
            New request
          </Link>
        ) : null
      }
      lead={msg ? <p className="mb-3 text-sm text-slate-600">{msg}</p> : null}
      pills={{
        options: DOC_STATUS_PILLS,
        of: (r) => r.status,
        guideTitle: "Request status",
        allTitle: "All requests",
      }}
      remote={{ path: "/orders/pump-over", dates: true, reloadKey: tick }}
      emptyMessage="No pump-over requests"
      columns={columns}
      searchPlaceholder="Document, customer, product"
      defaultOrderBy="orderDate"
      defaultSortDirection="DESC"
      exportName="pump_over_requests"
      inquiryTitle={(r) => r.documentNumber}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Stock · Pump-over · Inquiry"
            title={r.documentNumber}
            pills={<StatusPill tone="navy">{r.status}</StatusPill>}
          />
          <InquirySection title="Request">
            <ErpField label="Date" value={formatDate(r.orderDate)} />
            <ErpField label="Customer" value={partyName(r.customer)} />
            <ErpField label="Product" value={partyName(r.product)} />
            <ErpField label="Receiving depot" value={partyName(r.depot)} />
            <ErpField label="Quantity (m³)" value={formatQty(r.quantity)} />
            <ErpField label="Customer order" value={r.customerOrderNumber || "—"} />
          </InquirySection>
          {(r.vessels || []).length > 0 ? (
            <InquirySection title="Vessels">
              {(r.vessels || []).map((v, i) => (
                <ErpField
                  key={`${v.vessel?.id || i}-${v.stockStatus?.id || ""}-${v.vesselDate}`}
                  label={v.vessel?.code || v.vessel?.name || `Vessel ${i + 1}`}
                  value={`${formatDate(v.vesselDate)} · ${v.stockStatus ? stockStatusLabel(v.stockStatus) : "—"} · ${formatQty(v.quantity)} m³`}
                />
              ))}
            </InquirySection>
          ) : null}
          <Link
            href={`/stock/pump-over/${r.id}`}
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2 text-sm font-semibold text-white"
          >
            Open document
          </Link>
        </div>
      )}
    />
  );
}
