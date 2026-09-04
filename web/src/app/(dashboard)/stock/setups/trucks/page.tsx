"use client";

import { useCallback } from "react";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import type { Truck } from "@/lib/master";
import { TruckForm } from "./form";

const vehicleLabel: Record<string, string> = {
  straight: "Straight",
  semi: "Semi-trailer",
  pulling: "Pulling",
  pending: "Not configured",
};

export default function TrucksPage() {
  const columns = useCallback(
    () =>
      [
        {
          id: "plateNumber",
          header: "Plate",
          sortKey: "plateNumber",
          defaultVisible: true,
          cell: (r) => (
            <div className="font-medium text-slate-800">{r.plateNumber}</div>
          ),
        },
        {
          id: "trailer",
          header: "Trailer",
          sortKey: "trailer",
          defaultVisible: true,
          cell: (r) => r.trailer || "—",
        },
        {
          id: "vehicleType",
          header: "Type",
          sortKey: "vehicleType",
          defaultVisible: true,
          cell: (r) => vehicleLabel[r.vehicleType || ""] || r.vehicleType || "Not configured",
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Truck>[],
    [],
  );

  return (
    <MasterCatalogue<Truck>
      tableId="master-trucks"
      title="Trucks"
      subtitle="Horses and trailers used at the one-stop gantry"
      path="/master/trucks"
      orderBy="plateNumber"
      searchPlaceholder="Search plate"
      createLabel="Add truck"
      emptyMessage="No trucks found"
      columns={columns}
      inquiryTitle={(r) => r.plateNumber}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Trucks · Inquiry"
            title={r.plateNumber}
            pills={
              <StatusPill tone={r.isActive ? "navy" : "slate"}>
                {r.isActive ? "Active" : "Inactive"}
              </StatusPill>
            }
          />
          <InquirySection title="General">
            <ErpField label="Horse / plate" value={r.plateNumber} />
            <ErpField label="Trailer" value={r.trailer || "—"} />
            <ErpField label="Second trailer" value={r.trailerTwo || "—"} />
            <ErpField label="Type" value={vehicleLabel[r.vehicleType || ""] || r.vehicleType || "Not configured"} />
            <ErpField label="Loading" value={r.loadingType === "bottom" ? "Bottom" : "Top"} />
            <ErpField label="LNG / CNG" value={r.lngCng ? "Yes" : "No"} />
            <ErpField label="MPLW (kg)" value={r.mplw || "—"} />
            <ErpField label="GCWR (kg)" value={r.gcwr || "—"} />
            <ErpField label="Tare (kg)" value={r.tareWeight || "—"} />
          </InquirySection>
        </div>
      )}
      Form={TruckForm}
      configureHref={(r) => `/stock/setups/trucks/${r.id}`}
      configureLabel="Configure"
    />
  );
}
