"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, formatApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PRICE_WRITE_PERMS } from "@/lib/permissions";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import { ErpField, InquiryHeader, InquirySection, StatusPill } from "@/components/ApprovalReview";
import { Button, Field, Select, StatusBadge } from "@/components/ui";
import { DateInput } from "@/components/ui";
import { QtyInput } from "@/components/QtyInput";
import { CatalogueForm } from "@/app/(dashboard)/stock/setups/form";
import { DOC_STATUS_PILLS } from "@/components/StatusPills";
import { formatDate } from "@/lib/format";
import { qty } from "@/lib/master";
import { feeEditable, isoDay, type ExchangeRate } from "@/lib/billing";
import { creatorLabel } from "@/lib/creator";

const priceWrite = PRICE_WRITE_PERMS;

export default function FxPage() {
  const router = useRouter();
  const [tick, setTick] = useState(0);
  const { hasPermission } = useAuth();
  const canWrite = hasPermission(...priceWrite);
  const [form, setForm] = useState<ExchangeRate | "new" | null>(null);

  const columns: EnterpriseColumn<ExchangeRate>[] = useMemo(
    () => [
      {
        id: "effectiveFrom",
        header: "Effective",
        sortKey: "effectiveFrom",
        defaultVisible: true,
        cell: (r) => formatDate(r.effectiveFrom),
      },
      {
        id: "pair",
        header: "Pair",
        defaultVisible: true,
        cell: (r) => `${r.fromCurrency}/${r.toCurrency}`,
      },
      {
        id: "rate",
        header: "Rate",
        sortKey: "rate",
        defaultVisible: true,
        className: "text-right",
        cell: (r) => <span className="tabular-nums">{qty(r.rate)}</span>,
      },
      {
        id: "createdBy",
        header: "Created by",
        defaultVisible: true,
        cell: (r) => creatorLabel(r.createdBy),
      },
      {
        id: "status",
        header: "Status",
        sortKey: "status",
        defaultVisible: true,
        cell: (r) => <StatusBadge status={r.status} />,
      },
      {
        id: "_actions",
        fixed: true,
        header: "",
        className: "text-right",
        cell: (r) => (
          <div className="flex justify-end gap-3">
            <Link
              href={`/finance/setups/fx/${r.id}`}
              className="text-sm font-semibold text-brand-800"
              onClick={(e) => e.stopPropagation()}
            >
              Open
            </Link>
            {canWrite && feeEditable(r.status) ? (
              <Link
                href={`/finance/setups/fx/${r.id}`}
                className="text-sm font-semibold text-brand-800"
                onClick={(e) => e.stopPropagation()}
              >
                Edit
              </Link>
            ) : null}
          </div>
        ),
      },
    ],
    [canWrite],
  );

  return (
    <>
      <p className="mb-1 text-xs font-medium text-slate-500">Finance › Setups</p>
      <EnterpriseTable
        tableId="exchange-rates"
        title="Exchange rates"
        subtitle="Approved TZS-per-USD quotes auto-fill FSF, VSF, KOJ, and TBS batches"
        pills={{
          options: DOC_STATUS_PILLS,
          of: (r) => r.status,
          guideTitle: "Rate status",
          allTitle: "All rates",
        }}
        headerExtra={
          canWrite ? <Button onClick={() => setForm("new")}>Add rate</Button> : null
        }
        remote={{ path: "/billing/fx", dates: true, reloadKey: tick }}
        emptyMessage="No exchange rates yet. Add a TZS-per-USD quote, then submit it for approval."
        columns={columns}
        searchPlaceholder="Currency"
        rowSearch={(r) => `${r.fromCurrency} ${r.toCurrency} ${r.rate}`}
        defaultOrderBy="effectiveFrom"
        defaultSortDirection="DESC"
        dateOf={(r) => r.effectiveFrom}
        exportName="exchange_rates"
        inquiryTitle={(r) => `${r.fromCurrency}/${r.toCurrency}`}
        inquiry={(r) => (
          <div className="space-y-4">
            <InquiryHeader
              crumb="Billing · Setups · FX · Inquiry"
              title={`${r.fromCurrency} → ${r.toCurrency}`}
              pills={<StatusPill tone="navy">{r.status}</StatusPill>}
            />
            <InquirySection title="Rate">
              <ErpField label="From" value={r.fromCurrency} />
              <ErpField label="To" value={r.toCurrency} />
              <ErpField label="Rate" value={qty(r.rate)} />
              <ErpField label="Effective" value={formatDate(r.effectiveFrom)} />
            </InquirySection>
            <div className="flex flex-wrap gap-2">
              <Link href={`/finance/setups/fx/${r.id}`}>
                <Button variant="secondary">Open</Button>
              </Link>
              {canWrite && feeEditable(r.status) ? (
                <Link href={`/finance/setups/fx/${r.id}`}>
                  <Button>Edit</Button>
                </Link>
              ) : null}
            </div>
          </div>
        )}
      />
      {form ? (
        <FxForm
          row={form === "new" ? undefined : form}
          onClose={() => setForm(null)}
          onSaved={(id) => {
            setForm(null);
            setTick((n) => n + 1);
            if (id) router.push(`/finance/setups/fx/${id}`);
          }}
        />
      ) : null}
    </>
  );
}

function FxForm({
  row,
  onClose,
  onSaved,
}: {
  row?: ExchangeRate;
  onClose: () => void;
  onSaved: (id?: string) => void;
}) {
  const [effectiveFrom, setFrom] = useState(isoDay(row?.effectiveFrom));
  const [fromCurrency, setFromCcy] = useState(row?.fromCurrency || "USD");
  const [toCurrency, setToCcy] = useState(row?.toCurrency || "TZS");
  const [rate, setRate] = useState(row?.rate || "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    if (!rate.trim()) {
      setError("Rate is required.");
      return;
    }
    setSaving(true);
    try {
      const body = { effectiveFrom, fromCurrency, toCurrency, rate: rate.trim() };
      if (row) {
        await api.put(`/billing/fx/${row.id}`, body);
        onSaved(row.id);
      } else {
        const created = await api.post<{ id: string }>("/billing/fx", body);
        onSaved(created.id);
      }
    } catch (err) {
      setError(formatApiError(err, "Failed to save rate"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? "Edit exchange rate" : "Add exchange rate"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
    >
      <Field label="Effective from">
        <DateInput value={effectiveFrom} onChange={setFrom} />
      </Field>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="From">
          <Select value={fromCurrency} onChange={(e) => setFromCcy(e.target.value)}>
            <option value="USD">USD</option>
            <option value="TZS">TZS</option>
          </Select>
        </Field>
        <Field label="To">
          <Select value={toCurrency} onChange={(e) => setToCcy(e.target.value)}>
            <option value="TZS">TZS</option>
            <option value="USD">USD</option>
          </Select>
        </Field>
      </div>
      <Field label="Rate">
        <QtyInput unit="fx" value={rate} onChange={setRate} placeholder="2,600" />
      </Field>
    </CatalogueForm>
  );
}
