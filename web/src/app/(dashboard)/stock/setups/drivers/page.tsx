"use client";

import { useCallback, useState } from "react";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import {
  ErpField,
  InquiryHeader,
  StatusPill,
} from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import type { Driver } from "@/lib/master";
import { formatDate } from "@/lib/format";
import { DocumentFiles } from "@/components/DocumentFiles";
import { PartyWorkspace } from "../form";
import { DriverForm } from "./form";

type DriverPane = "general" | "licence";

const DRIVER_PANES: { id: DriverPane; label: string }[] = [
  { id: "general", label: "General" },
  { id: "licence", label: "Driving licence" },
];

export default function DriversPage() {
  const columns = useCallback(
    () =>
      [
        {
          id: "name",
          header: "Driver",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "licenseNumber",
          header: "License",
          sortKey: "licenseNumber",
          defaultVisible: true,
          cell: (r) => (
            <span className="font-mono text-xs text-slate-600">
              {r.licenseNumber || "—"}
            </span>
          ),
        },
        {
          id: "phone",
          header: "Phone",
          defaultVisible: true,
          cell: (r) => r.phone || "—",
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Driver>[],
    [],
  );

  return (
    <MasterCatalogue<Driver>
      tableId="master-drivers"
      title="Drivers"
      subtitle="Drivers assigned to a haulier and used on ILO truck lines"
      path="/master/drivers"
      orderBy="name"
      searchPlaceholder="Search name or license"
      createLabel="Add driver"
      emptyMessage="No drivers found"
      columns={columns}
      inquirySize="xl"
      inquiryTitle={(r) => r.name || "Driver"}
      inquiry={(r) => <DriverInquiry row={r} />}
      Form={DriverForm}
    />
  );
}

function DriverInquiry({ row }: { row: Driver }) {
  const [pane, setPane] = useState<DriverPane>("general");
  return (
    <div className="space-y-4">
      <InquiryHeader
        crumb="Stock · Setups · Drivers"
        title={row.name || "Driver"}
        subtitle={row.licenseNumber ? `Licence ${row.licenseNumber}` : undefined}
        pills={
          <StatusPill tone={row.isActive ? "navy" : "slate"}>
            {row.isActive ? "Active" : "Inactive"}
          </StatusPill>
        }
      />
      <PartyWorkspace panes={DRIVER_PANES} pane={pane} onPane={setPane}>
        {pane === "general" ? (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <ErpField label="Name" value={row.name} />
            <ErpField label="License" value={row.licenseNumber || "—"} />
            <ErpField label="Phone" value={row.phone || "—"} />
            <ErpField label="Email" value={row.email || "—"} />
            <ErpField label="License expires" value={formatDate(row.licenseExpires)} />
          </div>
        ) : (
          <DocumentFiles
            path={`/master/drivers/${row.id}`}
            draft={false}
            hint="Licence scans kept for history and renewal checks. Open Edit to upload or remove files."
          />
        )}
      </PartyWorkspace>
    </div>
  );
}
