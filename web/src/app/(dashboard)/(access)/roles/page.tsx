"use client";

import { useEffect, useMemo, useState } from "react";
import { api, ApiError, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ExportButton, ListSearch, useListQuery } from "@/lib/list";
import { DataTable } from "@/components/DataTable";
import { EditColumnsButton } from "@/components/EditColumns";
import { useColumnPrefs, type ColumnDef } from "@/lib/columnPrefs";
import { ListShell } from "@/components/ListShell";
import { Modal } from "@/components/Modal";
import { Paginator } from "@/components/Paginator";
import {
  Button,
  ErrorBanner,
  Field,
  Input,
  Spinner,
  Select,
} from "@/components/ui";
import { groupPermissions } from "@/lib/permissionGroups";
import { ROLE_FAMILIES, roleFamilyLabel } from "@/lib/roleFamilies";
import { InquiryHeader, StatusPill } from "@/components/ApprovalReview";
import type { Pagination, Permission, Role } from "@/lib/types";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import {
  PermissionEditBody,
  PermissionInquiryBody,
  PermissionTabs,
  usePermissionTab,
} from "../_components/RolePermissionMatrix";

export default function RolesPage() {
  const list = useListQuery({ orderBy: "name", sortDirection: "ASC"});
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<Role | null>(null);
  const [viewing, setViewing] = useState<Role | null>(null);
  const { hasPermission } = useAuth();
  const canManage = hasPermission(PERMISSIONS.rolesManage);

  const { data, loading, error, reload } = useAsync(
    () => api.get<Pagination<Role>>("/auth/roles", list.query),
    [list.search, list.page, list.pageSize, list.orderBy, list.sortDirection],
  );

  const catalog: ColumnDef<Role>[] = useMemo(
    () => [
      {
        id: "name",
        header: "Role",
        sortKey: "name",
        defaultVisible: true,
        cell: (r) => (
          <div className="font-medium text-slate-800">{r.name}</div>
        ),
      },
      {
        id: "category",
        header: "Category",
        sortKey: "category",
        defaultVisible: true,
        cell: (r) => (
          <span className="text-slate-700">{roleFamilyLabel(r.category)}</span>
        ),
      },
      {
        id: "description",
        header: "Description",
        sortKey: "description",
        defaultVisible: true,
        cell: (r) => (
          <span className="text-slate-600">{r.description || "—"}</span>
        ),
      },
      {
        id: "permissions",
        header: "Permissions",
        defaultVisible: true,
        className: "text-right",
        cell: (r) => (
          <span className="tabular-nums text-slate-800">
            {r.permissionCount ?? r.permissions?.length ?? 0}
          </span>
        ),
      },
      {
        id: "id",
        header: "ID",
        defaultVisible: false,
        cell: (r) => r.id,
      },
      {
        id: "_actions",
        fixed: true,
        header: "",
        className: "text-right",
        cell: (r) =>
          canManage ? (
            <Button
              variant="secondary"
              className="!px-3 !py-1"
              onClick={(e) => {
                e.stopPropagation();
                setEditing(r);
              }}
            >
              Manage
            </Button>
          ) : null,
      },
    ],
    [canManage],
  );

  const { columns, pref, save, restoreDefaults } = useColumnPrefs(
    "roles",
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
            <ExportButton path="/auth/roles" query={list.query} />
            {canManage ? (
              <Button onClick={() => setCreating(true)}>Add role</Button>
            ) : null}
          </div>
        }
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Search roles"
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
          emptyMessage="No roles found"
          sort={list.sort}
          onSort={list.onSort}
          onRowClick={(r) => setViewing(r)}
        />
      </ListShell>

      {viewing && (
        <ViewRoleModal
          role={viewing}
          canManage={canManage}
          onClose={() => setViewing(null)}
          onEdit={() => {
            setEditing(viewing);
            setViewing(null);
          }}
        />
      )}

      {creating && (
        <CreateRoleModal
          onClose={() => setCreating(false)}
          onSaved={() => {
            setCreating(false);
            reload();
          }}
        />
      )}

      {editing && (
        <EditRoleModal
          role={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            reload();
          }}
        />
      )}
    </div>
  );
}

function CreateRoleModal({
  onClose,
  onSaved,
}: {
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState("system");
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");

  async function save() {
    setFormError("");
    if (!name.trim() || !description.trim()) {
      setFormError("Name and description are required.");
      return;
    }
    setSaving(true);
    try {
      await api.post("/auth/roles", {
        name: name.trim(),
        description: description.trim(),
        category,
      });
      onSaved();
    } catch (err) {
      setFormError(formatApiError(err, "Failed to create role"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      title="Add role"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={save} loading={saving}>
            Create
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {formError && <ErrorBanner message={formError} />}
        <Field label="Name">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value.toUpperCase())}
            placeholder="e.g. APPROVER"
            maxLength={60}
          />
        </Field>
        <p className="text-xs text-slate-500">
          Letters and numbers only. Stored in uppercase.
        </p>
        <Field label="Category">
          <Select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          >
            {ROLE_FAMILIES.map((f) => (
              <option key={f} value={f}>
                {roleFamilyLabel(f)}
              </option>
            ))}
          </Select>
        </Field>
        <p className="text-xs text-slate-500">
          Workplace on user assignment: System, Terminal, Gantry, Finance,
          Billing. Terminal includes reception and ITT; pump-over is its own
          stock area.
        </p>
        <Field label="Description">
          <Input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="What this role can do"
          />
        </Field>
      </div>
    </Modal>
  );
}

function ViewRoleModal({
  role,
  canManage,
  onClose,
  onEdit,
}: {
  role: Role;
  canManage: boolean;
  onClose: () => void;
  onEdit: () => void;
}) {
  const { data: detail, loading, error } = useAsync(
    () => api.get<Role>(`/auth/roles/${role.id}`),
    [role.id],
  );
  const perms = detail?.permissions ?? role.permissions ?? [];
  const byArea = useMemo(() => groupPermissions(perms), [perms]);
  const { tab, setTab } = usePermissionTab(byArea);
  const active = byArea.find((a) => a.area === tab);

  return (
    <Modal
      open
      size="2xl"
      onClose={onClose}
      title="Role"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
          {canManage ? <Button onClick={onEdit}>Edit</Button> : null}
        </>
      }
    >
      {error ? <ErrorBanner message={error} /> : null}
      <div className="space-y-4">
        <InquiryHeader
          crumb="Roles · Inquiry"
          title={role.name}
          subtitle={detail?.description || role.description || undefined}
          pills={
            <>
              <StatusPill>
                {roleFamilyLabel(detail?.category || role.category)}
              </StatusPill>
              <StatusPill tone="navy">{perms.length} permissions</StatusPill>
            </>
          }
        />
        <PermissionTabs tab={tab} onChange={setTab} byArea={byArea} />
        {loading ? (
          <Spinner />
        ) : (
          <PermissionInquiryBody group={active} />
        )}
      </div>
    </Modal>
  );
}

function EditRoleModal({
  role,
  onClose,
  onSaved,
}: {
  role: Role;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [description, setDescription] = useState(role.description ?? "");
  const [category, setCategory] = useState(role.category || "system");
  const [permIds, setPermIds] = useState<number[]>([]);
  const [permSearch, setPermSearch] = useState("");
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");
  const [ready, setReady] = useState(false);

  const [confirmDelete, setConfirmDelete] = useState(false);

  const { data: detail, loading: detailLoading, error: detailError } = useAsync(
    () => api.get<Role>(`/auth/roles/${role.id}`),
    [role.id],
  );

  const {
    data: allPerms,
    loading: permsLoading,
    error: permsError,
  } = useAsync(async () => {
    // Catalogue is small but paginated (default 10) — pull every page so
    // Catalogue is paginated — pull every page so later modules are selectable.
    const collected: Permission[] = [];
    let page = 1;
    let totalPages = 1;
    do {
      const res = await api.get<Pagination<Permission>>("/auth/permissions", {
        page,
        pageSize: 100,
        orderBy: "module",
        sortDirection: "ASC",
      });
      collected.push(...(res?.items ?? []));
      totalPages = res?.totalPages || 1;
      page += 1;
    } while (page <= totalPages);
    return collected;
  }, []);

  // Seed checkboxes once role detail loads.
  useEffect(() => {
    if (detail && !ready) {
      setPermIds((detail.permissions ?? []).map((p) => p.id));
      setDescription(detail.description ?? role.description ?? "");
      setCategory(detail.category || role.category || "system");
      setReady(true);
    }
  }, [detail, ready, role.description]);

  const filtered = useMemo(() => {
    const q = permSearch.trim().toLowerCase();
    const list = allPerms ?? [];
    if (!q) return list;
    return list.filter(
      (p) =>
        p.code.toLowerCase().includes(q) ||
        p.description.toLowerCase().includes(q) ||
        p.module.toLowerCase().includes(q),
    );
  }, [allPerms, permSearch]);

  const byArea = useMemo(() => groupPermissions(filtered), [filtered]);
  const { tab, setTab } = usePermissionTab(byArea, permSearch);
  const active = byArea.find((a) => a.area === tab);

  function toggle(id: number) {
    setPermIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    );
  }

  function toggleModule(modulePerms: Permission[], on: boolean) {
    const ids = modulePerms.map((p) => p.id);
    setPermIds((prev) => {
      if (on) return [...new Set([...prev, ...ids])];
      return prev.filter((id) => !ids.includes(id));
    });
  }

  async function save() {
    setFormError("");
    if (!description.trim()) {
      setFormError("Description is required.");
      return;
    }
    setSaving(true);
    try {
      await api.patch(`/auth/roles/${role.id}`, {
        description: description.trim(),
        category,
      });
      await api.put(`/auth/roles/${role.id}/permissions`, {
        permIds });
      onSaved();
    } catch (err) {
      setFormError(formatApiError(err, "Failed to save role"));
    } finally {
      setSaving(false);
    }
  }

  async function remove() {
    if (!detail?.canDelete) return;
    setFormError("");
    setSaving(true);
    try {
      await api.del(`/auth/roles/${role.id}`);
      onSaved();
    } catch (err) {
      setFormError(
        err instanceof ApiError ? err.message : "Failed to delete role",
      );
      setConfirmDelete(false);
    } finally {
      setSaving(false);
    }
  }

  const loading = detailLoading || permsLoading || !ready;

  return (
    <>
    <Modal
      open
      size="2xl"
      onClose={onClose}
      title={`Role · ${role.name}`}
      footer={
        <>
          <Button
            variant="danger"
            onClick={() => setConfirmDelete(true)}
            disabled={!detail?.canDelete || saving}
            title={
              detail?.canDelete
                ? "Remove this unused role"
                : "Assigned to users or used on a workflow step"
            }
          >
            Delete
          </Button>
          <div className="flex-1" />
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={save} loading={saving} disabled={loading}>
            Save
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {(formError || detailError || permsError) && (
          <ErrorBanner message={formError || detailError || permsError} />
        )}

        <div className="grid gap-4 rounded-lg border border-slate-200 bg-white p-4 sm:grid-cols-2">
          <Field label="Category">
            <Select
              value={category}
              onChange={(e) => setCategory(e.target.value)}
            >
              {ROLE_FAMILIES.map((f) => (
                <option key={f} value={f}>
                  {roleFamilyLabel(f)}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Description">
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </Field>
        </div>

        <div className="flex flex-wrap items-end justify-between gap-2">
          <p className="text-xs text-slate-500">
            {permIds.length} permissions selected · Select all on an area tab or a module
          </p>
          <div className="w-56">
            <Input
              placeholder="Filter permissions"
              value={permSearch}
              onChange={(e) => setPermSearch(e.target.value)}
            />
          </div>
        </div>

        <PermissionTabs
          tab={tab}
          onChange={setTab}
          byArea={byArea}
          permIds={permIds}
        />

        {loading ? (
          <Spinner />
        ) : byArea.length === 0 ? (
          <p className="text-sm text-slate-500">No permissions match.</p>
        ) : (
          <PermissionEditBody
            group={active}
            permIds={permIds}
            onToggle={toggle}
            onToggleMany={toggleModule}
          />
        )}
      </div>
    </Modal>
    <ConfirmDialog
      open={confirmDelete}
      title="Delete role?"
      message={`Remove ${role.name} from Access? This cannot be undone.`}
      detail="Only unused roles can be removed — not assigned to users and not used as a workflow operator. After first assignment, deactivate the role instead."
      loading={saving}
      onConfirm={() => void remove()}
      onCancel={() => {
        if (!saving) setConfirmDelete(false);
      }}
    />
    </>
  );
}
