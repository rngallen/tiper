"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { QtyInput } from "@/components/QtyInput";
import { Button, Card, ErrorBanner, Field, Input, PageHeader, Textarea } from "@/components/ui";
import { DateInput } from "@/components/DateInput";
import { formatDate, formatQty, localISODate } from "@/lib/format";

type HoldParcel = {
  customerId: string;
  customerCode?: string;
  customerName?: string;
  productId: string;
  productCode?: string;
  productName?: string;
  vesselId: string;
  vesselCode?: string;
  vesselName?: string;
  vesselDate: string;
  stockStatusId: string;
  statusName?: string;
  quantity: string;
  cubicMeter?: string;
  metricTonne?: string;
};

function parcelKey(p: HoldParcel) {
  return `${p.customerId}:${p.productId}:${p.vesselId}:${p.vesselDate}:${p.stockStatusId}`;
}

export default function NewHoldReleasePage() {
  const router = useRouter();
  const today = localISODate(new Date());
  const parcels = useAsync(() => api.get<HoldParcel[]>("/ic/hold-parcels"), []);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const [form, setForm] = useState({ releaseDate: today, description: "", notes: "" });
  const [picked, setPicked] = useState<Record<string, string>>({});

  const rows = useMemo(() => {
    const all = Array.isArray(parcels.data) ? parcels.data : [];
    const q = search.trim().toLowerCase();
    if (!q) return all;
    return all.filter((p) =>
      [p.customerCode, p.customerName, p.productCode, p.vesselCode, p.vesselName, p.statusName]
        .join(" ")
        .toLowerCase()
        .includes(q),
    );
  }, [parcels.data, search]);

  async function save() {
    setError("");
    const lines = Object.entries(picked)
      .filter(([, qty]) => Number(qty) > 0)
      .map(([key, quantity]) => {
        const p = (parcels.data || []).find((row) => parcelKey(row) === key);
        if (!p) return null;
        const hold = Number(p.quantity);
        const qty = Number(quantity);
        const ratio = hold > 0 ? qty / hold : 0;
        return {
          customerId: p.customerId,
          productId: p.productId,
          vesselId: p.vesselId,
          vesselDate: `${String(p.vesselDate).slice(0, 10)}T00:00:00Z`,
          stockStatusId: p.stockStatusId,
          quantity,
          cubicMeter: p.cubicMeter ? String(Number(p.cubicMeter) * ratio) : undefined,
          metricTonne: p.metricTonne ? String(Number(p.metricTonne) * ratio) : undefined,
        };
      })
      .filter(Boolean);
    if (!form.description.trim()) {
      setError("Description is required.");
      return;
    }
    if (lines.length === 0) {
      setError("Select at least one held parcel and enter the paid quantity.");
      return;
    }
    setSaving(true);
    try {
      const created = await api.post<{ id: string }>("/ic/hold-releases", {
        releaseDate: `${form.releaseDate}T00:00:00Z`,
        description: form.description.trim(),
        notes: form.notes.trim(),
        lines,
      });
      router.push(`/stock/financial-hold/${created.id}`);
    } catch (e) {
      setError(formatApiError(e, "Could not create hold release"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="New financial hold release"
        subtitle="Record which parcel has been paid and at what quantity. Approval moves that volume from hold onto free stock."
        actions={
          <Link href="/stock/financial-hold" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="grid max-w-3xl gap-4 p-6 sm:grid-cols-2">
        <Field label="Date">
          <DateInput value={form.releaseDate} onChange={(releaseDate) => setForm({ ...form, releaseDate })} />
        </Field>
        <Field label="Description">
          <Input
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
            placeholder="e.g. Invoice 4521 paid"
          />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Notes">
            <Textarea value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} rows={3} />
          </Field>
        </div>
      </Card>
      <Card className="space-y-4 p-6">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="text-sm font-semibold text-slate-900">Open financial hold</h2>
            <p className="text-sm text-slate-600">Enter litres paid on each parcel. Leave blank to skip.</p>
          </div>
          <Field label="Search parcels">
            <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Customer, vessel, product" />
          </Field>
        </div>
        {parcels.loading ? <p className="text-sm text-slate-500">Loading hold stock…</p> : null}
        {parcels.error ? <ErrorBanner message={parcels.error} /> : null}
        <div className="overflow-x-auto rounded-lg border border-slate-200">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-left text-xs uppercase text-slate-500">
              <tr>
                <th className="px-3 py-2">Customer</th>
                <th className="px-3 py-2">Product</th>
                <th className="px-3 py-2">Vessel</th>
                <th className="px-3 py-2">Vessel date</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2 text-right">On hold (L)</th>
                <th className="px-3 py-2 text-right">Release (L)</th>
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-slate-500" colSpan={7}>
                    No parcels currently on financial hold
                  </td>
                </tr>
              ) : (
                rows.map((p) => {
                  const key = parcelKey(p);
                  return (
                    <tr key={key} className="border-t border-slate-100">
                      <td className="px-3 py-2">{p.customerName || p.customerCode}</td>
                      <td className="px-3 py-2">{p.productCode}</td>
                      <td className="px-3 py-2">{p.vesselName || p.vesselCode}</td>
                      <td className="px-3 py-2">{formatDate(p.vesselDate)}</td>
                      <td className="px-3 py-2">{p.statusName || "—"}</td>
                      <td className="px-3 py-2 text-right tabular-nums">{formatQty(p.quantity)}</td>
                      <td className="px-3 py-2">
                        <QtyInput
                          value={picked[key] || ""}
                          onChange={(quantity) =>
                            setPicked((prev) => {
                              const next = { ...prev };
                              if (!quantity) delete next[key];
                              else next[key] = quantity;
                              return next;
                            })
                          }
                        />
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => router.push("/stock/financial-hold")}>
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
