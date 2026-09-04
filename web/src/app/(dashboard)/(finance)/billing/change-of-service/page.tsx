"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { api } from "@/lib/api";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import { ErpField, InquiryHeader, InquirySection, StatusPill } from "@/components/ApprovalReview";
import { Button, StatusBadge } from "@/components/ui";
import { DOC_STATUS_PILLS } from "@/components/StatusPills";
import { formatDate } from "@/lib/format";
import { partyName } from "@/lib/ilr";
import { creatorLabel } from "@/lib/creator";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";

type Ref = { id: string; code?: string; name?: string };
type Cos = {
  id: string;
  documentNumber: string;
  effectiveDate: string;
  vesselDate?: string;
  status: string;
  fromCollection?: string;
  toCollection?: string;
  customer?: Ref;
  vessel?: Ref;
  product?: Ref;
  createdBy?: { name?: string; email?: string };
};

export default function ChangeOfServiceListPage() {
  const [tick, setTick] = useState(0);
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(PERMISSIONS.changeOfServiceCreate);
  const canSubmit = hasPermission(PERMISSIONS.changeOfServiceSubmit);

  async function submit(id: string) {
    setMsg("");
    try {
      await api.post(`/billing/change-of-service/${id}/submit`, {});
      setMsg("Submitted for approval");
      setTick((n) => n + 1);
    } catch (e) {
      setMsg(e instanceof Error ? e.message : "Could not submit");
    }
  }

  const columns: EnterpriseColumn<Cos>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        exportValue: (r) => r.documentNumber,
        cell: (r) => (
          <Link href={`/billing/change-of-service/${r.id}`} className="font-medium text-brand-800 hover:underline">
            {r.documentNumber}
          </Link>
        ),
      },
      {
        id: "effectiveDate",
        header: "Effective",
        sortKey: "effectiveDate",
        defaultVisible: true,
        exportValue: (r) => formatDate(r.effectiveDate),
        cell: (r) => formatDate(r.effectiveDate),
      },
      {
        id: "customer",
        header: "Customer",
        sortKey: "customer",
        defaultVisible: true,
        exportValue: (r) => r.customer?.name || r.customer?.code,
        cell: (r) => partyName(r.customer),
      },
      {
        id: "vessel",
        header: "Vessel",
        sortKey: "vessel",
        defaultVisible: true,
        exportValue: (r) => r.vessel?.name || r.vessel?.code,
        cell: (r) => partyName(r.vessel),
      },
      {
        id: "vesselDate",
        header: "Vessel date",
        sortKey: "vesselDate",
        defaultVisible: true,
        exportValue: (r) => formatDate(r.vesselDate),
        cell: (r) => formatDate(r.vesselDate),
      },
      {
        id: "delivery",
        header: "Delivery",
        defaultVisible: true,
        exportValue: (r) => collectionLabel(r),
        cell: (r) => collectionLabel(r),
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
              href={`/billing/change-of-service/${r.id}`}
              className="text-sm font-semibold text-brand-800"
              onClick={(e) => e.stopPropagation()}
            >
              Open
            </Link>
            {canSubmit && (r.status === "draft" || r.status === "returned") ? (
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
      tableId="billing-change-of-service"
      title="Change of service"
      subtitle="Switch delivery method on one customer vessel parcel — fees stay, FSF billing follows the new method"
      headerExtra={
        canCreate ? (
          <Link
            href="/billing/change-of-service/new"
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2.5 text-sm font-semibold text-white"
          >
            New change of service
          </Link>
        ) : null
      }
      lead={msg ? <p className="mb-3 text-sm text-slate-600">{msg}</p> : null}
      pills={{
        options: DOC_STATUS_PILLS,
        of: (r) => r.status,
        guideTitle: "Change of service status",
        allTitle: "All documents",
      }}
      remote={{ path: "/billing/change-of-service", dates: true, reloadKey: tick }}
      emptyMessage="No change of service documents"
      columns={columns}
      searchPlaceholder="Document, customer, vessel"
      defaultOrderBy="effectiveDate"
      defaultSortDirection="DESC"
      dateOf={(r) => r.effectiveDate}
      exportName="change_of_service"
      inquiryTitle={(r) => r.documentNumber}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Billing · Transactions · Change of service · Inquiry"
            title={r.documentNumber}
            pills={<StatusPill tone="navy">{r.status}</StatusPill>}
          />
          <InquirySection title="Parcel">
            <ErpField label="Effective" value={formatDate(r.effectiveDate)} />
            <ErpField label="Customer" value={partyName(r.customer)} />
            <ErpField label="Vessel" value={partyName(r.vessel)} />
            <ErpField label="Vessel date" value={formatDate(r.vesselDate)} />
            <ErpField label="Product" value={partyName(r.product)} />
            <ErpField label="Delivery" value={collectionLabel(r)} />
          </InquirySection>
          <Link
            href={`/billing/change-of-service/${r.id}`}
            className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2 text-sm font-semibold text-white"
          >
            Open document
          </Link>
        </div>
      )}
    />
  );
}

function collectionLabel(r: Cos) {
  return `${r.fromCollection || "—"} → ${r.toCollection || "—"}`;
}
