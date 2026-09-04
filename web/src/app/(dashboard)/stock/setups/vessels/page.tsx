"use client";

import { useCallback, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import { Field, Input, ToggleField } from "@/components/ui";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import type { Vessel } from "@/lib/master";
import { CatalogueForm } from "../form";

export default function VesselsPage() {
  const columns = useCallback(
    () =>
      [
        {
          id: "code",
          header: "Code",
          sortKey: "code",
          defaultVisible: true,
          cell: (r) => (
            <span className="font-mono text-xs text-slate-600">{r.code}</span>
          ),
        },
        {
          id: "name",
          header: "Vessel",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "imoNumber",
          header: "IMO",
          sortKey: "imoNumber",
          defaultVisible: true,
          cell: (r) => r.imoNumber || "—",
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Vessel>[],
    [],
  );

  return (
    <MasterCatalogue<Vessel>
      tableId="master-vessels"
      title="Vessels"
      subtitle="Ships that deliver product to the terminal"
      path="/master/vessels"
      orderBy="name"
      searchPlaceholder="Search name, code, IMO"
      createLabel="Add vessel"
      emptyMessage="No vessels found"
      columns={columns}
      inquiryTitle={(r) => r.name || r.code || "Vessel"}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Vessels · Inquiry"
            title={r.name || r.code || "Vessel"}
            pills={
              <StatusPill tone={r.isActive ? "navy" : "slate"}>
                {r.isActive ? "Active" : "Inactive"}
              </StatusPill>
            }
          />
          <InquirySection title="General">
            <ErpField label="Code" value={r.code} />
            <ErpField label="IMO number" value={r.imoNumber || "—"} />
          </InquirySection>
        </div>
      )}
      Form={VesselForm}
    />
  );
}

function VesselForm({
  row,
  onClose,
  onSaved,
}: {
  row?: Vessel;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(row?.name ?? "");
  const [imoNumber, setImo] = useState(row?.imoNumber ?? "");
  const [isActive, setActive] = useState(row?.isActive ?? true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    if (!name.trim()) {
      setError("Name is required.");
      return;
    }
    setSaving(true);
    try {
      const body = { name: name.trim(), imoNumber: imoNumber.trim(), isActive };
      if (row) await api.put(`/master/vessels/${row.id}`, body);
      else await api.post("/master/vessels", body);
      onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save vessel"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? "Edit vessel" : "Add vessel"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
    >
      <Field label="Name">
        <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      </Field>
      <Field label="IMO number">
        <Input value={imoNumber} onChange={(e) => setImo(e.target.value)} />
      </Field>
      <ToggleField label="Active" checked={isActive} onChange={setActive} />
    </CatalogueForm>
  );
}
