"use client";

import { useCallback, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { DateInput } from "@/components/DateInput";
import { Field, Input, ToggleField } from "@/components/ui";
import {
  ErpField,
  InquiryHeader,
  StatusPill,
} from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import { type Country, type Transporter } from "@/lib/master";
import { formatDate } from "@/lib/format";
import { CatalogueForm, PartyWorkspace } from "../form";

function isoDate(v?: string) {
  return (v || "").slice(0, 10);
}

type HaulerPane = "general" | "tax" | "address";

const HAULER_PANES: { id: HaulerPane; label: string }[] = [
  { id: "general", label: "General" },
  { id: "tax", label: "Tax & licence" },
  { id: "address", label: "Address" },
];

export default function TransportersPage() {
  const columns = useCallback(
    () =>
      [
        {
          id: "name",
          header: "Hauler",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "tinNumber",
          header: "TIN",
          defaultVisible: true,
          cell: (r) => r.tinNumber || "—",
        },
        {
          id: "phone",
          header: "Phone",
          defaultVisible: true,
          cell: (r) => r.phone || "—",
        },
        {
          id: "country",
          header: "Country",
          defaultVisible: false,
          cell: (r) => r.country?.name || r.countryCode || "—",
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Transporter>[],
    [],
  );

  return (
    <MasterCatalogue<Transporter>
      tableId="master-transporters"
      title="Haulers"
      subtitle="Haulers that collect product from the gantry"
      path="/master/transporters"
      orderBy="name"
      searchPlaceholder="Search name, TIN, or phone"
      createLabel="Add hauler"
      emptyMessage="No haulers found"
      columns={columns}
      inquiryTitle={(r) => r.name || "Hauler"}
      inquirySize="xl"
      inquiry={(r) => <HaulerInquiry row={r} />}
      Form={TransporterForm}
    />
  );
}

function HaulerInquiry({ row }: { row: Transporter }) {
  const [pane, setPane] = useState<HaulerPane>("general");
  return (
    <div className="space-y-4">
      <InquiryHeader
        crumb="Haulers · Inquiry"
        title={row.name || "Hauler"}
        pills={
          <StatusPill tone={row.isActive ? "navy" : "slate"}>
            {row.isActive ? "Active" : "Inactive"}
          </StatusPill>
        }
      />
      <PartyWorkspace panes={HAULER_PANES} pane={pane} onPane={setPane}>
        {pane === "general" ? (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <ErpField label="Name" value={row.name} />
            <ErpField label="Contact" value={row.contactPerson || "—"} />
            <ErpField label="Phone" value={row.phone || "—"} />
            <ErpField label="Email" value={row.email || "—"} />
          </div>
        ) : null}
        {pane === "tax" ? (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <ErpField label="TIN" value={row.tinNumber || "—"} />
            <ErpField label="VRN" value={row.vrnNumber || "—"} />
            <ErpField label="AEO / transit licence" value={row.license || "—"} />
            <ErpField label="AEO end" value={formatDate(row.aeoEndDate)} />
          </div>
        ) : null}
        {pane === "address" ? (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <ErpField label="Country" value={row.country?.name || row.countryCode || "—"} />
            <ErpField label="Address" value={row.address || "—"} />
            <ErpField label="Address 2" value={row.address2 || "—"} />
          </div>
        ) : null}
      </PartyWorkspace>
    </div>
  );
}

function TransporterForm({
  row,
  onClose,
  onSaved,
}: {
  row?: Transporter;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(row?.name ?? "");
  const [contactPerson, setContact] = useState(row?.contactPerson ?? "");
  const [phone, setPhone] = useState(row?.phone ?? "");
  const [email, setEmail] = useState(row?.email ?? "");
  const [tinNumber, setTin] = useState(row?.tinNumber ?? "");
  const [vrnNumber, setVrn] = useState(row?.vrnNumber ?? "");
  const [license, setLicense] = useState(row?.license ?? "");
  const [aeoEndDate, setAeo] = useState(isoDate(row?.aeoEndDate));
  const [countryCode, setCountry] = useState(row?.countryCode ?? "");
  const [countryLabel, setCountryLabel] = useState(row?.country?.name ?? row?.countryCode ?? "");
  const [address, setAddress] = useState(row?.address ?? "");
  const [address2, setAddress2] = useState(row?.address2 ?? "");
  const [isActive, setActive] = useState(row?.isActive ?? true);
  const [pane, setPane] = useState<HaulerPane>("general");
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
      const body = {
        name: name.trim(),
        contactPerson: contactPerson.trim(),
        phone: phone.trim(),
        email: email.trim(),
        tinNumber: tinNumber.trim(),
        vrnNumber: vrnNumber.trim(),
        license: license.trim(),
        aeoEndDate,
        countryCode,
        address: address.trim(),
        address2: address2.trim(),
        isActive,
      };
      if (row) await api.put(`/master/transporters/${row.id}`, body);
      else await api.post("/master/transporters", body);
      onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save hauler"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? "Edit hauler" : "Add hauler"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
      size="xl"
    >
      <PartyWorkspace panes={HAULER_PANES} pane={pane} onPane={setPane}>
        {pane === "general" ? (
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Name">
              <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
            </Field>
            <Field label="Contact person">
              <Input value={contactPerson} onChange={(e) => setContact(e.target.value)} />
            </Field>
            <Field label="Phone">
              <Input value={phone} onChange={(e) => setPhone(e.target.value)} />
            </Field>
            <Field label="Email">
              <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </Field>
            <div className="sm:col-span-2">
              <ToggleField label="Active" checked={isActive} onChange={setActive} />
            </div>
          </div>
        ) : null}
        {pane === "tax" ? (
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="TIN">
              <Input value={tinNumber} onChange={(e) => setTin(e.target.value)} />
            </Field>
            <Field label="VRN">
              <Input value={vrnNumber} onChange={(e) => setVrn(e.target.value)} />
            </Field>
            <Field label="AEO / transit licence">
              <Input value={license} onChange={(e) => setLicense(e.target.value)} />
            </Field>
            <Field label="AEO end date">
              <DateInput value={aeoEndDate} onChange={setAeo} />
            </Field>
          </div>
        ) : null}
        {pane === "address" ? (
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
        ) : null}
      </PartyWorkspace>
    </CatalogueForm>
  );
}
