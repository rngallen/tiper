"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { api, formatApiError } from "@/lib/api";
import { notify } from "@/lib/notify";
import { useAsync } from "@/lib/hooks";
import { useAuth } from "@/lib/auth";
import { PRICE_WRITE_PERMS, PRICE_SUBMIT_PERMS } from "@/lib/permissions";
import { QtyInput } from "@/components/QtyInput";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { CatalogueForm } from "@/app/(dashboard)/stock/setups/form";
import { DocumentMeta, ReviewTabBar, ErpField, StatusPill } from "@/components/ApprovalReview";
import { SectionRow, SectionTable } from "@/components/SectionTable";
import { Button, Card, ErrorBanner, Field, Input, PageHeader, Select, Spinner } from "@/components/ui";
import { DateInput } from "@/components/ui";
import {
  BILLING_UNITS,
  clampEffectiveToDoc,
  effectiveBeforeDocError,
  feeEditable,
  fetchApprovedFx,
  isoDay,
  productLabel,
  productOptions,
  type PriceBatch,
  type PriceLine,
} from "@/lib/billing";
import { creatorLabel } from "@/lib/creator";
import { BatchFiles } from "./BatchFiles";
import { BatchWorkflow } from "./BatchWorkflow";
import { FxRateField } from "./FxRateField";

const writePerms = PRICE_WRITE_PERMS;
const submitPerms = PRICE_SUBMIT_PERMS;

function emptyLine(): PriceLine {
  return { unit: "M3", sourceCurrencyCode: "USD", sourcePrice: "" };
}

export function PriceBatchEditor({
  kind,
  path,
  listHref,
  title,
}: {
  kind: string;
  path: string;
  listHref: string;
  title: string;
}) {
  const { id } = useParams<{ id: string }>();
  const doc = useAsync(() => api.get<PriceBatch>(`${path}/${id}`), [id, path]);
  const { hasPermission } = useAuth();
  const canWrite = hasPermission(...writePerms);
  const canSubmit = hasPermission(...submitPerms);
  const [tab, setTab] = useState<"lines" | "files" | "workflow">("lines");
  const [date, setDate] = useState("");
  const [effectiveFrom, setEff] = useState("");
  const [description, setDescription] = useState("");
  const [exchangeRate, setFx] = useState("");
  const [fxManual, setManual] = useState(false);
  const [fees, setFees] = useState<PriceLine[]>([emptyLine()]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [viewing, setViewing] = useState<PriceLine | null>(null);

  useEffect(() => {
    const row = doc.data;
    if (!row) return;
    setDate(isoDay(row.date));
    setEff(isoDay(row.effectiveFrom || row.date));
    setDescription(row.description || "");
    setFx(row.exchangeRate && Number(row.exchangeRate) ? String(row.exchangeRate) : "");
    setManual(Boolean(row.fxManual));
    setFees(row.fees?.length ? row.fees.map((f) => ({ ...f })) : [emptyLine()]);
  }, [doc.data]);

  const row = doc.data;
  const draft = feeEditable(row?.status) && canWrite;

  async function save() {
    setError("");
    const dateErr = effectiveBeforeDocError(date, effectiveFrom);
    if (dateErr) {
      setError(dateErr);
      notify.error(dateErr, { title: "Cannot save" });
      return;
    }
    const ready = fees.filter((f) => f.product?.id && f.sourcePrice.trim());
    if (ready.some((f) => !(Number(f.sourcePrice) > 0))) {
      const msg = "Each line amount must be greater than zero.";
      setError(msg);
      notify.error(msg, { title: "Cannot save" });
      return;
    }
    const seen = new Set<string>();
    for (const f of ready) {
      const key = `${f.product!.id}|${f.unit}|${f.sourceCurrencyCode}`;
      if (seen.has(key)) {
        const msg = `${productLabel(f.product)} / ${f.unit} / ${f.sourceCurrencyCode} is already on this batch.`;
        setError(msg);
        notify.error(msg, { title: "Cannot save" });
        return;
      }
      seen.add(key);
    }
    setSaving(true);
    try {
      await api.put(`${path}/${id}`, {
        date,
        effectiveFrom,
        description: description.trim(),
        exchangeRate: exchangeRate.trim() || "0",
        fxManual,
        fees: fees
          .filter((f) => f.product?.id && f.sourcePrice.trim())
          .map((f) => ({
            productId: f.product!.id,
            unit: f.unit,
            sourceCurrencyCode: f.sourceCurrencyCode,
            sourcePrice: f.sourcePrice.trim(),
          })),
      });
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not save batch"));
    } finally {
      setSaving(false);
    }
  }

  async function submit() {
    setError("");
    if (!fees.some((f) => f.product?.id && f.sourcePrice.trim())) {
      notify.error("Add at least one fee line before submitting for approval.", {
        title: "Cannot submit",
      });
      return;
    }
    try {
      await api.post(`${path}/${id}/submit`, {});
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not submit"));
    }
  }

  function patch(i: number, next: Partial<PriceLine>) {
    setFees((rows) => rows.map((r, idx) => (idx === i ? { ...r, ...next } : r)));
  }

  if (doc.loading && !row) return <Spinner />;
  if (!row) {
    return (
      <div className="space-y-4">
        <ErrorBanner message={doc.error || `${title} not found`} />
        <Link href={listHref} className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title={row.documentNumber}
        subtitle={`${title} · USD = TZS ÷ FX`}
        actions={
          <Link href={listHref} className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-6">
        <DocumentMeta crumb={`Finance · Setups · ${title}`}>
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Date">
            <DateInput
              value={date}
              onChange={(v) => {
                setDate(v);
                setEff((e) => clampEffectiveToDoc(v, e));
              }}
              disabled={!draft}
            />
          </Field>
          <Field label="Effective from">
            <DateInput value={effectiveFrom} onChange={setEff} min={date} disabled={!draft} />
          </Field>
          <div className="sm:col-span-2">
            <Field label="Description">
              <Input
                value={description}
                maxLength={200}
                disabled={!draft}
                onChange={(e) => setDescription(e.target.value)}
              />
            </Field>
          </div>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <ErpField label="Home currency" value={row.currencyCode || "—"} />
          <ErpField label="Created by" value={creatorLabel(row.createdBy)} />
        </div>
        <FxRateField
          rate={exchangeRate}
          manual={fxManual}
          disabled={!draft}
          onRate={setFx}
          onManual={(on) => {
            setManual(on);
            if (!on) void fetchApprovedFx(date).then((r) => r && setFx(r));
          }}
        />
        {draft ? (
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => void save()} loading={saving}>
              Save
            </Button>
            {canSubmit ? (
              <Button
                variant="secondary"
                onClick={() => void submit()}
                disabled={!fees.some((f) => f.product?.id && f.sourcePrice.trim())}
              >
                Submit for approval
              </Button>
            ) : null}
          </div>
        ) : null}
      </Card>
      <Card className="space-y-4 p-6">
        <ReviewTabBar
          tab={tab}
          onChange={setTab}
          ariaLabel={`${kind} batch sections`}
          ids={[
            { id: "lines", label: "Details" },
            { id: "files", label: "Attachments" },
            { id: "workflow", label: "Workflow" },
          ]}
        />
        {tab === "lines" ? (
          <div className="space-y-3">
            <p className="text-[13px] leading-5 text-slate-700">
              Unique together are product, unit, currency, and the batch effective date. Amount must be
              greater than zero. Click a row to view the line.
            </p>
            <SectionTable
              title="Products"
              actions={
                draft ? (
                  <Button variant="secondary" className="!px-3 !py-1.5" onClick={() => setFees((rows) => [...rows, emptyLine()])}>
                    Add product
                  </Button>
                ) : null
              }
              columns={[
                { label: "Product" },
                { label: "Unit" },
                { label: "Currency" },
                { label: "Amount" },
                { label: "" },
              ]}
              empty="Add a product line."
            >
              {fees.map((line, i) => (
                <SectionRow key={i} onClick={() => setViewing(line)}>
                  <td className="min-w-[16rem]" onClick={(e) => e.stopPropagation()}>
                    <AsyncSearchableSelect
                      value={line.product?.id || ""}
                      selectedLabel={productLabel(line.product)}
                      placeholder="Search product"
                      disabled={!draft}
                      onChange={(value, option) =>
                        patch(i, { product: value ? { id: value, name: option?.label } : undefined })
                      }
                      loadOptions={productOptions}
                    />
                  </td>
                  <td onClick={(e) => e.stopPropagation()}>
                    <Select value={line.unit} disabled={!draft} onChange={(e) => patch(i, { unit: e.target.value })}>
                      {BILLING_UNITS.map((u) => (
                        <option key={u.value} value={u.value}>
                          {u.label}
                        </option>
                      ))}
                    </Select>
                  </td>
                  <td onClick={(e) => e.stopPropagation()}>
                    <Select
                      value={line.sourceCurrencyCode}
                      disabled={!draft}
                      onChange={(e) => patch(i, { sourceCurrencyCode: e.target.value })}
                    >
                      <option value="USD">USD</option>
                      <option value="TZS">TZS</option>
                    </Select>
                  </td>
                  <td onClick={(e) => e.stopPropagation()}>
                    <QtyInput
                      unit="price"
                      value={line.sourcePrice}
                      disabled={!draft}
                      onChange={(sourcePrice) => patch(i, { sourcePrice })}
                    />
                  </td>
                  <td className="text-right">
                    {draft && fees.length > 1 ? (
                      <Button
                        variant="ghost"
                        onClick={(e) => {
                          e.stopPropagation();
                          setFees((rows) => rows.filter((_, idx) => idx !== i));
                        }}
                      >
                        Remove
                      </Button>
                    ) : null}
                  </td>
                </SectionRow>
              ))}
            </SectionTable>
            {viewing ? (
              <CatalogueForm
                title={productLabel(viewing.product) || "Fee line"}
                error=""
                saving={false}
                onClose={() => setViewing(null)}
                onSave={() => setViewing(null)}
                saveLabel="Close"
              >
                <ErpField label="Product" value={productLabel(viewing.product) || "—"} />
                <ErpField
                  label="Unit"
                  value={BILLING_UNITS.find((u) => u.value === viewing.unit)?.label || viewing.unit}
                />
                <ErpField label="Currency" value={viewing.sourceCurrencyCode} />
                <ErpField label="Amount" value={viewing.sourcePrice || "—"} />
                {viewing.homePrice ? <ErpField label="Home amount" value={viewing.homePrice} /> : null}
              </CatalogueForm>
            ) : null}
          </div>
        ) : null}
        {tab === "files" ? <BatchFiles path={`${path}/${id}`} draft={draft} /> : null}
        {tab === "workflow" ? <BatchWorkflow path={`${path}/${id}`} /> : null}
      </Card>
    </div>
  );
}
