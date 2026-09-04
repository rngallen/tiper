"use client";

import { useCallback, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import { Field, Input, ToggleField } from "@/components/ui";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import { searchMaster, type Destination, type District } from "@/lib/master";
import { CatalogueForm } from "../form";

export default function DistrictsPage() {
  const columns = useCallback(
    () =>
      [
        {
          id: "name",
          header: "District",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "destination",
          header: "Destination",
          defaultVisible: true,
          cell: (r) => r.destination?.name || "—",
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<District>[],
    [],
  );

  return (
    <MasterCatalogue<District>
      tableId="master-districts"
      title="Districts"
      subtitle="Delivery districts used on internal loading orders"
      path="/master/districts"
      orderBy="name"
      searchPlaceholder="Search district"
      createLabel="Add district"
      emptyMessage="No districts found"
      columns={columns}
      inquiryTitle={(r) => r.name || "District"}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Districts · Inquiry"
            title={r.name || "District"}
            pills={
              <StatusPill tone={r.isActive ? "navy" : "slate"}>
                {r.isActive ? "Active" : "Inactive"}
              </StatusPill>
            }
          />
          <InquirySection title="General">
            <ErpField label="Name" value={r.name} />
            <ErpField label="Destination" value={r.destination?.name || "—"} />
          </InquirySection>
        </div>
      )}
      Form={DistrictForm}
    />
  );
}

function DistrictForm({
  row,
  onClose,
  onSaved,
}: {
  row?: District;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(row?.name ?? "");
  const [destinationId, setDestinationId] = useState(row?.destination?.id ?? "");
  const [destinationLabel, setDestinationLabel] = useState(row?.destination?.name ?? "");
  const [isActive, setActive] = useState(row?.isActive ?? true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    if (!name.trim()) {
      setError("Name is required.");
      return;
    }
    if (!destinationId) {
      setError("Destination is required.");
      return;
    }
    setSaving(true);
    try {
      const body = { name: name.trim(), destinationId, isActive };
      if (row) await api.put(`/master/districts/${row.id}`, body);
      else await api.post("/master/districts", body);
      onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save district"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? "Edit district" : "Add district"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
    >
      <Field label="Destination">
        <AsyncSearchableSelect
          value={destinationId}
          selectedLabel={destinationLabel}
          placeholder="Search destination"
          emptyLabel="No matching destination"
          onChange={(value, option) => {
            setDestinationId(value);
            setDestinationLabel(option?.label || value);
          }}
          loadOptions={async (q) => {
            const rows = await searchMaster<Destination>("/master/destinations", q);
            return rows.map((d) => ({ value: d.id, label: d.name || "" }));
          }}
        />
      </Field>
      <Field label="Name">
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Ilala, Kinshasa"
          autoFocus
        />
      </Field>
      <ToggleField label="Active" checked={isActive} onChange={setActive} />
    </CatalogueForm>
  );
}
