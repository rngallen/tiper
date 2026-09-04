"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { Button, Card, ErrorBanner, PageHeader } from "@/components/ui";
import { localISODate } from "@/lib/format";
import { fetchLookups } from "@/lib/lookups";
import { useAsync } from "@/lib/hooks";
import { ReceiptHeaderForm } from "../_components/ReceiptHeaderForm";
import {
  emptyReceiptHeader,
  firstHeaderError,
  receiptHeaderPayload,
  validateReceiptHeader,
} from "../_lib/header";

export default function NewInternalReceiptPage() {
  const router = useRouter();
  const catalogs = useAsync(fetchLookups, []);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState(() => ({
    ...emptyReceiptHeader(),
    date: localISODate(new Date()),
    vesselDate: localISODate(new Date()),
    isProvision: true,
  }));
  const fieldErrors = validateReceiptHeader(form, "internal");

  async function save() {
    setError("");
    if (Object.keys(fieldErrors).length) {
      setError(firstHeaderError(fieldErrors));
      return;
    }
    setSaving(true);
    try {
      const created = await api.post<{ id: string }>("/ic/receipts", {
        ...receiptHeaderPayload(form, "internal"),
        details: [],
      });
      router.push(`/stock/receipts/${created.id}/edit`);
    } catch (e) {
      setError(formatApiError(e, "Could not create receipt"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="New internal receipt"
        subtitle="Complete every header field, then add customer parcels on the document"
        actions={
          <Link href="/stock/receipts" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-5 p-6">
        <p className="text-xs text-slate-500">
          Fields marked <span className="font-semibold text-red-600">*</span> are required.
        </p>
        <ReceiptHeaderForm
          kind="internal"
          form={form}
          errors={error ? fieldErrors : {}}
          catalogs={catalogs.data ?? undefined}
          onChange={setForm}
        />
        <div className="flex justify-end gap-2 border-t border-slate-100 pt-4">
          <Button variant="secondary" onClick={() => router.push("/stock/receipts")}>
            Cancel
          </Button>
          <Button loading={saving} onClick={() => void save()}>
            Create
          </Button>
        </div>
      </Card>
    </div>
  );
}
