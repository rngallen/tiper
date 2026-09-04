"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { QtyInput } from "@/components/QtyInput";
import { Button, Card, ErrorBanner, Field, Input, PageHeader, Select } from "@/components/ui";
import { DateInput } from "@/components/DateInput";
import { searchMaster } from "@/lib/master";
import { AMENDABLE_LINE_STATUSES, AmendmentKind, type OrderStatus } from "@/lib/domain";

type LineRef = { id: string; documentNumber?: string; requestedQty?: string; status?: string };

export default function NewAmendmentPage() {
  const router = useRouter();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    kind: AmendmentKind.Normal as AmendmentKind,
    iloId: "",
    iloLabel: "",
    requestedQty: "",
    expirationDate: "",
    notes: "",
  });

  async function save() {
    setError("");
    if (!form.iloId) {
      setError("Select the GLO to amend.");
      return;
    }
    setSaving(true);
    try {
      await api.post("/orders/amendments", {
        kind: form.kind,
        iloId: form.iloId,
        requestedQty: form.requestedQty,
        expirationDate: form.expirationDate,
        notes: form.notes,
      });
      router.push("/stock/amendments");
    } catch (e) {
      setError(formatApiError(e, "Could not create amendment"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="New loading amendment"
        subtitle="Extend and batch-cancel apply immediately; other kinds go to approval"
        actions={
          <Link href="/stock/amendments" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="grid max-w-3xl gap-4 p-6 sm:grid-cols-2">
        <Field label="Kind">
          <Select
            value={form.kind}
            onChange={(e) => setForm({ ...form, kind: e.target.value as AmendmentKind })}
          >
            <option value={AmendmentKind.Normal}>Normal (no increase)</option>
            <option value={AmendmentKind.QtyIncrease}>Quantity increase</option>
            <option value={AmendmentKind.QtyDecrease}>Quantity decrease</option>
            <option value={AmendmentKind.Product}>Product change</option>
            <option value={AmendmentKind.Cancel}>Cancel truck</option>
            <option value={AmendmentKind.BatchCancel}>Batch cancel (immediate)</option>
            <option value={AmendmentKind.Extend}>Extend expiration (immediate)</option>
          </Select>
        </Field>
        <Field label="GLO">
          <AsyncSearchableSelect
            value={form.iloId}
            selectedLabel={form.iloLabel}
            placeholder="Search open GLO"
            onChange={(value, option) =>
              setForm({ ...form, iloId: value, iloLabel: option?.label || "" })
            }
            loadOptions={async (q) =>
              (await searchMaster<LineRef>("/orders/loading-lines", q, { allDates: true }))
                .filter((l) => AMENDABLE_LINE_STATUSES.includes((l.status || "") as OrderStatus))
                .map((l) => ({
                  value: l.id,
                  label: `${l.documentNumber || l.id} — ${l.requestedQty || ""} (${l.status || ""})`,
                }))
            }
          />
        </Field>
        <Field label="New quantity">
          <QtyInput
            value={form.requestedQty}
            onChange={(requestedQty) => setForm({ ...form, requestedQty })}
          />
        </Field>
        <Field label="New expiry">
          <DateInput
            value={form.expirationDate}
            onChange={(expirationDate) => setForm({ ...form, expirationDate })}
          />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Notes">
            <Input value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
          </Field>
        </div>
        <div className="sm:col-span-2 flex justify-end gap-2">
          <Button variant="secondary" onClick={() => router.push("/stock/amendments")}>
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
