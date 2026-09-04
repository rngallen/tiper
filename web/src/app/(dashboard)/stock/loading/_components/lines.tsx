"use client";

import { useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { SectionRow, SectionTable } from "@/components/SectionTable";
import { Button, Card, Field, Input, Select } from "@/components/ui";
import { DateInput } from "@/components/DateInput";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { Modal } from "@/components/Modal";
import { QtyInput } from "@/components/QtyInput";
import { formatDate, formatQty, localISODate } from "@/lib/format";
import {
  searchMaster,
  type Destination,
  type Driver,
  type EwuraLicense,
  type Transporter,
  type Truck,
} from "@/lib/master";
import { partyName, searchCountryDestinations, truckComboPlate } from "@/lib/ilr";
import type { LineRow, Party } from "./types";

type Mode = "add" | "edit" | "view" | null;

const emptyForm = {
  productId: "",
  customerOrderNumber: "",
  requestedQty: "",
  transporterId: "",
  transporterLabel: "",
  driverId: "",
  driverLabel: "",
  truckId: "",
  truckLabel: "",
  ewuraLicense: "",
  licenseLabel: "",
  destinationId: "",
  destinationLabel: "",
  districtId: "",
  districtLabel: "",
  expirationDate: "",
};

export function LinesCard({
  uid,
  draft,
  transit,
  ilrDate,
  expiryDays,
  products,
  lines,
  onSaved,
}: {
  uid: string;
  draft: boolean;
  transit: boolean;
  ilrDate: string;
  expiryDays: number;
  products: Party[];
  lines: LineRow[];
  onSaved: () => void;
}) {
  const autoProduct = products.length === 1;
  const expDefault = (() => {
    const d = new Date(`${ilrDate}T00:00:00`);
    d.setDate(d.getDate() + expiryDays);
    return localISODate(d);
  })();
  const [mode, setMode] = useState<Mode>(null);
  const [editing, setEditing] = useState<LineRow | null>(null);
  const [copying, setCopying] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [err, setErr] = useState("");
  const [saving, setSaving] = useState(false);

  function formFromLine(row: LineRow, orderNumber: string) {
    const truck = row.truck;
    return {
      productId: row.product?.id || products[0]?.id || "",
      customerOrderNumber: orderNumber,
      requestedQty: row.requestedQty,
      transporterId: row.transporter?.id || "",
      transporterLabel: row.transporterName || row.transporter?.name || "",
      driverId: row.driver?.id || "",
      driverLabel: row.driverName || row.driver?.name || "",
      truckId: truck?.id || "",
      truckLabel: truck ? truckComboPlate(truck) : row.truckPlate || "",
      ewuraLicense: row.ewuraLicense || "",
      licenseLabel: row.ewuraLicense || "",
      destinationId: row.toDestination?.id || "",
      destinationLabel: row.toDestination?.name || row.destination || "",
      districtId: row.toDistrict?.id || "",
      districtLabel: transit ? "N/A" : row.toDistrict?.name || row.district || "",
      expirationDate: row.expirationDate ? String(row.expirationDate).slice(0, 10) : expDefault,
    };
  }

  function openAdd() {
    setEditing(null);
    setCopying(false);
    setForm({
      ...emptyForm,
      productId: products[0]?.id || "",
      expirationDate: expDefault,
      districtLabel: transit ? "N/A" : "",
    });
    setErr("");
    setMode("add");
  }

  function openView(row: LineRow) {
    setEditing(row);
    setCopying(false);
    setMode("view");
  }

  function openEdit(row: LineRow) {
    setEditing(row);
    setCopying(false);
    setForm(formFromLine(row, row.customerOrderNumber || ""));
    setErr("");
    setMode("edit");
  }

  function openCopy(row: LineRow) {
    setEditing(null);
    setCopying(true);
    setForm(formFromLine(row, ""));
    setErr("");
    setMode("add");
  }

  async function onLicense(value: string, label: string) {
    setForm((f) => ({ ...f, ewuraLicense: value, licenseLabel: label }));
    if (!value || transit) return;
    try {
      const place = await api.get<{
        destinationId?: string;
        destination?: string;
        districtId?: string;
        district?: string;
        regionName?: string;
        districtName?: string;
      }>("/master/places/from-license", { number: value });
      setForm((f) => ({
        ...f,
        destinationId: place.destinationId || "",
        destinationLabel: place.destination || place.regionName || "",
        districtId: place.districtId || "",
        districtLabel: place.district || place.districtName || "",
      }));
    } catch {
      /* license text still saved */
    }
  }

  function validateDates(): string {
    if (form.expirationDate && form.expirationDate < ilrDate) {
      return "Expiry date cannot be before the order date.";
    }
    return "";
  }

  async function save() {
    const dateErr = validateDates();
    if (dateErr) {
      setErr(dateErr);
      return;
    }
    setErr("");
    setSaving(true);
    const body = {
      productId: autoProduct ? products[0]?.id : form.productId,
      customerOrderNumber: form.customerOrderNumber,
      requestedQty: form.requestedQty,
      transporterId: form.transporterId,
      driverId: form.driverId,
      truckId: form.truckId,
      ewuraLicense: form.ewuraLicense,
      destinationId: form.destinationId,
      districtId: transit ? "" : form.districtId,
      destinationName: form.destinationLabel,
      districtName: transit ? "N/A" : form.districtLabel,
      expirationDate: form.expirationDate,
    };
    try {
      if (mode === "edit" && editing) {
        await api.put(`/orders/loading/${uid}/lines/${editing.id}`, body);
      } else {
        await api.post(`/orders/loading/${uid}/lines`, body);
      }
      setMode(null);
      onSaved();
    } catch (e) {
      setErr(formatApiError(e, "Could not save loading instruction"));
    } finally {
      setSaving(false);
    }
  }

  async function remove(lid: string) {
    try {
      await api.del(`/orders/loading/${uid}/lines/${lid}`);
      onSaved();
    } catch (e) {
      setErr(formatApiError(e, "Could not remove line"));
    }
  }

  function linePlate(ln: LineRow): string {
    if (ln.truck) return truckComboPlate(ln.truck);
    return (
      [ln.horsePlate, ln.trailerOnePlate, ln.trailerTwoPlate].filter(Boolean).join("/") ||
      ln.truckPlate ||
      "—"
    );
  }

  return (
    <Card className="space-y-3 p-5">
      {err && !mode ? <p className="text-sm text-red-600">{err}</p> : null}
      <SectionTable
        title="Loading instructions"
        actions={
          draft ? (
            <Button variant="secondary" className="!px-3 !py-1.5" onClick={openAdd}>
              Add instruction
            </Button>
          ) : null
        }
        columns={[
          { label: "ILO" },
          { label: "Order no" },
          { label: "Truck" },
          { label: "Driver" },
          { label: "Destination" },
          { label: "Product" },
          { label: "Qty (L)", align: "right" },
          { label: "" },
        ]}
        empty="No loading instructions"
      >
        {lines.map((ln) => (
          <SectionRow key={ln.id} onClick={() => openView(ln)}>
            <td className="font-semibold">{ln.documentNumber}</td>
            <td>{ln.customerOrderNumber || "—"}</td>
            <td>{linePlate(ln)}</td>
            <td>{ln.driverName || "—"}</td>
            <td>{ln.destination || "—"}</td>
            <td>{partyName(ln.product)}</td>
            <td className="text-right tabular-nums">{formatQty(ln.requestedQty)}</td>
            <td className="text-right" onClick={(e) => e.stopPropagation()}>
              {draft ? (
                <div className="flex justify-end gap-2">
                  <Button variant="ghost" className="!px-2 !py-1" onClick={() => openCopy(ln)}>
                    Copy
                  </Button>
                  <Button variant="ghost" className="!px-2 !py-1" onClick={() => openEdit(ln)}>
                    Edit
                  </Button>
                  <Button variant="ghost" className="!px-2 !py-1" onClick={() => void remove(ln.id)}>
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
        size="lg"
        onClose={() => setMode(null)}
        title={
          mode === "add"
            ? copying
              ? "Copy loading instruction"
              : "Add loading instruction"
            : mode === "edit"
              ? "Edit loading instruction"
              : "Loading instruction"
        }
        footer={
          mode === "view" ? (
            <>
              {draft && editing ? (
                <>
                  <Button variant="secondary" onClick={() => editing && openCopy(editing)}>
                    Copy
                  </Button>
                  <Button variant="secondary" onClick={() => editing && openEdit(editing)}>
                    Edit
                  </Button>
                </>
              ) : null}
              <Button onClick={() => setMode(null)}>Close</Button>
            </>
          ) : (
            <>
              <Button variant="secondary" onClick={() => setMode(null)}>
                Cancel
              </Button>
              <Button onClick={() => void save()} loading={saving}>
                Save
              </Button>
            </>
          )
        }
      >
        {mode === "view" && editing ? (
          <div className="grid gap-3 sm:grid-cols-2 text-sm">
            <div>
              <div className="text-xs text-slate-500">ILO</div>
              <div>{editing.documentNumber}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Customer order number</div>
              <div>{editing.customerOrderNumber || "—"}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Product</div>
              <div>{partyName(editing.product)}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Quantity (L)</div>
              <div className="tabular-nums">{formatQty(editing.requestedQty)}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Truck</div>
              <div>{linePlate(editing)}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Driver</div>
              <div>{editing.driverName || "—"}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Hauler</div>
              <div>{editing.transporterName || "—"}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">EWURA license</div>
              <div>{editing.ewuraLicense || "—"}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Destination</div>
              <div>{editing.destination || "—"}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">District</div>
              <div>{editing.district || "—"}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Expiry</div>
              <div>{formatDate(editing.expirationDate)}</div>
            </div>
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {err ? <p className="sm:col-span-2 text-sm text-red-600">{err}</p> : null}
            {copying ? (
              <p className="sm:col-span-2 text-sm text-slate-600">
                Copied from another instruction. Customer order number is blank — enter the new one, then change anything else that differs.
              </p>
            ) : null}
            <Field label="Customer order number">
              <Input
                value={form.customerOrderNumber}
                onChange={(e) => setForm((f) => ({ ...f, customerOrderNumber: e.target.value }))}
                placeholder={copying ? "Enter the new customer order number" : undefined}
              />
            </Field>
            <Field label="Order date">
              <DateInput value={ilrDate} disabled />
            </Field>
            <Field label="Expiry date">
              <DateInput
                min={ilrDate}
                value={form.expirationDate}
                onChange={(expirationDate) => setForm((f) => ({ ...f, expirationDate }))}
              />
            </Field>
            <Field label="Product">
              <Select
                value={form.productId || products[0]?.id || ""}
                disabled={autoProduct}
                onChange={(e) => setForm((f) => ({ ...f, productId: e.target.value }))}
              >
                {products.map((p) => (
                  <option key={p.id} value={p.id}>
                    {partyName(p)}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Requested quantity (Ltrs)">
              <QtyInput
                value={form.requestedQty}
                onChange={(requestedQty) => setForm((f) => ({ ...f, requestedQty }))}
              />
            </Field>
            <Field label="Hauler">
              <AsyncSearchableSelect
                value={form.transporterId}
                selectedLabel={form.transporterLabel}
                onChange={(value, option) =>
                  setForm((f) => ({ ...f, transporterId: value, transporterLabel: option?.label || "" }))
                }
                loadOptions={async (q) => {
                  const rows = await searchMaster<Transporter>("/master/transporters", q);
                  return rows.map((r) => ({ value: r.id, label: r.name || r.id }));
                }}
              />
            </Field>
            <Field label="Driver">
              <AsyncSearchableSelect
                value={form.driverId}
                selectedLabel={form.driverLabel}
                onChange={(value, option) =>
                  setForm((f) => ({ ...f, driverId: value, driverLabel: option?.label || "" }))
                }
                loadOptions={async (q) => {
                  const rows = await searchMaster<Driver>("/master/drivers", q);
                  return rows.map((r) => ({ value: r.id, label: r.name || r.id }));
                }}
              />
            </Field>
            <Field label="Truck">
              <AsyncSearchableSelect
                value={form.truckId}
                selectedLabel={form.truckLabel}
                onChange={(value, option) =>
                  setForm((f) => ({ ...f, truckId: value, truckLabel: option?.label || "" }))
                }
                loadOptions={async (q) => {
                  const rows = await searchMaster<Truck>("/master/trucks", q);
                  return rows.map((r) => ({
                    value: r.id,
                    label: truckComboPlate(r),
                  }));
                }}
              />
            </Field>
            <Field label="EWURA license">
              <AsyncSearchableSelect
                value={form.ewuraLicense}
                selectedLabel={form.licenseLabel}
                onChange={(value, option) => void onLicense(value, option?.label || value)}
                loadOptions={async (q) => {
                  const rows = await api.get<EwuraLicense[]>("/ewura/licenses/options", {
                    q: q || undefined,
                  });
                  return (Array.isArray(rows) ? rows : []).map((l) => ({
                    value: l.licenseNumber,
                    label: `${l.licenseNumber} — ${l.licensee}`,
                  }));
                }}
              />
            </Field>
            <Field label="Destination">
              {transit ? (
                <AsyncSearchableSelect
                  value={form.destinationId}
                  selectedLabel={form.destinationLabel}
                  placeholder="Country"
                  onChange={(value, option) =>
                    setForm((f) => ({
                      ...f,
                      destinationId: value,
                      destinationLabel: option?.label || "",
                      districtId: "",
                      districtLabel: "N/A",
                    }))
                  }
                  loadOptions={async (q) => {
                    const rows = await searchCountryDestinations(q);
                    return rows.map((r: Destination) => ({ value: r.id, label: r.name || r.id }));
                  }}
                />
              ) : (
                <Input value={form.destinationLabel} disabled placeholder="From EWURA license" />
              )}
            </Field>
            <Field label="District">
              <Input
                value={transit ? "N/A" : form.districtLabel}
                disabled
                placeholder={transit ? "N/A" : "From EWURA license"}
              />
            </Field>
          </div>
        )}
      </Modal>
    </Card>
  );
}
