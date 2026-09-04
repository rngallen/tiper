"use client";

import { useCallback, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import { Field, Input, ToggleField } from "@/components/ui";
import { SearchableSelect } from "@/components/SearchableSelect";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import type { Category, Product } from "@/lib/master";
import { CatalogueForm } from "../form";

export default function ProductsPage() {
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
          header: "Product",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "category",
          header: "Category",
          defaultVisible: true,
          cell: (r) => r.stockCategory?.name || "—",
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Product>[],
    [],
  );

  return (
    <MasterCatalogue<Product>
      tableId="master-products"
      title="Products"
      subtitle="Stock items handled at the depot (AGO, PMS, IK, HFO, …)"
      path="/master/products"
      orderBy="name"
      searchPlaceholder="Search code or name"
      createLabel="Add product"
      emptyMessage="No products found"
      columns={columns}
      inquiryTitle={(r) => r.name || r.code || "Product"}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Products · Inquiry"
            title={r.name || r.code || "Product"}
            pills={
              <StatusPill tone={r.isActive ? "navy" : "slate"}>
                {r.isActive ? "Active" : "Inactive"}
              </StatusPill>
            }
          />
          <InquirySection title="General">
            <ErpField label="Code" value={r.code} />
            <ErpField label="Category" value={r.stockCategory?.name || "—"} />
          </InquirySection>
        </div>
      )}
      Form={ProductForm}
    />
  );
}

function ProductForm({
  row,
  onClose,
  onSaved,
}: {
  row?: Product;
  onClose: () => void;
  onSaved: () => void;
}) {
  const cats = useAsync(() => api.get<Category[]>("/master/categories"), []);
  const [code, setCode] = useState(row?.code ?? "");
  const [name, setName] = useState(row?.name ?? "");
  const [stockCategoryId, setCat] = useState(row?.stockCategory?.id ?? "");
  const [isActive, setActive] = useState(row?.isActive ?? true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    if (!name.trim() || !code.trim()) {
      setError("Code and name are required.");
      return;
    }
    if (!stockCategoryId) {
      setError("Category is required.");
      return;
    }
    setSaving(true);
    try {
      const body = {
        code: code.trim(),
        name: name.trim(),
        unit: "L",
        stockCategoryId,
        isActive,
      };
      if (row) await api.put(`/master/products/${row.id}`, body);
      else await api.post("/master/products", body);
      onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save product"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? "Edit product" : "Add product"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
    >
      <Field
        label="Code"
        required
        hint={row ? "Code cannot be changed after the product is created." : undefined}
      >
        <Input
          value={code}
          onChange={(e) => setCode(e.target.value)}
          disabled={Boolean(row)}
          readOnly={Boolean(row)}
          placeholder="AGO"
          autoFocus={!row}
        />
      </Field>
      <Field label="Name" required>
        <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus={Boolean(row)} />
      </Field>
      <Field label="Category" required>
        <SearchableSelect
          value={stockCategoryId}
          placeholder="Search category"
          emptyLabel="No matching category"
          onChange={(value) => setCat(value)}
          options={(cats.data || [])
            .filter((c) => c.isActive !== false || c.id === stockCategoryId)
            .map((c) => ({ value: c.id, label: c.name || c.code || c.id, keywords: c.code }))}
        />
      </Field>
      <ToggleField label="Active" checked={isActive} onChange={setActive} />
    </CatalogueForm>
  );
}
