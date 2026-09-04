"use client";

import { useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { SectionRow, SectionTable } from "@/components/SectionTable";
import { Button, Card, Field, Input, Select } from "@/components/ui";
import { DateInput } from "@/components/DateInput";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { Modal } from "@/components/Modal";
import { QtyInput } from "@/components/QtyInput";
import { formatDate, formatQty } from "@/lib/format";
import { searchMaster, type Vessel } from "@/lib/master";
import { partyName } from "@/lib/ilr";
import type { Party, VesselRow } from "./types";

type Mode = "add" | "edit" | "view" | null;

const emptyForm = {
  vesselId: "",
  vesselLabel: "",
  productId: "",
  vesselDate: "",
  quantity: "",
  financialHold: false,
};

export function VesselsCard({
  uid,
  draft,
  ilrDate,
  products,
  vessels,
  onSaved,
}: {
  uid: string;
  draft: boolean;
  ilrDate: string;
  products: Party[];
  vessels: VesselRow[];
  onSaved: () => void;
}) {
  const [mode, setMode] = useState<Mode>(null);
  const [editing, setEditing] = useState<VesselRow | null>(null);
  const [form, setForm] = useState(emptyForm);
  const [err, setErr] = useState("");
  const [saving, setSaving] = useState(false);

  function openAdd() {
    setEditing(null);
    setForm({
      ...emptyForm,
      productId: products[0]?.id || "",
      vesselDate: ilrDate,
    });
    setErr("");
    setMode("add");
  }

  function openView(row: VesselRow) {
    setEditing(row);
    setMode("view");
  }

  function openEdit(row: VesselRow) {
    setEditing(row);
    setForm({
      vesselId: row.vessel?.id || "",
      vesselLabel: partyName(row.vessel),
      productId: row.product?.id || products[0]?.id || "",
      vesselDate: String(row.vesselDate).slice(0, 10),
      quantity: row.quantity,
      financialHold: row.financialHold,
    });
    setErr("");
    setMode("edit");
  }

  async function save() {
    setErr("");
    setSaving(true);
    const body = {
      vesselId: form.vesselId,
      productId: form.productId || products[0]?.id,
      vesselDate: form.vesselDate,
      quantity: form.quantity,
      financialHold: form.financialHold,
    };
    try {
      if (mode === "edit" && editing) {
        await api.put(`/orders/loading/${uid}/vessels/${editing.id}`, body);
      } else {
        await api.post(`/orders/loading/${uid}/vessels`, body);
      }
      setMode(null);
      onSaved();
    } catch (e) {
      setErr(formatApiError(e, "Could not save vessel"));
    } finally {
      setSaving(false);
    }
  }

  async function remove(id: string) {
    try {
      await api.del(`/orders/loading/${uid}/vessels/${id}`);
      onSaved();
    } catch (e) {
      setErr(formatApiError(e, "Could not remove vessel"));
    }
  }

  return (
    <Card className="space-y-3 p-5">
      {err && !mode ? <p className="text-sm text-red-600">{err}</p> : null}
      <SectionTable
        title="Vessels"
        actions={
          draft ? (
            <Button variant="secondary" className="!px-3 !py-1.5" onClick={openAdd}>
              Add vessel
            </Button>
          ) : null
        }
        columns={[
          { label: "Date" },
          { label: "Vessel" },
          { label: "Product" },
          { label: "Hold" },
          { label: "Qty (L)", align: "right" },
          { label: "" },
        ]}
        empty="No vessels"
      >
        {vessels.map((v) => (
          <SectionRow key={v.id} onClick={() => openView(v)}>
            <td>{formatDate(v.vesselDate)}</td>
            <td className="font-semibold">{v.vessel?.name || v.vessel?.code || "—"}</td>
            <td>{partyName(v.product)}</td>
            <td>{v.financialHold ? "Yes" : "No"}</td>
            <td className="text-right tabular-nums">{formatQty(v.quantity)}</td>
            <td className="text-right" onClick={(e) => e.stopPropagation()}>
              {draft ? (
                <div className="flex justify-end gap-2">
                  <Button variant="ghost" className="!px-2 !py-1" onClick={() => openEdit(v)}>
                    Edit
                  </Button>
                  <Button variant="ghost" className="!px-2 !py-1" onClick={() => void remove(v.id)}>
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
        title={mode === "add" ? "Add vessel" : mode === "edit" ? "Edit vessel" : "Vessel"}
        footer={
          mode === "view" ? (
            <>
              {draft && editing ? (
                <Button variant="secondary" onClick={() => editing && openEdit(editing)}>
                  Edit
                </Button>
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
              <div className="text-xs text-slate-500">Vessel date</div>
              <div>{formatDate(editing.vesselDate)}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Vessel</div>
              <div>{editing.vessel?.name || editing.vessel?.code}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Product</div>
              <div>{partyName(editing.product)}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Quantity (L)</div>
              <div className="tabular-nums">{formatQty(editing.quantity)}</div>
            </div>
            <div>
              <div className="text-xs text-slate-500">Financial hold</div>
              <div>{editing.financialHold ? "Yes" : "No"}</div>
            </div>
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {err ? <p className="sm:col-span-2 text-sm text-red-600">{err}</p> : null}
            <Field label="Vessel">
              <AsyncSearchableSelect
                value={form.vesselId}
                selectedLabel={form.vesselLabel}
                placeholder="Vessel"
                onChange={(value, option) =>
                  setForm((f) => ({ ...f, vesselId: value, vesselLabel: option?.label || "" }))
                }
                loadOptions={async (q) => {
                  const rows = await searchMaster<Vessel>("/master/vessels", q);
                  return rows.map((r) => ({ value: r.id, label: r.name || r.code || r.id }));
                }}
              />
            </Field>
            <Field label="Product">
              <Select
                value={form.productId || products[0]?.id || ""}
                onChange={(e) => setForm((f) => ({ ...f, productId: e.target.value }))}
              >
                {products.map((p) => (
                  <option key={p.id} value={p.id}>
                    {partyName(p)}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Vessel date">
              <DateInput
                max={ilrDate}
                value={form.vesselDate}
                onChange={(vesselDate) => setForm((f) => ({ ...f, vesselDate }))}
              />
            </Field>
            <Field label="Quantity (Ltrs)">
              <QtyInput value={form.quantity} onChange={(quantity) => setForm((f) => ({ ...f, quantity }))} />
            </Field>
            <Field label="Financial hold">
              <Select
                value={form.financialHold ? "yes" : "no"}
                onChange={(e) => setForm((f) => ({ ...f, financialHold: e.target.value === "yes" }))}
              >
                <option value="no">No</option>
                <option value="yes">Yes</option>
              </Select>
            </Field>
          </div>
        )}
      </Modal>
    </Card>
  );
}
