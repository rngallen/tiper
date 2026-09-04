"use client";

import { useMemo, useState, type ReactNode } from "react";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ExportButton, ListSearch, useListQuery } from "@/lib/list";
import { DataTable } from "@/components/DataTable";
import { EditColumnsButton } from "@/components/EditColumns";
import { useColumnPrefs, type ColumnDef } from "@/lib/columnPrefs";
import { Modal } from "@/components/Modal";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { Paginator } from "@/components/Paginator";
import { ActiveIcon, Button, ErrorBanner } from "@/components/ui";
import { ACTIVE_STATUS_PILLS } from "@/components/StatusPills";
import { ListShell } from "@/components/ListShell";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import type { Pagination } from "@/lib/types";
import { asPagination } from "@/lib/pagination";

export function ActiveCell({ active }: { active?: boolean }) {
  return <ActiveIcon active={Boolean(active)} />;
}

type CatalogueRow = { id: string; hasData?: boolean; name?: string; code?: string };

export function MasterCatalogue<T extends CatalogueRow>({
  tableId,
  title,
  subtitle,
  crumb = "Stock › Setups",
  path,
  orderBy,
  searchPlaceholder,
  createLabel,
  emptyMessage,
  columns: columnFactory,
  inquiryTitle,
  inquiry,
  Form,
  createPerms,
  updatePerms,
  deletePerms,
  noun,
  configureHref,
  configureLabel = "Configure",
  inquirySize = "md",
}: {
  tableId: string;
  title: string;
  subtitle: string;
  crumb?: string;
  path: string;
  orderBy: string;
  searchPlaceholder: string;
  createLabel: string;
  emptyMessage: string;
  columns: (ctx: { onEdit: (row: T) => void; canUpdate: boolean }) => ColumnDef<T>[];
  inquiryTitle: (row: T) => string;
  inquiry: (row: T) => ReactNode;
  Form: (props: {
    row?: T;
    onClose: () => void;
    onSaved: () => void;
  }) => ReactNode;
  createPerms?: string[];
  updatePerms?: string[];
  deletePerms?: string[];
  noun?: string;
  configureHref?: (row: T) => string;
  configureLabel?: string;
  inquirySize?: "md" | "lg" | "xl" | "2xl";
}) {
  const list = useListQuery({ orderBy, sortDirection: "ASC" });
  const [isActive, setIsActive] = useState("");
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<T | null>(null);
  const [viewing, setViewing] = useState<T | null>(null);
  const [pending, setPending] = useState<T | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(
    ...(createPerms ?? [PERMISSIONS.masterdataCreate, PERMISSIONS.masterdataManage]),
  );
  const canUpdate = hasPermission(
    ...(updatePerms ?? [PERMISSIONS.masterdataUpdate, PERMISSIONS.masterdataManage]),
  );
  const canDelete = hasPermission(
    ...(deletePerms ?? [PERMISSIONS.masterdataDelete, PERMISSIONS.masterdataManage]),
  );
  const label = noun || title.replace(/s$/, "").toLowerCase();

  const query = {
    ...list.query,
    isActive: isActive || undefined,
  };

  const { data, loading, error, reload } = useAsync(
    () => api.get<Pagination<T>>(path, query),
    [
      list.search,
      list.page,
      list.pageSize,
      list.orderBy,
      list.sortDirection,
      isActive,
    ],
  );

  function canRemove(r: T) {
    return canDelete && !r.hasData;
  }

  async function confirmDelete() {
    if (!pending) return;
    setDeleting(true);
    setDeleteError("");
    try {
      await api.del(`${path}/${pending.id}`);
      setPending(null);
      setViewing(null);
      reload();
    } catch (err) {
      setDeleteError(formatApiError(err, `Could not delete this ${label}`));
    } finally {
      setDeleting(false);
    }
  }

  const catalog = useMemo(
    () => [
      ...columnFactory({ onEdit: setEditing, canUpdate }),
      {
        id: "_actions",
        fixed: true,
        header: "",
        className: "text-right",
        cell: (r: T) =>
          canUpdate || canRemove(r) ? (
            <div className="flex justify-end gap-1">
              {canUpdate ? (
                configureHref ? (
                  <Link
                    href={configureHref(r)}
                    className="inline-flex items-center rounded-xl border border-slate-200 px-3 py-1 text-sm font-semibold text-brand-800"
                    onClick={(e) => e.stopPropagation()}
                  >
                    {configureLabel}
                  </Link>
                ) : (
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
                )
              ) : null}
              {canRemove(r) ? (
                <Button
                  variant="ghost"
                  className="!px-3 !py-1 text-red-700 hover:bg-red-50"
                  onClick={(e) => {
                    e.stopPropagation();
                    setDeleteError("");
                    setPending(r);
                  }}
                >
                  Delete
                </Button>
              ) : null}
            </div>
          ) : null,
      } satisfies ColumnDef<T>,
    ],
    [canUpdate, canDelete, columnFactory, configureHref, configureLabel],
  );

  const { columns, pref, save, restoreDefaults } = useColumnPrefs(tableId, catalog);
  const page = asPagination<T>(data);
  const pendingName = pending?.name || pending?.code || "";

  return (
    <div>
      {crumb ? (
        <p className="mb-1 text-xs font-medium text-slate-500">{crumb}</p>
      ) : null}
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
            {canCreate ? (
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
            placeholder={searchPlaceholder}
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
          emptyMessage={emptyMessage}
          sort={list.sort}
          onSort={list.onSort}
          onRowClick={(r) => setViewing(r)}
        />
      </ListShell>

      {viewing && (
        <Modal
          open
          size={inquirySize}
          title={inquiryTitle(viewing)}
          onClose={() => setViewing(null)}
          footer={
            <>
              {canRemove(viewing) ? (
                <Button
                  variant="danger"
                  onClick={() => {
                    setDeleteError("");
                    setPending(viewing);
                  }}
                >
                  Delete
                </Button>
              ) : null}
              <div className="flex-1" />
              <Button variant="secondary" onClick={() => setViewing(null)}>
                Close
              </Button>
              {canUpdate && configureHref ? (
                <Link
                  href={configureHref(viewing)}
                  className="inline-flex items-center rounded-xl bg-brand-800 px-4 py-2 text-sm font-semibold text-white"
                >
                  {configureLabel}
                </Link>
              ) : canUpdate ? (
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
          {inquiry(viewing)}
        </Modal>
      )}

      {creating && (
        <Form
          onClose={() => setCreating(false)}
          onSaved={() => {
            setCreating(false);
            reload();
          }}
        />
      )}
      {editing && (
        <Form
          row={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            reload();
          }}
        />
      )}

      <ConfirmDialog
        open={Boolean(pending)}
        title={`Delete ${label}?`}
        message={
          pendingName
            ? `Remove ${pendingName} from master data? This cannot be undone.`
            : `Remove this ${label}? This cannot be undone.`
        }
        detail="Only records with no documents or stock history can be removed. After first use, deactivate the record so history stays intact."
        loading={deleting}
        onConfirm={() => void confirmDelete()}
        onCancel={() => {
          if (!deleting) setPending(null);
        }}
      />
      {deleteError ? (
        <div className="fixed bottom-4 right-4 z-[90] max-w-md">
          <ErrorBanner message={deleteError} />
        </div>
      ) : null}
    </div>
  );
}
