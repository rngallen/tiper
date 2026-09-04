"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { api, ApiError, formatApiError } from "@/lib/api";
import { notify } from "@/lib/notify";
import { useAsync } from "@/lib/hooks";
import { useAuth } from "@/lib/auth";
import { PRICE_SUBMIT_PERMS, PRICE_WRITE_PERMS } from "@/lib/permissions";
import { CatalogueForm } from "@/app/(dashboard)/stock/setups/form";
import { DocumentMeta, ErpField, InquirySection, ReviewTabBar, StatusPill } from "@/components/ApprovalReview";
import { AsyncSearchableSelect, SearchableSelect } from "@/components/SearchableSelect";
import { Button, Card, ErrorBanner, Field, Input, PageHeader, Spinner } from "@/components/ui";
import { DateInput } from "@/components/ui";
import { QtyInput } from "@/components/QtyInput";
import { formatDate, parseQty } from "@/lib/format";
import {
  clampEffectiveToDoc,
  effectiveBeforeDocError,
  feeEditable,
  isoDay,
  miLossFraction,
  miLossPct,
  miLossPercentError,
  productLabel,
  productOptions,
  type MiLossBatch,
  type MiLossLine,
} from "@/lib/billing";
import { creatorLabel } from "@/lib/creator";
import { activeRows, catalogLabel, fetchLookups } from "@/lib/lookups";
import type { MasterRef } from "@/lib/master";
import { SectionGroupRow, SectionRow, SectionTable } from "@/components/SectionTable";
import { BatchFiles } from "../../_components/BatchFiles";
import { BatchWorkflow } from "../../_components/BatchWorkflow";

const writePerms = PRICE_WRITE_PERMS;
const submitPerms = PRICE_SUBMIT_PERMS;

function rateKey(line: MiLossLine, batchDay: string): string {
  return `${line.product?.id || ""}|${line.contractTypeCode}|${isoDay(line.effectiveFrom || batchDay)}`;
}

function refsFromBatch(row: MiLossBatch): MasterRef[] {
  const seen = new Map<string, MasterRef>();
  for (const p of row.products || []) {
    const id = p.product?.id || p.productId;
    if (!id) continue;
    seen.set(id, p.product?.id ? p.product : { id });
  }
  for (const l of row.lines || []) {
    if (l.product?.id && !seen.has(l.product.id)) seen.set(l.product.id, l.product);
  }
  return [...seen.values()];
}

export default function MiLossDocumentPage() {
  const { id } = useParams<{ id: string }>();
  const path = `/billing/miloss/batches/${id}`;
  const doc = useAsync(() => api.get<MiLossBatch>(path), [id]);
  const lookups = useAsync(fetchLookups, []);
  const contracts = activeRows(lookups.data?.contractTypes);
  const { hasPermission } = useAuth();
  const canWrite = hasPermission(...writePerms);
  const canSubmit = hasPermission(...submitPerms);
  const [tab, setTab] = useState<"files" | "workflow">("files");
  const [date, setDate] = useState("");
  const [effectiveFrom, setEff] = useState("");
  const [description, setDescription] = useState("");
  const [lines, setLines] = useState<MiLossLine[]>([]);
  const [products, setProducts] = useState<MasterRef[]>([]);
  const [editing, setEditing] = useState<{ line: MiLossLine; index: number } | "new" | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (doc.data) applyBatch(doc.data);
  }, [doc.data]);

  function applyBatch(row: MiLossBatch) {
    setDate(isoDay(row.date || row.effectiveFrom));
    setEff(isoDay(row.effectiveFrom));
    setDescription(row.description || "");
    const next = (row.products?.length
      ? row.products.flatMap((p) =>
          (p.rates || []).map((l) => ({
            ...l,
            product: l.product || p.product || (p.productId ? { id: p.productId } : undefined),
          })),
        )
      : row.lines || []
    ).map((l) => ({
      id: l.id,
      product: l.product,
      contractTypeCode: l.contractTypeCode,
      value: miLossPct(l.value),
      effectiveFrom: l.effectiveFrom || row.effectiveFrom,
    }));
    setLines(next);
    setProducts(refsFromBatch(row));
  }

  const row = doc.data;
  const draft = feeEditable(row?.status) && canWrite;

  async function save(nextLines = lines, nextProducts = products, silent = false): Promise<void> {
    const dateErr = effectiveBeforeDocError(date, effectiveFrom);
    if (dateErr) {
      notify.error(dateErr, { title: "Cannot save" });
      throw new Error(dateErr);
    }
    setSaving(true);
    try {
      const saved = await api.put<MiLossBatch>(
        path,
        {
          date,
          effectiveFrom,
          description: description.trim(),
          products: nextProducts
            .filter((p) => p.id)
            .map((p) => ({
              productId: p.id,
              rates: nextLines
                .filter((l) => l.product?.id === p.id && l.contractTypeCode && l.value.trim())
                .map((l) => ({
                  contractTypeCode: l.contractTypeCode,
                  value: miLossFraction(parseQty(l.value) || l.value),
                })),
            })),
        },
        silent ? { success: false } : undefined,
      );
      if (saved) applyBatch(saved);
      else {
        setLines(nextLines);
        setProducts(nextProducts);
      }
    } finally {
      setSaving(false);
    }
  }

  async function printPdf() {
    try {
      await api.download(`${path}/pdf`);
    } catch {
      /* message box from api */
    }
  }

  async function submit() {
    const ready = lines.filter((l) => l.product?.id && l.contractTypeCode && l.value.trim());
    if (ready.length === 0) {
      notify.error("Add at least one MI-loss rate before submitting for approval.", {
        title: "Cannot submit",
      });
      return;
    }
    try {
      await save(ready, products, true);
      await api.post(`${path}/submit`, {});
      doc.reload();
    } catch {
      /* message box from api */
    }
  }

  async function addProduct(value: string, option?: { label?: string }) {
    if (!value || products.some((p) => p.id === value)) return;
    const [code, ...rest] = (option?.label || "").split(" — ");
    const added = { id: value, code: code || undefined, name: rest.join(" — ") || option?.label };
    setProducts((prev) => (prev.some((p) => p.id === value) ? prev : [...prev, added]));
    setSaving(true);
    try {
      const saved = await api.post<MiLossBatch>(`${path}/products`, { productId: value }, { success: false });
      if (saved) applyBatch(saved);
    } catch {
      setProducts((prev) => prev.filter((p) => p.id !== value));
    } finally {
      setSaving(false);
    }
  }

  async function removeProduct(id: string) {
    const prevProducts = products;
    const prevLines = lines;
    setProducts((prev) => prev.filter((p) => p.id !== id));
    setLines((prev) => prev.filter((l) => l.product?.id !== id));
    setSaving(true);
    try {
      const saved = await api.del<MiLossBatch>(`${path}/products/${id}`, { success: false });
      if (saved) applyBatch(saved);
    } catch {
      setProducts(prevProducts);
      setLines(prevLines);
    } finally {
      setSaving(false);
    }
  }

  async function commitLine(line: MiLossLine) {
    if (!line.product?.id || !line.contractTypeCode || !line.value.trim()) {
      throw new Error("Product, contract, and MI-loss percent are required.");
    }
    const pctErr = miLossPercentError(parseQty(line.value) || line.value);
    if (pctErr) throw new Error(pctErr);
    const skip = editing && editing !== "new" && editing.index >= 0 ? editing.index : -1;
    const day = isoDay(effectiveFrom);
    if (lines.some((r, i) => i !== skip && rateKey(r, day) === rateKey(line, day))) {
      throw new Error("This product, contract, and effective date already have an MI-loss rate.");
    }
    const isNew = !editing || editing === "new" || editing.index < 0;
    if (isNew) {
      const saved = await api.post<MiLossBatch>(
        `${path}/rates`,
        {
          productId: line.product.id,
          contractTypeCode: line.contractTypeCode,
          value: miLossFraction(parseQty(line.value) || line.value),
        },
        { success: false },
      );
      if (saved) applyBatch(saved);
      setEditing(null);
      return;
    }
    const next = lines.map((r, i) => (i === editing.index ? line : r));
    await save(next, products, true);
    setEditing(null);
  }

  async function removeLine(index: number) {
    await save(
      lines.filter((_, idx) => idx !== index),
      products,
    );
  }

  function unusedContracts(productId?: string, keepCode?: string) {
    const used = new Set(
      lines
        .filter((l) => l.product?.id === productId && l.contractTypeCode && l.contractTypeCode !== keepCode)
        .map((l) => l.contractTypeCode),
    );
    return contracts.filter((c) => !used.has(c.code));
  }

  if (doc.loading && !row) return <Spinner />;
  if (!row) {
    return (
      <div className="space-y-4">
        <ErrorBanner message={doc.error || "MI loss batch not found"} />
        <Link href="/finance/setups/miloss" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title={row.documentNumber}
        subtitle="Choose products on the batch, then map each product to one or more contracts"
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="secondary" onClick={() => void printPdf()}>
              Print
            </Button>
            <Link href="/finance/setups/miloss" className="text-sm font-medium text-slate-600 hover:text-slate-900">
              Back to list
            </Link>
          </div>
        }
      />
      <Card className="space-y-4 p-6">
        <DocumentMeta crumb="Finance · Setups · MI loss">
          <StatusPill tone="navy">{row.status}</StatusPill>
        </DocumentMeta>
        <InquirySection title="Batch" columns={3}>
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
          <div className="sm:col-span-2 xl:col-span-1">
            <Field label="Description">
              <Input
                value={description}
                maxLength={200}
                disabled={!draft}
                onChange={(e) => setDescription(e.target.value)}
              />
            </Field>
          </div>
          <ErpField label="Created by" value={creatorLabel(row.createdBy)} />
          <ErpField label="Rates" value={String(lines.length)} />
        </InquirySection>
        <div>
          <p className="mb-1.5 text-[13px] font-semibold text-slate-900">Products on this batch</p>
          <p className="mb-2 text-[13px] leading-5 text-slate-700">
            Selecting a product saves it on this batch. Search then shows only products not yet added.
            On the row, add rates — unique together are product, contract, and effective date.
          </p>
          {draft ? (
            <div className="mb-3 max-w-md">
              <AsyncSearchableSelect
                key={products.map((p) => p.id).join("|")}
                value=""
                selectedLabel=""
                placeholder={saving ? "Saving…" : "Search to add a product"}
                emptyLabel="No other products available"
                disabled={saving}
                onChange={(value, option) => void addProduct(value, option)}
                loadOptions={(q) => productOptions(q, products.map((p) => p.id))}
              />
            </div>
          ) : null}
          <BatchItemsTable
            products={products}
            lines={lines}
            contracts={contracts}
            effectiveFrom={effectiveFrom}
            draft={draft}
            lookupsReady={!lookups.loading}
            onAddRate={(product) => {
              const open = unusedContracts(product.id);
              setEditing({
                line: {
                  product,
                  contractTypeCode: open[0]?.code || "",
                  value: "",
                  effectiveFrom,
                },
                index: -1,
              });
            }}
            onEditRate={(line, index) => setEditing({ line: { ...line }, index })}
            onRemoveRate={(index) => void removeLine(index)}
            onRemoveProduct={removeProduct}
          />
          {editing ? (
            <MiLossContractModal
              line={
                editing === "new"
                  ? { contractTypeCode: contracts[0]?.code || "", value: "", effectiveFrom }
                  : { ...editing.line, effectiveFrom: effectiveFrom || editing.line.effectiveFrom }
              }
              productLocked={editing !== "new"}
              contracts={unusedContracts(
                editing === "new" ? undefined : editing.line.product?.id,
                editing === "new" ? undefined : editing.line.contractTypeCode,
              )}
              onClose={() => setEditing(null)}
              onSave={commitLine}
            />
          ) : null}
        </div>
        {draft ? (
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => void save().catch(() => undefined)} loading={saving}>
              Save
            </Button>
            {canSubmit ? (
              <Button variant="secondary" onClick={() => void submit()} disabled={lines.length === 0}>
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
          ariaLabel="MI loss sections"
          ids={[
            { id: "files", label: "Attachments" },
            { id: "workflow", label: "Workflow" },
          ]}
        />
        {tab === "files" ? <BatchFiles path={path} draft={draft} /> : null}
        {tab === "workflow" ? <BatchWorkflow path={path} /> : null}
      </Card>
    </div>
  );
}

function BatchItemsTable({
  products,
  lines,
  contracts,
  effectiveFrom,
  draft,
  lookupsReady,
  onAddRate,
  onEditRate,
  onRemoveRate,
  onRemoveProduct,
}: {
  products: MasterRef[];
  lines: MiLossLine[];
  contracts: { code: string; name?: string }[];
  effectiveFrom: string;
  draft: boolean;
  lookupsReady: boolean;
  onAddRate: (product: MasterRef) => void;
  onEditRate: (line: MiLossLine, index: number) => void;
  onRemoveRate: (index: number) => void;
  onRemoveProduct: (id: string) => void;
}) {
  return (
    <SectionTable
      title="Products"
      columns={[
        { label: "Product" },
        { label: "Contract" },
        { label: "MI-loss %" },
        { label: "Effective from" },
        { label: "" },
      ]}
      empty={draft ? "Search above to add a product, then add MI-loss on that row." : "No products"}
    >
      {products.map((product) => {
        const rows = lines
          .map((line, index) => ({ line, index }))
          .filter((r) => r.line.product?.id === product.id);
        const used = new Set(rows.map((r) => r.line.contractTypeCode));
        const allMapped =
          lookupsReady && contracts.length > 0 && contracts.every((c) => used.has(c.code));
        return [
          <SectionGroupRow
            key={`${product.id}-group`}
            colSpan={5}
            actions={
              draft ? (
                <div className="flex justify-end gap-2">
                  {!allMapped ? (
                    <Button variant="ghost" onClick={() => onAddRate(product)}>
                      Add MI-loss
                    </Button>
                  ) : (
                    <span className="self-center text-sm font-medium text-slate-600">All contracts mapped</span>
                  )}
                  <Button variant="ghost" onClick={() => onRemoveProduct(product.id)}>
                    Remove
                  </Button>
                </div>
              ) : null
            }
          >
            {productLabel(product) || product.id}
            <span className="ml-2 text-[13px] font-medium text-slate-600">
              {rows.length === 0
                ? "No MI-loss yet"
                : `${rows.length} contract${rows.length === 1 ? "" : "s"}`}
            </span>
          </SectionGroupRow>,
          rows.length === 0 ? (
            <SectionRow key={`${product.id}-empty`}>
              <td colSpan={5} className="font-medium text-slate-700">
                {draft
                  ? "Click Add MI-loss to map a contract and percent for this product."
                  : "No MI-loss rates"}
              </td>
            </SectionRow>
          ) : (
            rows.map(({ line, index }) => (
              <SectionRow key={line.id || rateKey(line, effectiveFrom)}>
                <td className="text-slate-600">{product.code || ""}</td>
                <td className="font-semibold">
                  {catalogLabel(contracts.find((c) => c.code === line.contractTypeCode)) ||
                    line.contractTypeCode}
                </td>
                <td className="font-semibold tabular-nums">{line.value || "—"}</td>
                <td>{formatDate(line.effectiveFrom || effectiveFrom)}</td>
                <td className="text-right">
                  {draft ? (
                    <div className="flex justify-end gap-2">
                      <Button variant="ghost" onClick={() => onEditRate(line, index)}>
                        Edit
                      </Button>
                      <Button variant="ghost" onClick={() => onRemoveRate(index)}>
                        Remove
                      </Button>
                    </div>
                  ) : null}
                </td>
              </SectionRow>
            ))
          ),
        ];
      })}
    </SectionTable>
  );
}

function MiLossContractModal({
  line,
  productLocked,
  contracts,
  onClose,
  onSave,
}: {
  line: MiLossLine;
  productLocked: boolean;
  contracts: { code: string; name?: string }[];
  onClose: () => void;
  onSave: (line: MiLossLine) => void | Promise<void>;
}) {
  const [draft, setDraft] = useState<MiLossLine>(line);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  async function commit() {
    if (!draft.product?.id) {
      setError("Product is required.");
      return;
    }
    if (!draft.contractTypeCode) {
      setError("Contract is required.");
      return;
    }
    if (!draft.value.trim()) {
      setError("MI-loss percent is required.");
      return;
    }
    const pctErr = miLossPercentError(parseQty(draft.value) || draft.value);
    if (pctErr) {
      setError(pctErr);
      notify.error(pctErr, { title: "Check the values" });
      return;
    }
    setSaving(true);
    setError("");
    try {
      await onSave(draft);
    } catch (e) {
      const msg = formatApiError(e, "Could not save this MI-loss rate.");
      setError(msg);
      if (!(e instanceof ApiError)) {
        notify.error(msg, { title: "Cannot save rate" });
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={productLocked ? "Map contract" : "MI-loss rate"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void commit()}
      saveLabel="Save rate"
    >
      {productLocked ? (
        <ErpField label="Product" value={productLabel(draft.product) || "—"} />
      ) : (
        <Field label="Product">
          <AsyncSearchableSelect
            value={draft.product?.id || ""}
            selectedLabel={productLabel(draft.product)}
            placeholder="Search product"
            onChange={(value, option) =>
              setDraft((row) => ({ ...row, product: value ? { id: value, name: option?.label } : undefined }))
            }
            loadOptions={productOptions}
          />
        </Field>
      )}
      <Field label="Effective from">
        <DateInput value={isoDay(draft.effectiveFrom)} disabled />
        <p className="mt-1 text-xs font-medium text-slate-700">Taken from the batch.</p>
      </Field>
      <Field label="Contract" required>
        <SearchableSelect
          value={draft.contractTypeCode}
          placeholder="Search contract"
          emptyLabel="No remaining contract for this product"
          onChange={(value) => setDraft((row) => ({ ...row, contractTypeCode: value }))}
          options={contracts.map((c) => ({
            value: c.code,
            label: catalogLabel(c) || c.code,
            keywords: c.name,
          }))}
        />
      </Field>
      <Field label="MI-loss %">
        <QtyInput unit="miloss" value={draft.value} onChange={(value) => setDraft((row) => ({ ...row, value }))} />
        <p className="mt-1 text-xs font-medium text-slate-700">
          Percent, greater than 0 and less than 100. Typical values are around 0.5.
        </p>
      </Field>
    </CatalogueForm>
  );
}
