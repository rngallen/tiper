"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useAppearance } from "@/lib/appearance";
import { api, formatApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import {
  Button,
  ErrorBanner,
  Field,
  Input,
  PageHeader,
  PasswordInput,
  Select,
  Spinner,
} from "@/components/ui";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { TitleSelect } from "@/components/TitleSelect";
import {
  InquiryHeader,
  InquirySection,
  ReviewTabBar,
  StatusPill,
} from "@/components/ApprovalReview";
import { formatDateTime } from "@/lib/format";
import type {
  AppearanceSettings,
  MeResponse,
  Role,
  User,
} from "@/lib/types";

interface SubstituteAssignment {
  id?: string;
  delegateId?: string;
  delegateEmail?: string;
  delegateName?: string;
  processId?: string;
  processName?: string;
  nodeId?: string;
  nodeName?: string;
  startsAt: string;
  endsAt?: string | null;
  effective: boolean;
}

interface SubstituteView {
  canDelegate: boolean;
  delegatableSteps?: {
    nodeId: string;
    nodeName: string;
    processId: string;
    processName: string;
  }[];
  assignments?: SubstituteAssignment[];
}

type TabId = "profile" | "password" | "appearance" | "roles" | "delegation";

const TABS: { id: TabId; label: string }[] = [
  { id: "profile", label: "Profile" },
  { id: "password", label: "Password" },
  { id: "appearance", label: "Appearance" },
  { id: "delegation", label: "Substitute" },
  { id: "roles", label: "Roles" },
];

function parseTab(raw: string | null): TabId {
  if (
    raw === "password" ||
    raw === "appearance" ||
    raw === "roles" ||
    raw === "delegation"
  ) {
    return raw;
  }
  return "profile";
}

export default function ProfilePage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { mustChangePassword, refreshUser, isSuperUser } = useAuth();

  const tab = parseTab(searchParams.get("tab"));
  const forcePassword = mustChangePassword;

  const [me, setMe] = useState<MeResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");

  async function loadProfile() {
    setLoadError("");
    setLoading(true);
    try {
      const data = await api.get<MeResponse>("/auth/profile");
      setMe(data);
    } catch (err) {
      setLoadError(formatApiError(err, "Failed to load profile"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadProfile();
  }, []);

  useEffect(() => {
    if (forcePassword && tab !== "password") {
      router.replace("/profile?tab=password");
    }
  }, [forcePassword, tab, router]);

  function setTab(next: TabId) {
    if (forcePassword && next !== "password") return;
    const qs = next === "profile" ? "/profile" : `/profile?tab=${next}`;
    router.replace(qs);
  }

  const user = me?.user;

  return (
    <div className="space-y-6">
      <PageHeader
        title="My profile"
        subtitle={
          forcePassword
            ? "Set a new password before continuing"
            : "Account, security, appearance, substitutes, and roles"
        }
      />

      {!forcePassword && (
        <ReviewTabBar tab={tab} onChange={setTab} ids={TABS} ariaLabel="Profile sections" />
      )}

      {loading ? (
        <Spinner />
      ) : loadError ? (
        <ErrorBanner message={loadError} />
      ) : forcePassword || tab === "password" ? (
        <PasswordTab force={forcePassword} />
      ) : tab === "appearance" ? (
        <AppearanceTab
          settings={user?.profile?.appearanceSettings}
          onSaved={async () => {
            await loadProfile();
            await refreshUser();
          }}
        />
      ) : tab === "roles" ? (
        <RolesTab
          roles={user?.roles ?? []}
          isSuperUser={Boolean(me?.isSuperUser || isSuperUser)}
          permissions={me?.permissions ?? []}
        />
      ) : tab === "delegation" ? (
        <DelegationTab />
      ) : (
        <ProfileTab
          user={user}
          onSaved={async () => {
            await loadProfile();
            await refreshUser();
          }}
        />
      )}
    </div>
  );
}

function ProfileTab({
  user,
  onSaved,
}: {
  user?: User;
  onSaved: () => Promise<void>;
}) {
  const [firstName, setFirstName] = useState(user?.firstName ?? "");
  const [lastName, setLastName] = useState(user?.lastName ?? "");
  const [phoneNumber, setPhoneNumber] = useState(
    user?.profile?.phoneNumber ?? "",
  );
  const [title, setTitle] = useState(user?.profile?.title ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    setFirstName(user?.firstName ?? "");
    setLastName(user?.lastName ?? "");
    setPhoneNumber(user?.profile?.phoneNumber ?? "");
    setTitle(user?.profile?.title ?? "");
  }, [user]);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!title.trim()) {
      setError("Select a job title.");
      return;
    }
    setSaving(true);
    try {
      await api.patch("/auth/profile", {
        firstName,
        lastName,
        profile: { phoneNumber, title },
      });
      await onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to update profile"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={save} className="space-y-4">
      <div className="flex justify-end">
        <Button type="submit" loading={saving}>
          Save profile
        </Button>
      </div>
      {error && <ErrorBanner message={error} />}
      <InquiryHeader
        crumb="Profile · Account"
        title={[user?.firstName, user?.lastName].filter(Boolean).join(" ") || "My profile"}
        pills={
          <>
            <StatusPill tone="navy">{user?.email || "—"}</StatusPill>
            {title ? <StatusPill>{title}</StatusPill> : null}
          </>
        }
      />
      <InquirySection title="Identity" columns={3}>
        <Field label="Email">
          <Input value={user?.email ?? ""} disabled />
        </Field>
        <Field label="Job title">
          <TitleSelect
            value={title}
            onChange={setTitle}
            placeholder="Search job title…"
          />
        </Field>
        <Field label="First name">
          <Input
            value={firstName}
            onChange={(e) => setFirstName(e.target.value)}
            required
          />
        </Field>
        <Field label="Last name">
          <Input
            value={lastName}
            onChange={(e) => setLastName(e.target.value)}
            required
          />
        </Field>
        <Field label="Phone">
          <Input
            type="text"
            inputMode="numeric"
            pattern="[1-9][0-9]{6,14}"
            maxLength={15}
            value={phoneNumber}
            onChange={(e) =>
              setPhoneNumber(e.target.value.replace(/\D/g, "").slice(0, 15))
            }
            placeholder="255711223344"
            autoComplete="tel"
          />
        </Field>
      </InquirySection>
      <p className="text-xs text-slate-500">
        Phone: digits only, 7–15 characters, no leading zero (e.g. 255711223344).
      </p>
    </form>
  );
}

function PasswordTab({ force }: { force: boolean }) {
  const router = useRouter();
  const { changePassword } = useAuth();
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const rules = passwordRules(newPassword, oldPassword, confirmPassword);
  const allOk = rules.every((r) => r.ok) && oldPassword.length > 0;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!allOk) {
      setError("Meet every password requirement before updating.");
      return;
    }
    setSaving(true);
    try {
      await changePassword(oldPassword, newPassword, confirmPassword);
      router.replace("/dashboard");
    } catch (err) {
      setError(formatApiError(err, "Failed to change password"));
      setSaving(false);
    }
  }

  return (
    <form onSubmit={submit} className="max-w-lg space-y-4">
      <InquiryHeader
        crumb="Profile · Password"
        title="Change password"
        pills={
          force ? <StatusPill tone="amber">Required</StatusPill> : undefined
        }
      />
      {force ? (
        <div className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-950">
          For security, you must choose a new password before using the application.
        </div>
      ) : (
        <p className="text-sm text-slate-500">
          Choose a password that meets the policy below. You will be signed in
          with the new password after it is saved.
        </p>
      )}
      {error && <ErrorBanner message={error} />}
      <InquirySection title="Credentials" columns={1}>
        <Field label="Current password">
          <PasswordInput
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            required
            autoComplete="current-password"
          />
        </Field>
        <Field label="New password">
          <PasswordInput
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            required
            minLength={8}
            maxLength={150}
            autoComplete="new-password"
          />
        </Field>
        <Field label="Confirm new password">
          <PasswordInput
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            required
            minLength={8}
            maxLength={150}
            autoComplete="new-password"
          />
        </Field>
        <ul className="space-y-1.5 pt-1" aria-live="polite">
          {rules.map((r) => (
            <li
              key={r.label}
              className={`flex items-center gap-2 text-xs ${
                r.ok ? "text-emerald-700" : "text-slate-500"
              }`}
            >
              <span
                className={`inline-block h-1.5 w-1.5 rounded-full ${
                  r.ok ? "bg-emerald-600" : "bg-slate-300"
                }`}
                aria-hidden
              />
              {r.label}
            </li>
          ))}
        </ul>
        <p className="text-xs text-slate-500">
          The new password must also differ from your last five passwords.
        </p>
      </InquirySection>
      <Button type="submit" loading={saving} disabled={!allOk}>
        Update password
      </Button>
    </form>
  );
}

const PASSWORD_SPECIAL = /[!@#$%^&*()_+\-=[\]{};:'",.<>?]/;

function passwordRules(
  password: string,
  oldPassword: string,
  confirm: string,
): { ok: boolean; label: string }[] {
  return [
    { ok: password.length >= 8, label: "At least 8 characters" },
    { ok: /[a-zA-Z]/.test(password), label: "Contains a letter" },
    { ok: /[0-9]/.test(password), label: "Contains a number" },
    {
      ok: PASSWORD_SPECIAL.test(password),
      label: "Contains a special character",
    },
    {
      ok: Boolean(password) && password !== oldPassword,
      label: "Different from current password",
    },
    {
      ok: Boolean(confirm) && password === confirm,
      label: "New password and confirmation match",
    },
  ];
}

function AppearanceTab({
  settings,
  onSaved,
}: {
  settings?: AppearanceSettings;
  onSaved: () => Promise<void>;
}) {
  const { applyLocal, clearLocal } = useAppearance();
  const [theme, setTheme] = useState(settings?.theme || "light");
  const [compactMode, setCompactMode] = useState(
    settings?.compactMode === undefined ? true : Boolean(settings.compactMode),
  );
  const [largeText, setLargeText] = useState(Boolean(settings?.largeText));
  const [sidebarState, setSidebarState] = useState(
    settings?.sidebarState === undefined ? true : Boolean(settings.sidebarState),
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    setTheme(settings?.theme || "light");
    setCompactMode(
      settings?.compactMode === undefined ? true : Boolean(settings.compactMode),
    );
    setLargeText(Boolean(settings?.largeText));
    setSidebarState(
      settings?.sidebarState === undefined
        ? true
        : Boolean(settings.sidebarState),
    );
  }, [settings]);

  // Live preview while editing; revert to saved profile values on leave.
  useEffect(() => {
    applyLocal({ theme, compactMode, largeText, sidebarState });
  }, [theme, compactMode, largeText, sidebarState, applyLocal]);

  useEffect(() => {
    return () => {
      clearLocal();
    };
  }, [clearLocal]);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setSaving(true);
    try {
      // Backend route uses the existing spelling "apperance-settings".
      await api.patch("/auth/profile/apperance-settings", {
        theme,
        compactMode,
        largeText,
        sidebarState,
      });
      applyLocal({ theme, compactMode, largeText, sidebarState });
      await onSaved();
    } catch (err) {
      setError(formatApiError(err, "Failed to save appearance"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={save} className="space-y-4">
      <div className="flex justify-end">
        <Button type="submit" loading={saving}>
          Save appearance
        </Button>
      </div>
      {error && <ErrorBanner message={error} />}
      <InquiryHeader
        crumb="Profile · Appearance"
        title="Display preferences"
        pills={<StatusPill>{theme}</StatusPill>}
      />
      <InquirySection title="Theme">
        <Field label="Theme">
          <select
            value={theme}
            onChange={(e) => setTheme(e.target.value)}
            className="pp-control"
          >
            <option value="system">System</option>
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </Field>
      </InquirySection>
      <section className="rounded-lg border border-slate-200 bg-white p-4">
        <h4 className="mb-3 text-xs font-bold uppercase tracking-wide text-brand-800">
          Layout
        </h4>
        <div className="flex flex-wrap items-center gap-x-8 gap-y-2">
          <label className="inline-flex items-center gap-2 text-sm font-semibold text-slate-900">
            <input
              type="checkbox"
              checked={compactMode}
              onChange={(e) => setCompactMode(e.target.checked)}
            />
            Compact mode
          </label>
          <label className="inline-flex items-center gap-2 text-sm font-semibold text-slate-900">
            <input
              type="checkbox"
              checked={largeText}
              onChange={(e) => setLargeText(e.target.checked)}
            />
            Large text
          </label>
          <label className="inline-flex items-center gap-2 text-sm font-semibold text-slate-900">
            <input
              type="checkbox"
              checked={sidebarState}
              onChange={(e) => setSidebarState(e.target.checked)}
            />
            Expanded sidebar by default
            <span className="font-normal text-slate-500">
              (sidebar arrow toggles anytime)
            </span>
          </label>
        </div>
      </section>
    </form>
  );
}

const ACTION_ORDER = [
  "read",
  "create",
  "update",
  "delete",
  "submit",
  "complete",
  "manage",
  "tasks",
  "run",
  "post",
  "balances",
];

function actionRank(action: string): number {
  const i = ACTION_ORDER.indexOf(action);
  return i >= 0 ? i : ACTION_ORDER.length;
}

function groupPermissions(codes: string[]): { resource: string; actions: { action: string; code: string }[] }[] {
  const map = new Map<string, { action: string; code: string }[]>();
  for (const code of codes) {
    const parts = code.split(".");
    const resource =
      parts.length >= 3 ? `${parts[0]}.${parts[1]}` : parts[0] || code;
    const action = parts.length >= 3 ? parts.slice(2).join(".") : parts[1] || code;
    const list = map.get(resource) || [];
    list.push({ action, code });
    map.set(resource, list);
  }
  return [...map.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([resource, actions]) => ({
      resource,
      actions: actions.sort(
        (a, b) =>
          actionRank(a.action) - actionRank(b.action) ||
          a.action.localeCompare(b.action),
      ),
    }));
}

const RESOURCE_LABELS: Record<string, string> = {
  "orders.ilr": "Internal loading",
  "orders.pumpover": "Pump-over requests",
  "orders.pumpreport": "Pump-over reports",
  "orders.compartmentalization": "Compartmentalization",
  "orders.amendment": "Loading amendments",
  "inventory.receipts": "Vessel receipts",
  "inventory.itt": "In-tank transfers",
  "inventory.zerolization": "Zerolization",
};

function resourceLabel(resource: string): string {
  if (!resource) return resource;
  if (RESOURCE_LABELS[resource]) return RESOURCE_LABELS[resource];
  return resource.charAt(0).toUpperCase() + resource.slice(1);
}

function DelegationTab() {
  const { user: meUser } = useAuth();
  const [view, setView] = useState<SubstituteView | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [processId, setProcessId] = useState("");
  const [delegateId, setDelegateId] = useState("");
  const [delegateLabel, setDelegateLabel] = useState("");
  const [nodeId, setNodeId] = useState("");
  const [endsAt, setEndsAt] = useState("");

  const processes = useMemo(() => {
    const map = new Map<string, string>();
    for (const s of view?.delegatableSteps || []) {
      if (s.processId && !map.has(s.processId)) {
        map.set(s.processId, s.processName);
      }
    }
    return [...map.entries()].map(([id, name]) => ({ id, name }));
  }, [view?.delegatableSteps]);

  const stepsForProcess = useMemo(
    () =>
      (view?.delegatableSteps || []).filter((s) => s.processId === processId),
    [view?.delegatableSteps, processId],
  );

  async function load() {
    setError("");
    setLoading(true);
    try {
      const data = await api.get<SubstituteView>("/workflow/substitute/me");
      setView(data);
    } catch (err) {
      setError(formatApiError(err, "Failed to load delegation settings"));
      setView(null);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  useEffect(() => {
    if (!processId && processes.length === 1) {
      setProcessId(processes[0].id);
    }
  }, [processId, processes]);

  async function save() {
    setError("");
    if (!processId) {
      setError("Select a workflow.");
      return;
    }
    if (!delegateId) {
      setError("Select a substitute user.");
      return;
    }
    setSaving(true);
    try {
      const data = await api.put<SubstituteView>("/workflow/substitute/me", {
        processId,
        delegateUserId: delegateId,
        nodeId: nodeId || "",
        endsAt: endsAt ? new Date(endsAt).toISOString() : "",
      });
      setView(data);
    } catch (err) {
      setError(formatApiError(err, "Failed to save delegation"));
    } finally {
      setSaving(false);
    }
  }

  async function removeAssignment(id?: string) {
    if (!id) return;
    setError("");
    setSaving(true);
    try {
      const data = await api.del<SubstituteView>(`/workflow/substitute/me/${id}`);
      setView(data);
    } catch (err) {
      setError(formatApiError(err, "Failed to remove delegation"));
    } finally {
      setSaving(false);
    }
  }

  async function clearAll() {
    setError("");
    setSaving(true);
    try {
      const data = await api.put<SubstituteView>("/workflow/substitute/me", {
        delegateUserId: "",
      });
      setView(data);
      setDelegateId("");
      setDelegateLabel("");
      setNodeId("");
      setEndsAt("");
    } catch (err) {
      setError(formatApiError(err, "Failed to clear delegations"));
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <Spinner />;
  if (!view?.canDelegate) {
    return (
      <div className="space-y-4">
        <InquiryHeader
          crumb="Profile · Substitute"
          title="Approval delegation"
        />
        {error && <ErrorBanner message={error} />}
        <section className="rounded-lg border border-slate-200 bg-white p-4">
          <p className="text-sm text-slate-600">
            Only users assigned to an approval step can appoint a substitute. Ask an
            administrator to assign the matching role under Users → Edit.
          </p>
        </section>
      </div>
    );
  }

  const assignments = view.assignments || [];

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap justify-end gap-2">
        <Button onClick={save} loading={saving} disabled={!delegateId || !processId}>
          Save assignment
        </Button>
        <Button
          variant="secondary"
          onClick={clearAll}
          loading={saving}
          disabled={assignments.length === 0}
        >
          Clear all
        </Button>
      </div>
      {error && <ErrorBanner message={error} />}
      <InquiryHeader
        crumb="Profile · Substitute"
        title="Approval delegation"
        pills={
          <StatusPill tone="navy">
            {assignments.length} assignment{assignments.length === 1 ? "" : "s"}
          </StatusPill>
        }
      />
      <p className="text-sm text-slate-600">
        Appoint a different substitute per workflow — for example one person on
        gantry loading and another on vessel receipts. While the assignment is
        effective they receive the same email and SMS as you would for new
        approval tasks, marked as covering for you. Their Approvals inbox shows
        your pending work; the trail records they acted <em>on behalf of</em> you.
        The substitute needs Approvals access (
        <code className="text-xs">workflow.tasks</code>).
      </p>

      <section className="overflow-hidden rounded-lg border border-slate-200 bg-white">
        <h4 className="border-b border-slate-100 px-4 py-3 text-xs font-bold uppercase tracking-wide text-brand-800">
          Current assignments
        </h4>
        {assignments.length === 0 ? (
          <p className="px-4 py-6 text-sm text-slate-500">
            No substitutes set. Add one below for each workflow you operate.
          </p>
        ) : (
          <table className="pp-ledger min-w-full text-left text-sm">
            <thead className="pp-table-head">
              <tr>
                <th>Workflow</th>
                <th>Step</th>
                <th>Substitute</th>
                <th>Ends</th>
                <th>Status</th>
                <th> </th>
              </tr>
            </thead>
            <tbody>
              {assignments.map((a) => (
                <tr key={a.id} className="border-t border-slate-100">
                  <td className="px-4 py-3 font-medium text-slate-900">
                    {a.processName || "—"}
                  </td>
                  <td className="px-4 py-3 text-slate-600">
                    {a.nodeName || "All my steps"}
                  </td>
                  <td className="px-4 py-3">
                    <div className="font-medium text-slate-900">
                      {a.delegateName || a.delegateEmail || "—"}
                    </div>
                    {a.delegateName && a.delegateEmail ? (
                      <div className="text-xs text-slate-500">
                        {a.delegateEmail}
                      </div>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 text-slate-600">
                    {a.endsAt ? formatDateTime(a.endsAt) : "Open-ended"}
                  </td>
                  <td className="px-4 py-3">
                    <StatusPill tone={a.effective ? "emerald" : "slate"}>
                      {a.effective ? "Active" : "Inactive"}
                    </StatusPill>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      type="button"
                      variant="secondary"
                      className="!px-2 !py-1 text-xs"
                      onClick={() => removeAssignment(a.id)}
                      loading={saving}
                    >
                      Remove
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <InquirySection title="Add or replace by workflow">
        <Field label="Workflow">
          <Select
            value={processId}
            onChange={(e) => {
              setProcessId(e.target.value);
              setNodeId("");
            }}
          >
            <option value="">Select workflow…</option>
            {processes.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Limit to step (optional)">
          <Select
            value={nodeId}
            onChange={(e) => setNodeId(e.target.value)}
            disabled={!processId}
          >
            <option value="">All my steps on this workflow</option>
            {stepsForProcess.map((s) => (
              <option key={s.nodeId} value={s.nodeId}>
                {s.nodeName}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Substitute">
          <AsyncSearchableSelect
            value={delegateId}
            selectedLabel={delegateLabel}
            placeholder="Search user by name or email…"
            loadOptions={async (q) => {
              const rows = await api.get<
                {
                  id: string;
                  email: string;
                  firstName?: string;
                  lastName?: string;
                }[]
              >("/auth/users/search", {
                search: q || undefined,
                pageSize: 20,
                excludeSelf: true,
              });
              const selfId = meUser?.id;
              const selfEmail = (meUser?.email || "").toLowerCase();
              return (rows || [])
                .filter((u) => {
                  if (selfId && u.id === selfId) return false;
                  if (selfEmail && (u.email || "").toLowerCase() === selfEmail) {
                    return false;
                  }
                  return true;
                })
                .map((u) => {
                  const name = [u.firstName, u.lastName]
                    .filter(Boolean)
                    .join(" ");
                  return {
                    value: u.id,
                    label: `${name || u.email} · ${u.email}`,
                  };
                });
            }}
            onChange={(value, option) => {
              if (value && value === meUser?.id) return;
              setDelegateId(value);
              setDelegateLabel(option?.label || "");
            }}
          />
        </Field>
        <Field label="Ends at (optional)">
          <Input
            type="datetime-local"
            value={endsAt}
            onChange={(e) => setEndsAt(e.target.value)}
          />
        </Field>
      </InquirySection>
    </div>
  );
}

function RolesTab({
  roles,
  isSuperUser,
  permissions,
}: {
  roles: Role[];
  isSuperUser: boolean;
  permissions: string[];
}) {
  const groups = useMemo(() => groupPermissions(permissions), [permissions]);
  const [showPerms, setShowPerms] = useState(!isSuperUser);

  return (
    <div className="space-y-4">
      <InquiryHeader
        crumb="Profile · Roles"
        title="Assigned roles & permissions"
        pills={
          isSuperUser ? (
            <StatusPill tone="navy">Super user</StatusPill>
          ) : (
            <StatusPill>
              {roles.length} role{roles.length === 1 ? "" : "s"}
            </StatusPill>
          )
        }
      />
      {isSuperUser && (
        <div className="rounded-lg bg-brand-50 px-3 py-2 text-sm text-brand-900">
          You are a super user — all permissions are granted.
        </div>
      )}
      <section className="rounded-lg border border-slate-200 bg-white p-4">
        <h4 className="mb-3 text-xs font-bold uppercase tracking-wide text-brand-800">
          Assigned roles
        </h4>
        {roles.length === 0 ? (
          <p className="mt-2 text-sm text-slate-500">No roles assigned.</p>
        ) : (
          <ul className="mt-3 space-y-2">
            {roles.map((role) => (
              <li
                key={role.id}
                className="rounded-lg border border-slate-100 bg-slate-50 px-3 py-2"
              >
                <p className="text-sm font-medium text-slate-900">{role.name}</p>
                {role.description && (
                  <p className="text-xs text-slate-500">{role.description}</p>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
      <section className="rounded-lg border border-slate-200 bg-white p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h4 className="text-xs font-bold uppercase tracking-wide text-brand-800">
            Effective permissions
          </h4>
          {isSuperUser && groups.length > 0 && (
            <button
              type="button"
              className="text-xs font-medium text-brand-800 hover:underline"
              onClick={() => setShowPerms((v) => !v)}
            >
              {showPerms ? "Hide breakdown" : "Show breakdown"}
            </button>
          )}
        </div>
        {groups.length === 0 ? (
          <p className="mt-2 text-sm text-slate-500">No permissions listed.</p>
        ) : !showPerms ? (
          <p className="mt-2 text-sm text-slate-500">
            {groups.length} modules · {permissions.length} permissions
          </p>
        ) : (
          <ul className="mt-3 divide-y divide-slate-100 rounded-lg border border-slate-100">
            {groups.map(({ resource, actions }) => (
              <li
                key={resource}
                className="flex flex-col gap-2 px-3 py-2.5 sm:flex-row sm:items-center sm:gap-4"
              >
                <span className="w-28 shrink-0 text-sm font-medium text-slate-800">
                  {resourceLabel(resource)}
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {actions.map(({ action, code }) => (
                    <span
                      key={code}
                      title={code}
                      className="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-700"
                    >
                      {action}
                    </span>
                  ))}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
