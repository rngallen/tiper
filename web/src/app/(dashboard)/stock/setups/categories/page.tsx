"use client";

import { useCallback, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import { Field, Input, ToggleField } from "@/components/ui";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import type { Category } from "@/lib/master";
import { CatalogueForm } from "../form";

export default function ProductCategoriesPage() {
  const columns = useCallback(
    () =>
      [
        {
          id: "name",
          header: "Category",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Category>[],
    [],
  );

  return (
    <MasterCatalogue<Category>
      tableId="master-categories"
      title="Product categories"
      subtitle="Required grouping for every stock item (Petroleum products)"
      path="/master/categories"
      orderBy="name"
      searchPlaceholder="Search category"
      createLabel="Add category"
      emptyMessage="No categories found"
      noun="category"
      columns={columns}
      inquiryTitle={(r) => r.name || "Category"}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Product categories · Inquiry"
            title={r.name || "Category"}
            pills={
              <StatusPill tone={r.isActive ? "navy" : "slate"}>
                {r.isActive ? "Active" : "Inactive"}
              </StatusPill>
            }
          />
          <InquirySection title="General">
            <ErpField label="Name" value={r.name} />
          </InquirySection>
        </div>
      )}
      Form={CategoryForm}
    />
  );
}

function CategoryForm({
  row,
  onClose,
  onSaved,
}: {
  row?: Category;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(row?.name ?? "");
  const [isActive, setActive] = useState(row?.isActive ?? true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    if (!name.trim()) {
      setError("Name is required.");
      return;
    }
    setSaving(true);
    try {
      const body = { name: name.trim(), isActive };
      if (row) await api.put(`/master/categories/${row.id}`, body);
      else await api.post("/master/categories", body);
      onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save category"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? "Edit category" : "Add category"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
    >
      <Field label="Name">
        <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      </Field>
      <ToggleField label="Active" checked={isActive} onChange={setActive} />
    </CatalogueForm>
  );
}
