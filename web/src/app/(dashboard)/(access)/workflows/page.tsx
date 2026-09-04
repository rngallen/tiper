"use client";

import { useMemo, useState } from "react";
import { ChevronDown, ChevronUp, Trash2 } from "lucide-react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { DataTable, type Column } from "@/components/DataTable";
import { Modal } from "@/components/Modal";
import { SearchableSelect } from "@/components/SearchableSelect";
import {
  Button,
  ErrorBanner,
  Field,
  Input,
  Select,
} from "@/components/ui";
import { ListSearch } from "@/lib/list";
import { Paginator } from "@/components/Paginator";
import { ListShell } from "@/components/ListShell";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { WorkflowDiagram } from "@/components/WorkflowDiagram";
import { UserPoolPicker, type PoolUser } from "@/components/UserPoolPicker";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  ReviewTabBar,
  StatusPill,
} from "@/components/ApprovalReview";

interface PoolMember {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
}

interface OperatorRole {
  id: number;
  name: string;
}

interface WorkflowNode {
  id: string;
  name: string;
  status: string;
  sequence: number;
  quorumMode?: string;
  operatorRoles?: OperatorRole[];
}

interface DiagramEdge {
  from: string;
  to: string;
  actType: string;
  name?: string;
  code?: string;
}

interface WorkflowProcess {
  id: string;
  code: string;
  name: string;
  amendmentMode: "creator" | "pool" | string;
  note?: string;
  nodes?: WorkflowNode[];
}

interface ApprovalStep {
  id: string;
  name: string;
  sequence: number;
  quorumMode: string;
  roleId: number;
  roleName?: string;
}

interface ProcessView {
  process: WorkflowProcess;
  diagram: DiagramEdge[];
  initiatorPool: PoolMember[];
  poolMemberIds: string[];
  notifyOnSubmit?: PoolMember[];
  notifyOnComplete?: PoolMember[];
  notifyOnReject?: PoolMember[];
  amendmentModes: string[];
  operatorRoleOptions?: OperatorRole[];
  approvalSteps?: ApprovalStep[];
}

function toPoolUsers(members?: PoolMember[]): PoolUser[] {
  return (members || []).map((m) => ({
    id: m.id,
    email: m.email,
    firstName: m.firstName,
    lastName: m.lastName,
  }));
}

function memberNames(members?: PoolMember[]): string {
  const rows = members || [];
  if (rows.length === 0) return "None";
  return rows
    .map((m) => {
      const name = [m.firstName, m.lastName].filter(Boolean).join(" ").trim();
      return name || m.email;
    })
    .join(", ");
}

function processStepCount(view: ProcessView) {
  if (view.approvalSteps && view.approvalSteps.length > 0) {
    return view.approvalSteps.length;
  }
  const running = (view.process.nodes || []).filter((n) => n.status === "running");
  return running.length || (view.process.nodes || []).length;
}

function watcherCounts(view: ProcessView) {
  return {
    submit: view.notifyOnSubmit?.length ?? 0,
    complete: view.notifyOnComplete?.length ?? 0,
    reject: view.notifyOnReject?.length ?? 0,
  };
}

type SetupTab = "steps" | "amendment" | "notifications";
type NotifyEventTab = "submit" | "complete" | "reject";
type InquiryTab = "definition" | "notifications" | "flow";

function automaticOutcomeLabel(mode: string, event: NotifyEventTab): string {
  if (event === "submit") {
    return "First-step operators (action-required task)";
  }
  return mode === "pool"
    ? "Initiator pool (and the original initiator if not in the pool)"
    : "Original initiator";
}

interface EditableStep {
  key: string;
  id?: string;
  name: string;
  roleId: number;
  quorumMode: "any" | "all";
}

export default function WorkflowsPage() {
  const { hasPermission } = useAuth();
  const canManage = hasPermission(PERMISSIONS.workflowManage);
  const [editing, setEditing] = useState<ProcessView | null>(null);
  const [editTab, setEditTab] = useState<SetupTab>("steps");
  const [viewingDiagram, setViewingDiagram] = useState<ProcessView | null>(null);
  const [diagramLoading, setDiagramLoading] = useState(false);
  const [diagramError, setDiagramError] = useState("");

  const { data, loading, error, reload } = useAsync(
    () => api.get<ProcessView[]>("/workflow/processes"),
    [],
  );
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    const rows = data || [];
    if (!q) return rows;
    return rows.filter((r) => {
      const hay = `${r.process.code} ${r.process.name} ${r.process.note || ""}`.toLowerCase();
      return hay.includes(q);
    });
  }, [data, search]);
  const paged = useMemo(() => {
    const total = filtered.length;
    const totalPages = Math.max(1, Math.ceil(total / pageSize));
    const safePage = Math.min(page, totalPages);
    const start = (safePage - 1) * pageSize;
    return {
      items: filtered.slice(start, start + pageSize),
      page: safePage,
      pageSize,
      totalPages,
      itemsCount: total,
    };
  }, [filtered, page, pageSize]);

  async function openProcess(
    processId: string,
    mode: "configure" | "diagram" | "amendment" | "notifications",
  ) {
    setDiagramError("");
    if (mode === "diagram") setDiagramLoading(true);
    try {
      const view = await api.get<ProcessView>(
        `/workflow/processes/${processId}`,
      );
      if (mode === "diagram") {
        setViewingDiagram(view);
      } else {
        setEditTab(
          mode === "amendment"
            ? "amendment"
            : mode === "notifications"
              ? "notifications"
              : "steps",
        );
        setEditing(view);
      }
    } catch (err) {
      setDiagramError(formatApiError(err, "Failed to load workflow process"));
    } finally {
      setDiagramLoading(false);
    }
  }

  const columns: Column<ProcessView>[] = [
    {
      header: "Process",
      cell: (r) => (
        <div>
          <div className="font-medium text-slate-800">{r.process.name}</div>
          <div className="text-xs text-slate-400">{r.process.code}</div>
        </div>
      ),
    },
    {
      header: "Steps",
      className: "text-right tabular-nums",
      cell: (r) => processStepCount(r),
    },
    {
      header: "Returns to",
      cell: (r) =>
        r.process.amendmentMode === "pool"
          ? "Initiator pool"
          : "Original initiator",
    },
    {
      header: "Pool",
      cell: (r) => {
        const n = r.initiatorPool?.length ?? 0;
        const label = `${n} user${n === 1 ? "" : "s"}`;
        return (
          <span className="text-slate-700">
            {r.process.amendmentMode === "pool"
              ? label
              : n === 0
                ? "Unused"
                : `${label} (unused)`}
          </span>
        );
      },
    },
    {
      header: "Watchers",
      cell: (r) => {
        const c = watcherCounts(r);
        const total = c.submit + c.complete + c.reject;
        return (
          <div>
            <div className="text-slate-800">{total} user{total === 1 ? "" : "s"}</div>
            <div className="text-xs text-slate-400">
              {c.submit} submit · {c.complete} approved · {c.reject} rejected
            </div>
          </div>
        );
      },
    },
    {
      header: "",
      className: "text-right",
      cell: (r) => (
        <div className="flex justify-end gap-2">
          {canManage ? (
            <Button
              variant="secondary"
              className="!px-3 !py-1"
              onClick={(e) => {
                e.stopPropagation();
                void openProcess(r.process.id, "configure");
              }}
              title="Steps, amendment pool, and watchers"
            >
              Configure
            </Button>
          ) : null}
        </div>
      ),
    },
  ];

  return (
    <div>
      <ListShell
        lead={
          <>
            {!canManage ? (
              <p className="mb-4 text-sm text-slate-600">
                You can view this page but need workflow.manage to change configuration.
              </p>
            ) : null}
            {diagramError && !viewingDiagram && !editing ? (
              <div className="mb-4">
                <ErrorBanner message={diagramError} />
              </div>
            ) : null}
          </>
        }
        search={
          <ListSearch
            value={search}
            onChange={(v) => {
              setSearch(v);
              setPage(1);
            }}
            placeholder="Search process name or code"
          />
        }
        footer={
          !loading && !error ? (
            <Paginator
              embedded
              data={paged}
              page={paged.page}
              onPage={setPage}
              pageSize={pageSize}
              onPageSize={(n) => {
                setPageSize(n);
                setPage(1);
              }}
            />
          ) : null
        }
      >
        <DataTable
          plain
          columns={columns}
          rows={paged.items}
          loading={loading || diagramLoading}
          error={error}
          emptyMessage={search.trim() ? "No workflows match that search" : "No workflow processes seeded yet"}
          onRowClick={(r) => void openProcess(r.process.id, "diagram")}
        />
      </ListShell>

      {editing && (
        <ConfigureProcessModal
          initial={editing}
          initialTab={editTab}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            reload();
          }}
        />
      )}
      {viewingDiagram && (
        <ViewProcessModal
          view={viewingDiagram}
          canManage={canManage}
          onClose={() => setViewingDiagram(null)}
          onConfigure={() => {
            setEditing(viewingDiagram);
            setEditTab("steps");
            setViewingDiagram(null);
          }}
        />
      )}
    </div>
  );
}

function ViewProcessModal({
  view,
  canManage,
  onClose,
  onConfigure,
}: {
  view: ProcessView;
  canManage: boolean;
  onClose: () => void;
  onConfigure: () => void;
}) {
  const [tab, setTab] = useState<InquiryTab>("definition");
  const stepCount = processStepCount(view);
  const watchers = watcherCounts(view);
  const watcherTotal = watchers.submit + watchers.complete + watchers.reject;
  return (
    <Modal
      open
      size="xl"
      onClose={onClose}
      title="Workflow"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
          {canManage ? (
            <Button onClick={onConfigure}>Configure</Button>
          ) : null}
        </>
      }
    >
      <div className="space-y-4">
        <InquiryHeader
          crumb="Workflows · Inquiry"
          title={view.process.name}
          pills={
            <>
              <StatusPill tone="navy">
                {view.process.code || "Process"}
              </StatusPill>
              <StatusPill>{stepCount} steps</StatusPill>
              <StatusPill>{watcherTotal} watchers</StatusPill>
            </>
          }
        />
        <ReviewTabBar
          tab={tab}
          onChange={setTab}
          ids={[
            { id: "definition", label: "Definition" },
            { id: "notifications", label: "Notifications", count: watcherTotal },
            { id: "flow", label: "Flow" },
          ]}
        />
        {tab === "definition" ? (
          <InquirySection title="General">
            <ErpField
              label="Amendment"
              value={
                view.process.amendmentMode === "pool"
                  ? "Initiator pool — any member"
                  : "Original initiator only"
              }
            />
            <ErpField
              label="Note"
              value={
                view.process.note || "Approval chain for this process."
              }
            />
          </InquirySection>
        ) : tab === "notifications" ? (
          <InquirySection title="Information mail" columns={1}>
            <p className="text-xs text-slate-500">
              Watchers receive email only. They do not get an Approvals inbox
              item and cannot approve or reject.
            </p>
            <section className="overflow-hidden rounded-lg border border-slate-200">
              <table className="pp-ledger min-w-full text-left text-sm">
                <thead className="pp-table-head">
                  <tr>
                    <th className="w-36">Event</th>
                    <th>Always notified</th>
                    <th>Additional watchers</th>
                  </tr>
                </thead>
                <tbody>
                  {(
                    [
                      ["submit", "On submit", view.notifyOnSubmit],
                      ["complete", "On approved", view.notifyOnComplete],
                      ["reject", "On rejected", view.notifyOnReject],
                    ] as const
                  ).map(([event, label, members]) => (
                    <tr key={event}>
                      <td className="font-semibold text-slate-900">{label}</td>
                      <td className="text-slate-700">
                        {automaticOutcomeLabel(
                          view.process.amendmentMode,
                          event,
                        )}
                      </td>
                      <td className="text-slate-700">{memberNames(members)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </section>
          </InquirySection>
        ) : (
          <WorkflowDiagram
            nodes={view.process.nodes || []}
            edges={view.diagram || []}
            heightClass="h-[min(38rem,calc(90vh-16rem))]"
          />
        )}
      </div>
    </Modal>
  );
}

function stepsFromProcess(view: ProcessView): EditableStep[] {
  const fromApi = (view.approvalSteps || [])
    .slice()
    .sort((a, b) => a.sequence - b.sequence);
  if (fromApi.length > 0) {
    return fromApi.map((n, i) => ({
      key: n.id || `step-${i}`,
      id: n.id,
      name: n.name,
      roleId: n.roleId || 0,
      quorumMode: n.quorumMode === "all" ? "all" : "any",
    }));
  }
  return (view.process.nodes || [])
    .filter((n) => n.status === "running")
    .slice()
    .sort((a, b) => a.sequence - b.sequence)
    .map((n, i) => ({
      key: n.id || `step-${i}`,
      id: n.id,
      name: n.name,
      roleId: n.operatorRoles?.[0]?.id ?? 0,
      quorumMode: n.quorumMode === "all" ? "all" : "any",
    }));
}

function ConfigureProcessModal({
  initial,
  initialTab = "steps",
  onClose,
  onSaved,
}: {
  initial: ProcessView;
  initialTab?: SetupTab;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [tab, setTab] = useState<SetupTab>(initialTab);
  const [notifyTab, setNotifyTab] = useState<NotifyEventTab>("complete");
  const [mode, setMode] = useState(initial.process.amendmentMode || "pool");
  const [poolMembers, setPoolMembers] = useState<PoolUser[]>(() =>
    toPoolUsers(initial.initiatorPool),
  );
  const [notifySubmit, setNotifySubmit] = useState<PoolUser[]>(() =>
    toPoolUsers(initial.notifyOnSubmit),
  );
  const [notifyComplete, setNotifyComplete] = useState<PoolUser[]>(() =>
    toPoolUsers(initial.notifyOnComplete),
  );
  const [notifyReject, setNotifyReject] = useState<PoolUser[]>(() =>
    toPoolUsers(initial.notifyOnReject),
  );
  const [steps, setSteps] = useState<EditableStep[]>(() =>
    stepsFromProcess(initial),
  );
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");

  const roles = initial.operatorRoleOptions || [];
  const selectedIds = poolMembers.map((m) => m.id);

  function updateStep(key: string, patch: Partial<EditableStep>) {
    setSteps((prev) =>
      prev.map((s) => (s.key === key ? { ...s, ...patch } : s)),
    );
  }

  function moveStep(index: number, dir: -1 | 1) {
    setSteps((prev) => {
      const next = [...prev];
      const j = index + dir;
      if (j < 0 || j >= next.length) return prev;
      [next[index], next[j]] = [next[j], next[index]];
      return next;
    });
  }

  function addStep() {
    const defaultRole = roles[0]?.id ?? 0;
    setSteps((prev) => [
      ...prev,
      {
        key: `new-${Date.now()}`,
        name: "",
        roleId: defaultRole,
        quorumMode: "any",
      },
    ]);
  }

  function removeStep(key: string) {
    setSteps((prev) => (prev.length <= 1 ? prev : prev.filter((s) => s.key !== key)));
  }

  async function save() {
    setFormError("");
    for (let i = 0; i < steps.length; i++) {
      if (!steps[i].name.trim()) {
        setFormError(`Step ${i + 1} needs a name`);
        setTab("steps");
        return;
      }
    }

    setSaving(true);
    try {
      if (mode !== initial.process.amendmentMode) {
        await api.patch(
          `/workflow/processes/${initial.process.id}/amendment-mode`,
          { amendmentMode: mode },
        );
      }
      if (mode === "pool") {
        await api.put(
          `/workflow/processes/${initial.process.id}/initiator-pool`,
          { userIds: selectedIds },
        );
      }
      await api.put(
        `/workflow/processes/${initial.process.id}/notifications`,
        {
          notifyOnSubmit: notifySubmit.map((m) => m.id),
          notifyOnComplete: notifyComplete.map((m) => m.id),
          notifyOnReject: notifyReject.map((m) => m.id),
        },
      );
      await api.put<ProcessView>(
        `/workflow/processes/${initial.process.id}/steps`,
        {
          steps: steps.map((s) => ({
            id: s.id || "",
            name: s.name.trim(),
            roleId: s.roleId,
            quorumMode: s.quorumMode,
          })),
        },
      );
      onSaved();
    } catch (err) {
      setFormError(formatApiError(err, "Failed to save workflow settings"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      open
      size="xl"
      onClose={onClose}
      title="Configure workflow"
      footer={
        <>
          {tab === "steps" ? (
            <Button type="button" variant="secondary" onClick={addStep}>
              Add step
            </Button>
          ) : null}
          <div className="flex-1" />
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={save} loading={saving}>
            Save
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {formError && <ErrorBanner message={formError} />}

        <InquiryHeader
          crumb="Workflows · Setup"
          title={initial.process.name}
          pills={
            <>
              <StatusPill tone="navy">
                {initial.process.code || "Process"}
              </StatusPill>
              <StatusPill>{steps.length} steps</StatusPill>
            </>
          }
        />

        <ReviewTabBar
          tab={tab}
          onChange={setTab}
          ids={[
            { id: "steps", label: "Steps" },
            { id: "amendment", label: "Amendment" },
            {
              id: "notifications",
              label: "Notifications",
              count:
                notifySubmit.length +
                notifyComplete.length +
                notifyReject.length,
            },
          ]}
        />

        {tab === "steps" ? (
          <div className="space-y-3">
            <p className="text-xs text-slate-500">
              Assign a role on a step so it waits for those users (Access →
              Users). Empty role skips the step. Quorum is any one operator, or
              every assigned operator.
            </p>
            <section className="overflow-hidden rounded-lg border border-slate-200 bg-white">
              <table className="pp-ledger min-w-full text-left text-sm">
                <thead className="pp-table-head">
                  <tr>
                    <th className="w-12">#</th>
                    <th>Name</th>
                    <th>Operator role</th>
                    <th>Quorum</th>
                    <th className="w-28"> </th>
                  </tr>
                </thead>
                <tbody>
                  {steps.map((step, index) => (
                    <tr key={step.key}>
                      <td className="font-semibold text-slate-600">
                        {index + 1}
                      </td>
                      <td>
                        <Input
                          value={step.name}
                          onChange={(e) =>
                            updateStep(step.key, { name: e.target.value })
                          }
                          placeholder="e.g. Legal"
                        />
                      </td>
                      <td>
                        <SearchableSelect
                          value={step.roleId ? String(step.roleId) : ""}
                          placeholder="Search operator role…"
                          emptyLabel="No matching role"
                          options={roles.map((r) => ({
                            value: String(r.id),
                            label: r.name,
                          }))}
                          onChange={(value) =>
                            updateStep(step.key, {
                              roleId: Number(value) || 0,
                            })
                          }
                        />
                      </td>
                      <td>
                        <Select
                          value={step.quorumMode}
                          onChange={(e) =>
                            updateStep(step.key, {
                              quorumMode: e.target.value as "any" | "all",
                            })
                          }
                        >
                          <option value="any">Any one</option>
                          <option value="all">All assigned</option>
                        </Select>
                      </td>
                      <td>
                        <div className="flex justify-end gap-0.5">
                          <Button
                            type="button"
                            variant="ghost"
                            className="!px-2 !py-1"
                            disabled={index === 0}
                            onClick={() => moveStep(index, -1)}
                            title="Move up"
                          >
                            <ChevronUp className="h-4 w-4" />
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            className="!px-2 !py-1"
                            disabled={index === steps.length - 1}
                            onClick={() => moveStep(index, 1)}
                            title="Move down"
                          >
                            <ChevronDown className="h-4 w-4" />
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            className="!px-2 !py-1 text-red-700"
                            disabled={steps.length <= 1}
                            onClick={() => removeStep(step.key)}
                            title="Remove step"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </section>
          </div>
        ) : tab === "amendment" ? (
          <div className="space-y-4">
            <InquirySection title="Amendment">
              <div className="sm:col-span-2">
                <Field label="Who amends after soft reject">
                  <Select
                    value={mode}
                    onChange={(e) => setMode(e.target.value)}
                  >
                    <option value="pool">
                      Initiator pool — any member can amend
                    </option>
                    <option value="creator">Original initiator only</option>
                  </Select>
                </Field>
                <p className="mt-2 text-xs text-slate-500">
                  {mode === "pool"
                    ? "Any assigned initiator-pool member can amend a returned document in Approvals. The same people are emailed on full approval and hard reject."
                    : poolMembers.length > 0
                      ? `Returned documents go to the original submitter’s Approvals inbox. The initiator is emailed on approval and hard reject. The ${poolMembers.length} assigned pool member${poolMembers.length === 1 ? "" : "s"} ${poolMembers.length === 1 ? "is" : "are"} unused in this mode and will apply again if you switch back to initiator pool.`
                      : "Returned documents go to the original submitter’s Approvals inbox. The initiator is emailed on full approval and hard reject. Switch to initiator pool to assign members who may amend on their behalf."}
                </p>
              </div>
            </InquirySection>
            {mode === "pool" ? (
              <UserPoolPicker
                title="Initiator pool"
                members={poolMembers}
                onChange={setPoolMembers}
                helpText="Give users the matching initiator role under Users → Edit, then add them here. Soft-rejected documents appear in their Approvals inbox."
              />
            ) : null}
          </div>
        ) : (
          <div className="space-y-4">
            <p className="text-xs text-slate-500">
              Additional watchers receive information mail only — no Approvals
              inbox item and no Approve / Reject. For a shared pool mailbox,
              create a user with that address and add them on the events you
              want copied.
            </p>
            <ReviewTabBar
              tab={notifyTab}
              onChange={setNotifyTab}
              ariaLabel="Notification events"
              ids={[
                { id: "submit", label: "On submit", count: notifySubmit.length },
                {
                  id: "complete",
                  label: "On approved",
                  count: notifyComplete.length,
                },
                {
                  id: "reject",
                  label: "On rejected",
                  count: notifyReject.length,
                },
              ]}
            />
            <InquirySection title="Always notified" columns={1}>
              <ErpField
                label="System recipients"
                value={automaticOutcomeLabel(mode, notifyTab)}
              />
              <p className="text-xs text-slate-500">
                {notifyTab === "submit"
                  ? "First-step operators already get an action-required email. Do not add them here unless they also need a separate information copy."
                  : notifyTab === "complete"
                    ? "Sent when the last approval step agrees and the request is fully approved."
                    : "Sent when a request is terminated. Soft reject (return for correction) is a task for amenders, not this list."}
              </p>
            </InquirySection>
            <UserPoolPicker
              title="Additional watchers"
              members={
                notifyTab === "submit"
                  ? notifySubmit
                  : notifyTab === "complete"
                    ? notifyComplete
                    : notifyReject
              }
              onChange={
                notifyTab === "submit"
                  ? setNotifySubmit
                  : notifyTab === "complete"
                    ? setNotifyComplete
                    : setNotifyReject
              }
              helpText="These users are copied in addition to the system recipients above."
            />
          </div>
        )}
      </div>
    </Modal>
  );
}
