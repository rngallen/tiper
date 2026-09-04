"use client";

import { useMemo, useState } from "react";
import { api } from "@/lib/api";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import { QtyInput } from "@/components/QtyInput";
import { Button } from "@/components/ui";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { qty } from "@/lib/master";

type Party = { id: string; code?: string; name?: string };
type Line = {
  id: string;
  documentNumber: string;
  truckPlate: string;
  driverName: string;
  destination: string;
  requestedQty: string;
  loadedQty: string;
  status: string;
  transporterName?: string;
  district?: string;
  request?: {
    documentNumber?: string;
    customer?: Party;
    product?: Party;
  };
};

export default function DeliveriesPage() {
  const [tick, setTick] = useState(0);
  const [qtyById, setQty] = useState<Record<string, string>>({});
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canComplete = hasPermission(PERMISSIONS.ilrComplete);

  async function complete(id: string, requested: string) {
    setMsg("");
    try {
      await api.post(`/orders/loading-lines/${id}/complete`, {
        quantity: qtyById[id] || requested,
      });
      setMsg("Loading posted to the stock ledger");
      setTick((n) => n + 1);
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "Could not complete loading");
    }
  }

  const columns: EnterpriseColumn<Line>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "ILO",
        sortKey: "documentNumber",
        defaultVisible: true,
        exportValue: (r) => r.documentNumber,
        cell: (r) => <div className="font-medium text-slate-800">{r.documentNumber}</div>,
      },
      {
        id: "glr",
        header: "ILR",
        defaultVisible: true,
        exportValue: (r) => r.request?.documentNumber,
        cell: (r) => r.request?.documentNumber || "—",
      },
      {
        id: "customer",
        header: "Customer",
        defaultVisible: true,
        exportValue: (r) => r.request?.customer?.code,
        cell: (r) => r.request?.customer?.code || "—",
      },
      {
        id: "product",
        header: "Product",
        defaultVisible: true,
        exportValue: (r) => r.request?.product?.code,
        cell: (r) => r.request?.product?.code || "—",
      },
      {
        id: "truckPlate",
        header: "Truck",
        sortKey: "truckPlate",
        defaultVisible: true,
        exportValue: (r) => r.truckPlate,
        cell: (r) => r.truckPlate || "—",
      },
      {
        id: "destination",
        header: "Destination",
        defaultVisible: true,
        exportValue: (r) => r.destination,
        cell: (r) => r.destination || "—",
      },
      {
        id: "requestedQty",
        header: "Requested",
        sortKey: "requestedQty",
        defaultVisible: true,
        className: "text-right",
        exportValue: (r) => r.requestedQty,
        cell: (r) => <span className="tabular-nums">{qty(r.requestedQty)}</span>,
      },
      {
        id: "loaded",
        header: "Loaded",
        className: "text-right",
        defaultVisible: true,
        exportValue: (r) => qtyById[r.id] ?? r.requestedQty,
        cell: (r) => (
          <div className="w-28">
            <QtyInput
              value={qtyById[r.id] ?? r.requestedQty}
              onChange={(value) => setQty((prev) => ({ ...prev, [r.id]: value }))}
            />
          </div>
        ),
      },
      {
        id: "_actions",
        fixed: true,
        header: "",
        className: "text-right",
        cell: (r) =>
          canComplete ? (
          <Button
            className="!px-3 !py-1"
            onClick={(e) => {
              e.stopPropagation();
              void complete(r.id, r.requestedQty);
            }}
          >
            Complete
          </Button>
          ) : null,
      },
    ],
    [qtyById, canComplete],
  );

  return (
    <EnterpriseTable
        tableId="orders-deliveries"
        title="Gantry completion"
        subtitle="Open truck lines after a loading request is approved. Post actual loaded quantity here until ALMA is connected."
        lead={msg ? <p className="mb-3 text-sm text-slate-600">{msg}</p> : null}
        remote={{ path: "/orders/loading-lines", extraQuery: { status: "open" }, dates: true, reloadKey: tick }}
        emptyMessage="No open truck lines. Approve a gantry loading request first."
        columns={columns}
        searchPlaceholder="ILO, truck, customer"
        defaultOrderBy="documentNumber"
        exportName="gantry_completion"
        inquiryTitle={(r) => r.documentNumber}
        inquiry={(r) => (
          <div className="space-y-4">
            <InquiryHeader
              crumb="Stock · Completion · Inquiry"
              title={r.documentNumber}
              pills={<StatusPill tone="navy">{r.status}</StatusPill>}
            />
            <InquirySection title="Line">
              <ErpField label="ILR" value={r.request?.documentNumber || "—"} />
              <ErpField label="Customer" value={r.request?.customer?.code || "—"} />
              <ErpField label="Product" value={r.request?.product?.code || "—"} />
              <ErpField label="Truck" value={r.truckPlate || "—"} />
              <ErpField label="Driver" value={r.driverName || "—"} />
              <ErpField label="Hauler" value={r.transporterName || "—"} />
              <ErpField label="Destination" value={r.destination || "—"} />
              <ErpField label="District" value={r.district || "—"} />
              <ErpField label="Requested (m³)" value={qty(r.requestedQty)} />
              <ErpField label="Loaded (m³)" value={qty(r.loadedQty)} />
            </InquirySection>
          </div>
        )}
    />
  );
}
