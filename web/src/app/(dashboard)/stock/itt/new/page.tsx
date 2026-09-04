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

export default function NewITTPage() {
  const router = useRouter();
  const today = localISODate(new Date());
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    transferDate: today,
    fromCustomerId: "",
    fromCustomerLabel: "",
    toCustomerId: "",
    toCustomerLabel: "",
    productId: "",
    productLabel: "",
    vesselId: "",
    vesselLabel: "",
    vesselDate: today,
    stockStatusId: "",
    stockStatusLabel: "",
    quantity: "",
  });

  function patch(p: Partial<typeof form>) {
    setForm((prev) => ({ ...prev, ...p }));
  }

  async function save() {
    setError("");
    if (!form.fromCustomerId || !form.toCustomerId || !form.productId || !form.vesselId || !form.stockStatusId || !form.quantity) {
      setError("From, to, product, vessel, status, and quantity are required.");
      return;
    }
    if (form.fromCustomerId === form.toCustomerId) {
      setError("From and to customers must be different.");
      return;
    }
    setSaving(true);
    try {
      const created = await api.post<{ id: string }>("/ic/itt", {
        transferDate: `${form.transferDate}T00:00:00Z`,
        fromCustomerId: form.fromCustomerId,
        toCustomerId: form.toCustomerId,
        productId: form.productId,
        vesselId: form.vesselId,
        vesselDate: form.vesselDate,
        stockStatusId: form.stockStatusId,
        quantity: form.quantity,
      });
      router.push(`/stock/itt/${created.id}`);
    } catch (e) {
      setError(formatApiError(e, "Could not create in-tank transfer"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="New in-tank transfer"
        subtitle="Change ownership of a parcel without moving product off the terminal. On approval the ledger posts two dated lines (sender out, receiver in). Receipt figures stay as received; later storage billing follows the receiver for the transferred quantity."
        actions={
          <Link href="/stock/itt" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="grid max-w-3xl gap-4 p-6 sm:grid-cols-2">
        <Field label="Date">
          <DateInput value={form.transferDate} onChange={(transferDate) => patch({ transferDate })} />
        </Field>
        <Field label="Quantity (m³)">
          <QtyInput value={form.quantity} onChange={(quantity) => patch({ quantity })} />
        </Field>
        <Field label="From customer">
          <AsyncSearchableSelect
            value={form.fromCustomerId}
            selectedLabel={form.fromCustomerLabel}
            placeholder="Search customer"
            onChange={(value, option) => patch({ fromCustomerId: value, fromCustomerLabel: option?.label || "" })}
            loadOptions={async (q) => (await searchMaster<MasterRef>("/master/customers", q)).map(masterOption)}
          />
        </Field>
        <Field label="To customer">
          <AsyncSearchableSelect
            value={form.toCustomerId}
            selectedLabel={form.toCustomerLabel}
            placeholder="Search customer"
            onChange={(value, option) => patch({ toCustomerId: value, toCustomerLabel: option?.label || "" })}
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
        <Field label="Status">
          <AsyncSearchableSelect
            value={form.stockStatusId}
            selectedLabel={form.stockStatusLabel}
            placeholder="Search status"
            onChange={(value, option) => patch({ stockStatusId: value, stockStatusLabel: option?.label || "" })}
            loadOptions={async (q) =>
              (await searchMaster<StockStatusRef>("/master/statuses", q)).map((s) => ({
                value: s.id,
                label: stockStatusLabel(s),
              }))
            }
          />
        </Field>
        <Field label="Vessel">
          <AsyncSearchableSelect
            value={form.vesselId}
            selectedLabel={form.vesselLabel}
            placeholder="Search vessel"
            onChange={(value, option) => patch({ vesselId: value, vesselLabel: option?.label || "" })}
            loadOptions={async (q) => (await searchMaster<MasterRef>("/master/vessels", q)).map(masterOption)}
          />
        </Field>
        <Field label="Vessel date">
          <DateInput value={form.vesselDate} onChange={(vesselDate) => patch({ vesselDate })} />
        </Field>
        <div className="sm:col-span-2 flex justify-end gap-2">
          <Button variant="secondary" onClick={() => router.push("/stock/itt")}>
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
