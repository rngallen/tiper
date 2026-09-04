"use client";

import { useCallback, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { Field, Input, ToggleField } from "@/components/ui";
import { ErpField, InquiryHeader, StatusPill } from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import type { Country, Supplier } from "@/lib/master";
import {
  CatalogueForm,
  PartyWorkspace,
  RecordPlaceholder,
} from "../form";
import { BillingAccountsPanel } from "../BillingAccounts";
import { PartyDocuments } from "../PartyDocuments";

type RelatedPane = "address" | "billing" | "documents";
type FormPane = "general" | RelatedPane;

const RELATED_PANES: { id: RelatedPane; label: string }[] = [
  { id: "address", label: "Address" },
  { id: "billing", label: "Billing" },
  { id: "documents", label: "Documents" },
];

const FORM_PANES: { id: FormPane; label: string }[] = [
  { id: "general", label: "General" },
  ...RELATED_PANES,
];

export default function SuppliersPage() {
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
          header: "Supplier",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "email",
          header: "Email",
          sortKey: "email",
          defaultVisible: true,
          cell: (r) => r.email || "—",
        },
        {
          id: "phone",
          header: "Phone",
          defaultVisible: true,
          cell: (r) => r.phone || "—",
        },
        {
          id: "tinNumber",
          header: "TIN",
          defaultVisible: false,
          cell: (r) => r.tinNumber || "—",
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Supplier>[],
    [],
  );

  return (
    <MasterCatalogue<Supplier>
      tableId="master-suppliers"
      title="Suppliers"
      subtitle="Vendors billed for SRT first-cycle storage"
      path="/master/suppliers"
      orderBy="name"
      searchPlaceholder="Search name, code, email, phone"
      createLabel="Add supplier"
      emptyMessage="No suppliers found"
      columns={columns}
      inquiryTitle={(r) => r.name || r.code || "Supplier"}
      inquirySize="xl"
      inquiry={(r) => <SupplierInquiry row={r} />}
      Form={SupplierForm}
    />
  );
}

function SupplierInquiry({ row }: { row: Supplier }) {
  const [pane, setPane] = useState<RelatedPane>("billing");
  return (
    <div className="space-y-4">
      <InquiryHeader
        crumb="Stock · Setups · Suppliers"
        title={row.name || row.code || "Supplier"}
        subtitle={row.code ? `Code ${row.code}` : undefined}
        pills={
          <StatusPill tone={row.isActive ? "navy" : "slate"}>
            {row.isActive ? "Active" : "Inactive"}
          </StatusPill>
        }
      >
        <ErpField label="Email" value={row.email || "—"} boxed={false} />
        <ErpField label="Phone" value={row.phone || "—"} boxed={false} />
        <ErpField label="TIN" value={row.tinNumber || "—"} boxed={false} />
      </InquiryHeader>
      <PartyWorkspace
        panes={RELATED_PANES}
        pane={pane}
        onPane={setPane}
        banner={row.name || row.code || "Supplier"}
        hint={
          pane === "billing"
            ? "Sage AR mappings used when this supplier is billed."
            : pane === "documents"
              ? "TIN certificates, contracts, and registration files for this vendor."
              : undefined
        }
      >
        {pane === "address" ? (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <ErpField label="Contact" value={row.contactPerson || "—"} />
            <ErpField label="Mobile" value={row.mobile || "—"} />
            <ErpField label="Country" value={row.country?.name || row.countryCode || "—"} />
            <ErpField label="Address" value={row.address || "—"} />
            <ErpField label="Address 2" value={row.address2 || "—"} />
          </div>
        ) : pane === "billing" ? (
          <BillingAccountsPanel
            partyKind="supplier"
            partyId={row.id}
            readOnly
            showHelp={false}
          />
        ) : (
          <PartyDocuments path={`/master/suppliers/${row.id}`} readOnly />
        )}
      </PartyWorkspace>
    </div>
  );
}

function SupplierForm({
  row,
  onClose,
  onSaved,
}: {
  row?: Supplier;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [pane, setPane] = useState<FormPane>("general");
  const [name, setName] = useState(row?.name ?? "");
  const [email, setEmail] = useState(row?.email ?? "");
  const [phone, setPhone] = useState(row?.phone ?? "");
  const [mobile, setMobile] = useState(row?.mobile ?? "");
  const [contactPerson, setContact] = useState(row?.contactPerson ?? "");
  const [tinNumber, setTin] = useState(row?.tinNumber ?? "");
  const [countryCode, setCountry] = useState(row?.countryCode ?? "");
  const [countryLabel, setCountryLabel] = useState(row?.country?.name ?? row?.countryCode ?? "");
  const [address, setAddress] = useState(row?.address ?? "");
  const [address2, setAddress2] = useState(row?.address2 ?? "");
  const [isActive, setActive] = useState(row?.isActive ?? true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [savedId, setSavedId] = useState(row?.id ?? "");
  const isCreate = !row;

  async function save() {
    setError("");
    if (!name.trim()) {
      setError("Name is required.");
      setPane("general");
      return;
    }
    setSaving(true);
    try {
      const body = {
        name: name.trim(),
        email: email.trim(),
        phone: phone.trim(),
        mobile: mobile.trim(),
        contactPerson: contactPerson.trim(),
        tinNumber: tinNumber.trim(),
        countryCode,
        address: address.trim(),
        address2: address2.trim(),
        isActive,
      };
      if (savedId) {
        await api.put(`/master/suppliers/${savedId}`, body);
        onSaved();
        return;
      }
      const created = await api.post<Supplier>("/master/suppliers", body);
      setSavedId(created.id);
      setPane("billing");
    } catch (err) {
      setError(formatApiError(err, "Failed to save supplier"));
    } finally {
      setSaving(false);
    }
  }

  function dismiss() {
    if (savedId && isCreate) onSaved();
    else onClose();
  }

  const hint =
    pane === "billing"
      ? savedId
        ? "One Sage account per fee and currency. The same account cannot be used by another party."
        : "Create the supplier first, then map Sage AR accounts."
      : pane === "documents"
        ? savedId
          ? "TIN certificates, contracts, and registration files for this vendor."
          : "Create the supplier first, then attach documents."
        : undefined;

  return (
    <CatalogueForm
      title={row ? "Supplier" : "Add supplier"}
      error={error}
      saving={saving}
      onClose={dismiss}
      onSave={() => void save()}
      saveLabel={savedId && isCreate ? "Done" : row ? "Save" : "Create"}
      size="xl"
    >
      <PartyWorkspace
        panes={FORM_PANES}
        pane={pane}
        onPane={setPane}
        banner={name.trim() || row?.name || row?.code || (savedId ? "Supplier" : undefined)}
        hint={hint}
      >
        {pane === "general" ? (
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <Field label="Name" required>
                <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
              </Field>
            </div>
            <Field label="Contact person">
              <Input value={contactPerson} onChange={(e) => setContact(e.target.value)} />
            </Field>
            <Field label="TIN">
              <Input value={tinNumber} onChange={(e) => setTin(e.target.value)} />
            </Field>
            <Field label="Email">
              <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </Field>
            <Field label="Phone">
              <Input value={phone} onChange={(e) => setPhone(e.target.value)} />
            </Field>
            <Field label="Mobile">
              <Input value={mobile} onChange={(e) => setMobile(e.target.value)} />
            </Field>
            <div className="flex items-end">
              <ToggleField
                compact
                label="Active"
                description="Inactive suppliers are kept for history but hidden from new billing"
                checked={isActive}
                onChange={setActive}
              />
            </div>
          </div>
        ) : pane === "address" ? (
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <Field label="Country">
                <AsyncSearchableSelect
                  value={countryCode}
                  selectedLabel={countryLabel}
                  placeholder="Search country"
                  emptyLabel="No matching country"
                  onChange={(value, option) => {
                    setCountry(value);
                    setCountryLabel(option?.label || value);
                  }}
                  loadOptions={async (q) => {
                    const rows = await api.get<Country[]>("/master/countries", {
                      search: q || undefined,
                    });
                    return (Array.isArray(rows) ? rows : []).map((c) => ({
                      value: c.code,
                      label: `${c.code} — ${c.name}`,
                    }));
                  }}
                />
              </Field>
            </div>
            <Field label="Address">
              <Input value={address} onChange={(e) => setAddress(e.target.value)} />
            </Field>
            <Field label="Address 2">
              <Input value={address2} onChange={(e) => setAddress2(e.target.value)} />
            </Field>
          </div>
        ) : !savedId ? (
          <RecordPlaceholder>
            {pane === "billing"
              ? "Billing mappings appear here after you create the supplier."
              : "Documents appear here after you create the supplier."}
          </RecordPlaceholder>
        ) : pane === "billing" ? (
          <BillingAccountsPanel
            partyKind="supplier"
            partyId={savedId}
            showHelp={false}
          />
        ) : (
          <PartyDocuments
            path={`/master/suppliers/${savedId}`}
            deleteDetail="Remove this file from the supplier record? This cannot be undone."
          />
        )}
      </PartyWorkspace>
    </CatalogueForm>
  );
}
