"use client";

import { useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import {
  CatalogueForm,
  PartyWorkspace,
  RecordPlaceholder,
} from "../form";
import { Field, Input, ToggleField } from "@/components/ui";
import { ErpField, InquiryHeader, StatusPill } from "@/components/ApprovalReview";
import type { Customer, EwuraLicense } from "@/lib/master";
import { BillingAccountsPanel } from "../BillingAccounts";
import { PartyDocuments } from "../PartyDocuments";

type RelatedPane = "billing" | "documents";
type FormPane = "general" | RelatedPane;

const RELATED_PANES: { id: RelatedPane; label: string }[] = [
  { id: "billing", label: "Billing" },
  { id: "documents", label: "Documents" },
];

const FORM_PANES: { id: FormPane; label: string }[] = [
  { id: "general", label: "General" },
  ...RELATED_PANES,
];

export function CustomerInquiry({ row }: { row: Customer }) {
  const [pane, setPane] = useState<RelatedPane>("billing");
  return (
    <div className="space-y-4">
      <InquiryHeader
        crumb="Stock · Setups · Customers"
        title={row.name || row.code || "Customer"}
        subtitle={row.code ? `Code ${row.code}` : undefined}
        pills={
          <StatusPill tone={row.isActive ? "navy" : "slate"}>
            {row.isActive ? "Active" : "Inactive"}
          </StatusPill>
        }
      >
        <ErpField label="KYC number" value={row.kycNumber || "—"} boxed={false} />
        <ErpField label="VRN" value={row.vrnNumber || "—"} boxed={false} />
        <ErpField label="EWURA license" value={row.ewuraLicense || "—"} boxed={false} />
        <ErpField label="Email" value={row.email || "—"} boxed={false} />
        <ErpField label="Phone" value={row.phone || "—"} boxed={false} />
        <ErpField label="TIN" value={row.tinNumber || "—"} boxed={false} />
      </InquiryHeader>
      <PartyWorkspace
        panes={RELATED_PANES}
        pane={pane}
        onPane={setPane}
        banner={row.name || row.code || "Customer"}
        hint={
          pane === "billing"
            ? "Sage AR mappings used when this customer is billed."
            : "Registration files copied onto new ILRs, pump-over requests, and ITTs."
        }
      >
        {pane === "billing" ? (
          <BillingAccountsPanel
            partyKind="customer"
            partyId={row.id}
            readOnly
            showHelp={false}
          />
        ) : (
          <PartyDocuments path={`/master/customers/${row.id}`} readOnly lifecycle />
        )}
      </PartyWorkspace>
    </div>
  );
}

export function CustomerForm({
  row,
  onClose,
  onSaved,
}: {
  row?: Customer;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [pane, setPane] = useState<FormPane>("general");
  const [name, setName] = useState(row?.name ?? "");
  const [email, setEmail] = useState(row?.email ?? "");
  const [phone, setPhone] = useState(row?.phone ?? "");
  const [tinNumber, setTin] = useState(row?.tinNumber ?? "");
  const [kycNumber, setKyc] = useState(row?.kycNumber ?? "");
  const [vrnNumber, setVrn] = useState(row?.vrnNumber ?? "");
  const [ewuraLicense, setLicense] = useState(row?.ewuraLicense ?? "");
  const [licenseLabel, setLicenseLabel] = useState(row?.ewuraLicense ?? "");
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
    if (!kycNumber.trim()) {
      setError("KYC number is required.");
      setPane("general");
      return;
    }
    if (!ewuraLicense.trim()) {
      setError("EWURA license is required.");
      setPane("general");
      return;
    }
    setSaving(true);
    try {
      const body = {
        name: name.trim(),
        email: email.trim(),
        phone: phone.trim(),
        tinNumber: tinNumber.trim(),
        kycNumber: kycNumber.trim(),
        vrnNumber: vrnNumber.trim(),
        ewuraLicense: ewuraLicense.trim(),
        isActive,
      };
      if (savedId) {
        await api.put(`/master/customers/${savedId}`, body);
        onSaved();
        return;
      }
      const created = await api.post<Customer>("/master/customers", body);
      setSavedId(created.id);
      setPane("billing");
    } catch (err) {
      setError(formatApiError(err, "Failed to save customer"));
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
        : "Create the customer first, then map Sage AR accounts."
      : pane === "documents"
        ? savedId
          ? "Active files are copied onto new ILRs, pump-over requests, and ITTs."
          : "Create the customer first, then attach registration documents."
        : undefined;

  return (
    <CatalogueForm
      title={row ? "Customer" : "Add customer"}
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
        banner={name.trim() || row?.name || row?.code || (savedId ? "Customer" : undefined)}
        hint={hint}
      >
        {pane === "general" ? (
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <Field label="Name" required>
                <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
              </Field>
            </div>
            <Field label="Email">
              <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </Field>
            <Field label="Phone">
              <Input value={phone} onChange={(e) => setPhone(e.target.value)} />
            </Field>
            <Field
              label="KYC number"
              required
              hint="Know Your Customer reference. Required on every new customer."
            >
              <Input value={kycNumber} onChange={(e) => setKyc(e.target.value)} />
            </Field>
            <Field label="VRN">
              <Input value={vrnNumber} onChange={(e) => setVrn(e.target.value)} />
            </Field>
            <Field label="TIN">
              <Input value={tinNumber} onChange={(e) => setTin(e.target.value)} />
            </Field>
            <Field
              label="EWURA license"
              required
              hint="Choosing a licence copies email, phone, and TIN when those fields are empty. You can edit them afterwards."
            >
              <AsyncSearchableSelect
                value={ewuraLicense}
                selectedLabel={licenseLabel}
                placeholder="Search license number or licensee"
                emptyLabel="No matching license"
                onChange={(value, option) => {
                  setLicense(value);
                  setLicenseLabel(option?.label || value);
                  if (!email.trim()) setEmail(option?.email ?? "");
                  if (!phone.trim()) setPhone(option?.phone ?? "");
                  if (!tinNumber.trim()) setTin(option?.tinNumber ?? "");
                }}
                loadOptions={async (q) => {
                  const rows = await api.get<EwuraLicense[]>("/ewura/licenses/options", {
                    q: q || undefined,
                  });
                  return (Array.isArray(rows) ? rows : []).map((l) => ({
                    value: l.licenseNumber,
                    label: `${l.licenseNumber} — ${l.licensee}`,
                    keywords: `${l.tinNumber || ""} ${l.licenseClass || ""}`,
                    email: l.email,
                    phone: l.phone,
                    tinNumber: l.tinNumber,
                  }));
                }}
              />
            </Field>
            <ToggleField
              compact
              label="Active"
              description="Hidden from new receipts and orders when off"
              checked={isActive}
              onChange={setActive}
            />
          </div>
        ) : !savedId ? (
          <RecordPlaceholder>
            {pane === "billing"
              ? "Billing mappings appear here after you create the customer."
              : "Documents appear here after you create the customer."}
          </RecordPlaceholder>
        ) : pane === "billing" ? (
          <BillingAccountsPanel
            partyKind="customer"
            partyId={savedId}
            showHelp={false}
          />
        ) : (
          <PartyDocuments
            path={`/master/customers/${savedId}`}
            lifecycle
            deleteDetail="Only files that have never been copied onto an ILR, pump-over, or ITT can be deleted. Used files stay for history — deactivate them so they are not copied onto new work."
          />
        )}
      </PartyWorkspace>
    </CatalogueForm>
  );
}
