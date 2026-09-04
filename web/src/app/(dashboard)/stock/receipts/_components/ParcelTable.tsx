"use client";

import { useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { SectionRow, SectionTable } from "@/components/SectionTable";
import { Button, Card, Field, Input, Select } from "@/components/ui";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { Modal } from "@/components/Modal";
import { QtyInput } from "@/components/QtyInput";
import { formatQty } from "@/lib/format";
import { masterOption, searchMaster } from "@/lib/master";
import { partyName } from "@/lib/ilr";
import { stockStatusLabel, type StockStatusRef } from "@/lib/status";
import { activeRows, catalogLabel, type BillingLookups } from "@/lib/lookups";
import { addQty, fromLitres } from "../_lib/qty";
import type { Receipt, ReceiptDetail } from "../_lib/types";

type Ref = { id: string; code?: string; name?: string };

const empty = {
  customerId: "",
  customerLabel: "",
  depotId: "",
  depotLabel: "",
  stockStatusId: "",
  stockStatusLabel: "",
  collectionMethod: "",
  contractTypeCode: "",
  pricingNature: "",
  nextBillingDays: 15,
  financialHold: false,
  quantity: "",
  isProration: false,
};

export function ParcelTable({
  row,
  catalogs,
  onSaved,
}: {
  row: Receipt;
  catalogs: BillingLookups | undefined;
  onSaved: () => void;
}) {
  const draft = row.status === "draft";
  const internal = row.receiptType !== "external";
  const delivery = activeRows(catalogs?.deliveryMethods);
  const contracts = activeRows(catalogs?.contractTypes);
  const pricing = activeRows(catalogs?.pricingNatures);
  const cycles = activeRows(catalogs?.billingCycles);
  const [mode, setMode] = useState<"add" | "edit" | null>(null);
  const [editing, setEditing] = useState<ReceiptDetail | null>(null);
  const [form, setForm] = useState(empty);
  const [err, setErr] = useState("");
  const [saving, setSaving] = useState(false);

  const derived = fromLitres(form.quantity, row.density || "0");

  function openAdd() {
    setEditing(null);
    setForm({
      ...empty,
      collectionMethod: delivery[0]?.code || "",
      contractTypeCode: contracts[0]?.code || "",
      pricingNature: pricing[0]?.code || "",
      nextBillingDays: cycles[0]?.days || 15,
    });
    setErr("");
    setMode("add");
  }

  function openEdit(d: ReceiptDetail) {
    setEditing(d);
    setForm({
      customerId: d.customer?.id || "",
      customerLabel: partyName(d.customer),
      depotId: d.depot?.id || "",
      depotLabel: partyName(d.depot),
      stockStatusId: d.stockStatus?.id || "",
      stockStatusLabel: d.stockStatus?.name || "",
      collectionMethod: d.collectionMethod || delivery[0]?.code || "",
      contractTypeCode: d.contractTypeCode || contracts[0]?.code || "",
      pricingNature: d.pricingNature || pricing[0]?.code || "",
      nextBillingDays: d.nextBillingDays || 15,
      financialHold: Boolean(d.financialHold),
      quantity: d.quantity || "",
      isProration: Boolean(d.stockStatus?.isProration),
    });
    setErr("");
    setMode("edit");
  }

  async function save() {
    setErr("");
    if (!form.customerId || !form.quantity) {
      setErr("Customer and quantity in litres are required.");
      return;
    }
    if (internal && !form.stockStatusId) {
      setErr("Status is required.");
      return;
    }
    if (!internal && !form.depotId) {
      setErr("Receiving depot is required.");
      return;
    }
    if (internal && form.isProration && !(Number(form.quantity) < 0)) {
      setErr("Proration (PRORATA) parcel quantity must be negative.");
      return;
    }
    setSaving(true);
    const body = {
      customerId: form.customerId,
      depotId: form.depotId || undefined,
      stockStatusId: internal ? form.stockStatusId || undefined : undefined,
      collectionMethod: internal ? form.collectionMethod : "",
      contractTypeCode: internal ? form.contractTypeCode : "",
      pricingNature: internal ? form.pricingNature : "",
      nextBillingDays: internal ? form.nextBillingDays : undefined,
      financialHold: internal && form.financialHold,
      quantity: form.quantity,
    };
    try {
      if (mode === "edit" && editing) {
        await api.put(`/ic/receipts/${row.id}/details/${editing.id}`, body);
      } else {
        await api.post(`/ic/receipts/${row.id}/details`, body);
      }
      setMode(null);
      onSaved();
    } catch (e) {
      setErr(formatApiError(e, "Could not save parcel"));
    } finally {
      setSaving(false);
    }
  }

  async function remove(id: string) {
    try {
      await api.del(`/ic/receipts/${row.id}/details/${id}`);
      onSaved();
    } catch (e) {
      setErr(formatApiError(e, "Could not remove parcel"));
    }
  }

  const lines = row.details || [];

  return (
    <Card className="space-y-3 p-5">
      {err && !mode ? <p className="text-sm text-red-600">{err}</p> : null}
      <SectionTable
        title={internal ? "Customer parcels" : "Depot quantities"}
        actions={
          draft ? (
            <Button variant="secondary" className="!px-3 !py-1.5" onClick={openAdd}>
              {internal ? "Add parcel" : "Add depot"}
            </Button>
          ) : null
        }
        columns={[
          { label: "Customer" },
          { label: internal ? "Status" : "Depot" },
          ...(internal ? [{ label: "Delivery" }, { label: "Hold" }] : []),
          { label: "Outturn (L)", align: "right" as const },
          { label: "m³", align: "right" as const },
          { label: "MT", align: "right" as const },
          ...(internal
            ? [
                { label: "Line loss (L)", align: "right" as const },
                { label: "Received (L)", align: "right" as const },
              ]
            : []),
          { label: "" },
        ]}
        empty={
          draft
            ? internal
              ? "No parcels yet — click Add parcel"
              : "No depot lines yet — click Add depot"
            : "No lines"
        }
      >
        {lines.map((d) => (
          <SectionRow key={d.id}>
            <td className="font-semibold">{partyName(d.customer)}</td>
            <td>{internal ? d.stockStatus?.name || "—" : partyName(d.depot)}</td>
            {internal ? <td>{d.collectionMethod || "—"}</td> : null}
            {internal ? <td>{d.financialHold ? "Yes" : "—"}</td> : null}
            <td className="text-right tabular-nums">{formatQty(d.quantity, "l")}</td>
            <td className="text-right tabular-nums">{formatQty(d.cubicMeter, "m3")}</td>
            <td className="text-right tabular-nums">{formatQty(d.metricTonne, "mt")}</td>
            {internal ? <td className="text-right tabular-nums">{formatQty(d.lineLoss, "l")}</td> : null}
            {internal ? (
              <td className="text-right tabular-nums">{formatQty(addQty(d.quantity, d.lineLoss))}</td>
            ) : null}
            <td className="text-right">
              {draft ? (
                <div className="flex justify-end gap-2">
                  <Button variant="ghost" className="!px-2 !py-1" onClick={() => openEdit(d)}>
                    Edit
                  </Button>
                  <Button variant="ghost" className="!px-2 !py-1" onClick={() => void remove(d.id)}>
                    Remove
                  </Button>
                </div>
              ) : null}
            </td>
          </SectionRow>
        ))}
      </SectionTable>

      <Modal
        open={mode !== null}
        onClose={() => setMode(null)}
        title={mode === "edit" ? (internal ? "Edit parcel" : "Edit depot line") : internal ? "Add parcel" : "Add depot line"}
        size="lg"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setMode(null)}>
              Cancel
            </Button>
            <Button loading={saving} onClick={() => void save()}>
              Save
            </Button>
          </div>
        }
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Customer">
            <AsyncSearchableSelect
              value={form.customerId}
              selectedLabel={form.customerLabel}
              placeholder="Search customer"
              onChange={(value, option) => setForm({ ...form, customerId: value, customerLabel: option?.label || "" })}
              loadOptions={async (q) => (await searchMaster<Ref>("/master/customers", q)).map(masterOption)}
            />
          </Field>
          {internal ? (
            <Field label="Status">
              <AsyncSearchableSelect
                value={form.stockStatusId}
                selectedLabel={form.stockStatusLabel}
                placeholder="Local, transit, mining, proration"
                onChange={(value, option) =>
                  setForm({
                    ...form,
                    stockStatusId: value,
                    stockStatusLabel: option?.label || "",
                    isProration: Boolean(option?.isProration),
                  })
                }
                loadOptions={async (q) =>
                  (await searchMaster<StockStatusRef>("/master/statuses", q)).map((s) => ({
                    value: s.id,
                    label: stockStatusLabel(s),
                    isProration: s.isProration,
                  }))
                }
              />
            </Field>
          ) : (
            <Field label="Receiving depot">
              <AsyncSearchableSelect
                value={form.depotId}
                selectedLabel={form.depotLabel}
                placeholder="Search depot"
                onChange={(value, option) => setForm({ ...form, depotId: value, depotLabel: option?.label || "" })}
                loadOptions={async (q) => (await searchMaster<Ref>("/master/depots", q)).map(masterOption)}
              />
            </Field>
          )}
          {internal ? (
            <>
              <Field label="Delivery">
                <Select
                  value={form.collectionMethod}
                  onChange={(e) => setForm({ ...form, collectionMethod: e.target.value })}
                >
                  {delivery.map((r) => (
                    <option key={r.code} value={r.code}>{catalogLabel(r)}</option>
                  ))}
                </Select>
              </Field>
              <Field label="Pricing">
                <Select value={form.pricingNature} onChange={(e) => setForm({ ...form, pricingNature: e.target.value })}>
                  {pricing.map((r) => (
                    <option key={r.code} value={r.code}>{catalogLabel(r)}</option>
                  ))}
                </Select>
              </Field>
              <Field label="Contract type">
                <Select
                  value={form.contractTypeCode}
                  onChange={(e) => setForm({ ...form, contractTypeCode: e.target.value })}
                >
                  {contracts.map((r) => (
                    <option key={r.code} value={r.code}>{catalogLabel(r)}</option>
                  ))}
                </Select>
              </Field>
              <Field label="Next billing">
                <Select
                  value={String(form.nextBillingDays)}
                  onChange={(e) => setForm({ ...form, nextBillingDays: Number(e.target.value) })}
                >
                  {(cycles.length ? cycles : [{ days: 15, description: "15-day next billing" }]).map((r) => (
                    <option key={r.days} value={r.days}>{r.description || `${r.days} days`}</option>
                  ))}
                </Select>
              </Field>
              <label className="flex items-center gap-2 self-end text-sm">
                <input
                  type="checkbox"
                  checked={form.financialHold}
                  onChange={(e) => setForm({ ...form, financialHold: e.target.checked })}
                />
                Financial hold
              </label>
            </>
          ) : null}
          <div className="grid gap-4 sm:col-span-2 sm:grid-cols-3">
            <Field
              label="Quantity (L)"
              hint={form.isProration ? "PRORATA parcels must be a negative quantity and are not billed." : undefined}
            >
              <QtyInput value={form.quantity} onChange={(quantity) => setForm({ ...form, quantity })} />
            </Field>
            <Field label="Cubic metres (m³)">
              <Input disabled value={derived.cubicMeter} />
            </Field>
            <Field label="Metric tonnes (MT)">
              <Input disabled value={derived.metricTonne} />
            </Field>
          </div>
          <p className="sm:col-span-2 text-xs text-slate-500">
            {internal
              ? `m³ = litres ÷ 1,000. MT = m³ × header density (${row.density || "0"}).`
              : "m³ = litres ÷ 1,000."}
          </p>
          {err ? <p className="sm:col-span-2 text-sm text-red-600">{err}</p> : null}
        </div>
      </Modal>
    </Card>
  );
}
