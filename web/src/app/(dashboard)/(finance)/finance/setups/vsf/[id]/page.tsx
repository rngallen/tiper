"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { api, formatApiError } from "@/lib/api";
import { notify } from "@/lib/notify";
import { useAsync } from "@/lib/hooks";
import { useAuth } from "@/lib/auth";
import { PRICE_WRITE_PERMS } from "@/lib/permissions";
import { QtyInput } from "@/components/QtyInput";
import { SearchableSelect } from "@/components/SearchableSelect";
import { CatalogueForm } from "@/app/(dashboard)/stock/setups/form";
import { DocumentMeta, ReviewTabBar, ErpField, StatusPill } from "@/components/ApprovalReview";
import { SectionPanel, SectionRow, SectionTable } from "@/components/SectionTable";
import { Button, Card, ErrorBanner, Field, Input, PageHeader, Select, Spinner } from "@/components/ui";
import { DateInput } from "@/components/ui";
import {
  clampEffectiveToDoc,
  effectiveBeforeDocError,
  feeEditable,
  fetchApprovedFx,
  fetchApprovedMiLoss,
  isoDay,
  latestMiLossAsOf,
  miLossBatchLabel,
  miLossOnOrBefore,
  miLossPct,
  productLabel,
  type MiLossBatch,
  type MiLossLine,
  type VarContract,
  type VarProduct,
  type VariableBatch,
} from "@/lib/billing";
import { creatorLabel } from "@/lib/creator";
import { qty } from "@/lib/master";
import { catalogLabel, fetchLookups } from "@/lib/lookups";
import { BatchFiles } from "../../_components/BatchFiles";
import { BatchWorkflow } from "../../_components/BatchWorkflow";
import { FxRateField } from "../../_components/FxRateField";

const writePerms = PRICE_WRITE_PERMS;

export default function VariableBatchPage() {
  const { id } = useParams<{ id: string }>();
  const path = `/billing/vsf/batches/${id}`;
  const doc = useAsync(() => api.get<VariableBatch>(path), [id]);
  const lookups = useAsync(fetchLookups, []);
  const approvedMi = useAsync(fetchApprovedMiLoss, []);
  const { hasPermission } = useAuth();
  const canWrite = hasPermission(...writePerms);
  const [tab, setTab] = useState<"lines" | "files" | "workflow">("lines");
  const [date, setDate] = useState("");
  const [effectiveFrom, setEff] = useState("");
  const [description, setDescription] = useState("");
  const [exchangeRate, setFx] = useState("");
  const [fxManual, setManual] = useState(false);
  const [miLossBatchId, setMiLossId] = useState("");
  const [products, setProducts] = useState<VarProduct[]>([]);
  const [editing, setEditing] = useState<{ index: number; product: VarProduct } | "new" | null>(null);
  const [viewing, setViewing] = useState<VarProduct | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const row = doc.data;
    if (!row) return;
    setDate(isoDay(row.date));
    setEff(isoDay(row.effectiveFrom || row.date));
    setDescription(row.description || "");
    setFx(row.exchangeRate && Number(row.exchangeRate) ? String(row.exchangeRate) : "");
    setManual(Boolean(row.fxManual));
    setMiLossId(row.miLossBatch?.id || "");
    setProducts(row.products?.length ? row.products.map((p) => ({ ...p })) : []);
  }, [doc.data]);

  const row = doc.data;
  const draft = feeEditable(row?.status) && canWrite;
  const contractNames = lookups.data?.contractTypes || [];

  const approvedBatches = useMemo(() => {
    const rows = [...(approvedMi.data || [])];
    const extra = row?.miLossBatch;
    if (extra && !rows.some((b) => b.id === extra.id)) {
      rows.push({
        id: extra.id,
        documentNumber: extra.documentNumber,
        date: extra.date,
        effectiveFrom: extra.effectiveFrom || "",
        description: extra.description,
        status: "approved",
      });
    }
    return rows;
  }, [approvedMi.data, row?.miLossBatch]);
  const asOfBatches = useMemo(
    () => miLossOnOrBefore(approvedBatches, effectiveFrom || date),
    [approvedBatches, effectiveFrom, date],
  );

  const asOf = (effectiveFrom || date).slice(0, 10);
  const asOfRef = useRef(asOf);

  useEffect(() => {
    if (!draft) return;
    const latest = latestMiLossAsOf(approvedBatches, asOf);
    const dateChanged = asOfRef.current !== asOf;
    asOfRef.current = asOf;
    setMiLossId((current) => {
      const keep = !dateChanged && current && asOfBatches.some((b) => b.id === current);
      return keep ? current : latest?.id || "";
    });
  }, [draft, asOf, asOfBatches, approvedBatches]);

  const source = useAsync(
    () =>
      miLossBatchId ? api.get<MiLossBatch>(`/billing/miloss/batches/${miLossBatchId}`) : Promise.resolve(null),
    [miLossBatchId],
  );
  const sourceLines = source.data?.lines || [];

  const takenIds = useMemo(() => new Set(products.map((p) => p.product?.id).filter(Boolean) as string[]), [products]);
  const availableLines = useMemo(() => {
    const seen = new Set<string>();
    return sourceLines.filter((l) => {
      const id = l.product?.id;
      if (!id || takenIds.has(id) || seen.has(id)) return false;
      seen.add(id);
      return true;
    });
  }, [sourceLines, takenIds]);

  async function save(next = products, silent = false): Promise<void> {
    if (!date || !effectiveFrom) {
      notify.error("Date and effective from are required.", { title: "Cannot save" });
      throw new Error("Date and effective from are required.");
    }
    const dateErr = effectiveBeforeDocError(date, effectiveFrom);
    if (dateErr) {
      notify.error(dateErr, { title: "Cannot save" });
      throw new Error(dateErr);
    }
    if (!miLossBatchId) {
      notify.error("Select an approved MI-loss batch.", { title: "Cannot save" });
      throw new Error("Select an approved MI-loss batch.");
    }
    setSaving(true);
    try {
      await api.put(
        path,
        {
          date,
          effectiveFrom,
          description: description.trim(),
          exchangeRate: exchangeRate.trim() || "0",
          fxManual,
          miLossBatchId,
          products: next
            .filter((p) => p.product?.id && p.ewuraPrice.trim() && p.density.trim())
            .map((p) => ({
              productId: p.product!.id,
              ewuraPrice: p.ewuraPrice.trim(),
              density: p.density.trim(),
            })),
        },
        silent ? { success: false } : undefined,
      );
      setProducts(next);
      doc.reload();
    } finally {
      setSaving(false);
    }
  }

  async function submit() {
    if (!products.some((p) => p.product?.id && p.ewuraPrice.trim() && p.density.trim())) {
      notify.error("Add at least one product before submitting for approval.", {
        title: "Cannot submit",
      });
      return;
    }
    try {
      await save(products, true);
      await api.post(`${path}/submit`, {});
      doc.reload();
    } catch {
      /* message box from api / save */
    }
  }

  async function commitProduct(product: VarProduct) {
    if (!product.product?.id || !product.ewuraPrice.trim() || !product.density.trim()) {
      throw new Error("Product, EWURA price, and density are required.");
    }
    const n = Number(product.density);
    if (!Number.isFinite(n) || n <= 0 || n >= 2) {
      throw new Error("Density must be MT per m³ (e.g. 0.84).");
    }
    if (!sourceLines.some((l) => l.product?.id === product.product?.id)) {
      throw new Error("Choose a product from the selected MI-loss batch.");
    }
    const skip = editing !== "new" && editing ? editing.index : -1;
    if (products.some((r, i) => i !== skip && r.product?.id === product.product?.id)) {
      throw new Error("This product is already on the card.");
    }
    const rates = sourceLines.filter((l) => l.product?.id === product.product?.id);
    const nextProduct: VarProduct = {
      ...product,
      contracts: rates.length
        ? rates.map((l) => ({
            contractTypeCode: l.contractTypeCode,
            miLossValue: l.value,
          }))
        : product.contracts,
    };
    const next =
      editing === "new"
        ? [...products, nextProduct]
        : editing
          ? products.map((r, i) => (i === editing.index ? nextProduct : r))
          : products;
    await save(next);
    setEditing(null);
  }

  async function removeProduct(index: number) {
    const next = products.filter((_, i) => i !== index);
    await save(next).catch(() => undefined);
  }

  function startAdd() {
    if (!draft) return;
    if (!miLossBatchId) {
      notify.error("Select an MI-loss batch first.", { title: "Cannot add product" });
      return;
    }
    if (source.loading) return;
    if (availableLines.length === 0) {
      notify.error(
        sourceLines.length === 0
          ? "This MI-loss batch has no product rates."
          : "Every product on this MI-loss batch is already on the card.",
        { title: "Cannot add product" },
      );
      return;
    }
    setEditing("new");
  }

  function changeMiLoss(id: string) {
    if (id !== miLossBatchId && products.length > 0) {
      setProducts([]);
      notify.warning(
        "Products were cleared because the MI-loss source changed. Add products from the new batch.",
      );
    }
    setMiLossId(id);
  }

  if (doc.loading && !row) return <Spinner />;
  if (!row) {
    return (
      <div className="space-y-4">
        <ErrorBanner message={doc.error || "Variable storage fee batch not found"} />
        <Link href="/finance/setups/vsf" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  const selectedSource = source.data || approvedBatches.find((b) => b.id === miLossBatchId) || row.miLossBatch;
  const suggested = latestMiLossAsOf(asOfBatches, effectiveFrom || date);

  return (
    <div className="space-y-4">
      <PageHeader
        title={row.documentNumber}
        subtitle="EWURA price and density per product · MI-loss from an approved MI-loss batch"
        actions={
          <Link href="/finance/setups/vsf" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      <Card className="space-y-4 p-6">
        <DocumentMeta crumb="Finance · Setups · Variable storage fee">
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
          <Field label="MI-loss batch">
            <Select value={miLossBatchId} disabled={!draft} onChange={(e) => changeMiLoss(e.target.value)}>
              {asOfBatches.length === 0 ? (
                <option value="">No approved MI-loss effective on or before this date</option>
              ) : null}
              {asOfBatches.map((b) => (
                <option key={b.id} value={b.id}>
                  {miLossBatchLabel(b)}
                </option>
              ))}
            </Select>
            {selectedSource && asOfBatches.some((b) => b.id === miLossBatchId) ? (
              <p className="mt-1 text-xs text-slate-500">
                Latest eligible is selected automatically. Using{" "}
                <Link href={`/finance/setups/miloss/${miLossBatchId}`} className="font-medium text-brand-800 hover:underline">
                  {miLossBatchLabel(selectedSource)}
                </Link>
                {suggested && suggested.id !== miLossBatchId
                  ? ` · you can switch to ${suggested.documentNumber}`
                  : null}
                . Only batches effective on or before this variable storage fee date are listed.
              </p>
            ) : draft ? (
              <p className="mt-1 text-xs text-slate-500">
                Set Effective from to the MI-loss date or later, or{" "}
                <Link href="/finance/setups/miloss" className="font-medium text-brand-800 hover:underline">
                  approve an MI-loss batch
                </Link>{" "}
                first.
              </p>
            ) : null}
          </Field>
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
        <div className="grid gap-3 sm:grid-cols-2">
          <ErpField label="Home currency" value={row.currencyCode || "—"} />
          <ErpField label="Created by" value={creatorLabel(row.createdBy)} />
        </div>
        {draft ? (
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => void save().catch(() => undefined)} loading={saving}>
              Save
            </Button>
            <Button
              variant="secondary"
              onClick={() => void submit()}
              disabled={!products.some((p) => p.product?.id && p.ewuraPrice.trim() && p.density.trim())}
            >
              Submit for approval
            </Button>
          </div>
        ) : null}
      </Card>
      <Card className="space-y-4 p-6">
        <ReviewTabBar
          tab={tab}
          onChange={setTab}
          ariaLabel="Variable storage fee sections"
          ids={[
            { id: "lines", label: "Products", count: products.length },
            { id: "files", label: "Attachments" },
            { id: "workflow", label: "Workflow" },
          ]}
        />
        {tab === "lines" ? (
          <VsfLines
            products={products}
            sourceLines={sourceLines}
            contracts={contractNames}
            exchangeRate={exchangeRate}
            draft={draft}
            canAdd={Boolean(miLossBatchId) && !source.loading}
            onAdd={startAdd}
            onEdit={(i, p) => setEditing({ index: i, product: { ...p } })}
            onRemove={(i) => void removeProduct(i)}
            onView={setViewing}
          />
        ) : null}
        {editing ? (
          <ProductModal
            product={
              editing === "new"
                ? { ewuraPrice: "", density: "", product: availableLines[0]?.product }
                : editing.product
            }
            lines={editing === "new" ? availableLines : sourceLines}
            contracts={contractNames}
            lockProduct={editing !== "new"}
            onClose={() => setEditing(null)}
            onSave={commitProduct}
          />
        ) : null}
        {viewing ? (
          <CatalogueForm
            title={productLabel(viewing.product) || "Product"}
            error=""
            saving={false}
            onClose={() => setViewing(null)}
            onSave={() => setViewing(null)}
            saveLabel="Close"
          >
            <ErpField label="Product" value={productLabel(viewing.product) || "—"} />
            <ErpField label="EWURA TZS/L" value={qty(viewing.ewuraPrice) || "—"} />
            <ErpField label="Density" value={qty(viewing.density) || "—"} />
            <ErpField label="Wholesale USD/m³" value={qty(viewing.wholesaleCm) || "—"} />
            <ErpField label="Wholesale USD/MT" value={qty(viewing.wholesaleMt) || "—"} />
          </CatalogueForm>
        ) : null}
        {tab === "files" ? <BatchFiles path={path} draft={draft} /> : null}
        {tab === "workflow" ? <BatchWorkflow path={path} /> : null}
      </Card>
    </div>
  );
}

function VsfLines({
  products,
  sourceLines,
  contracts,
  exchangeRate,
  draft,
  canAdd,
  onAdd,
  onEdit,
  onRemove,
  onView,
}: {
  products: VarProduct[];
  sourceLines: MiLossLine[];
  contracts: { code: string; name?: string }[];
  exchangeRate: string;
  draft: boolean;
  canAdd: boolean;
  onAdd: () => void;
  onEdit: (index: number, product: VarProduct) => void;
  onRemove: (index: number) => void;
  onView: (product: VarProduct) => void;
}) {
  const contractRows = products.flatMap((p, index) => {
    const rates: VarContract[] =
      p.contracts && p.contracts.length > 0
        ? p.contracts
        : sourceLines
            .filter((l) => l.product?.id === p.product?.id)
            .map((l) => ({ contractTypeCode: l.contractTypeCode, miLossValue: l.value }));
    return rates.map((c) => ({ product: p, index, contract: c }));
  });

  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-600">
        Each product appears once. EWURA TZS/L and density are constant across contracts. Wholesale m³
        = (EWURA × 1,000) / FX. Fee = MI-loss × wholesale. Click a product row to view it.
      </p>
      <SectionTable
        title="Products"
        actions={
          draft ? (
            <Button variant="secondary" className="!px-3 !py-1.5" onClick={onAdd} disabled={!canAdd}>
              Add product
            </Button>
          ) : null
        }
        columns={[
          { label: "Product" },
          { label: "EWURA TZS/L" },
          { label: "Density" },
          { label: "USD/m³" },
          { label: "USD/MT" },
          { label: "" },
        ]}
        empty="No products yet. Add a product from the MI-loss batch."
      >
        {products.map((p, i) => (
          <SectionRow key={p.product?.id || i} onClick={() => onView(p)}>
            <td className="font-semibold">{productLabel(p.product) || "—"}</td>
            <td className="tabular-nums">{qty(p.ewuraPrice)}</td>
            <td className="tabular-nums">{qty(p.density)}</td>
            <td className="tabular-nums">{qty(p.wholesaleCm) || "—"}</td>
            <td className="tabular-nums">{qty(p.wholesaleMt) || "—"}</td>
            <td className="text-right">
              {draft ? (
                <div className="flex justify-end gap-2">
                  <Button
                    variant="ghost"
                    onClick={(e) => {
                      e.stopPropagation();
                      onEdit(i, p);
                    }}
                  >
                    Edit
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={(e) => {
                      e.stopPropagation();
                      onRemove(i);
                    }}
                  >
                    Remove
                  </Button>
                </div>
              ) : null}
            </td>
          </SectionRow>
        ))}
      </SectionTable>
      <SectionTable
        title="Product × contract"
        columns={[
          { label: "Product" },
          { label: "Contract" },
          { label: "MI-loss %" },
          { label: "USD/m³" },
          { label: "USD/MT" },
          { label: "TZS/m³" },
          { label: "TZS/MT" },
        ]}
        empty="Contract fees appear after products are saved."
      >
        {contractRows.map((row, i) => (
          <SectionRow key={`${row.product.product?.id || row.index}-${row.contract.contractTypeCode}-${i}`}>
            <td className="font-semibold">{productLabel(row.product.product) || "—"}</td>
            <td>
              {catalogLabel(contracts.find((t) => t.code === row.contract.contractTypeCode)) ||
                row.contract.contractTypeCode ||
                "—"}
            </td>
            <td className="tabular-nums">{miLossPct(row.contract.miLossValue) || "—"}</td>
            <td className="tabular-nums">{qty(row.contract.feeUsdCm) || "—"}</td>
            <td className="tabular-nums">{qty(row.contract.feeUsdMt) || "—"}</td>
            <td className="tabular-nums">{qty(row.contract.feeTzsCm) || "—"}</td>
            <td className="tabular-nums">{qty(row.contract.feeTzsMt) || "—"}</td>
          </SectionRow>
        ))}
      </SectionTable>
      <SectionPanel title="Computation">
        <div className="grid gap-3 p-4 sm:grid-cols-2 lg:grid-cols-4">
          <ErpField label="FX (TZS/USD)" value={qty(exchangeRate) || "—"} />
          <ErpField label="Products" value={String(products.length)} />
          <ErpField label="Contract lines" value={String(contractRows.length)} />
          <ErpField
            label="Wholesale USD/m³"
            value={
              products
                .map((p) => qty(p.wholesaleCm))
                .filter(Boolean)
                .join(" · ") || "—"
            }
          />
        </div>
        <p className="border-t border-slate-200 px-4 py-3 text-[13px] leading-5 text-slate-700">
          Wholesale m³ = (EWURA TZS/L × 1,000) ÷ FX. Wholesale MT = wholesale m³ × density. Fee =
          MI-loss fraction × wholesale. One EWURA price applies to every contract on that product.
        </p>
      </SectionPanel>
    </div>
  );
}

function ProductModal({
  product,
  lines,
  contracts,
  lockProduct,
  onClose,
  onSave,
}: {
  product: VarProduct;
  lines: MiLossLine[];
  contracts: { code: string; name?: string }[];
  lockProduct: boolean;
  onClose: () => void;
  onSave: (p: VarProduct) => void | Promise<void>;
}) {
  const [draft, setDraft] = useState<VarProduct>(product);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const options = useMemo(() => {
    const seen = new Map<string, { value: string; label: string }>();
    for (const l of lines) {
      if (l.product?.id && !seen.has(l.product.id)) {
        seen.set(l.product.id, { value: l.product.id, label: productLabel(l.product) });
      }
    }
    return [...seen.values()];
  }, [lines]);

  const productLines = lines.filter((l) => l.product?.id === draft.product?.id);

  async function commit() {
    if (!draft.product?.id) {
      setError("Product is required.");
      notify.error("Product is required.", { title: "Check the values" });
      return;
    }
    if (!draft.ewuraPrice.trim()) {
      setError("EWURA price is required.");
      notify.error("EWURA price is required.", { title: "Check the values" });
      return;
    }
    const n = Number(draft.density);
    if (!Number.isFinite(n) || n <= 0 || n >= 2) {
      const msg = "Density must be MT per m³ (e.g. 0.84).";
      setError(msg);
      notify.error(msg, { title: "Check the values" });
      return;
    }
    setSaving(true);
    setError("");
    try {
      await onSave(draft);
    } catch (e) {
      const msg = formatApiError(e, "Could not save this product.");
      setError(msg);
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title="Product price"
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void commit()}
      saveLabel="Save product"
    >
      <Field label="Product">
        <SearchableSelect
          value={draft.product?.id || ""}
          disabled={lockProduct}
          placeholder="Product on the MI-loss batch"
          emptyLabel="No remaining products on this MI-loss batch"
          onChange={(value, option) =>
            setDraft((row) => ({
              ...row,
              product: value ? { id: value, name: option?.label } : undefined,
            }))
          }
          options={options}
        />
      </Field>
      <div className="space-y-2">
        <p className="text-[13px] font-semibold text-slate-900">Contracts from MI-loss</p>
        {productLines.length === 0 ? (
          <p className="text-sm text-slate-600">Choose a product to see its contracts.</p>
        ) : (
          productLines.map((l) => (
            <div key={l.contractTypeCode} className="grid gap-3 sm:grid-cols-2">
              <ErpField
                label="Contract"
                value={catalogLabel(contracts.find((t) => t.code === l.contractTypeCode)) || l.contractTypeCode || "—"}
              />
              <ErpField label="MI-loss %" value={miLossPct(l.value) || "—"} />
            </div>
          ))
        )}
      </div>
      <Field label="EWURA TZS/L">
        <QtyInput
          unit="price"
          value={draft.ewuraPrice}
          onChange={(ewuraPrice) => setDraft((row) => ({ ...row, ewuraPrice }))}
        />
      </Field>
      <Field label="Density (MT/m³)">
        <QtyInput unit="density" value={draft.density} onChange={(density) => setDraft((row) => ({ ...row, density }))} />
      </Field>
    </CatalogueForm>
  );
}
