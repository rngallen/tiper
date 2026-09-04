"use client";

import { useMemo } from "react";
import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
} from "@/components/ApprovalReview";
import { formatDate } from "@/lib/format";
import { qty } from "@/lib/master";

type Balance = {
  customerCode: string;
  customerName: string;
  productCode: string;
  productName: string;
  vesselCode: string;
  vesselDate: string;
  statusName: string;
  isTransit?: boolean;
  final: string;
  provision: string;
  financialHold: string;
  reserved?: string;
  freeToOrder: string;
};

export default function BalancesPage() {
  const { data, loading, error } = useAsync(
    () => api.get<Balance[]>("/ic/balances"),
    [],
  );

  const columns: EnterpriseColumn<Balance>[] = useMemo(
    () => [
      {
        id: "customerName",
        header: "Customer",
        sortKey: "customerName",
        defaultVisible: true,
        exportValue: (r) => r.customerName || r.customerCode,
        cell: (r) => (
          <div>
            <div className="font-medium text-slate-800">{r.customerName || r.customerCode}</div>
            {r.customerName && r.customerCode ? (
              <div className="font-mono text-xs text-slate-500">{r.customerCode}</div>
            ) : null}
          </div>
        ),
      },
      {
        id: "productCode",
        header: "Product",
        sortKey: "productCode",
        defaultVisible: true,
        exportValue: (r) => r.productCode,
        cell: (r) => (
          <div>
            <div className="font-medium text-slate-800">{r.productCode}</div>
            {r.productName ? <div className="text-xs text-slate-500">{r.productName}</div> : null}
          </div>
        ),
      },
      {
        id: "vesselCode",
        header: "Vessel",
        sortKey: "vesselCode",
        defaultVisible: true,
        exportValue: (r) => r.vesselCode,
        cell: (r) => r.vesselCode || "—",
      },
      {
        id: "vesselDate",
        header: "Vessel date",
        sortKey: "vesselDate",
        defaultVisible: false,
        className: "text-right",
        exportValue: (r) => formatDate(r.vesselDate),
        cell: (r) => formatDate(r.vesselDate),
      },
      {
        id: "statusName",
        header: "Status",
        sortKey: "statusName",
        defaultVisible: true,
        exportValue: (r) => r.statusName,
        cell: (r) => r.statusName || "—",
      },
      {
        id: "final",
        header: "Final",
        sortKey: "final",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.final,
        cell: (r) => <span className="tabular-nums">{qty(r.final)}</span>,
      },
      {
        id: "provision",
        header: "Provision",
        sortKey: "provision",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.provision,
        cell: (r) => <span className="tabular-nums">{qty(r.provision)}</span>,
      },
      {
        id: "financialHold",
        header: "Hold",
        sortKey: "financialHold",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.financialHold,
        cell: (r) => <span className="tabular-nums">{qty(r.financialHold)}</span>,
      },
      {
        id: "reserved",
        header: "Reserved",
        sortKey: "reserved",
        defaultVisible: false,
        className: "text-right",
        exportValue: (r) => r.reserved,
        cell: (r) => <span className="tabular-nums">{qty(r.reserved)}</span>,
      },
      {
        id: "freeToOrder",
        header: "Free",
        sortKey: "freeToOrder",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.freeToOrder,
        cell: (r) => <span className="tabular-nums font-medium">{qty(r.freeToOrder)}</span>,
      },
    ],
    [],
  );

  return (
    <EnterpriseTable
        tableId="ic-balances"
        title="Stock balances"
        subtitle="On-hand by customer, product, vessel, and status"
        rows={data}
        loading={loading}
        error={error}
        emptyMessage="No balances"
        columns={columns}
        searchPlaceholder="Customer, product, vessel, status"
        rowSearch={(r) =>
          `${r.customerCode} ${r.customerName} ${r.productCode} ${r.productName} ${r.vesselCode} ${r.statusName}`
        }
        defaultOrderBy="customerName"
        exportName="stock_balances"
        inquiryTitle={(r) => r.customerName || r.customerCode}
        inquiry={(r) => (
          <div className="space-y-4">
            <InquiryHeader
              crumb="Stock · Balances · Inquiry"
              title={r.customerName || r.customerCode}
              subtitle={r.productName || r.productCode}
            />
            <InquirySection title="Parcel">
              <ErpField label="Customer" value={r.customerName || r.customerCode} />
              <ErpField label="Code" value={r.customerCode} />
              <ErpField label="Product" value={`${r.productCode}${r.productName ? ` — ${r.productName}` : ""}`} />
              <ErpField label="Vessel" value={r.vesselCode || "—"} />
              <ErpField label="Vessel date" value={formatDate(r.vesselDate)} />
              <ErpField label="Status" value={r.statusName || "—"} />
            </InquirySection>
            <InquirySection title="Quantities (m³)">
              <ErpField label="Final" value={qty(r.final)} />
              <ErpField label="Provision" value={qty(r.provision)} />
              <ErpField label="Financial hold" value={qty(r.financialHold)} />
              <ErpField label="Reserved" value={qty(r.reserved)} />
              <ErpField label="Free to order" value={qty(r.freeToOrder)} />
            </InquirySection>
          </div>
        )}
    />
  );
}
