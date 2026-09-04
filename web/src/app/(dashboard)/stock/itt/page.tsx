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
import { creatorLabel } from "@/lib/creator";

import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";

type Ref = { id: string; code?: string; name?: string };
type ITT = {
  id: string;
  documentNumber: string;
  transferDate: string;
  quantity: string;
  status: string;
  fromCustomer?: Ref;
  toCustomer?: Ref;
  product?: Ref;
  vessel?: Ref;
  createdBy?: { name?: string; email?: string };
};

export default function ITTPage() {
  const [tick, setTick] = useState(0);
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(PERMISSIONS.ittCreate);
  const canSubmit = hasPermission(PERMISSIONS.ittSubmit);

  async function submit(id: string) {
    setMsg("");
    try {
      await api.post(`/ic/itt/${id}/submit`, {});
      setMsg("Submitted for approval");
      setTick((n) => n + 1);
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "Could not submit");
    }
  }

  const columns: EnterpriseColumn<ITT>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        exportValue: (r) => r.documentNumber,
        cell: (r) => (
          <Link href={`/stock/itt/${r.id}`} className="font-medium text-brand-800 hover:underline">
            {r.documentNumber}
          </Link>
        ),
      },
      {
        id: "transferDate",
        header: "Date",
        sortKey: "transferDate",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => formatDate(r.transferDate),
        cell: (r) => formatDate(r.transferDate),
      },
      {
        id: "from",
        header: "From",
        defaultVisible: true,
        exportValue: (r) => r.fromCustomer?.name || r.fromCustomer?.code,
        cell: (r) => r.fromCustomer?.name || r.fromCustomer?.code || "—",
      },
      {
        id: "to",
        header: "To",
        defaultVisible: true,
        exportValue: (r) => r.toCustomer?.name || r.toCustomer?.code,
        cell: (r) => r.toCustomer?.name || r.toCustomer?.code || "—",
      },
      {
        id: "product",
        header: "Product",
        defaultVisible: true,
        exportValue: (r) => r.product?.code,
        cell: (r) => r.product?.code || "—",
      },
      {
        id: "quantity",
        header: "Quantity",
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
              href={`/stock/itt/${r.id}`}
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
      tableId="ic-itt"
      title="In-tank transfers"
		subtitle="Ownership changes between customers on the same parcel. Stock moves on approval: before that date the sender still holds the full receipt volume."
      headerExtra={
        canCreate ? (
          <Link
            href="/stock/itt/new"
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2.5 text-sm font-semibold text-white"
          >
            New ITT
          </Link>
        ) : null
      }
      lead={msg ? <p className="mb-3 text-sm text-slate-600">{msg}</p> : null}
      pills={{
        options: DOC_STATUS_PILLS,
        of: (r) => r.status,
        guideTitle: "ITT status",
        allTitle: "All transfers",
      }}
      remote={{ path: "/ic/itt", dates: true, reloadKey: tick }}
      emptyMessage="No in-tank transfers"
      columns={columns}
      searchPlaceholder="Document, customer, product"
      defaultOrderBy="transferDate"
      defaultSortDirection="DESC"
      exportName="in_tank_transfers"
      inquiryTitle={(r) => r.documentNumber}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Stock · Terminal · ITT · Inquiry"
            title={r.documentNumber}
            pills={<StatusPill tone="navy">{r.status}</StatusPill>}
          />
          <InquirySection title="Transfer">
            <ErpField label="Date" value={formatDate(r.transferDate)} />
            <ErpField label="From" value={partyName(r.fromCustomer)} />
            <ErpField label="To" value={partyName(r.toCustomer)} />
            <ErpField label="Product" value={partyName(r.product)} />
            <ErpField label="Quantity (m³)" value={formatQty(r.quantity)} />
            <ErpField label="Vessel" value={partyName(r.vessel)} />
          </InquirySection>
          <Link
            href={`/stock/itt/${r.id}`}
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2 text-sm font-semibold text-white"
          >
            Open document
          </Link>
        </div>
      )}
    />
  );
}
