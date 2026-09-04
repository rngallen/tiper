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
import type { Destination } from "@/lib/master";
import { CatalogueForm } from "../form";

export default function DestinationsPage() {
  const columns = useCallback(
    () =>
      [
        {
          id: "name",
          header: "Destination",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "isCountry",
          header: "Kind",
          defaultVisible: true,
          cell: (r) => (r.isCountry ? "Country" : "Region"),
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Destination>[],
    [],
  );

  return (
    <MasterCatalogue<Destination>
      tableId="master-destinations"
      title="Destinations"
      subtitle="Delivery countries and inland regions used on gantry lines"
      path="/master/destinations"
      orderBy="name"
      searchPlaceholder="Search destination"
      createLabel="Add destination"
      emptyMessage="No destinations found"
      columns={columns}
      inquiryTitle={(r) => r.name || "Destination"}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Destinations · Inquiry"
            title={r.name || "Destination"}
            pills={
              <StatusPill tone={r.isActive ? "navy" : "slate"}>
                {r.isActive ? "Active" : "Inactive"}
              </StatusPill>
            }
          />
          <InquirySection title="General">
            <ErpField label="Name" value={r.name} />
            <ErpField label="Kind" value={r.isCountry ? "Country" : "Region"} />
          </InquirySection>
        </div>
      )}
      Form={DestinationForm}
    />
  );
}

function DestinationForm({
  row,
  onClose,
  onSaved,
}: {
  row?: Destination;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(row?.name ?? "");
  const [isCountry, setCountry] = useState(row?.isCountry ?? false);
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
      const body = { name: name.trim(), isCountry, isActive };
      if (row) await api.put(`/master/destinations/${row.id}`, body);
      else await api.post("/master/destinations", body);
      onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save destination"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? "Edit destination" : "Add destination"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
    >
      <Field label="Name">
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Congo, Dar es Salaam"
          autoFocus
        />
      </Field>
      <ToggleField
        label="Country"
        description="On for a country (Congo, Rwanda). Off for a domestic region."
        checked={isCountry}
        onChange={setCountry}
      />
      <ToggleField label="Active" checked={isActive} onChange={setActive} />
    </CatalogueForm>
  );
}
