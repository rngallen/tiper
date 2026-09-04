"use client";

import { useCallback, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { Field, Input, ToggleField } from "@/components/ui";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import { searchMaster, type Customer, type Depot } from "@/lib/master";
import { CatalogueForm } from "../form";

export default function DepotsPage() {
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
          header: "Depot",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "customer",
          header: "KOJ customer",
          defaultVisible: true,
          cell: (r) => r.customer?.name || "—",
        },
        {
          id: "isInternal",
          header: "TIPER",
          defaultVisible: true,
          cell: (r) => (r.isInternal ? "Internal" : "External"),
        },
        {
          id: "ewuraLicense",
          header: "EWURA license",
          defaultVisible: false,
          cell: (r) => r.ewuraLicense || "—",
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Depot>[],
    [],
  );

  return (
    <MasterCatalogue<Depot>
      tableId="master-depots"
      title="Depots"
      subtitle="TIPER and pump-over destination depots"
      path="/master/depots"
      orderBy="name"
      searchPlaceholder="Search name or code"
      createLabel="Add depot"
      emptyMessage="No depots found"
      columns={columns}
      inquiryTitle={(r) => r.name || r.code || "Depot"}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Depots · Inquiry"
            title={r.name || r.code || "Depot"}
            pills={
              <StatusPill tone={r.isActive ? "navy" : "slate"}>
                {r.isActive ? "Active" : "Inactive"}
              </StatusPill>
            }
          />
          <InquirySection title="General">
            <ErpField label="Code" value={r.code} />
            <ErpField label="Customer (KOJ billing only)" value={r.customer?.name || "—"} />
            <ErpField label="Type" value={r.isInternal ? "TIPER (internal)" : "External"} />
            <ErpField label="EWURA license" value={r.ewuraLicense || "—"} />
          </InquirySection>
        </div>
      )}
      Form={DepotForm}
    />
  );
}

function DepotForm({
  row,
  onClose,
  onSaved,
}: {
  row?: Depot;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [code, setCode] = useState(row?.code ?? "");
  const [name, setName] = useState(row?.name ?? "");
  const [ewuraLicense, setLicense] = useState(row?.ewuraLicense ?? "");
  const [isInternal, setInternal] = useState(row?.isInternal ?? false);
  const [customerId, setCustomer] = useState(row?.customer?.id ?? "");
  const [customerLabel, setCustomerLabel] = useState(
    row?.customer ? `${row.customer.code ?? ""} — ${row.customer.name ?? ""}`.trim() : "",
  );
  const [isActive, setActive] = useState(row?.isActive ?? true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    if (!name.trim()) {
      setError("Name is required.");
      return;
    }
    if (!customerId) {
      setError("Select a customer with a KOJ fee billing account.");
      return;
    }
    setSaving(true);
    try {
      const body = {
        code: code.trim(),
        name: name.trim(),
        ewuraLicense: ewuraLicense.trim(),
        isInternal,
        customerId,
        isActive,
      };
      if (row) await api.put(`/master/depots/${row.id}`, body);
      else await api.post("/master/depots", body);
      onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save depot"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? "Edit depot" : "Add depot"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
    >
      <Field label="Code">
        <Input
          value={code}
          onChange={(e) => setCode(e.target.value)}
          disabled={Boolean(row)}
          autoFocus={!row}
          placeholder={row ? undefined : "Assigned on save if left blank"}
        />
      </Field>
      <Field label="Name">
        <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus={Boolean(row)} />
      </Field>
      <Field
        label="Customer (KOJ billing only)"
        hint="KOJ facility fee for stock at this depot is billed to this customer. The customer must have a KOJ Sage mapping."
      >
        <AsyncSearchableSelect
          value={customerId}
          selectedLabel={customerLabel}
          placeholder="Search customer with a KOJ billing account"
          emptyLabel="No matching KOJ-billed customer"
          onChange={(value, option) => {
            setCustomer(value);
            setCustomerLabel(option?.label || value);
          }}
          loadOptions={async (q) => {
            const rows = await searchMaster<Customer>("/master/customers", q, { feeCode: "KOJ" });
            return rows.map((c) => ({
              value: c.id,
              label: `${c.code} — ${c.name}`,
              keywords: c.tinNumber,
            }));
          }}
        />
      </Field>
      <Field label="EWURA license">
        <Input value={ewuraLicense} onChange={(e) => setLicense(e.target.value)} />
      </Field>
      <ToggleField
        label="TIPER (internal)"
        description="Mark the Kigamboni depot. Leave off for pump-over destinations."
        checked={isInternal}
        onChange={setInternal}
      />
      <ToggleField label="Active" checked={isActive} onChange={setActive} />
    </CatalogueForm>
  );
}
