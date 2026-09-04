"use client";

import { useMemo, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ExportButton, ListSearch, useListQuery } from "@/lib/list";
import { DataTable } from "@/components/DataTable";
import { EditColumnsButton } from "@/components/EditColumns";
import { useColumnPrefs, type ColumnDef } from "@/lib/columnPrefs";
import { Modal } from "@/components/Modal";
import { Paginator } from "@/components/Paginator";
import { ActiveCell } from "@/components/MasterCatalogue";
import { Button, Field, Input, ToggleField } from "@/components/ui";
import { ACTIVE_STATUS_PILLS } from "@/components/StatusPills";
import { ListShell } from "@/components/ListShell";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { asPagination } from "@/lib/pagination";
import { CatalogueForm } from "@/app/(dashboard)/stock/setups/form";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import type { CatalogRow, CycleRow } from "@/lib/lookups";

export type FlagCol = { key: string; label: string; short: string };

function flagOn(row: object, key: string): boolean {
  return Boolean((row as Record<string, unknown>)[key]);
}

const writePerms = [
  PERMISSIONS.masterdataCreate,
  PERMISSIONS.masterdataUpdate,
  PERMISSIONS.masterdataManage,
  PERMISSIONS.billingManage,
];

export function CatalogSetup<T extends CatalogRow>({
  tableId,
  title,
  subtitle,
  path,
  createLabel,
  singular,
  extra,
}: {
  tableId: string;
  title: string;
  subtitle: string;
  path: string;
  createLabel: string;
  singular: string;
  extra: FlagCol[];
}) {
  const list = useListQuery({ orderBy: "code", sortDirection: "ASC" });
  const [isActive, setIsActive] = useState("");
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<T | null>(null);
  const [viewing, setViewing] = useState<T | null>(null);
  const { hasPermission } = useAuth();
  const canWrite = hasPermission(...writePerms);

  const query = { ...list.query, isActive: isActive || undefined };
  const { data, loading, error, reload } = useAsync(
    () => api.get<T[]>(path, query),
    [list.search, list.page, list.pageSize, list.orderBy, list.sortDirection, isActive],
  );

  const catalog = useMemo(
    () =>
      [
        {
          id: "code",
          header: "Code",
          sortKey: "code",
          defaultVisible: true,
          cell: (r: T) => (
            <span className="font-mono text-xs text-slate-600">{r.code}</span>
          ),
        },
        {
          id: "name",
          header: "Name",
          sortKey: "name",
          defaultVisible: true,
          cell: (r: T) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        ...extra.map((e) => ({
          id: e.key,
          header: e.short,
          defaultVisible: true,
          cell: (r: T) => (flagOn(r, e.key) ? "Yes" : "—"),
        })),
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r: T) => <ActiveCell active={r.isActive} />,
        },
        {
          id: "_actions",
          fixed: true,
          header: "",
          className: "text-right",
          cell: (r: T) =>
            canWrite ? (
              <Button
                variant="secondary"
                className="!px-3 !py-1"
                onClick={(e) => {
                  e.stopPropagation();
                  setEditing(r);
                }}
              >
                Edit
              </Button>
            ) : null,
        },
      ] satisfies ColumnDef<T>[],
    [canWrite, extra],
  );

  const { columns, pref, save, restoreDefaults } = useColumnPrefs(tableId, catalog);
  const page = asPagination<T>(data);

  return (
    <div>
      <p className="mb-1 text-xs font-medium text-slate-500">Finance › Setups</p>
      <ListShell
        title={title}
        subtitle={subtitle}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <EditColumnsButton
              catalog={catalog}
              pref={pref}
              onApply={save}
              onRestore={restoreDefaults}
            />
            <ExportButton path={path} query={query} />
            {canWrite ? (
              <Button onClick={() => setCreating(true)}>{createLabel}</Button>
            ) : null}
          </div>
        }
        pills={{
          options: ACTIVE_STATUS_PILLS,
          value: isActive,
          allTitle: "All records",
          guideTitle: "Record status",
          onChange: (id) => {
            setIsActive(id);
            list.setPage(1);
          },
        }}
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Search code or name"
          />
        }
        footer={
          !loading && !error ? (
            <Paginator
              embedded
              data={page}
              page={list.page}
              onPage={list.setPage}
              pageSize={list.pageSize}
              onPageSize={list.setPageSize}
            />
          ) : null
        }
      >
        <DataTable
          plain
          columns={columns}
          rows={page?.items}
          loading={loading}
          error={error}
          emptyMessage={`No ${title.toLowerCase()} found`}
          sort={list.sort}
          onSort={list.onSort}
          onRowClick={(r) => setViewing(r)}
        />
      </ListShell>

      {viewing ? (
        <Modal
          open
          size="md"
          title={viewing.name || viewing.code}
          onClose={() => setViewing(null)}
          footer={
            <>
              <Button variant="secondary" onClick={() => setViewing(null)}>
                Close
              </Button>
              {canWrite ? (
                <Button
                  onClick={() => {
                    setEditing(viewing);
                    setViewing(null);
                  }}
                >
                  Edit
                </Button>
              ) : null}
            </>
          }
        >
          <div className="space-y-4">
            <InquiryHeader
              crumb={`${title} · Inquiry`}
              title={viewing.name || viewing.code}
              pills={
                <StatusPill tone={viewing.isActive === false ? "slate" : "navy"}>
                  {viewing.isActive === false ? "Inactive" : "Active"}
                </StatusPill>
              }
            />
            <InquirySection title="General">
              <ErpField label="Code" value={viewing.code} />
              <ErpField label="Name" value={viewing.name} />
            </InquirySection>
            {extra.length > 0 ? (
              <InquirySection title="Engine flags">
                {extra.map((e) => (
                  <ErpField key={e.key} label={e.label} value={flagOn(viewing, e.key) ? "Yes" : "No"} />
                ))}
              </InquirySection>
            ) : null}
          </div>
        </Modal>
      ) : null}

      {creating || editing ? (
        <NamedForm
          path={path}
          extra={extra}
          singular={singular}
          row={editing ?? undefined}
          onClose={() => {
            setCreating(false);
            setEditing(null);
          }}
          onSaved={() => {
            setCreating(false);
            setEditing(null);
            reload();
          }}
        />
      ) : null}
    </div>
  );
}

function NamedForm({
  path,
  extra,
  singular,
  row,
  onClose,
  onSaved,
}: {
  path: string;
  extra: FlagCol[];
  singular: string;
  row?: CatalogRow & Record<string, unknown>;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [code, setCode] = useState(row?.code || "");
  const [name, setName] = useState(row?.name || "");
  const [flags, setFlags] = useState<Record<string, boolean>>(
    Object.fromEntries(extra.map((e) => [e.key, Boolean(row?.[e.key])])),
  );
  const [active, setActive] = useState(row?.isActive !== false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    if (!code.trim() || !name.trim()) {
      setError("Code and name are required.");
      return;
    }
    setSaving(true);
    try {
      const body = {
        code: code.trim().toUpperCase(),
        name: name.trim(),
        isActive: active,
        ...flags,
      };
      if (row) await api.put(`${path}/${encodeURIComponent(row.code)}`, body);
      else await api.post(path, body);
      onSaved();
    } catch (e) {
      setError(formatApiError(e, "Could not save"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? `Edit ${row.code}` : `Add ${singular}`}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
    >
      <Field label="Code">
        <Input
          value={code}
          disabled={Boolean(row)}
          onChange={(e) => setCode(e.target.value)}
          autoFocus={!row}
        />
      </Field>
      <Field label="Name">
        <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus={Boolean(row)} />
      </Field>
      {extra.map((e) => (
        <ToggleField
          key={e.key}
          label={e.label}
          checked={Boolean(flags[e.key])}
          onChange={(on) => setFlags((f) => ({ ...f, [e.key]: on }))}
        />
      ))}
      <ToggleField label="Active" checked={active} onChange={setActive} />
    </CatalogueForm>
  );
}

export function CycleSetup() {
  const list = useListQuery({ orderBy: "days", sortDirection: "ASC" });
  const [isActive, setIsActive] = useState("");
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<CycleRow | null>(null);
  const [viewing, setViewing] = useState<CycleRow | null>(null);
  const { hasPermission } = useAuth();
  const canWrite = hasPermission(...writePerms);
  const path = "/master/billing-cycles";
  const query = { ...list.query, isActive: isActive || undefined };
  const { data, loading, error, reload } = useAsync(
    () => api.get<CycleRow[]>(path, query),
    [list.search, list.page, list.pageSize, list.orderBy, list.sortDirection, isActive],
  );

  const catalog = useMemo(
    () =>
      [
        {
          id: "days",
          header: "Days",
          sortKey: "days",
          defaultVisible: true,
          cell: (r: CycleRow) => (
            <span className="tabular-nums font-medium text-slate-800">{r.days}</span>
          ),
        },
        {
          id: "description",
          header: "Description",
          defaultVisible: true,
          cell: (r: CycleRow) => r.description,
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r: CycleRow) => <ActiveCell active={r.isActive} />,
        },
        {
          id: "_actions",
          fixed: true,
          header: "",
          className: "text-right",
          cell: (r: CycleRow) =>
            canWrite ? (
              <Button
                variant="secondary"
                className="!px-3 !py-1"
                onClick={(e) => {
                  e.stopPropagation();
                  setEditing(r);
                }}
              >
                Edit
              </Button>
            ) : null,
        },
      ] satisfies ColumnDef<CycleRow>[],
    [canWrite],
  );

  const { columns, pref, save, restoreDefaults } = useColumnPrefs("finance-cycles", catalog);
  const page = asPagination<CycleRow>(data);

  return (
    <div>
      <p className="mb-1 text-xs font-medium text-slate-500">Finance › Setups</p>
      <ListShell
        title="Billing cycles"
        subtitle="First-cycle length in days — 15, 30, or 45. Collection method is separate."
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <EditColumnsButton
              catalog={catalog}
              pref={pref}
              onApply={save}
              onRestore={restoreDefaults}
            />
            <ExportButton path={path} query={query} />
            {canWrite ? <Button onClick={() => setCreating(true)}>Add cycle</Button> : null}
          </div>
        }
        pills={{
          options: ACTIVE_STATUS_PILLS,
          value: isActive,
          allTitle: "All records",
          guideTitle: "Record status",
          onChange: (id) => {
            setIsActive(id);
            list.setPage(1);
          },
        }}
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Search description"
          />
        }
        footer={
          !loading && !error ? (
            <Paginator
              embedded
              data={page}
              page={list.page}
              onPage={list.setPage}
              pageSize={list.pageSize}
              onPageSize={list.setPageSize}
            />
          ) : null
        }
      >
        <DataTable
          plain
          columns={columns}
          rows={page?.items}
          loading={loading}
          error={error}
          emptyMessage="No billing cycles found"
          sort={list.sort}
          onSort={list.onSort}
          onRowClick={(r) => setViewing(r)}
        />
      </ListShell>

      {viewing ? (
        <Modal
          open
          size="md"
          title={`${viewing.days}-day cycle`}
          onClose={() => setViewing(null)}
          footer={
            <>
              <Button variant="secondary" onClick={() => setViewing(null)}>
                Close
              </Button>
              {canWrite ? (
                <Button
                  onClick={() => {
                    setEditing(viewing);
                    setViewing(null);
                  }}
                >
                  Edit
                </Button>
              ) : null}
            </>
          }
        >
          <div className="space-y-4">
            <InquiryHeader
              crumb="Billing cycles · Inquiry"
              title={`${viewing.days}-day cycle`}
              pills={
                <StatusPill tone={viewing.isActive === false ? "slate" : "navy"}>
                  {viewing.isActive === false ? "Inactive" : "Active"}
                </StatusPill>
              }
            />
            <InquirySection title="General">
              <ErpField label="Days" value={String(viewing.days)} />
              <ErpField label="Description" value={viewing.description} />
            </InquirySection>
          </div>
        </Modal>
      ) : null}

      {creating || editing ? (
        <CycleForm
          row={editing ?? undefined}
          onClose={() => {
            setCreating(false);
            setEditing(null);
          }}
          onSaved={() => {
            setCreating(false);
            setEditing(null);
            reload();
          }}
        />
      ) : null}
    </div>
  );
}

function CycleForm({
  row,
  onClose,
  onSaved,
}: {
  row?: CycleRow;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [days, setDays] = useState(row ? String(row.days) : "15");
  const [description, setDescription] = useState(row?.description || "");
  const [active, setActive] = useState(row?.isActive !== false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    if (!description.trim() || !Number(days)) {
      setError("Days and description are required.");
      return;
    }
    setSaving(true);
    try {
      const body = {
        days: Number(days),
        description: description.trim(),
        isActive: active,
      };
      if (row) await api.put(`/master/billing-cycles/${row.days}`, body);
      else await api.post("/master/billing-cycles", body);
      onSaved();
    } catch (e) {
      setError(formatApiError(e, "Could not save"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title={row ? `Edit ${row.days}-day cycle` : "Add billing cycle"}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel={row ? "Save" : "Create"}
    >
      <Field label="Days">
        <Input
          value={days}
          disabled={Boolean(row)}
          inputMode="numeric"
          onChange={(e) => setDays(e.target.value)}
        />
      </Field>
      <Field label="Description">
        <Input value={description} onChange={(e) => setDescription(e.target.value)} autoFocus />
      </Field>
      <ToggleField label="Active" checked={active} onChange={setActive} />
    </CatalogueForm>
  );
}
