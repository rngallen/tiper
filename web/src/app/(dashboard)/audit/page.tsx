"use client";

import { useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ExportButton, ListSearch, useListQuery } from "@/lib/list";
import { DataTable } from "@/components/DataTable";
import { EditColumnsButton } from "@/components/EditColumns";
import { useColumnPrefs, type ColumnDef } from "@/lib/columnPrefs";
import { Paginator } from "@/components/Paginator";
import { Select, Button } from "@/components/ui";
import { DateRangeFields, FilterField, ListShell } from "@/components/ListShell";
import { Modal } from "@/components/Modal";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  ReviewTabBar,
  StatusPill,
} from "@/components/ApprovalReview";
import { FieldChanges } from "@/components/FieldChanges";
import {
  formatDateTime,
  formatFieldChange,
  formatFieldLabel,
  localISODate,
  titleCase,
  toApiDate,
} from "@/lib/format";
import type { AuditEntry, Pagination } from "@/lib/types";

/** Closed catalogues — keep values aligned with pkg/types/audit.go. */
const AUDIT_MODULES = [
  "Auth",
  "User",
  "Profile",
  "Role",
  "Title",
  "Customer",
  "Inventory",
  "Billing",
  "Ewura",
  "Orders",
  "Reports",
  "Workflow",
  "Settings",
  "System",
] as const;

const AUDIT_ACTIONS = [
  "login",
  "refresh_token",
  "logout",
  "create",
  "update",
  "delete",
  "approve",
  "reject",
  "initiate",
  "dispatch",
  "sync",
] as const;

// Audit defaults to 30 days (not lib/format defaultFromISO's 730) for a tighter trail view.
function defaultFromISO() {
  const d = new Date();
  d.setDate(d.getDate() - 30);
  return localISODate(d);
}

export default function AuditPage() {
  const list = useListQuery({
    orderBy: "createdAt",
    sortDirection: "DESC",
  });
  const [module, setModule] = useState("");
  const [action, setAction] = useState("");
  const [fromDate, setFromDate] = useState(defaultFromISO);
  const [toDate, setToDate] = useState(() => localISODate(new Date()));
  const [preview, setPreview] = useState<AuditEntry | null>(null);
  const [previewTab, setPreviewTab] = useState<"entry" | "changes">("entry");

  const filters = useMemo(
    () => ({
      ...list.query,
      module: module || undefined,
      action: action || undefined,
      fromDate: toApiDate(fromDate),
      toDate: toApiDate(toDate),
    }),
    [list.query, module, action, fromDate, toDate],
  );

  const { data, loading, error } = useAsync(
    () => api.get<Pagination<AuditEntry>>("/audit", filters),
    [
      list.search,
      list.page,
      list.pageSize,
      list.orderBy,
      list.sortDirection,
      module,
      action,
      fromDate,
      toDate,
    ],
  );

  const catalog: ColumnDef<AuditEntry>[] = useMemo(
    () => [
      {
        id: "createdAt",
        header: "When",
        sortKey: "createdAt",
        defaultVisible: true,
        cell: (r) => formatDateTime(r.createdAt),
      },
      {
        id: "userName",
        header: "Actor",
        sortKey: "userName",
        defaultVisible: true,
        cell: (r) => (
          <div className="text-slate-800">{r.userName || "system"}</div>
        ),
      },
      {
        id: "module",
        header: "Module",
        sortKey: "module",
        defaultVisible: true,
        cell: (r) => titleCase(r.module),
      },
      {
        id: "action",
        header: "Action",
        sortKey: "action",
        defaultVisible: true,
        cell: (r) => titleCase(r.action),
      },
      {
        id: "description",
        header: "Description",
        sortKey: "description",
        defaultVisible: true,
        cell: (r) => (
          <div className="max-w-md whitespace-normal">
            <div className="text-slate-800">{r.description || "—"}</div>
            {r.changes && Object.keys(r.changes).length > 0 && (
              <ul className="mt-1.5 space-y-0.5 border-l-2 border-brand-100 pl-2 text-xs text-slate-500">
                {Object.entries(r.changes).map(([field, change]) => (
                  <li key={field}>
                    <span className="font-medium text-slate-600">
                      {formatFieldLabel(field)}:
                    </span>{" "}
                    {formatFieldChange(change)}
                  </li>
                ))}
              </ul>
            )}
          </div>
        ),
      },
      {
        id: "ipAddress",
        header: "IP address",
        defaultVisible: false,
        cell: (r) => r.ipAddress || "—",
      },
      {
        id: "id",
        header: "ID",
        defaultVisible: false,
        cell: (r) => r.id,
      },
    ],
    [],
  );

  const { columns, pref, save, restoreDefaults } = useColumnPrefs(
    "audit",
    catalog,
  );

  return (
    <div>
      <ListShell
        title="Audit trail"
        subtitle="Append-only record of system actions"
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <EditColumnsButton
              catalog={catalog}
              pref={pref}
              onApply={save}
              onRestore={restoreDefaults}
            />
            <ExportButton path="/audit" query={filters} />
          </div>
        }
        filters={
          <>
            <DateRangeFields
              from={fromDate}
              to={toDate}
              onFrom={(iso) => {
                setFromDate(iso);
                list.setPage(1);
              }}
              onTo={(iso) => {
                setToDate(iso);
                list.setPage(1);
              }}
              onClear={() => {
                setFromDate("");
                setToDate("");
                list.setPage(1);
              }}
            />
            <FilterField label="Module">
              <Select
                value={module}
                onChange={(e) => {
                  setModule(e.target.value);
                  list.setPage(1);
                }}
                aria-label="Filter by module"
              >
                <option value="">All</option>
                {AUDIT_MODULES.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </Select>
            </FilterField>
            <FilterField label="Action">
              <Select
                value={action}
                onChange={(e) => {
                  setAction(e.target.value);
                  list.setPage(1);
                }}
                aria-label="Filter by action"
              >
                <option value="">All</option>
                {AUDIT_ACTIONS.map((a) => (
                  <option key={a} value={a}>
                    {titleCase(a)}
                  </option>
                ))}
              </Select>
            </FilterField>
          </>
        }
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Actor / description"
          />
        }
        footer={
          !loading && !error ? (
            <Paginator
              embedded
              data={data}
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
          rows={data?.items}
          loading={loading}
          error={error}
          emptyMessage="No audit entries in this date range"
          sort={list.sort}
          onSort={list.onSort}
          onRowClick={(r) => {
            setPreviewTab("entry");
            setPreview(r);
          }}
        />
      </ListShell>

      {preview && (
        <Modal
          open
          size="xl"
          title="Audit entry"
          onClose={() => setPreview(null)}
          footer={
            <Button variant="secondary" onClick={() => setPreview(null)}>
              Close
            </Button>
          }
        >
          <div className="space-y-4">
            <InquiryHeader
              crumb="Audit · Inquiry"
              title={preview.description || titleCase(preview.action)}
              pills={
                <>
                  <StatusPill tone="navy">{titleCase(preview.module)}</StatusPill>
                  <StatusPill>{titleCase(preview.action)}</StatusPill>
                </>
              }
            />
            <ReviewTabBar
              tab={previewTab}
              onChange={setPreviewTab}
              ids={[
                { id: "entry", label: "Entry" },
                {
                  id: "changes",
                  label: "Field changes",
                  count: preview.changes
                    ? Object.keys(preview.changes).length
                    : 0,
                },
              ]}
            />
            {previewTab === "entry" ? (
              <InquirySection title="Entry">
                <ErpField
                  label="When"
                  value={formatDateTime(preview.createdAt)}
                />
                <ErpField label="Actor" value={preview.userName || "system"} />
                <ErpField label="Module" value={titleCase(preview.module)} />
                <ErpField label="Action" value={titleCase(preview.action)} />
                <ErpField label="IP address" value={preview.ipAddress || "—"} />
                <ErpField
                  label="Description"
                  value={preview.description || "—"}
                />
              </InquirySection>
            ) : (
              <section className="rounded-lg border border-slate-200 bg-white p-4">
                <h4 className="mb-3 text-xs font-bold uppercase tracking-wide text-brand-800">
                  Field changes
                </h4>
                <FieldChanges
                  changes={preview.changes}
                  emptyMessage="No field-level changes were recorded for this entry."
                />
              </section>
            )}
          </div>
        </Modal>
      )}
    </div>
  );
}
