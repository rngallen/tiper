"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { Button, Card, Field, Input, PageHeader, ToggleField } from "@/components/ui";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { QtyInput } from "@/components/QtyInput";
import { stockStatusLabel, type StockStatusRef } from "@/lib/status";
import { searchMaster, type Customer } from "@/lib/master";
import { asRows } from "@/lib/pagination";
import { localISODate } from "@/lib/format";
import { DateInput } from "@/components/DateInput";
import { partyName, searchILRProducts } from "@/lib/ilr";

function partyLabel(row: { code?: string; name?: string }): string {
  return partyName(row) === "—" ? "" : partyName(row);
}

export default function NewLoadingPage() {
  const router = useRouter();
  const statuses = useAsync(
    () => api.get<StockStatusRef[] | { items?: StockStatusRef[] }>("/master/statuses"),
    [],
  );
  const statusList = useMemo(
    () => asRows<StockStatusRef>(statuses.data),
    [statuses.data],
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    orderDate: localISODate(new Date()),
    description: "",
    customerId: "",
    customerLabel: "",
    productId: "",
    productLabel: "",
    byProductId: "",
    byProductLabel: "",
    stockStatusId: "",
    stockStatusLabel: "",
    quantity: "",
    byProductQuantity: "",
    loadingOrderAvailable: false,
    validContract: false,
  });

  const selectedStatus = statusList.find((s) => s.id === form.stockStatusId);
  const transit = Boolean(selectedStatus?.isTransit);

  function patch(p: Partial<typeof form>) {
    setForm((prev) => ({ ...prev, ...p }));
  }

  async function save() {
    setError("");
    if (!form.description.trim()) {
      setError("Description is required.");
      return;
    }
    if (!form.customerId || !form.productId || !form.stockStatusId || !form.quantity) {
      setError("Customer, product, product status, and quantity are required.");
      return;
    }
    if (!transit && form.byProductId && !form.byProductQuantity) {
      setError("By-product quantity is required.");
      return;
    }
    setSaving(true);
    try {
      const created = await api.post<{ id: string }>("/orders/loading", {
        orderDate: `${form.orderDate}T00:00:00Z`,
        description: form.description.trim(),
        customerId: form.customerId,
        productId: form.productId,
        byProductId: transit ? "" : form.byProductId,
        byProductQuantity: transit || !form.byProductId ? "0" : form.byProductQuantity,
        stockStatusId: form.stockStatusId,
        quantity: form.quantity,
        loadingOrderAvailable: form.loadingOrderAvailable,
        validContract: form.validContract,
      });
      router.push(`/stock/loading/${created.id}/edit`);
    } catch (e) {
      setError(formatApiError(e, "Could not create internal loading request"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="New internal loading request"
        subtitle="Save the header, then complete vessels, stock, trucks, and attachments"
        actions={
          <Link href="/stock/loading" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {error ? (
        <p className="rounded-xl border border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700">{error}</p>
      ) : null}
      <Card className="space-y-4 p-6">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Request date">
            <DateInput value={form.orderDate} onChange={(orderDate) => patch({ orderDate })} />
          </Field>
          <Field label="Description">
            <Input value={form.description} onChange={(e) => patch({ description: e.target.value })} />
          </Field>
          <Field label="Customer">
            <AsyncSearchableSelect
              value={form.customerId}
              selectedLabel={form.customerLabel}
              placeholder="Search customer"
              onChange={(value, option) => patch({ customerId: value, customerLabel: option?.label || "" })}
              loadOptions={async (q) => {
                const rows = await searchMaster<Customer>("/master/customers", q);
                return rows.map((r) => ({ value: r.id, label: r.name || partyLabel(r) }));
              }}
            />
          </Field>
          <Field label="Product status">
            <AsyncSearchableSelect
              value={form.stockStatusId}
              selectedLabel={form.stockStatusLabel}
              placeholder="Local / transit / …"
              onChange={(value, option) =>
                patch({
                  stockStatusId: value,
                  stockStatusLabel: option?.label || "",
                  ...(statusList.find((s) => s.id === value)?.isTransit
                    ? { byProductId: "", byProductLabel: "", byProductQuantity: "" }
                    : {}),
                })
              }
              loadOptions={async (q) =>
                statusList
                  .filter((s) => !q || stockStatusLabel(s).toLowerCase().includes(q.toLowerCase()))
                  .map((s) => ({ value: s.id, label: stockStatusLabel(s) }))
              }
            />
          </Field>
          <Field label="Product">
            <AsyncSearchableSelect
              value={form.productId}
              selectedLabel={form.productLabel}
              placeholder="AGO or PMS"
              onChange={(value, option) =>
                patch({
                  productId: value,
                  productLabel: option?.label || "",
                  byProductId: "",
                  byProductLabel: "",
                  byProductQuantity: "",
                })
              }
              loadOptions={async (q) => {
                const rows = await searchILRProducts(q);
                return rows.map((r) => ({ value: r.id, label: partyLabel(r) }));
              }}
            />
          </Field>
          <Field label="Requested quantity (Ltrs)">
            <QtyInput value={form.quantity} onChange={(quantity) => patch({ quantity })} />
          </Field>
          {!transit ? (
            <>
              <Field label="By-product (optional)">
                <AsyncSearchableSelect
                  value={form.byProductId}
                  selectedLabel={form.byProductLabel}
                  placeholder={form.productId ? "The other gantry grade" : "Select product first"}
                  onChange={(value, option) =>
                    patch({ byProductId: value, byProductLabel: option?.label || "" })
                  }
                  loadOptions={async (q) => {
                    const rows = await searchILRProducts(q, form.productId || undefined);
                    return rows.map((r) => ({ value: r.id, label: partyLabel(r) }));
                  }}
                />
              </Field>
              <Field label="By-product quantity (Ltrs)">
                <QtyInput
                  value={form.byProductQuantity}
                  onChange={(byProductQuantity) => patch({ byProductQuantity })}
                  disabled={!form.byProductId}
                />
              </Field>
            </>
          ) : null}
          <div className="sm:col-span-2 grid gap-4 sm:grid-cols-2">
            <ToggleField
              label="Customer loading order available"
              checked={form.loadingOrderAvailable}
              onChange={(v) => patch({ loadingOrderAvailable: v })}
            />
            <ToggleField
              label="Valid contract available"
              checked={form.validContract}
              onChange={(v) => patch({ validContract: v })}
            />
          </div>
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => router.push("/stock/loading")}>
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
