"use client";

import { useCallback, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import { Field, Input, Select, ToggleField } from "@/components/ui";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import type { ColumnDef } from "@/lib/columnPrefs";
import { stockStatusLabel, type StockStatusRef } from "@/lib/status";
import { CatalogueForm } from "../form";

type StatusRow = StockStatusRef & {
  isActive?: boolean;
};

function kindOf(s: StatusRow): string {
  if (s.isMining) return "Mining";
  if (s.isLocal) return "Local";
  if (s.isProration) return "Proration";
  if (s.isTransit) return "Transit";
  return "—";
}

export default function StatusesPage() {
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
          header: "Status",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => (
            <div className="font-medium text-slate-800">{stockStatusLabel(r)}</div>
          ),
        },
        {
          id: "kind",
          header: "Class",
          defaultVisible: true,
          cell: (r) => kindOf(r),
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<StatusRow>[],
    [],
  );

  return (
    <MasterCatalogue<StatusRow>
      tableId="master-statuses"
      title="Stock statuses"
      subtitle="Book-stock buckets. Add a new corridor here as class Transit (parent Transit) — no code change."
      path="/master/statuses"
      orderBy="name"
      searchPlaceholder="Search status"
      createLabel="Add status"
      emptyMessage="No stock statuses found"
      columns={columns}
      inquiryTitle={(r) => stockStatusLabel(r)}
      inquiry={(r) => (
        <div className="space-y-4">
          <InquiryHeader
            crumb="Statuses · Inquiry"
            title={stockStatusLabel(r)}
            pills={
              <StatusPill tone={r.isActive ? "navy" : "slate"}>
                {r.isActive ? "Active" : "Inactive"}
              </StatusPill>
            }
          />
          <InquirySection title="General">
            <ErpField label="Code" value={r.code} />
            <ErpField label="Class" value={kindOf(r)} />
            <ErpField label="Parent" value={r.parent?.name || "—"} />
          </InquirySection>
        </div>
      )}
      Form={StatusForm}
    />
  );
}

function StatusForm({
  row,
  onClose,
  onSaved,
}: {
  row?: StatusRow;
  onClose: () => void;
  onSaved: () => void;
}) {
  const all = useAsync(() => api.get<StatusRow[]>("/master/statuses"), []);
  const [name, setName] = useState(row?.name ?? "");
  const [kind, setKind] = useState(
    row?.isMining
      ? "mining"
      : row?.isProration
        ? "proration"
        : row?.isLocal
          ? "local"
          : "transit",
  );
  const [parentId, setParent] = useState(row?.parent?.id ?? "");
  const [isActive, setActive] = useState(row?.isActive ?? true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const transitRoots = (all.data || []).filter((s) => s.isTransit && !s.parent);

  async function save() {
    setError("");
    if (!name.trim()) {
      setError("Name is required.");
      return;
    }
    setSaving(true);
    try {
      const body = {
        name: name.trim(),
        isTransit: kind === "transit",
        isLocal: kind === "local",
        isMining: kind === "mining",
        isProration: kind === "proration",
        parentId: kind === "transit" ? parentId : "",
        isActive,
      };
      if (row) await api.put(`/master/statuses/${row.id}`, body);
      else await api.post("/master/statuses", body);
      onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save status"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? "Edit status" : "Add status"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
    >
      <Field label="Name">
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Uganda"
          autoFocus
        />
      </Field>
      <Field label="Class">
        <Select value={kind} onChange={(e) => setKind(e.target.value)}>
          <option value="transit">Transit (rolls into transit totals)</option>
          <option value="local">Local</option>
          <option value="mining">Mining (local subclass)</option>
          <option value="proration">Proration</option>
        </Select>
      </Field>
      {kind === "transit" && (
        <Field label="Parent">
          <Select value={parentId} onChange={(e) => setParent(e.target.value)}>
            <option value="">Transit (default)</option>
            {transitRoots
              .filter((s) => s.id !== row?.id)
              .map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
          </Select>
        </Field>
      )}
      <ToggleField label="Active" checked={isActive} onChange={setActive} />
    </CatalogueForm>
  );
}
