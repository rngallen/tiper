"use client";

import { useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { Field, Input, ToggleField } from "@/components/ui";
import { DateInput } from "@/components/DateInput";
import { DocumentFiles } from "@/components/DocumentFiles";
import { normalizePlate } from "@/lib/format";
import type { Driver } from "@/lib/master";
import { CatalogueForm, PartyWorkspace } from "../form";

export function DriverForm({
  row,
  onClose,
  onSaved,
}: {
  row?: Driver;
  onClose: () => void;
  onSaved: () => void;
}) {
  const editing = Boolean(row);
  const [name, setName] = useState(row?.name ?? "");
  const [licenseNumber, setLicense] = useState(row?.licenseNumber ?? "");
  const [licenseExpires, setExpires] = useState(
    row?.licenseExpires ? String(row.licenseExpires).slice(0, 10) : "",
  );
  const [phone, setPhone] = useState(row?.phone ?? "");
  const [email, setEmail] = useState(row?.email ?? "");
  const [isActive, setActive] = useState(row?.isActive ?? true);
  const [pane, setPane] = useState<"general" | "licence">("general");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    if (!name.trim() || !licenseNumber.trim()) {
      setError("Name and licence number are required.");
      return;
    }
    setSaving(true);
    try {
      if (row) {
        await api.put(`/master/drivers/${row.id}`, {
          name: name.trim(),
          licenseExpires,
          phone: phone.trim(),
          email: email.trim(),
          isActive,
        });
      } else {
        await api.post("/master/drivers", {
          name: name.trim(),
          licenseNumber: licenseNumber.trim(),
        });
      }
      onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save driver"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={editing ? "Edit driver" : "Add driver"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={editing ? "Save" : "Create"}
      size={editing ? "xl" : "md"}
    >
      {editing ? (
        <PartyWorkspace
          panes={[
            { id: "general", label: "General" },
            { id: "licence", label: "Driving licence" },
          ]}
          pane={pane}
          onPane={setPane}
        >
          {pane === "general" ? (
            <DriverDetailsFields
              name={name}
              setName={setName}
              licenseNumber={licenseNumber}
              setLicense={setLicense}
              licenseExpires={licenseExpires}
              setExpires={setExpires}
              phone={phone}
              setPhone={setPhone}
              email={email}
              setEmail={setEmail}
              isActive={isActive}
              setActive={setActive}
              editing
            />
          ) : (
            <DocumentFiles
              path={`/master/drivers/${row!.id}`}
              draft
              hint="Keep scans of the current card and renewals. The drop zone stays at the top."
              layout="stack"
            />
          )}
        </PartyWorkspace>
      ) : (
        <>
          <DriverDetailsFields
            name={name}
            setName={setName}
            licenseNumber={licenseNumber}
            setLicense={setLicense}
            licenseExpires={licenseExpires}
            setExpires={setExpires}
            phone={phone}
            setPhone={setPhone}
            email={email}
            setEmail={setEmail}
            isActive={isActive}
            setActive={setActive}
            editing={false}
          />
          <p className="text-xs leading-5 text-slate-500">
            Licence expiry defaults to yesterday until it is set. After create, attach licence
            scans and complete phone, email, and expiry on edit.
          </p>
        </>
      )}
    </CatalogueForm>
  );
}

function DriverDetailsFields({
  name,
  setName,
  licenseNumber,
  setLicense,
  licenseExpires,
  setExpires,
  phone,
  setPhone,
  email,
  setEmail,
  isActive,
  setActive,
  editing,
}: {
  name: string;
  setName: (v: string) => void;
  licenseNumber: string;
  setLicense: (v: string) => void;
  licenseExpires: string;
  setExpires: (v: string) => void;
  phone: string;
  setPhone: (v: string) => void;
  email: string;
  setEmail: (v: string) => void;
  isActive: boolean;
  setActive: (v: boolean) => void;
  editing: boolean;
}) {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <Field label="Name">
        <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      </Field>
      <Field label="License number">
        <Input
          value={licenseNumber}
          onChange={(e) => setLicense(normalizePlate(e.target.value))}
          disabled={editing}
        />
      </Field>
      {editing ? (
        <>
          <Field label="License expires">
            <DateInput value={licenseExpires} onChange={setExpires} />
          </Field>
          <Field label="Phone">
            <Input value={phone} onChange={(e) => setPhone(e.target.value)} />
          </Field>
          <Field label="Email">
            <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </Field>
          <div className="flex items-end">
            <ToggleField label="Active" checked={isActive} onChange={setActive} />
          </div>
        </>
      ) : null}
    </div>
  );
}
