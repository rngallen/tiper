"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { useAuth } from "@/lib/auth";
import { PRICE_WRITE_PERMS } from "@/lib/permissions";
import { QtyInput } from "@/components/QtyInput";
import { DocumentMeta, ReviewTabBar, ErpField, StatusPill } from "@/components/ApprovalReview";
import { Button, Card, ErrorBanner, Field, PageHeader, Select, Spinner } from "@/components/ui";
import { DateInput } from "@/components/ui";
import { feeEditable, isoDay, type ExchangeRate } from "@/lib/billing";
import { creatorLabel } from "@/lib/creator";
import { BatchFiles } from "../../_components/BatchFiles";
import { BatchWorkflow } from "../../_components/BatchWorkflow";

const writePerms = PRICE_WRITE_PERMS;

export default function FxDocumentPage() {
  const { id } = useParams<{ id: string }>();
  const path = `/billing/fx/${id}`;
  const doc = useAsync(() => api.get<ExchangeRate>(path), [id]);
  const { hasPermission } = useAuth();
  const canWrite = hasPermission(...writePerms);
  const [tab, setTab] = useState<"rate" | "files" | "workflow">("rate");
  const [effectiveFrom, setFrom] = useState("");
  const [fromCurrency, setFromCcy] = useState("USD");
  const [toCurrency, setToCcy] = useState("TZS");
  const [rate, setRate] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const row = doc.data;
    if (!row) return;
    setFrom(isoDay(row.effectiveFrom));
    setFromCcy(row.fromCurrency || "USD");
    setToCcy(row.toCurrency || "TZS");
    setRate(row.rate || "");
  }, [doc.data]);

  const row = doc.data;
  const draft = feeEditable(row?.status) && canWrite;

  async function save() {
    setError("");
    setSaving(true);
    try {
      await api.put(path, { effectiveFrom, fromCurrency, toCurrency, rate: rate.trim() });
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not save rate"));
    } finally {
      setSaving(false);
    }
  }

  async function submit() {
    setError("");
    try {
      await api.post(`${path}/submit`, {});
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not submit"));
    }
  }

  if (doc.loading && !row) return <Spinner />;
  if (!row) {
    return (
      <div className="space-y-4">
        <ErrorBanner message={doc.error || "Exchange rate not found"} />
        <Link href="/finance/setups/fx" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title={`${row.fromCurrency}/${row.toCurrency}`}
        subtitle="Approved quote used by FSF, VSF, KOJ, and TBS · USD = TZS ÷ rate"
        actions={
          <Link href="/finance/setups/fx" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-6">
        <DocumentMeta crumb="Finance · Setups · Exchange rate">
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        <ReviewTabBar
          tab={tab}
          onChange={setTab}
          ariaLabel="Exchange rate sections"
          ids={[
            { id: "rate", label: "Rate" },
            { id: "files", label: "Attachments" },
            { id: "workflow", label: "Workflow" },
          ]}
        />
        {tab === "rate" ? (
          <div className="max-w-lg space-y-3">
            <ErpField label="Created by" value={creatorLabel(row.createdBy)} />
            <Field label="Effective from">
              <DateInput value={effectiveFrom} onChange={setFrom} disabled={!draft} />
            </Field>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field label="From">
                <Select value={fromCurrency} onChange={(e) => setFromCcy(e.target.value)} disabled={!draft}>
                  <option value="USD">USD</option>
                  <option value="TZS">TZS</option>
                </Select>
              </Field>
              <Field label="To">
                <Select value={toCurrency} onChange={(e) => setToCcy(e.target.value)} disabled={!draft}>
                  <option value="TZS">TZS</option>
                  <option value="USD">USD</option>
                </Select>
              </Field>
            </div>
            <Field label="Rate (TZS per 1 USD when USD→TZS)">
              <QtyInput unit="fx" value={rate} onChange={setRate} disabled={!draft} />
            </Field>
            {draft ? (
              <div className="flex flex-wrap gap-2">
                <Button onClick={() => void save()} loading={saving}>
                  Save
                </Button>
                <Button variant="secondary" onClick={() => void submit()}>
                  Submit for approval
                </Button>
              </div>
            ) : null}
          </div>
        ) : null}
        {tab === "files" ? <BatchFiles path={path} draft={draft} /> : null}
        {tab === "workflow" ? <BatchWorkflow path={path} /> : null}
      </Card>
    </div>
  );
}
