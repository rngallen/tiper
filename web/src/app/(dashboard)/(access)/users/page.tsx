"use client";

import { useMemo, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ExportButton, ListSearch, useListQuery } from "@/lib/list";
import { DataTable } from "@/components/DataTable";
import { EditColumnsButton } from "@/components/EditColumns";
import { useColumnPrefs, type ColumnDef } from "@/lib/columnPrefs";
import { Modal } from "@/components/Modal";
import { Paginator } from "@/components/Paginator";
import { TitleSelect } from "@/components/TitleSelect";
import {
  ActiveIcon,
  Button,
  ErrorBanner,
  Field,
  Input,
  LockedIcon,
  Spinner,
  ToggleField,
} from "@/components/ui";
import { formatDateTime } from "@/lib/format";
import type { Pagination, Role, User } from "@/lib/types";
import {
  ROLE_FAMILIES,
  roleFamilyLabel,
  roleFamilyOf,
  type RoleFamily,
} from "@/lib/roleFamilies";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { ACTIVE_STATUS_PILLS } from "@/components/StatusPills";
import { ListShell } from "@/components/ListShell";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  ReviewTabBar,
  StatusPill,
} from "@/components/ApprovalReview";

export default function UsersPage() {
  const list = useListQuery({ orderBy: "email", sortDirection: "ASC" });
  const [isActive, setIsActive] = useState(""); // "" | "true" | "false"
  const [editing, setEditing] = useState<User | null>(null);
  const [viewing, setViewing] = useState<User | null>(null);
  const [creating, setCreating] = useState(false);
  const { hasPermission } = useAuth();

  const canManage = hasPermission(PERMISSIONS.usersManage);

  const filters = {
    ...list.query,
    isActive: isActive || undefined,
  };

  const { data, loading, error, reload } = useAsync(
    () => api.get<Pagination<User>>("/auth/users", filters),
    [
      list.search,
      list.page,
      list.pageSize,
      list.orderBy,
      list.sortDirection,
      isActive,
    ],
  );

  const catalog: ColumnDef<User>[] = useMemo(
    () => [
      {
        id: "user",
        header: "User",
        sortKey: "email",
        defaultVisible: true,
        cell: (r) => (
          <div>
            <div className="font-medium text-slate-800">
              {[r.firstName, r.lastName].filter(Boolean).join(" ") || "—"}
            </div>
            <div className="text-xs text-slate-400">{r.email}</div>
          </div>
        ),
      },
      {
        id: "firstName",
        header: "First name",
        sortKey: "firstname",
        defaultVisible: false,
        cell: (r) => r.firstName || "—",
      },
      {
        id: "lastName",
        header: "Last name",
        sortKey: "lastname",
        defaultVisible: false,
        cell: (r) => r.lastName || "—",
      },
      {
        id: "email",
        header: "Email",
        sortKey: "email",
        defaultVisible: false,
        cell: (r) => r.email || "—",
      },
      {
        id: "title",
        header: "Title",
        sortKey: "title",
        defaultVisible: true,
        cell: (r) => r.profile?.title || "—",
      },
      {
        id: "phoneNumber",
        header: "Phone",
        defaultVisible: false,
        cell: (r) => r.profile?.phoneNumber || "—",
      },
      {
        id: "roles",
        header: "Roles",
        defaultVisible: true,
        cell: (r) =>
          r.roles && r.roles.length > 0
            ? r.roles.map((role) => role.name).join(", ")
            : "—",
      },
      {
        id: "isActive",
        header: "Active",
        sortKey: "active",
        defaultVisible: true,
        className: "w-20 text-center",
        cell: (r) => <ActiveIcon active={!!r.isActive} />,
      },
      {
        id: "isLocked",
        header: "Locked",
        defaultVisible: true,
        className: "w-20 text-center",
        cell: (r) => <LockedIcon locked={!!r.isLocked} />,
      },
      {
        id: "lastLogin",
        header: "Last login",
        defaultVisible: false,
        cell: (r) => (r.lastLogin ? formatDateTime(r.lastLogin) : "Never"),
      },
      {
        id: "mustChangePassword",
        header: "Must change password",
        defaultVisible: false,
        cell: (r) => (r.mustChangePassword ? "Yes" : "No"),
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
            onClick={(e) => {
              e.stopPropagation();
              setEditing(r);
            }}
            className="!px-3 !py-1"
          >
            Edit
          </Button>
          ) : null,
      },
    ],
    [canManage],
  );

  const { columns, pref, save, restoreDefaults } = useColumnPrefs(
    "users",
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
            <ExportButton path="/auth/users" query={filters} />
            {canManage ? (
              <Button onClick={() => setCreating(true)}>Add user</Button>
            ) : null}
          </div>
        }
        pills={{
          options: ACTIVE_STATUS_PILLS,
          value: isActive,
          allTitle: "All users",
          guideTitle: "User status",
          onChange: (id) => {
            setIsActive(id);
            list.setPage(1);
          },
        }}
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Name / email"
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
          emptyMessage="No users found"
          sort={list.sort}
          onSort={list.onSort}
          onRowClick={(r) => setViewing(r)}
        />
      </ListShell>

      {viewing && (
        <ViewUserModal
          user={viewing}
          canManage={canManage}
          onClose={() => setViewing(null)}
          onEdit={() => {
            setEditing(viewing);
            setViewing(null);
          }}
        />
      )}

      {creating && (
        <CreateUserModal
          onClose={() => setCreating(false)}
          onSaved={() => {
            setCreating(false);
            reload();
          }}
        />
      )}

      {editing && (
        <EditUserModal
          user={editing}
          canManage={canManage}
          onClose={() => setEditing(null)}
          onRefresh={reload}
          onSaved={() => {
            setEditing(null);
            reload();
          }}
        />
      )}
    </div>
  );
}

function CreateUserModal({
  onClose,
  onSaved,
}: {
  onClose: () => void;
  onSaved: () => void;
}) {
  const [form, setForm] = useState({
    email: "",
    firstName: "",
    lastName: "",
    phoneNumber: "",
    title: "" });
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");
  const [created, setCreated] = useState(false);
  const [notified, setNotified] = useState(false);

  async function save() {
    setFormError("");
    if (!form.email || !form.firstName || !form.lastName) {
      setFormError("Email, first name and last name are required.");
      return;
    }
    setSaving(true);
    try {
      const res = await api.post<{ notified?: boolean }>("/auth/users", {
        email: form.email,
        firstName: form.firstName,
        lastName: form.lastName,
        profile: { phoneNumber: form.phoneNumber, title: form.title },
      });
      setNotified(Boolean(res?.notified));
      setCreated(true);
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Failed to create user");
    } finally {
      setSaving(false);
    }
  }

  if (created) {
    return (
      <Modal
        open
        onClose={onSaved}
        title="User created"
        footer={<Button onClick={onSaved}>Done</Button>}
      >
        <p className="text-sm text-slate-600">
          Account created for <strong>{form.email}</strong>.{" "}
          {notified
            ? "A temporary password was sent to them by email. They must change it on first sign-in."
            : "Mail is not configured, so they did not receive a password. Enable SMTP or SMS, then use Reset password so they get sign-in details."}
        </p>
      </Modal>
    );
  }

  return (
    <Modal
      open
      onClose={onClose}
      title="Add user"
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
        <div className="grid grid-cols-2 gap-3">
          <Field label="First name">
            <Input
              value={form.firstName}
              onChange={(e) => setForm({ ...form, firstName: e.target.value })}
            />
          </Field>
          <Field label="Last name">
            <Input
              value={form.lastName}
              onChange={(e) => setForm({ ...form, lastName: e.target.value })}
            />
          </Field>
          <Field label="Email">
            <Input
              type="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
          </Field>
          <Field label="Job title">
            <TitleSelect
              value={form.title}
              onChange={(title) => setForm({ ...form, title })}
              placeholder="Organisation position…"
            />
          </Field>
          <Field label="Phone">
            <Input
              type="text"
              inputMode="numeric"
              pattern="[1-9][0-9]{6,14}"
              maxLength={15}
              value={form.phoneNumber}
              placeholder="255711223344"
              onChange={(e) =>
                setForm({
                  ...form,
                  phoneNumber: e.target.value.replace(/\D/g, "").slice(0, 15),
                })
              }
            />
          </Field>
        </div>
        <p className="text-xs text-slate-500">
          Phone: digits only, 7–15 characters, no leading zero (e.g. 255711223344).
        </p>
      </div>
    </Modal>
  );
}

type UserEditorTab = "profile" | "roles";

function userFullName(user: User): string {
  return [user.firstName, user.lastName].filter(Boolean).join(" ") || user.email;
}

function UserStatusPills({ user }: { user: User }) {
  return (
    <>
      <StatusPill tone={user.isActive ? "emerald" : "amber"}>
        {user.isActive ? "Active" : "Inactive"}
      </StatusPill>
      {user.isLocked ? <StatusPill tone="amber">Locked</StatusPill> : null}
    </>
  );
}

function EditUserModal({
  user,
  canManage,
  onClose,
  onSaved,
  onRefresh,
}: {
  user: User;
  canManage: boolean;
  onClose: () => void;
  onSaved: () => void;
  onRefresh?: () => void;
}) {
  const [tab, setTab] = useState<UserEditorTab>("profile");
  const [form, setForm] = useState({
    firstName: user.firstName ?? "",
    lastName: user.lastName ?? "",
    isActive: user.isActive ?? true,
    phoneNumber: (user.profile?.phoneNumber ?? "").replace(/\D/g, "").slice(0, 15),
    title: user.profile?.title ?? "" });
  const [roleIds, setRoleIds] = useState<number[]>(
    (user.roles ?? []).map((r) => r.id),
  );
  const [saving, setSaving] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [formError, setFormError] = useState("");
  const [actionMsg, setActionMsg] = useState("");
  const [isLocked, setIsLocked] = useState(user.isLocked ?? false);

  const {
    data: roles,
    loading: rolesLoading,
    error: rolesError,
  } = useAsync(
    () => api.get<Role[]>("/auth/roles/options"),
    [],
  );

  function toggleRole(id: number) {
    setRoleIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    );
  }

  async function unlock() {
    setFormError("");
    setActionMsg("");
    setSaving(true);
    try {
      await api.post(`/auth/users/${user.id}/unlock`);
      setIsLocked(false);
      setActionMsg("Account unlocked. The user can sign in again.");
      onRefresh?.();
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Failed to unlock user");
    } finally {
      setSaving(false);
    }
  }

  async function resetPassword() {
    setFormError("");
    setActionMsg("");
    if (isLocked) {
      setFormError("Unlock the account before resetting the password.");
      return;
    }
    setSaving(true);
    try {
      const res = await api.post<{ notified?: boolean; message?: string }>(
        `/auth/users/${user.id}/reset-password`,
      );
      setActionMsg(
        res?.notified
          ? "Password reset. A temporary password was sent to the user by email."
          : "Password reset, but mail is not configured so they did not receive it. Enable SMTP or SMS and try again.",
      );
      onRefresh?.();
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Failed to reset password");
    } finally {
      setSaving(false);
    }
  }

  async function removeUser() {
    setFormError("");
    setSaving(true);
    try {
      await api.del(`/auth/users/${user.id}`);
      onSaved();
    } catch (err) {
      setFormError(
        err instanceof ApiError ? err.message : "Failed to delete user",
      );
      setConfirmDelete(false);
    } finally {
      setSaving(false);
    }
  }

  async function saveProfile() {
    setFormError("");
    setActionMsg("");
    if (!form.firstName || !form.lastName) {
      setFormError("First and last name are required.");
      return;
    }
    setSaving(true);
    try {
      await api.patch(`/auth/users/${user.id}`, {
        firstName: form.firstName,
        lastName: form.lastName,
        isActive: form.isActive,
        profile: { phoneNumber: form.phoneNumber, title: form.title },
      });
      onSaved();
    } catch (err) {
      setFormError(
        err instanceof ApiError ? err.message : "Failed to update profile",
      );
    } finally {
      setSaving(false);
    }
  }

  async function saveRoles() {
    setFormError("");
    setActionMsg("");
    setSaving(true);
    try {
      await api.put(`/auth/users/${user.id}/roles`, { rolesIds: roleIds });
      onRefresh?.();
    } catch (err) {
      setFormError(
        err instanceof ApiError ? err.message : "Failed to update roles",
      );
    } finally {
      setSaving(false);
    }
  }

  const disabled = !canManage;
  const name = userFullName(user);

  return (
    <>
    <Modal
      open
      size="xl"
      onClose={onClose}
      title="Edit user"
      footer={
        <>
          {canManage && user.canDelete ? (
            <Button variant="danger" onClick={() => setConfirmDelete(true)} disabled={saving}>
              Delete
            </Button>
          ) : null}
          <div className="flex-1" />
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
          {canManage && (
            <Button
              onClick={tab === "profile" ? saveProfile : saveRoles}
              loading={saving}
            >
              {tab === "profile" ? "Save profile" : "Save roles"}
            </Button>
          )}
        </>
      }
    >
      <div className="space-y-4">
        <InquiryHeader
          crumb="Users · Maintain"
          title={name}
          subtitle={user.email}
          pills={<UserStatusPills user={{ ...user, isLocked }} />}
        />
        <ReviewTabBar
          tab={tab}
          ariaLabel="User sections"
          onChange={(id) => {
            setTab(id);
            setFormError("");
          }}
          ids={[
            { id: "profile", label: "Profile" },
            { id: "roles", label: "Roles", count: roleIds.length },
          ]}
        />
        {formError && <ErrorBanner message={formError} />}
        {actionMsg && (
          <p className="rounded-lg border border-brand-100 bg-brand-50 px-3 py-2 text-sm text-brand-900">
            {actionMsg}
          </p>
        )}

        {tab === "profile" ? (
          <div className="space-y-4">
            {isLocked && (
              <div className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
                This account is locked after too many failed sign-in attempts.
                Unlock it before resetting the password or changing active
                status.
              </div>
            )}
            <InquirySection title="Personal">
              <Field label="First name">
                <Input
                  value={form.firstName}
                  disabled={disabled}
                  onChange={(e) =>
                    setForm({ ...form, firstName: e.target.value })
                  }
                />
              </Field>
              <Field label="Last name">
                <Input
                  value={form.lastName}
                  disabled={disabled}
                  onChange={(e) =>
                    setForm({ ...form, lastName: e.target.value })
                  }
                />
              </Field>
              <Field label="Job title">
                <TitleSelect
                  value={form.title}
                  disabled={disabled}
                  onChange={(title) => setForm({ ...form, title })}
                  placeholder="Organisation position…"
                />
              </Field>
              <Field label="Phone">
                <Input
                  type="text"
                  inputMode="numeric"
                  pattern="[1-9][0-9]{6,14}"
                  maxLength={15}
                  value={form.phoneNumber}
                  disabled={disabled}
                  placeholder="255711223344"
                  onChange={(e) =>
                    setForm({
                      ...form,
                      phoneNumber: e.target.value.replace(/\D/g, "").slice(0, 15),
                    })
                  }
                />
              </Field>
              <p className="sm:col-span-2 text-xs text-slate-500">
                Job title is the organisation position, not a workflow role.
                Phone: 7–15 digits, no leading zero.
              </p>
            </InquirySection>

            <div className="grid gap-4 sm:grid-cols-2">
              <InquirySection title="Account" columns={1}>
                <ToggleField
                  label="Account active"
                  description={
                    isLocked
                      ? "Unlock the account before changing active status."
                      : "Inactive users cannot sign in."
                  }
                  checked={form.isActive}
                  disabled={disabled || isLocked}
                  onChange={(isActive) => setForm({ ...form, isActive })}
                />
              </InquirySection>
              {canManage ? (
                <InquirySection title="Security" columns={1}>
                  <p className="text-xs text-slate-500">
                    Unlock a locked account before issuing a password reset.
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <Button
                      variant={isLocked ? "primary" : "secondary"}
                      onClick={unlock}
                      loading={saving}
                      disabled={!isLocked}
                      title={
                        isLocked
                          ? "Clear lockout so the user can sign in"
                          : "Account is not locked"
                      }
                    >
                      Unlock account
                    </Button>
                    <Button
                      variant="secondary"
                      onClick={resetPassword}
                      loading={saving}
                      disabled={isLocked}
                      title={
                        isLocked
                          ? "Unlock the account before resetting the password"
                          : "Issue a temporary password"
                      }
                    >
                      Reset password
                    </Button>
                  </div>
                </InquirySection>
              ) : null}
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            <p className="text-xs text-slate-500">
              Roles control what this person can do. Job title is unchanged when
              you save this tab.
            </p>
            {rolesLoading ? (
              <Spinner />
            ) : rolesError ? (
              <ErrorBanner message={rolesError} />
            ) : roles && roles.length > 0 ? (
              <RoleChecklist
                roles={roles}
                selectedIds={roleIds}
                disabled={disabled}
                onToggle={toggleRole}
              />
            ) : (
              <p className="text-sm text-slate-500">No roles defined yet.</p>
            )}
          </div>
        )}
      </div>
    </Modal>
    <ConfirmDialog
      open={confirmDelete}
      title="Delete user?"
      message={`Remove ${user.email}? This cannot be undone.`}
      detail="Only allowed if this person has never signed in. The email can then be reused."
      loading={saving}
      onConfirm={() => void removeUser()}
      onCancel={() => {
        if (!saving) setConfirmDelete(false);
      }}
    />
    </>
  );
}

function ViewUserModal({
  user,
  onClose,
  onEdit,
  canManage,
}: {
  user: User;
  onClose: () => void;
  onEdit: () => void;
  canManage: boolean;
}) {
  const [tab, setTab] = useState<UserEditorTab>("profile");
  const name = userFullName(user);
  const assigned = user.roles ?? [];
  const grouped = groupRoles(assigned);

  return (
    <Modal
      open
      size="xl"
      onClose={onClose}
      title="User"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
          {canManage ? <Button onClick={onEdit}>Edit</Button> : null}
        </>
      }
    >
      <div className="space-y-4">
        <InquiryHeader
          crumb="Users · Inquiry"
          title={name}
          subtitle={user.email}
          pills={<UserStatusPills user={user} />}
        />
        <ReviewTabBar
          tab={tab}
          onChange={setTab}
          ariaLabel="User sections"
          ids={[
            { id: "profile", label: "Profile" },
            { id: "roles", label: "Roles", count: assigned.length },
          ]}
        />
        {tab === "profile" ? (
          <div className="space-y-4">
            <InquirySection title="Personal">
              <ErpField label="First name" value={user.firstName || "—"} />
              <ErpField label="Last name" value={user.lastName || "—"} />
              <ErpField label="Job title" value={user.profile?.title || "—"} />
              <ErpField label="Phone" value={user.profile?.phoneNumber || "—"} />
            </InquirySection>
            <InquirySection title="Account">
              <ErpField
                label="Status"
                value={user.isActive ? "Active" : "Inactive"}
              />
              <ErpField label="Locked" value={user.isLocked ? "Yes" : "No"} />
              <ErpField
                label="Last login"
                value={
                  user.lastLogin ? formatDateTime(user.lastLogin) : "Never"
                }
              />
            </InquirySection>
          </div>
        ) : assigned.length === 0 ? (
          <div className="rounded-lg border border-dashed border-slate-200 bg-slate-50 px-4 py-8 text-center">
            <p className="text-sm font-medium text-slate-700">No roles assigned</p>
            <p className="mt-1 text-xs text-slate-500">
              {canManage
                ? "Open Edit to assign roles from the catalogue."
                : "This user has not been given any roles."}
            </p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border border-slate-200">
            <table className="pp-ledger min-w-full text-left text-sm">
              <thead className="pp-table-head">
                <tr>
                  <th>Role</th>
                  <th>Description</th>
                </tr>
              </thead>
              <tbody>
                {grouped.flatMap(({ group, roles: list }) => [
                  <tr key={`g-${group}`}>
                    <td
                      colSpan={2}
                      className="bg-slate-50 text-[11px] font-bold uppercase tracking-wide text-brand-800"
                    >
                      {roleFamilyLabel(group)}
                    </td>
                  </tr>,
                  ...list.map((role) => (
                    <tr key={role.id}>
                      <td className="font-medium text-slate-800">{role.name}</td>
                      <td className="text-slate-600">
                        {role.description || "—"}
                      </td>
                    </tr>
                  )),
                ])}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Modal>
  );
}

function groupRoles(roles: Role[]) {
  const buckets = new Map<RoleFamily, Role[]>();
  for (const r of roles) {
    const g = roleFamilyOf(r.category);
    const list = buckets.get(g) || [];
    list.push(r);
    buckets.set(g, list);
  }
  return ROLE_FAMILIES.filter((g) => (buckets.get(g) || []).length > 0).map(
    (g) => ({ group: g, roles: buckets.get(g) || [] }),
  );
}

function RoleChecklist({
  roles,
  selectedIds,
  disabled,
  onToggle,
}: {
  roles: Role[];
  selectedIds: number[];
  disabled?: boolean;
  onToggle: (id: number) => void;
}) {
  const [q, setQ] = useState("");
  const [assignedOnly, setAssignedOnly] = useState(false);
  const [family, setFamily] = useState<"all" | RoleFamily>("all");
  const byId = useMemo(() => new Map(roles.map((r) => [r.id, r])), [roles]);
  const selectedRoles = selectedIds
    .map((id) => byId.get(id))
    .filter((r): r is Role => Boolean(r));

  const familyCounts = useMemo(() => {
    const counts = new Map<RoleFamily, number>();
    for (const r of roles) {
      const g = roleFamilyOf(r.category);
      counts.set(g, (counts.get(g) || 0) + 1);
    }
    return counts;
  }, [roles]);

  const filtered = useMemo(() => {
    const needle = q.trim().toLowerCase();
    return roles.filter((r) => {
      if (assignedOnly && !selectedIds.includes(r.id)) return false;
      if (family !== "all" && roleFamilyOf(r.category) !== family) return false;
      if (!needle) return true;
      return (
        r.name.toLowerCase().includes(needle) ||
        (r.description || "").toLowerCase().includes(needle)
      );
    });
  }, [roles, q, assignedOnly, selectedIds, family]);

  const grouped = groupRoles(filtered);

  return (
    <div className="space-y-3">
      {selectedRoles.length > 0 ? (
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="mr-1 text-[11px] font-semibold uppercase tracking-wide text-slate-500">
            {selectedRoles.length} of {roles.length}
          </span>
          {selectedRoles.map((r) => (
            <span
              key={r.id}
              className="inline-flex items-center gap-1 rounded-md border border-brand-200 bg-brand-50 px-2 py-0.5 text-xs font-medium text-brand-900"
            >
              {r.name}
              {!disabled ? (
                <button
                  type="button"
                  className="text-slate-400 hover:text-slate-700"
                  onClick={() => onToggle(r.id)}
                  aria-label={`Remove ${r.name}`}
                >
                  ×
                </button>
              ) : null}
            </span>
          ))}
        </div>
      ) : (
        <p className="text-xs text-slate-500">
          None assigned. Tick roles in the catalogue.
        </p>
      )}

      <div className="flex flex-wrap items-center gap-2">
        {(
          [
            ["all", "All"] as const,
            ...ROLE_FAMILIES.map((g) => [g, roleFamilyLabel(g)] as const),
          ] as const
        ).map(([id, label]) => {
          const active = family === id;
          const count =
            id === "all"
              ? roles.length
              : familyCounts.get(id) ?? 0;
          if (id !== "all" && count === 0) return null;
          return (
            <button
              key={id}
              type="button"
              onClick={() => setFamily(id)}
              className={`rounded-full px-2.5 py-1 text-xs font-semibold transition ${
                active
                  ? "bg-brand-800 text-white"
                  : "bg-slate-100 text-slate-600 hover:bg-slate-200"
              }`}
            >
              {label}
              <span className="ml-1 tabular-nums opacity-80">{count}</span>
            </button>
          );
        })}
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <Input
          placeholder="Find a role"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          className="min-w-[12rem] flex-1"
        />
        <label className="flex items-center gap-1.5 text-xs font-medium text-slate-600">
          <input
            type="checkbox"
            className="h-3.5 w-3.5 rounded border-slate-300 text-brand-800"
            checked={assignedOnly}
            onChange={(e) => setAssignedOnly(e.target.checked)}
          />
          Assigned only
        </label>
      </div>

      <div className="max-h-[22rem] overflow-auto rounded-lg border border-slate-200">
        {grouped.length === 0 ? (
          <p className="px-4 py-6 text-center text-sm text-slate-500">
            No roles match these filters.
          </p>
        ) : (
          <table className="pp-ledger min-w-full text-left text-sm">
            <thead className="pp-table-head sticky top-0">
              <tr>
                <th className="w-10"></th>
                <th>Role</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              {grouped.flatMap(({ group, roles: list }) => [
                <tr key={`g-${group}`}>
                  <td
                    colSpan={3}
                    className="bg-slate-50 text-[11px] font-bold uppercase tracking-wide text-brand-800"
                  >
                    {roleFamilyLabel(group)}
                  </td>
                </tr>,
                ...list.map((role) => {
                  const on = selectedIds.includes(role.id);
                  return (
                    <tr
                      key={role.id}
                      className={`${disabled ? "" : "cursor-pointer"} ${on ? "bg-brand-50/60" : ""}`}
                      onClick={() => {
                        if (!disabled) onToggle(role.id);
                      }}
                    >
                      <td>
                        <input
                          type="checkbox"
                          className="h-4 w-4 rounded border-slate-300 text-brand-800"
                          checked={on}
                          disabled={disabled}
                          onChange={() => onToggle(role.id)}
                          onClick={(e) => e.stopPropagation()}
                        />
                      </td>
                      <td className="font-medium text-slate-800">{role.name}</td>
                      <td className="text-slate-600">
                        {role.description || "—"}
                      </td>
                    </tr>
                  );
                }),
              ])}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
