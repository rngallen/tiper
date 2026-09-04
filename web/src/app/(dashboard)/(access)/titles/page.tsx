"use client";

import { useMemo, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ExportButton, ListSearch, useListQuery } from "@/lib/list";
import { DataTable } from "@/components/DataTable";
import { Modal } from "@/components/Modal";
import { Paginator } from "@/components/Paginator";
import {
  Button,
  ErrorBanner,
  Field,
  Input,
  StatusBadge,
} from "@/components/ui";
import { ListShell } from "@/components/ListShell";
import type { Pagination, Title } from "@/lib/types";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { EditColumnsButton } from "@/components/EditColumns";
import { useColumnPrefs, type ColumnDef } from "@/lib/columnPrefs";
import { InquiryHeader, StatusPill } from "@/components/ApprovalReview";

export default function TitlesPage() {
  const list = useListQuery({ orderBy: "name", sortDirection: "ASC" });
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<Title | null>(null);
  const [viewing, setViewing] = useState<Title | null>(null);
  const [pending, setPending] = useState<Title | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(
    PERMISSIONS.titlesCreate,
    PERMISSIONS.titlesManage,
  );
  const canUpdate = hasPermission(
    PERMISSIONS.titlesUpdate,
    PERMISSIONS.titlesManage,
  );
  const canDelete = hasPermission(
    PERMISSIONS.titlesDelete,
    PERMISSIONS.titlesManage,
  );

  const { data, loading, error, reload } = useAsync(
    () => api.get<Pagination<Title>>("/auth/titles", list.query),
    [list.search, list.page, list.pageSize, list.orderBy, list.sortDirection],
  );

  async function confirmRemove() {
    if (!pending) return;
    setDeleting(true);
    setDeleteError("");
    try {
      await api.del(`/auth/titles/${pending.id}`);
      setPending(null);
      reload();
    } catch (err) {
      setDeleteError(formatApiError(err, "Failed to delete title"));
    } finally {
      setDeleting(false);
    }
  }

  const catalog: ColumnDef<Title>[] = useMemo(
    () => [
      {
        id: "name",
        header: "Title",
        sortKey: "name",
        defaultVisible: true,
        cell: (r) => (
          <div className="font-medium text-slate-800">{r.name}</div>
        ),
      },
      {
        id: "hasData",
        header: "Status",
        defaultVisible: true,
        cell: (r) => (
          <StatusBadge
            status={r.hasData ? "approved" : "pending"}
            label={r.hasData ? "In use" : "Unused"}
          />
        ),
      },
      {
        id: "id",
        header: "ID",
        defaultVisible: false,
        cell: (r) => (
          <span className="font-mono text-xs text-slate-600">{r.id}</span>
        ),
      },
      {
        id: "_actions",
        fixed: true,
        header: "",
        className: "text-right",
        cell: (r) => (
          <div className="flex justify-end gap-2">
            {canUpdate ? (
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
            ) : null}
            {canDelete && !r.hasData ? (
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
        ),
      },
    ],
    [canUpdate, canDelete],
  );

  const { columns, pref, save, restoreDefaults } = useColumnPrefs(
    "titles",
    catalog,
  );

  return (
    <div>
      <ListShell
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <EditColumnsButton
              catalog={catalog}
              pref={pref}
              onApply={save}
              onRestore={restoreDefaults}
            />
            <ExportButton path="/auth/titles" query={list.query} />
            {canCreate ? (
              <Button onClick={() => setCreating(true)}>Add title</Button>
            ) : null}
          </div>
        }
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Search titles"
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
          emptyMessage="No titles yet — add one so user profiles can pick a job title"
          sort={list.sort}
          onSort={list.onSort}
          onRowClick={(r) => setViewing(r)}
        />
      </ListShell>

      {viewing && (
        <Modal
          open
          title="Title"
          onClose={() => setViewing(null)}
          footer={
            <>
              <Button variant="secondary" onClick={() => setViewing(null)}>
                Close
              </Button>
              {canUpdate ? (
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
          <InquiryHeader
            crumb="Titles · Inquiry"
            title={viewing.name}
            subtitle={
              viewing.hasData
                ? "Assigned to at least one user"
                : "Not assigned to any user"
            }
            pills={
              <StatusPill tone={viewing.hasData ? "emerald" : "amber"}>
                {viewing.hasData ? "In use" : "Unused"}
              </StatusPill>
            }
          />
        </Modal>
      )}

      {creating && (
        <TitleModal
          onClose={() => setCreating(false)}
          onSaved={() => {
            setCreating(false);
            reload();
          }}
        />
      )}

      {editing && (
        <TitleModal
          title={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            reload();
          }}
        />
      )}
      <ConfirmDialog
        open={Boolean(pending)}
        title="Delete title?"
        message={
          pending
            ? `Remove “${pending.name}” from the catalogue? This cannot be undone.`
            : "Remove this title?"
        }
        detail="Only unused titles can be removed. After a title is assigned to a user, keep it and stop using it on new profiles."
        loading={deleting}
        onConfirm={() => void confirmRemove()}
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

function TitleModal({
  title,
  onClose,
  onSaved,
}: {
  title?: Title;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(title?.name ?? "");
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");

  async function save() {
    setFormError("");
    const trimmed = name.trim();
    if (!trimmed) {
      setFormError("Name is required.");
      return;
    }
    setSaving(true);
    try {
      if (title) {
        await api.patch(`/auth/titles/${title.id}`, { name: trimmed });
      } else {
        await api.post("/auth/titles", { name: trimmed });
      }
      onSaved();
    } catch (err) {
      setFormError(formatApiError(err, "Failed to save title"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      title={title ? "Edit title" : "Add title"}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={save} loading={saving}>
            {title ? "Save" : "Create"}
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {formError && <ErrorBanner message={formError} />}
        <Field label="Name">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Head of Finance"
            maxLength={60}
            autoFocus
          />
        </Field>
        <p className="text-xs text-slate-500">
          Shown in profile and user forms. Max 60 characters.
        </p>
      </div>
    </Modal>
  );
}
