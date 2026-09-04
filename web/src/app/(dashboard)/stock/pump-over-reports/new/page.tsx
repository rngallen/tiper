"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { QtyInput } from "@/components/QtyInput";
import { Button, Card, ErrorBanner, Field, PageHeader } from "@/components/ui";
import { searchMaster } from "@/lib/master";
import type { PumpOverRequest } from "@/lib/pumpOver";

export default function NewPumpOverReportPage() {
  const router = useRouter();
  const params = useSearchParams();
  const preselect = params.get("request") || "";
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    requestId: preselect,
    requestLabel: "",
    actualDelivered: "",
    actualReceived: "",
  });

  useEffect(() => {
    if (!preselect) return;
    void api.get<PumpOverRequest>(`/orders/pump-over/${preselect}`).then((r) => {
      setForm((prev) => ({
        ...prev,
        requestId: r.id,
        requestLabel: `${r.documentNumber} — ${r.customer?.code || ""} ${r.quantity}`.trim(),
        actualDelivered: prev.actualDelivered || r.quantity,
      }));
    }).catch(() => undefined);
  }, [preselect]);

  async function save() {
    setError("");
    if (!form.requestId || !form.actualDelivered) {
      setError("Approved request and delivered quantity are required.");
      return;
    }
    setSaving(true);
    try {
      const created = await api.post<{ id: string }>("/orders/pump-over-reports", {
        requestId: form.requestId,
        actualDelivered: form.actualDelivered,
        actualReceived: form.actualReceived || form.actualDelivered,
      });
      router.push(`/stock/pump-over-reports/${created.id}`);
    } catch (e) {
      setError(formatApiError(e, "Could not create execution report"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="New pump-over report"
        subtitle="Post delivered and received quantities against an approved pump-over request"
        actions={
          <Link
            href="/stock/pump-over-reports"
            className="text-sm font-medium text-slate-600 hover:text-slate-900"
          >
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="grid max-w-3xl gap-4 p-6 sm:grid-cols-2">
        <Field label="Approved request">
          <AsyncSearchableSelect
            value={form.requestId}
            selectedLabel={form.requestLabel}
            placeholder="Search approved request"
            onChange={(value, option) =>
              setForm({ ...form, requestId: value, requestLabel: option?.label || "" })
            }
            loadOptions={async (q) =>
              (await searchMaster<PumpOverRequest>("/orders/pump-over", q, {
                status: "approved",
                allDates: true,
              })).map((r) => ({
                value: r.id,
                label: `${r.documentNumber} — ${r.customer?.code || ""} ${r.quantity}`.trim(),
              }))
            }
          />
        </Field>
        <Field label="Delivered (m³)">
          <QtyInput
            value={form.actualDelivered}
            onChange={(actualDelivered) => setForm({ ...form, actualDelivered })}
          />
        </Field>
        <Field label="Received (m³)">
          <QtyInput
            value={form.actualReceived}
            onChange={(actualReceived) => setForm({ ...form, actualReceived })}
          />
        </Field>
        <div className="sm:col-span-2 flex justify-end gap-2">
          <Button variant="secondary" onClick={() => router.push("/stock/pump-over-reports")}>
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
