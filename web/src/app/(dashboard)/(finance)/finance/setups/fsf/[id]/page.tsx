"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { api, formatApiError } from "@/lib/api";
import { notify } from "@/lib/notify";
import { useAsync } from "@/lib/hooks";
import { useAuth } from "@/lib/auth";
import { PRICE_WRITE_PERMS } from "@/lib/permissions";
import { DocumentMeta, ReviewTabBar, ErpField, StatusPill } from "@/components/ApprovalReview";
import { SectionRow, SectionTable } from "@/components/SectionTable";
import { Button, Card, ErrorBanner, Field, Input, PageHeader, Select, Spinner, ToggleField } from "@/components/ui";
import { DateInput } from "@/components/ui";
import { QtyInput } from "@/components/QtyInput";
import { parseQty } from "@/lib/format";
import { CatalogueForm } from "@/app/(dashboard)/stock/setups/form";
import {
  BILLING_UNITS,
  clampEffectiveToDoc,
  effectiveBeforeDocError,
  feeEditable,
  fetchApprovedFx,
  isoDay,
  type FcfBatch,
  type FcfLine,
  type FcfTier,
} from "@/lib/billing";
import { creatorLabel } from "@/lib/creator";
import { qty } from "@/lib/master";
import { activeRows, catalogLabel, fetchLookups, type CatalogRow } from "@/lib/lookups";
import { BatchFiles } from "../../_components/BatchFiles";
import { BatchWorkflow } from "../../_components/BatchWorkflow";
import { FxRateField } from "../../_components/FxRateField";

const writePerms = PRICE_WRITE_PERMS;

function emptyLine(lookups?: {
  tender?: string;
  route?: string;
  delivery?: string;
  procurement?: string;
  days?: number;
}): FcfLine {
  const days = lookups?.days || 15;
  return {
    classOfTrade: lookups?.tender || "",
    procurementMethod: lookups?.procurement || "",
    dischargeRoute: lookups?.route || "",
    collectionMethod: lookups?.delivery || "",
    isPromotional: false,
    firstDays: days,
    firstChargeTo: "customer",
    firstRateKind: "flat",
    firstUnit: "M3",
    firstSourceCurrencyCode: "USD",
    firstSourcePrice: "",
    nthDays: days === 45 ? 45 : 30,
    nthRateKind: "flat",
    nthUnit: "M3",
    nthSourceCurrencyCode: "USD",
    nthSourcePrice: "",
    tiers: [],
  };
}

function lineReady(l: FcfLine): boolean {
  return Boolean(l.classOfTrade && l.dischargeRoute);
}

export default function FcfBatchPage() {
  const { id } = useParams<{ id: string }>();
  const path = `/billing/fsf/batches/${id}`;
  const doc = useAsync(() => api.get<FcfBatch>(path), [id]);
  const catalogs = useAsync(fetchLookups, []);
  const { hasPermission } = useAuth();
  const canWrite = hasPermission(...writePerms);
  const [tab, setTab] = useState<"lines" | "files" | "workflow">("lines");
  const [date, setDate] = useState("");
  const [effectiveFrom, setEff] = useState("");
  const [description, setDescription] = useState("");
  const [exchangeRate, setFx] = useState("");
  const [fxManual, setManual] = useState(false);
  const [lines, setLines] = useState<FcfLine[]>([]);
  const [editing, setEditing] = useState<{ index: number; line: FcfLine } | "new" | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const tenders = activeRows(catalogs.data?.tenders);
  const routes = activeRows(catalogs.data?.routes);
  const delivery = activeRows(catalogs.data?.deliveryMethods);
  const procurement = activeRows(catalogs.data?.procurementMethods);
  const periods = activeRows(catalogs.data?.billingCycles);
  const defaults = {
    tender: tenders[0]?.code,
    route: routes[0]?.code,
    delivery: delivery[0]?.code,
    procurement: procurement[0]?.code,
    days: periods[0]?.days || 15,
  };

  useEffect(() => {
    const row = doc.data;
    if (!row) return;
    setDate(isoDay(row.date));
    setEff(isoDay(row.effectiveFrom || row.date));
    setDescription(row.description || "");
    setFx(row.exchangeRate && Number(row.exchangeRate) ? String(row.exchangeRate) : "");
    setManual(Boolean(row.fxManual));
    setLines(
      row.lines?.length
        ? row.lines.map((l) => ({
            ...emptyLine(defaults),
            ...l,
            firstRateKind: l.firstRateKind === "tier" ? "tier" : "flat",
            nthRateKind: l.nthRateKind === "tier" ? "tier" : "flat",
            tiers: l.tiers || [],
          }))
        : [],
    );
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
    setSaving(true);
    try {
      await api.put(path, {
        date,
        effectiveFrom,
        description: description.trim(),
        exchangeRate: parseQty(exchangeRate) || "0",
        fxManual,
        lines: lines.filter(lineReady).map((l) => ({
          ...l,
          firstDays: Number(l.firstDays) || 15,
          nthDays: Number(l.nthDays) || 30,
          firstSourcePrice: parseQty(l.firstSourcePrice),
          nthSourcePrice: parseQty(l.nthSourcePrice),
          tiers: (l.tiers || []).map((t) => ({
            ...t,
            fromQty: parseQty(t.fromQty),
            toQty: t.toQty ? parseQty(t.toQty) : "",
            sourcePrice: parseQty(t.sourcePrice),
          })),
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
    if (!lines.some(lineReady)) {
      notify.error("Add at least one FSF pricing line before submitting for approval.", {
        title: "Cannot submit",
      });
      return;
    }
    try {
      await api.post(`${path}/submit`, {});
      doc.reload();
    } catch (e) {
      setError(formatApiError(e, "Could not submit"));
    }
  }

  function patch(i: number, next: Partial<FcfLine>) {
    setLines((rows) => rows.map((r, idx) => (idx === i ? { ...r, ...next } : r)));
  }

  function commitModal(line: FcfLine) {
    if (!lineReady(line)) return;
    if (editing === "new") {
      setLines((rows) => [...rows, line]);
    } else if (editing) {
      patch(editing.index, line);
    }
    setEditing(null);
  }

  if (doc.loading && !row) return <Spinner />;
  if (!row) {
    return (
      <div className="space-y-4">
        <ErrorBanner message={doc.error || "Fixed storage fee batch not found"} />
        <Link href="/finance/setups/fsf" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title={row.documentNumber}
        subtitle="Fixed storage fee — first billing on confirmed reception, then repeating second billing until the parcel is finished"
        actions={
          <Link href="/finance/setups/fsf" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-6">
        <DocumentMeta crumb="Finance · Setups · Fixed storage fee">
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
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <ErpField label="Home currency" value={row.currencyCode || "—"} />
          <ErpField label="Created by" value={creatorLabel(row.createdBy)} />
          <div className="sm:col-span-2">
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
          </div>
        </div>
        <p className="text-xs leading-5 text-slate-500">
          Billing uses the latest approved list whose effective date is on or before the receipt billing
          date. Home currency comes from company setup. Source rates convert with FX (TZS per 1 USD).
        </p>
        {draft ? (
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => void save()} loading={saving}>
              Save
            </Button>
            <Button variant="secondary" onClick={() => void submit()} disabled={!lines.some(lineReady)}>
              Submit for approval
            </Button>
          </div>
        ) : null}
      </Card>
      <Card className="space-y-4 p-6">
        <ReviewTabBar
          tab={tab}
          onChange={setTab}
          ariaLabel="Fixed storage fee sections"
          ids={[
            { id: "lines", label: "Pricing models" },
            { id: "files", label: "Attachments" },
            { id: "workflow", label: "Workflow" },
          ]}
        />
        {tab === "lines" ? (
          <div className="space-y-4">
            <p className="text-sm text-slate-600">
              One pricing model per class of trade. First billing posts when a surveyed receipt is approved.
              Second billing repeats until stock is finished. Flat = one unit price; volume tier = whole-volume slab.
            </p>
            <SectionTable
              title="Pricing models"
              actions={
                draft ? (
                  <Button variant="secondary" className="!px-3 !py-1.5" onClick={() => setEditing("new")}>
                    Add pricing model
                  </Button>
                ) : null
              }
              columns={[
                { label: "Tender" },
                { label: "Route" },
                { label: "Delivery" },
                { label: "First" },
                { label: "Second" },
                { label: "" },
              ]}
              empty="No pricing models yet."
            >
              {lines.map((line, i) => (
                <SectionRow key={i}>
                  <td className="font-semibold">
                    {catalogLabel(tenders.find((t) => t.code === line.classOfTrade) || { code: line.classOfTrade, name: line.classOfTrade })}
                  </td>
                  <td>{line.dischargeRoute}</td>
                  <td>{line.collectionMethod || "Any"}</td>
                  <td className="tabular-nums">
                    {line.firstDays}d · {line.firstRateKind === "tier" ? `${(line.tiers || []).filter((t) => t.phase === "first").length} slabs` : `${qty(line.firstSourcePrice)} ${line.firstUnit}`}
                  </td>
                  <td className="tabular-nums">
                    {line.nthDays}d · {line.nthRateKind === "tier" ? `${(line.tiers || []).filter((t) => t.phase === "nth").length} slabs` : `${qty(line.nthSourcePrice)} ${line.nthUnit}`}
                  </td>
                  <td className="text-right">
                    {draft ? (
                      <div className="flex justify-end gap-2">
                        <Button variant="ghost" onClick={() => setEditing({ index: i, line: { ...line, tiers: [...(line.tiers || [])] } })}>
                          Edit
                        </Button>
                        <Button variant="ghost" onClick={() => setLines((rows) => rows.filter((_, idx) => idx !== i))}>
                          Remove
                        </Button>
                      </div>
                    ) : null}
                  </td>
                </SectionRow>
              ))}
            </SectionTable>
            {editing ? (
              <PricingModal
                line={editing === "new" ? emptyLine(defaults) : editing.line}
                tenders={tenders}
                routes={routes}
                delivery={delivery}
                procurement={procurement}
                periods={periods}
                onClose={() => setEditing(null)}
                onSave={commitModal}
              />
            ) : null}
          </div>
        ) : null}
        {tab === "files" ? <BatchFiles path={path} draft={draft} /> : null}
        {tab === "workflow" ? <BatchWorkflow path={path} /> : null}
      </Card>
    </div>
  );
}

function PricingModal({
  line,
  tenders,
  routes,
  delivery,
  procurement,
  periods,
  onClose,
  onSave,
}: {
  line: FcfLine;
  tenders: CatalogRow[];
  routes: CatalogRow[];
  delivery: CatalogRow[];
  procurement: CatalogRow[];
  periods: { days: number; description?: string }[];
  onClose: () => void;
  onSave: (line: FcfLine) => void;
}) {
  const [draft, setDraft] = useState<FcfLine>(line);
  const [error, setError] = useState("");

  function commit() {
    if (!draft.classOfTrade || !draft.dischargeRoute || !draft.procurementMethod) {
      setError("Tender, procurement, and route are required.");
      return;
    }
    onSave(draft);
  }

  return (
    <CatalogueForm
      title="Pricing model"
      error={error}
      saving={false}
      onClose={onClose}
      onSave={commit}
      saveLabel="Add to batch"
      size="2xl"
    >
      <PricingCard
        line={draft}
        draft
        tenders={tenders}
        routes={routes}
        delivery={delivery}
        procurement={procurement}
        periods={periods}
        onChange={(next) => setDraft((row) => ({ ...row, ...next }))}
      />
    </CatalogueForm>
  );
}

function PricingCard({
  line,
  draft,
  tenders,
  routes,
  delivery,
  procurement,
  periods,
  onChange,
  onRemove,
}: {
  line: FcfLine;
  draft: boolean;
  tenders: CatalogRow[];
  routes: CatalogRow[];
  delivery: CatalogRow[];
  procurement: CatalogRow[];
  periods: { days: number; description?: string }[];
  onChange: (next: Partial<FcfLine>) => void;
  onRemove?: () => void;
}) {
  const cycleOpts = periods.length
    ? periods
    : [
        { days: 15, description: "15-day" },
        { days: 30, description: "30-day" },
        { days: 45, description: "45-day" },
      ];

  return (
    <div className="space-y-4 rounded-xl border border-slate-200 p-4">
      <div className="grid gap-3 sm:grid-cols-5">
        <Field label="Tender">
          <Select value={line.classOfTrade} disabled={!draft} onChange={(e) => onChange({ classOfTrade: e.target.value })}>
            {tenders.map((t) => (
              <option key={t.code} value={t.code}>{catalogLabel(t)}</option>
            ))}
          </Select>
        </Field>
        <Field label="Procurement">
          <Select value={line.procurementMethod} disabled={!draft} onChange={(e) => onChange({ procurementMethod: e.target.value })}>
            {procurement.map((t) => (
              <option key={t.code} value={t.code}>{catalogLabel(t)}</option>
            ))}
          </Select>
        </Field>
        <Field label="Route">
          <Select value={line.dischargeRoute} disabled={!draft} onChange={(e) => onChange({ dischargeRoute: e.target.value })}>
            {routes.map((t) => (
              <option key={t.code} value={t.code}>{catalogLabel(t)}</option>
            ))}
          </Select>
        </Field>
        <Field label="Delivery">
          <Select
            value={line.collectionMethod || ""}
            disabled={!draft}
            onChange={(e) => onChange({ collectionMethod: e.target.value })}
          >
            <option value="">Any</option>
            {delivery.map((t) => (
              <option key={t.code} value={t.code}>{catalogLabel(t)}</option>
            ))}
          </Select>
        </Field>
        <div className="flex items-end">
          <ToggleField
            label="Promotional"
            checked={Boolean(line.isPromotional)}
            onChange={(isPromotional) => onChange({ isPromotional })}
            disabled={!draft}
          />
        </div>
      </div>
      <PhaseBlock
        title="First billing — on confirmed reception"
        draft={draft}
        days={line.firstDays}
        chargeTo={line.firstChargeTo}
        showCharge
        rateKind={line.firstRateKind}
        unit={line.firstUnit}
        currency={line.firstSourceCurrencyCode}
        price={line.firstSourcePrice}
        cycles={cycleOpts}
        tiers={(line.tiers || []).filter((t) => t.phase === "first")}
        onDays={(firstDays) => onChange({ firstDays })}
        onCharge={(firstChargeTo) => onChange({ firstChargeTo })}
        onKind={(firstRateKind) => onChange({ firstRateKind })}
        onUnit={(firstUnit) => onChange({ firstUnit })}
        onCurrency={(firstSourceCurrencyCode) => onChange({ firstSourceCurrencyCode })}
        onPrice={(firstSourcePrice) => onChange({ firstSourcePrice })}
        onTiers={(next) =>
          onChange({
            tiers: [...(line.tiers || []).filter((t) => t.phase !== "first"), ...next.map((t) => ({ ...t, phase: "first" as const }))],
          })
        }
      />
      <PhaseBlock
        title="Second billing — repeats until product finished (customer)"
        draft={draft}
        days={line.nthDays}
        rateKind={line.nthRateKind}
        unit={line.nthUnit}
        currency={line.nthSourceCurrencyCode}
        price={line.nthSourcePrice}
        cycles={cycleOpts}
        tiers={(line.tiers || []).filter((t) => t.phase === "nth")}
        onDays={(nthDays) => onChange({ nthDays })}
        onKind={(nthRateKind) => onChange({ nthRateKind })}
        onUnit={(nthUnit) => onChange({ nthUnit })}
        onCurrency={(nthSourceCurrencyCode) => onChange({ nthSourceCurrencyCode })}
        onPrice={(nthSourcePrice) => onChange({ nthSourcePrice })}
        onTiers={(next) =>
          onChange({
            tiers: [...(line.tiers || []).filter((t) => t.phase !== "nth"), ...next.map((t) => ({ ...t, phase: "nth" as const }))],
          })
        }
      />
      {onRemove ? (
        <Button variant="ghost" onClick={onRemove}>
          Remove model
        </Button>
      ) : null}
    </div>
  );
}

function PhaseBlock({
  title,
  draft,
  days,
  chargeTo,
  showCharge,
  rateKind,
  unit,
  currency,
  price,
  cycles,
  tiers,
  onDays,
  onCharge,
  onKind,
  onUnit,
  onCurrency,
  onPrice,
  onTiers,
}: {
  title: string;
  draft: boolean;
  days: number;
  chargeTo?: string;
  showCharge?: boolean;
  rateKind: "flat" | "tier";
  unit: string;
  currency: string;
  price: string;
  cycles: { days: number; description?: string }[];
  tiers: FcfTier[];
  onDays: (n: number) => void;
  onCharge?: (v: string) => void;
  onKind: (v: "flat" | "tier") => void;
  onUnit: (v: string) => void;
  onCurrency: (v: string) => void;
  onPrice: (v: string) => void;
  onTiers: (rows: FcfTier[]) => void;
}) {
  return (
    <div className="space-y-3 rounded-lg bg-slate-50 p-3">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-600">{title}</h3>
      <div className="grid gap-3 sm:grid-cols-5">
        <Field label="Period (days)">
          <Select value={String(days)} disabled={!draft} onChange={(e) => onDays(Number(e.target.value))}>
            {cycles.map((p) => (
              <option key={p.days} value={p.days}>
                {p.description || `${p.days} days`}
              </option>
            ))}
          </Select>
        </Field>
        {showCharge && onCharge ? (
          <Field label="Charge to">
            <Select value={chargeTo || "customer"} disabled={!draft} onChange={(e) => onCharge(e.target.value)}>
              <option value="customer">Customer</option>
              <option value="supplier">Supplier</option>
            </Select>
          </Field>
        ) : null}
        <Field label="Rate">
          <Select value={rateKind} disabled={!draft} onChange={(e) => onKind(e.target.value as "flat" | "tier")}>
            <option value="flat">Flat</option>
            <option value="tier">Volume tier</option>
          </Select>
        </Field>
        <Field label="Unit">
          <Select value={unit} disabled={!draft} onChange={(e) => onUnit(e.target.value)}>
            {BILLING_UNITS.map((u) => (
              <option key={u.value} value={u.value}>{u.label}</option>
            ))}
          </Select>
        </Field>
        <Field label="Ccy">
          <Select value={currency} disabled={!draft} onChange={(e) => onCurrency(e.target.value)}>
            <option value="USD">USD</option>
            <option value="TZS">TZS</option>
          </Select>
        </Field>
      </div>
      {rateKind === "flat" ? (
        <Field label="Unit price">
          <QtyInput unit="price" value={price} onChange={onPrice} disabled={!draft} />
        </Field>
      ) : (
        <div className="space-y-2">
          <div className="grid grid-cols-[1fr_1fr_1fr_auto] gap-2 text-xs uppercase tracking-wide text-slate-500">
            <span>From</span>
            <span>To (blank = open)</span>
            <span>Price</span>
            <span />
          </div>
          {tiers.map((t, i) => (
            <div key={i} className="grid grid-cols-[1fr_1fr_1fr_auto] items-center gap-2">
              <QtyInput
                unit="m3"
                value={t.fromQty}
                disabled={!draft}
                onChange={(fromQty) => onTiers(tiers.map((row, idx) => (idx === i ? { ...row, fromQty } : row)))}
              />
              <QtyInput
                unit="m3"
                value={t.toQty || ""}
                disabled={!draft}
                onChange={(toQty) => onTiers(tiers.map((row, idx) => (idx === i ? { ...row, toQty } : row)))}
              />
              <QtyInput
                unit="price"
                value={t.sourcePrice}
                disabled={!draft}
                onChange={(sourcePrice) => onTiers(tiers.map((row, idx) => (idx === i ? { ...row, sourcePrice } : row)))}
              />
              {draft ? (
                <Button variant="ghost" onClick={() => onTiers(tiers.filter((_, idx) => idx !== i))}>
                  Remove
                </Button>
              ) : null}
            </div>
          ))}
          {draft ? (
            <Button
              variant="secondary"
              onClick={() =>
                onTiers([...tiers, { phase: tiers[0]?.phase || "first", fromQty: "0", toQty: "", sourcePrice: "" }])
              }
            >
              Add slab
            </Button>
          ) : null}
        </div>
      )}
    </div>
  );
}
