"use client";

import { useCallback, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { Field, Input, ToggleField } from "@/components/ui";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  ReviewTabBar,
  StatusPill,
} from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import { searchMaster, type Product, type Tank } from "@/lib/master";
import { qty } from "@/lib/master";
import { DocumentFiles } from "@/components/DocumentFiles";
import { QtyInput } from "@/components/QtyInput";
import { CatalogueForm, PartyWorkspace } from "../form";

type TankTab = "details" | "calibration";

function litres(v: string | number | undefined): number {
  const n = Number(String(v ?? "").replace(/,/g, "").trim());
  return Number.isFinite(n) ? n : NaN;
}

function usableLitres(cap?: string | number, dead?: string | number): string {
  const c = litres(cap);
  const d = litres(dead);
  if (!Number.isFinite(c) || !Number.isFinite(d)) return "—";
  return qty(c - d);
}

export default function TanksPage() {
  const columns = useCallback(
    () =>
      [
        {
          id: "code",
          header: "Code",
          sortKey: "code",
          defaultVisible: true,
          cell: (r) => (
            <span className="font-mono text-xs text-slate-600">{r.code}</span>
          ),
        },
        {
          id: "name",
          header: "Tank",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "product",
          header: "Product",
          defaultVisible: true,
          cell: (r) => r.product?.name || "—",
        },
        {
          id: "maximumCapacity",
          header: "Capacity (L)",
          defaultVisible: true,
          cell: (r) => qty(r.maximumCapacity),
        },
        {
          id: "deadStock",
          header: "Dead stock (L)",
          defaultVisible: false,
          cell: (r) => qty(r.deadStock),
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Tank>[],
    [],
  );

  return (
    <MasterCatalogue<Tank>
      tableId="master-tanks"
      title="Tanks"
      subtitle="Physical storage tanks at TIPER"
      path="/master/tanks"
      orderBy="code"
      searchPlaceholder="Search code or name"
      createLabel="Add tank"
      emptyMessage="No tanks found"
      columns={columns}
      inquiryTitle={(r) => r.name || r.code || "Tank"}
      inquirySize="xl"
      inquiry={(r) => <TankInquiry row={r} />}
      Form={TankForm}
    />
  );
}

function TankInquiry({ row }: { row: Tank }) {
  const [tab, setTab] = useState<TankTab>("details");
  return (
    <div className="space-y-4">
      <InquiryHeader
        crumb="Stock · Setups · Tanks"
        title={row.name || row.code || "Tank"}
        subtitle={row.code ? `Code ${row.code}` : undefined}
        pills={
          <StatusPill tone={row.isActive ? "navy" : "slate"}>
            {row.isActive ? "Active" : "Inactive"}
          </StatusPill>
        }
      />
      <ReviewTabBar
        tab={tab}
        onChange={setTab}
        ariaLabel="Tank record"
        ids={[
          { id: "details", label: "Tank details" },
          { id: "calibration", label: "Calibration" },
        ]}
      />
      {tab === "details" ? (
        <InquirySection title="General">
          <ErpField label="Code" value={row.code || "—"} />
          <ErpField label="Name" value={row.name || "—"} />
          <ErpField label="Product" value={row.product?.name || "—"} />
          <ErpField label="Capacity (L)" value={qty(row.maximumCapacity)} />
          <ErpField label="Dead stock (L)" value={qty(row.deadStock)} />
          <ErpField
            label="Usable volume (L)"
            value={usableLitres(row.maximumCapacity, row.deadStock)}
          />
        </InquirySection>
      ) : (
        <InquirySection title="Calibration files" columns={1}>
          <DocumentFiles
            path={`/master/tanks/${row.id}`}
            draft={false}
            hint="Charts and certificates are read-only here. Open Edit → Calibration to upload or remove files."
          />
        </InquirySection>
      )}
    </div>
  );
}

function TankForm({
  row,
  onClose,
  onSaved,
}: {
  row?: Tank;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [code, setCode] = useState(row?.code ?? "");
  const [name, setName] = useState(row?.name ?? "");
  const [maximumCapacity, setCap] = useState(qty(row?.maximumCapacity).replace("—", ""));
  const [deadStock, setDead] = useState(qty(row?.deadStock).replace("—", ""));
  const [productId, setProduct] = useState(row?.product?.id ?? "");
  const [productLabel, setProductLabel] = useState(
    row?.product ? `${row.product.code ?? ""} — ${row.product.name ?? ""}`.trim() : "",
  );
  const [isActive, setActive] = useState(row?.isActive ?? true);
  const [pane, setPane] = useState<"details" | "calibration">("details");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    if (!code.trim() || !name.trim()) {
      setError("Code and name are required.");
      return;
    }
    if (!productId) {
      setError("Product is required.");
      return;
    }
    const cap = litres(maximumCapacity);
    const dead = litres(deadStock);
    if (!Number.isFinite(cap) || !Number.isFinite(dead)) {
      setError("Capacity and dead stock must be numbers.");
      return;
    }
    if (!(cap > dead)) {
      setError("Capacity must be greater than dead stock.");
      return;
    }
    setSaving(true);
    try {
      const body = {
        code: code.trim(),
        name: name.trim(),
        maximumCapacity,
        deadStock,
        productId,
        isActive,
      };
      if (row) await api.put(`/master/tanks/${row.id}`, body);
      else await api.post("/master/tanks", body);
      onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save tank"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? "Edit tank" : "Add tank"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
      size="xl"
    >
      {row ? (
        <PartyWorkspace
          panes={[
            { id: "details", label: "Tank details" },
            { id: "calibration", label: "Calibration" },
          ]}
          pane={pane}
          onPane={setPane}
        >
          {pane === "details" ? (
            <TankDetailsFields
              row={row}
              code={code}
              setCode={setCode}
              name={name}
              setName={setName}
              productId={productId}
              productLabel={productLabel}
              setProduct={setProduct}
              setProductLabel={setProductLabel}
              maximumCapacity={maximumCapacity}
              setCap={setCap}
              deadStock={deadStock}
              setDead={setDead}
              isActive={isActive}
              setActive={setActive}
            />
          ) : (
            <DocumentFiles
              path={`/master/tanks/${row.id}`}
              draft
              hint="Calibration charts and tank certificates. Drop zone stays at the top so it stays visible as the list grows."
              layout="stack"
            />
          )}
        </PartyWorkspace>
      ) : (
        <>
          <TankDetailsFields
            code={code}
            setCode={setCode}
            name={name}
            setName={setName}
            productId={productId}
            productLabel={productLabel}
            setProduct={setProduct}
            setProductLabel={setProductLabel}
            maximumCapacity={maximumCapacity}
            setCap={setCap}
            deadStock={deadStock}
            setDead={setDead}
            isActive={isActive}
            setActive={setActive}
          />
          <p className="text-xs leading-5 text-slate-500">
            Capacity must be greater than dead stock. After create, open Edit to
            attach calibration charts and certificates.
          </p>
        </>
      )}
    </CatalogueForm>
  );
}

function TankDetailsFields({
  row,
  code,
  setCode,
  name,
  setName,
  productId,
  productLabel,
  setProduct,
  setProductLabel,
  maximumCapacity,
  setCap,
  deadStock,
  setDead,
  isActive,
  setActive,
}: {
  row?: Tank;
  code: string;
  setCode: (v: string) => void;
  name: string;
  setName: (v: string) => void;
  productId: string;
  productLabel: string;
  setProduct: (v: string) => void;
  setProductLabel: (v: string) => void;
  maximumCapacity: string;
  setCap: (v: string) => void;
  deadStock: string;
  setDead: (v: string) => void;
  isActive: boolean;
  setActive: (v: boolean) => void;
}) {
  return (
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Code">
          <Input
            value={code}
            onChange={(e) => setCode(e.target.value)}
            disabled={Boolean(row)}
            autoFocus={!row}
          />
        </Field>
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus={Boolean(row)} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Product">
            <AsyncSearchableSelect
              value={productId}
              selectedLabel={productLabel}
              placeholder="Search product"
              emptyLabel="No matching product"
              onChange={(value, option) => {
                setProduct(value);
                setProductLabel(option?.label || value);
              }}
              loadOptions={async (q) => {
                const rows = await searchMaster<Product>("/master/products", q);
                return rows.map((p) => ({
                  value: p.id,
                  label: `${p.code} — ${p.name}`,
                }));
              }}
            />
          </Field>
        </div>
        <Field label="Maximum capacity (L)">
          <QtyInput value={maximumCapacity} onChange={setCap} unit="l" />
        </Field>
        <Field label="Dead stock (L)">
          <QtyInput value={deadStock} onChange={setDead} unit="l" />
        </Field>
        <div className="sm:col-span-2">
          <ToggleField
            label="Active"
            description="Inactive tanks are hidden from new receipts and dips"
            checked={isActive}
            onChange={setActive}
          />
        </div>
      </div>
  );
}
