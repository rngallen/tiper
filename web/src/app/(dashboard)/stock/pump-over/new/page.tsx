"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { QtyInput } from "@/components/QtyInput";
import { Button, Card, ErrorBanner, Field, Input, PageHeader } from "@/components/ui";
import { DateInput } from "@/components/DateInput";
import { localISODate } from "@/lib/format";
import { masterOption, searchMaster, type MasterRef } from "@/lib/master";
import { stockStatusLabel, type StockStatusRef } from "@/lib/status";
import { emptyPumpOverVessel } from "@/lib/pumpOver";

export default function NewPumpOverPage() {
  const router = useRouter();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const today = localISODate(new Date());
  const [form, setForm] = useState({
    orderDate: today,
    customerId: "",
    customerLabel: "",
    productId: "",
    productLabel: "",
    depotId: "",
    depotLabel: "",
    customerOrderNumber: "",
    vessels: [emptyPumpOverVessel(today)],
  });

  function patch(p: Partial<typeof form>) {
    setForm((prev) => ({ ...prev, ...p }));
  }

  async function save() {
    setError("");
    const vessels = form.vessels.filter((v) => v.vesselId && v.stockStatusId && v.quantity);
    if (!form.customerId || !form.productId || !form.depotId) {
      setError("Customer, product, and receiving depot are required.");
      return;
    }
    if (vessels.length === 0) {
      setError("Add at least one vessel with stock status and quantity.");
      return;
    }
    const quantity = vessels.reduce((sum, v) => sum + Number(v.quantity || 0), 0).toString();
    setSaving(true);
    try {
      const created = await api.post<{ id: string }>("/orders/pump-over", {
        orderDate: `${form.orderDate}T00:00:00Z`,
        customerId: form.customerId,
        productId: form.productId,
        depotId: form.depotId,
        quantity,
        customerOrderNumber: form.customerOrderNumber,
        vessels: vessels.map((v) => ({
          vesselId: v.vesselId,
          vesselDate: v.vesselDate,
          stockStatusId: v.stockStatusId,
          quantity: v.quantity,
        })),
      });
      router.push(`/stock/pump-over/${created.id}`);
    } catch (e) {
      setError(formatApiError(e, "Could not create pump-over request"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="New pump-over request"
        subtitle="Save the request, then submit it for approval. Execution quantities are posted as a separate report."
        actions={
          <Link href="/stock/pump-over" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-6">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Request date">
            <DateInput value={form.orderDate} onChange={(orderDate) => patch({ orderDate })} />
          </Field>
          <Field label="Customer order">
            <Input
              value={form.customerOrderNumber}
              onChange={(e) => patch({ customerOrderNumber: e.target.value })}
            />
          </Field>
          <Field label="Customer">
            <AsyncSearchableSelect
              value={form.customerId}
              selectedLabel={form.customerLabel}
              placeholder="Search customer"
              onChange={(value, option) => patch({ customerId: value, customerLabel: option?.label || "" })}
              loadOptions={async (q) => (await searchMaster<MasterRef>("/master/customers", q)).map(masterOption)}
            />
          </Field>
          <Field label="Product">
            <AsyncSearchableSelect
              value={form.productId}
              selectedLabel={form.productLabel}
              placeholder="Search product"
              onChange={(value, option) => patch({ productId: value, productLabel: option?.label || "" })}
              loadOptions={async (q) => (await searchMaster<MasterRef>("/master/products", q)).map(masterOption)}
            />
          </Field>
          <Field label="Receiving depot">
            <AsyncSearchableSelect
              value={form.depotId}
              selectedLabel={form.depotLabel}
              placeholder="Search depot"
              onChange={(value, option) => patch({ depotId: value, depotLabel: option?.label || "" })}
              loadOptions={async (q) => (await searchMaster<MasterRef>("/master/depots", q)).map(masterOption)}
            />
          </Field>
        </div>
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium text-slate-800">Vessels</h3>
            <Button
              variant="secondary"
              className="!px-3 !py-1"
              onClick={() =>
                patch({ vessels: [...form.vessels, emptyPumpOverVessel(form.orderDate)] })
              }
            >
              Add vessel
            </Button>
          </div>
          <p className="text-xs text-slate-500">
            Each vessel can have its own stock status (Local, Transit, Congo, …).
          </p>
          {form.vessels.map((line, i) => (
            <div key={i} className="grid gap-3 rounded-xl border border-slate-100 bg-slate-50 p-3 sm:grid-cols-2">
              <Field label="Vessel">
                <AsyncSearchableSelect
                  value={line.vesselId}
                  selectedLabel={line.vesselLabel}
                  placeholder="Search vessel"
                  onChange={(value, option) => {
                    patch({
                      vessels: form.vessels.map((v, j) =>
                        j === i ? { ...v, vesselId: value, vesselLabel: option?.label || "" } : v,
                      ),
                    });
                  }}
                  loadOptions={async (q) => (await searchMaster<MasterRef>("/master/vessels", q)).map(masterOption)}
                />
              </Field>
              <Field label="Stock status">
                <AsyncSearchableSelect
                  value={line.stockStatusId}
                  selectedLabel={line.stockStatusLabel}
                  placeholder="Search status"
                  onChange={(value, option) => {
                    patch({
                      vessels: form.vessels.map((v, j) =>
                        j === i ? { ...v, stockStatusId: value, stockStatusLabel: option?.label || "" } : v,
                      ),
                    });
                  }}
                  loadOptions={async (q) =>
                    (await searchMaster<StockStatusRef>("/master/statuses", q)).map((s) => ({
                      value: s.id,
                      label: stockStatusLabel(s),
                    }))
                  }
                />
              </Field>
              <Field label="Vessel date">
                <DateInput
                  value={line.vesselDate}
                  onChange={(vesselDate) => {
                    patch({
                      vessels: form.vessels.map((v, j) =>
                        j === i ? { ...v, vesselDate } : v,
                      ),
                    });
                  }}
                />
              </Field>
              <Field label="Quantity (m³)">
                <QtyInput
                  value={line.quantity}
                  onChange={(quantity) => {
                    patch({
                      vessels: form.vessels.map((v, j) => (j === i ? { ...v, quantity } : v)),
                    });
                  }}
                />
              </Field>
              {form.vessels.length > 1 ? (
                <div className="sm:col-span-2">
                  <Button
                    variant="secondary"
                    className="!px-3 !py-1"
                    onClick={() => patch({ vessels: form.vessels.filter((_, j) => j !== i) })}
                  >
                    Remove
                  </Button>
                </div>
              ) : null}
            </div>
          ))}
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => router.push("/stock/pump-over")}>
            Cancel
          </Button>
          <Button onClick={() => void save()} loading={saving}>
            Create
          </Button>
        </div>
      </Card>
    </div>
  );
}
