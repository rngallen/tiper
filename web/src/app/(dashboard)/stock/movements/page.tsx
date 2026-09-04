"use client";

import { useMemo } from "react";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import { StatusBadge } from "@/components/ui";
import { formatDate, formatQty } from "@/lib/format";

type Movement = {
  id: string;
  transactionDate: string;
  transactionType: string;
  quantity: string;
  cubicMeter?: string;
  metricTonne?: string;
  isProvision: boolean;
  financialHold?: boolean;
  referenceType: string;
  vesselDate?: string;
};

export default function MovementsPage() {
  const columns: EnterpriseColumn<Movement>[] = useMemo(
    () => [
      {
        id: "transactionDate",
        header: "Date",
        sortKey: "transactionDate",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => formatDate(r.transactionDate),
        cell: (r) => formatDate(r.transactionDate),
      },
      {
        id: "transactionType",
        header: "Type",
        sortKey: "transactionType",
        defaultVisible: true,
        exportValue: (r) => r.transactionType,
        cell: (r) => <StatusBadge status={r.transactionType} />,
      },
      {
        id: "quantity",
        header: "Quantity",
        sortKey: "quantity",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.quantity,
        cell: (r) => <span className="tabular-nums">{formatQty(r.quantity, "m3")}</span>,
      },
      {
        id: "isProvision",
        header: "Provision",
        sortKey: "isProvision",
        defaultVisible: true,
        exportValue: (r) => (r.isProvision ? "Yes" : "No"),
        cell: (r) => (r.isProvision ? "Yes" : "No"),
      },
      {
        id: "financialHold",
        header: "Hold",
        sortKey: "financialHold",
        defaultVisible: false,
        exportValue: (r) => (r.financialHold ? "Yes" : "No"),
        cell: (r) => (r.financialHold ? "Yes" : "No"),
      },
      {
        id: "referenceType",
        header: "Reference",
        sortKey: "referenceType",
        defaultVisible: true,
        exportValue: (r) => r.referenceType,
        cell: (r) => r.referenceType || "—",
      },
    ],
    [],
  );

  return (
    <EnterpriseTable
        tableId="ic-movements"
        title="Stock movements"
        subtitle="Ledger events that built the current book stock"
        pills={{
          options: [
            { id: "final", label: "Final", meaning: "Posted final quantity." },
            { id: "provision", label: "Provision", meaning: "Provisional quantity pending final." },
          ],
          of: (r) => (r.isProvision ? "provision" : "final"),
          guideTitle: "Movement type",
          allTitle: "All movements",
        }}
        remote={{
          path: "/ic/movements",
          dates: true,
          laneQuery: (lane) =>
            lane === "provision"
              ? { provision: true }
              : lane === "final"
                ? { provision: false }
                : {},
        }}
        emptyMessage="No movements"
        columns={columns}
        searchPlaceholder="Type or reference"
        rowSearch={(r) => `${r.transactionType} ${r.referenceType} ${r.quantity}`}
        defaultOrderBy="transactionDate"
        defaultSortDirection="DESC"
        dateOf={(r) => r.transactionDate}
        exportName="stock_movements"
        inquiryTitle={(r) => r.transactionType || "Movement"}
        inquiry={(r) => (
          <div className="space-y-4">
            <InquiryHeader
              crumb="Stock · Movements · Inquiry"
              title={r.transactionType || "Movement"}
              pills={
                <StatusPill tone={r.isProvision ? "amber" : "navy"}>
                  {r.isProvision ? "Provision" : "Final"}
                </StatusPill>
              }
            />
            <InquirySection title="Ledger">
              <ErpField label="Date" value={formatDate(r.transactionDate)} />
              <ErpField label="Type" value={r.transactionType} />
              <ErpField label="Quantity (m³)" value={formatQty(r.quantity, "m3")} />
              <ErpField label="Cubic metres" value={formatQty(r.cubicMeter, "m3")} />
              <ErpField label="Metric tonnes" value={formatQty(r.metricTonne, "mt")} />
              <ErpField label="Vessel date" value={formatDate(r.vesselDate)} />
              <ErpField label="Reference" value={r.referenceType || "—"} />
              <ErpField label="Financial hold" value={r.financialHold ? "Yes" : "No"} />
            </InquirySection>
          </div>
        )}
    />
  );
}
