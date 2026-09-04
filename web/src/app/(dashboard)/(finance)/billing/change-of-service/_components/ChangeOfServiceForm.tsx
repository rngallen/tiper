"use client";

import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { Button, Field, Select, Textarea } from "@/components/ui";
import { DateInput } from "@/components/DateInput";
import { formatDate, localISODate } from "@/lib/format";
import { masterOption, qty, searchMaster, type MasterRef } from "@/lib/master";
import { activeRows, catalogLabel, fetchLookups } from "@/lib/lookups";

export type CosPayload = {
  effectiveDate: string;
  customerId: string;
  parcelId: string;
  toCollection: string;
  notes?: string;
};

export type CosFormValue = {
  effectiveDate?: string;
  customerId?: string;
  customerLabel?: string;
  parcelId?: string;
  vesselId?: string;
  vesselDate?: string;
  fromCollection?: string;
  toCollection?: string;
  notes?: string;
};

export type CosParcel = {
  id: string;
  receiptNumber?: string;
  vesselId: string;
  vesselName?: string;
  vesselDate: string;
  productId: string;
  productCode?: string;
  productName?: string;
  collectionMethod?: string;
  contractTypeCode?: string;
  quantity?: string;
};

function day(v?: string): string {
  return (v || "").slice(0, 10);
}

function vesselLabel(p: CosParcel): string {
  return p.vesselName || p.vesselId;
}

function productLabel(p: CosParcel): string {
  const code = p.productCode ? `${p.productCode} — ` : "";
  return `${code}${p.productName || p.productId}`;
}

function parcelCaption(p: CosParcel): string {
  const method = p.collectionMethod || "—";
  const vol = p.quantity ? ` · ${qty(p.quantity)}` : "";
  return `${productLabel(p)} · ${method}${vol}`;
}

const EMPTY_PARCELS: CosParcel[] = [];

export function ChangeOfServiceForm({
  initial,
  saving,
  submitLabel,
  onCancel,
  onSubmit,
  onError,
}: {
  initial?: CosFormValue;
  saving: boolean;
  submitLabel: string;
  onCancel: () => void;
  onSubmit: (body: CosPayload) => void;
  onError: (message: string) => void;
}) {
  const today = localISODate(new Date());
  const catalogs = useAsync(fetchLookups, []);
  const [form, setForm] = useState({
    effectiveDate: (initial?.effectiveDate || today).slice(0, 10),
    customerId: initial?.customerId || "",
    customerLabel: initial?.customerLabel || "",
    vesselId: initial?.vesselId || "",
    vesselDate: day(initial?.vesselDate),
    parcelId: initial?.parcelId || "",
    toCollection: initial?.toCollection || "",
    notes: initial?.notes || "",
  });

  function patch(p: Partial<typeof form>) {
    setForm((prev) => ({ ...prev, ...p }));
  }

  const parcels = useAsync(
    () =>
      form.customerId
        ? api.get<CosParcel[]>(`/billing/change-of-service/parcels`, { customerId: form.customerId })
        : Promise.resolve([] as CosParcel[]),
    [form.customerId],
  );
  const rows = Array.isArray(parcels.data) ? parcels.data : EMPTY_PARCELS;
  const delivery = activeRows(catalogs.data?.deliveryMethods);

  const vessels = useMemo(() => {
    const seen = new Map<string, string>();
    for (const p of rows) {
      if (p.vesselId && !seen.has(p.vesselId)) seen.set(p.vesselId, vesselLabel(p));
    }
    return [...seen.entries()].map(([id, label]) => ({ id, label }));
  }, [rows]);

  const dates = useMemo(() => {
    const seen = new Set<string>();
    for (const p of rows) {
      if (p.vesselId !== form.vesselId) continue;
      const d = day(p.vesselDate);
      if (d) seen.add(d);
    }
    return [...seen].sort().reverse();
  }, [rows, form.vesselId]);

  const matches = useMemo(
    () => rows.filter((p) => p.vesselId === form.vesselId && day(p.vesselDate) === form.vesselDate),
    [rows, form.vesselId, form.vesselDate],
  );

  const selected = matches.find((p) => p.id === form.parcelId) || matches[0];
  const fromCollection = selected?.collectionMethod || "";

  useEffect(() => {
    if (!form.customerId) return;
    if (!form.vesselId && vessels[0]) {
      patch({ vesselId: vessels[0].id, vesselDate: "", parcelId: "" });
      return;
    }
    if (form.vesselId && !form.vesselDate && dates[0]) {
      patch({ vesselDate: dates[0], parcelId: "" });
      return;
    }
    if (matches.length > 0 && !matches.some((p) => p.id === form.parcelId)) {
      patch({ parcelId: matches[0].id });
    }
  }, [form.customerId, form.vesselId, form.vesselDate, form.parcelId, vessels, dates, matches]);

  function save() {
    if (!form.customerId) {
      onError("Customer is required.");
      return;
    }
    if (!selected?.id) {
      onError("Select the vessel and vessel date for the parcel to change.");
      return;
    }
    if (!form.toCollection) {
      onError("New delivery method is required.");
      return;
    }
    if (form.toCollection === fromCollection) {
      onError("Choose a different delivery method. Fees stay on the customer.");
      return;
    }
    onSubmit({
      effectiveDate: `${form.effectiveDate}T00:00:00Z`,
      customerId: form.customerId,
      parcelId: selected.id,
      toCollection: form.toCollection,
      notes: form.notes.trim() || undefined,
    });
  }

  return (
    <div className="grid max-w-3xl gap-4 sm:grid-cols-2">
      <p className="sm:col-span-2 text-sm leading-6 text-slate-600">
        Change delivery method on one approved vessel parcel. All fees stay mapped — only collection
        (pump-over vs gantry loading) changes how that parcel is billed.
      </p>
      <Field label="Effective date">
        <DateInput value={form.effectiveDate} onChange={(effectiveDate) => patch({ effectiveDate })} />
      </Field>
      <Field label="Customer">
        <AsyncSearchableSelect
          value={form.customerId}
          selectedLabel={form.customerLabel}
          placeholder="Search customer"
          onChange={(value, option) =>
            patch({
              customerId: value,
              customerLabel: option?.label || "",
              vesselId: "",
              vesselDate: "",
              parcelId: "",
              toCollection: "",
            })
          }
          loadOptions={async (q) => (await searchMaster<MasterRef>("/master/customers", q)).map(masterOption)}
        />
      </Field>
      <Field label="Vessel">
        <Select
          value={form.vesselId}
          disabled={!form.customerId || parcels.loading}
          onChange={(e) => patch({ vesselId: e.target.value, vesselDate: "", parcelId: "" })}
        >
          <option value="">{parcels.loading ? "Loading parcels…" : "Select vessel"}</option>
          {vessels.map((v) => (
            <option key={v.id} value={v.id}>
              {v.label}
            </option>
          ))}
        </Select>
      </Field>
      <Field label="Vessel date">
        <Select
          value={form.vesselDate}
          disabled={!form.vesselId}
          onChange={(e) => patch({ vesselDate: e.target.value, parcelId: "" })}
        >
          <option value="">Select date</option>
          {dates.map((d) => (
            <option key={d} value={d}>
              {formatDate(d)}
            </option>
          ))}
        </Select>
      </Field>
      {matches.length > 1 ? (
        <div className="sm:col-span-2">
          <Field label="Parcel">
            <Select value={form.parcelId} onChange={(e) => patch({ parcelId: e.target.value })}>
              {matches.map((p) => (
                <option key={p.id} value={p.id}>
                  {parcelCaption(p)}
                  {p.receiptNumber ? ` · ${p.receiptNumber}` : ""}
                </option>
              ))}
            </Select>
          </Field>
        </div>
      ) : selected ? (
        <div className="sm:col-span-2 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700">
          <span className="font-semibold text-brand-800">{parcelCaption(selected)}</span>
          {selected.receiptNumber ? ` · ${selected.receiptNumber}` : ""}
        </div>
      ) : form.customerId && !parcels.loading ? (
        <p className="sm:col-span-2 text-sm text-slate-600">
          No approved vessel parcels for this customer.
        </p>
      ) : null}
      <Field label="Current delivery">
        <input
          className="pp-control bg-slate-50"
          value={catalogLabel(delivery.find((d) => d.code === fromCollection)) || fromCollection || "—"}
          disabled
          readOnly
        />
      </Field>
      <Field label="New delivery method">
        <Select value={form.toCollection} onChange={(e) => patch({ toCollection: e.target.value })}>
          <option value="">Select method</option>
          {delivery.map((d) => (
            <option key={d.code} value={d.code} disabled={d.code === fromCollection}>
              {catalogLabel(d)}
            </option>
          ))}
        </Select>
      </Field>
      <div className="sm:col-span-2">
        <Field label="Notes">
          <Textarea value={form.notes} onChange={(e) => patch({ notes: e.target.value })} rows={3} />
        </Field>
      </div>
      <div className="sm:col-span-2 flex justify-end gap-2">
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
        <Button onClick={() => void save()} loading={saving}>
          {submitLabel}
        </Button>
      </div>
    </div>
  );
}
