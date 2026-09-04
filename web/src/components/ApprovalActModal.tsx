"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { formatDateTime, formatWait, waitOverdue } from "@/lib/format";
import type { WorkflowEvent, WorkflowTask } from "@/lib/types";
import {
  approvalApiPath,
  approvalAttachLockedCaption,
  approvalAttachPath,
  approvalDocumentHref,
  approvalDocumentLabel,
} from "@/lib/approvals";
import { DocumentFiles } from "@/components/DocumentFiles";
import { WorkflowDiagram } from "@/components/WorkflowDiagram";
import {
  ApprovalHistory,
  ErpField,
  InquiryHeader,
  InquirySection,
  PriorComments,
  ReviewTabBar,
  StatusPill,
} from "@/components/ApprovalReview";
import { Modal } from "@/components/Modal";
import { Button, ErrorBanner, Field, Select, Textarea } from "@/components/ui";

type Tab = "document" | "files" | "flow" | "trail" | "decision";
type ActKind = "agree" | "reject" | "transfer" | "back";

type FlowNode = { id: string; name: string; status?: string; sequence?: number };

type DocumentFacts = {
  id?: string;
  documentNumber?: string;
  status?: string;
  description?: string;
  fromCurrency?: string;
  toCurrency?: string;
  rate?: string;
  effectiveFrom?: string;
  customerName?: string;
  customerCode?: string;
  product?: string;
  quantity?: string;
  vessel?: string;
  batchNumber?: string;
  amount?: string;
  currencyCode?: string;
};

type InstanceFlow = {
  id: string;
  no: string;
  summary: string;
  status: string;
  currentNodeName?: string;
  visitedNodeNames?: string[];
  docId?: string;
  documentNumber?: string;
  attachmentCount?: number;
  document?: DocumentFacts | null;
  process: {
    process: { name: string; nodes?: FlowNode[] };
    diagram: { from: string; to: string; actType: string; name?: string }[];
  };
  history: WorkflowEvent[];
};

export function ApprovalActModal({
  queueLabel,
  task,
  onClose,
  onDone,
}: {
  queueLabel: string;
  task: WorkflowTask;
  onClose: () => void;
  onDone: () => void;
}) {
  const [tab, setTab] = useState<Tab>("document");
  const [pending, setPending] = useState<ActKind | null>(null);
  const [totalRejection, setTotalRejection] = useState(false);
  const [targetNode, setTargetNode] = useState("");
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState("");

  const isAmendment = (task.nodeStatus || "").toLowerCase() === "draft";

  const flow = useAsync(
    () => api.get<InstanceFlow>(`/workflow/instances/${task.instanceId}`),
    [task.instanceId],
  );

  const history = flow.data?.history || [];
  const docId = task.docId || flow.data?.docId || flow.data?.document?.id || "";
  const apiPath = approvalApiPath(task.docContentType, docId);
  const docLabel = approvalDocumentLabel({
    ...task,
    documentNumber: task.documentNumber || flow.data?.documentNumber || flow.data?.document?.documentNumber,
    no: task.no || flow.data?.no,
  });
  const href = approvalDocumentHref(task.docContentType, docId);
  const moduleAttach = approvalAttachPath(task.docContentType, docId);
  const attachPath =
    isAmendment && moduleAttach ? moduleAttach : `/workflow/instances/${task.instanceId}`;
  const canAddFiles = Boolean(isAmendment && moduleAttach);
  const fileCount = flow.data?.attachmentCount;
  const detail = useAsync(
    () => (apiPath ? api.get<Record<string, unknown>>(apiPath) : Promise.resolve(null)),
    [apiPath],
  );
  const processNodes = useMemo(() => {
    const nodes = flow.data?.process?.process?.nodes || [];
    return nodes.filter((n) => {
      const st = (n.status || "").toLowerCase();
      if (st === "completed" || st === "rejected") return false;
      if (n.name && n.name === task.nodeName) return false;
      return true;
    });
  }, [flow.data, task.nodeName]);

  function openAct(kind: ActKind) {
    setPending(kind);
    setTotalRejection(false);
    setTargetNode("");
    setComment("");
    setFormError("");
  }

  async function submit() {
    if (!pending) return;
    setFormError("");
    if ((pending === "agree" || pending === "reject") && !comment.trim()) {
      setFormError("Add a comment for the audit trail.");
      return;
    }
    if ((pending === "transfer" || pending === "back") && !targetNode) {
      setFormError("Select the step to send this document to.");
      return;
    }
    setBusy(true);
    try {
      await api.post(`/workflow/instances/${task.instanceId}/act`, {
        action: pending,
        comment: comment.trim(),
        totalRejection: pending === "reject" ? totalRejection : false,
        targetNode: pending === "transfer" || pending === "back" ? targetNode : undefined,
      });
      onDone();
    } catch (err) {
      setFormError(formatApiError(err, "Could not complete the decision"));
    } finally {
      setBusy(false);
    }
  }

  const agreeLabel = isAmendment ? "Resubmit" : "Approve";

  return (
    <Modal
      open
      size="2xl"
      title={`Review ${docLabel}`}
      onClose={onClose}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          {href ? (
            <Link
              href={href}
              className="inline-flex items-center rounded-xl border border-slate-200 px-4 py-2 text-sm font-semibold text-brand-800"
            >
              Open document
            </Link>
          ) : null}
        </>
      }
    >
      <div className="space-y-4">
        {formError ? <ErrorBanner message={formError} /> : null}
        <InquiryHeader
          crumb={`Approvals · ${queueLabel}`}
          title={docLabel}
          subtitle={queueLabel}
          pills={
            <StatusPill tone={waitOverdue(task.createdAt) ? "amber" : "navy"}>
              {waitOverdue(task.createdAt) ? "Overdue" : "Pending"} · {formatWait(task.createdAt)}
            </StatusPill>
          }
        />
        <ReviewTabBar
          tab={tab}
          onChange={setTab}
          ariaLabel="Review sections"
          ids={[
            { id: "document", label: "Document" },
            { id: "files", label: "Attachments", count: fileCount || undefined },
            { id: "flow", label: "Workflow" },
            { id: "trail", label: "Approval trail", count: history.length || undefined },
            { id: "decision", label: "Decision" },
          ]}
        />

        {tab === "document" ? (
          <DocumentPanel
            task={task}
            queueLabel={queueLabel}
            facts={flow.data?.document}
            detail={detail.data}
            loading={detail.loading || flow.loading}
            history={history}
          />
        ) : null}

        {tab === "files" ? (
          <DocumentFiles
            path={attachPath}
            draft={canAddFiles}
            hint="Supporting files on this document. Approvers can preview and download."
            lockedCaption={approvalAttachLockedCaption(task.docContentType)}
          />
        ) : null}

        {tab === "flow" ? (
          <FlowBody flow={flow.data} loading={flow.loading} error={flow.error} fallbackName={task.nodeName} />
        ) : null}

        {tab === "trail" ? (
          flow.loading ? (
            <p className="text-sm text-slate-500">Loading trail…</p>
          ) : (
            <ApprovalHistory events={history} />
          )
        ) : null}

        {tab === "decision" ? (
          <div className="space-y-4">
            {isAmendment ? (
              <p className="rounded-lg border border-orange-200 bg-orange-50 px-3 py-2 text-sm text-orange-950">
                This document was returned for correction. Resubmit after you fix it, or reject to stop the workflow.
              </p>
            ) : null}
            <InquirySection title="This step" columns={3}>
              <ErpField label="Step" value={task.nodeName || "—"} />
              <ErpField label="Received" value={formatDateTime(task.createdAt)} />
              <ErpField label="On behalf of" value={task.onBehalfOfName || "You"} />
            </InquirySection>
            <div>
              <p className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                Choose action
              </p>
              <p className="mb-3 text-sm text-slate-600">
                Each action opens a comment box. Comments start empty — write what belongs on the trail.
              </p>
              <div className="flex flex-wrap gap-2">
                <Button type="button" onClick={() => openAct("agree")}>
                  {agreeLabel}
                </Button>
                <Button type="button" variant="secondary" onClick={() => openAct("reject")}>
                  Reject
                </Button>
                {!isAmendment ? (
                  <>
                    <Button type="button" variant="secondary" onClick={() => openAct("transfer")}>
                      Send to
                    </Button>
                    <Button type="button" variant="secondary" onClick={() => openAct("back")}>
                      Back to
                    </Button>
                  </>
                ) : null}
              </div>
            </div>
            <InquirySection title="Previous comments" columns={1}>
              <PriorComments events={history} />
            </InquirySection>
          </div>
        ) : null}
      </div>
      {pending ? (
        <Modal
          open
          layer="nested"
          size="md"
          title={confirmTitle(pending, isAmendment)}
          onClose={busy ? () => undefined : () => setPending(null)}
          footer={
            <>
              <Button variant="secondary" onClick={() => setPending(null)} disabled={busy}>
                Cancel
              </Button>
              <Button
                variant={pending === "reject" && totalRejection ? "danger" : "primary"}
                loading={busy}
                onClick={() => void submit()}
              >
                {confirmLabel(pending, totalRejection, isAmendment)}
              </Button>
            </>
          }
        >
          <div className="space-y-4">
            {formError ? <ErrorBanner message={formError} /> : null}
            <p className="text-sm text-slate-700">{confirmHint(pending, totalRejection, isAmendment)}</p>
            {pending === "reject" ? (
              <label className="flex items-start gap-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-800">
                <input
                  type="checkbox"
                  className="mt-1 h-4 w-4 accent-brand-800"
                  checked={totalRejection}
                  onChange={(e) => setTotalRejection(e.target.checked)}
                />
                <span>
                  <span className="font-semibold">Terminate this request (total rejection)</span>
                  <span className="mt-0.5 block text-xs text-slate-600">
                    Leave unchecked to return the document for correction. That is the default.
                  </span>
                </span>
              </label>
            ) : null}
            {pending === "transfer" || pending === "back" ? (
              <Field
                label={pending === "transfer" ? "Send to (in-progress step)" : "Back to (in-progress step)"}
              >
                <Select value={targetNode} onChange={(e) => setTargetNode(e.target.value)}>
                  <option value="">Select step…</option>
                  {processNodes.map((n) => (
                    <option key={n.id} value={n.id}>
                      {n.name}
                    </option>
                  ))}
                </Select>
                {processNodes.length === 0 ? (
                  <p className="mt-1 text-xs text-amber-800">No other in-progress steps are available.</p>
                ) : null}
              </Field>
            ) : null}
            <Field label={pending === "agree" || pending === "reject" ? "Comment (required)" : "Comment (optional)"}>
              <Textarea
                rows={4}
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder={commentPlaceholder(pending, totalRejection, isAmendment)}
              />
            </Field>
          </div>
        </Modal>
      ) : null}
    </Modal>
  );
}

export function ApprovalViewModal({
  queueLabel,
  instanceId,
  onClose,
}: {
  queueLabel: string;
  instanceId: string;
  onClose: () => void;
}) {
  const [tab, setTab] = useState<"trail" | "files" | "flow">("trail");
  const flow = useAsync(
    () => api.get<InstanceFlow>(`/workflow/instances/${instanceId}`),
    [instanceId],
  );
  return (
    <Modal
      open
      size="2xl"
      title={`${queueLabel} · ${flow.data?.documentNumber || flow.data?.no || "trail"}`}
      onClose={onClose}
      footer={
        <Button variant="secondary" onClick={onClose}>
          Close
        </Button>
      }
    >
      <ReviewTabBar
        tab={tab}
        onChange={setTab}
        ariaLabel="Decision record"
        ids={[
          { id: "trail", label: "Approval trail", count: flow.data?.history?.length || undefined },
          { id: "files", label: "Attachments", count: flow.data?.attachmentCount || undefined },
          { id: "flow", label: "Workflow" },
        ]}
      />
      <div className="mt-4">
        {tab === "trail" ? (
          flow.loading ? (
            <p className="text-sm text-slate-500">Loading trail…</p>
          ) : (
            <ApprovalHistory events={flow.data?.history} />
          )
        ) : null}
        {tab === "files" ? (
          <DocumentFiles
            path={`/workflow/instances/${instanceId}`}
            draft={false}
            hint="Supporting files on this document."
          />
        ) : null}
        {tab === "flow" ? (
          <FlowBody flow={flow.data} loading={flow.loading} error={flow.error} />
        ) : null}
      </div>
    </Modal>
  );
}

function confirmTitle(action: ActKind, amendment: boolean): string {
  if (action === "agree") return amendment ? "Resubmit document" : "Approve document";
  if (action === "reject") return "Reject or return";
  if (action === "transfer") return "Send to another step";
  return "Send back";
}

function confirmLabel(action: ActKind, totalRejection: boolean, amendment: boolean): string {
  if (action === "agree") return amendment ? "Resubmit" : "Approve";
  if (action === "reject") return totalRejection ? "Reject" : "Return for correction";
  if (action === "transfer") return "Send to";
  return "Back to";
}

function confirmHint(action: ActKind, totalRejection: boolean, amendment: boolean): string {
  if (action === "agree") {
    return amendment
      ? "Confirm you have corrected the document. Your comment is stored on the approval trail."
      : "Confirm approval. Your comment is stored on the approval trail.";
  }
  if (action === "reject") {
    return totalRejection
      ? "This terminates the workflow. The document cannot be resubmitted."
      : "The document returns to the initiator for correction. The workflow stays open until they resubmit.";
  }
  if (action === "transfer") {
    return "Move this document to another in-progress step without rejecting it.";
  }
  return "Return this document to an earlier in-progress step without terminating the workflow.";
}

function commentPlaceholder(action: ActKind, totalRejection: boolean, amendment: boolean): string {
  if (action === "agree") {
    return amendment ? "What did you correct?" : "Why are you approving?";
  }
  if (action === "reject") {
    return totalRejection ? "Why is this request terminated?" : "What should the initiator correct?";
  }
  return "Optional note for the audit trail";
}

function FlowBody({
  flow,
  loading,
  error,
  fallbackName,
}: {
  flow?: InstanceFlow | null;
  loading: boolean;
  error?: string | null;
  fallbackName?: string;
}) {
  if (loading) return <p className="text-sm text-slate-500">Loading workflow…</p>;
  if (error || !flow) return <ErrorBanner message={error || "Could not load workflow"} />;
  const nodes = flow.process?.process?.nodes || [];
  const edges = flow.process?.diagram || [];
  return (
    <div className="space-y-3">
      <p className="text-sm text-slate-600">
        {flow.process?.process?.name || "Approval"} · {flow.no}
        {flow.currentNodeName ? ` · current step: ${flow.currentNodeName}` : ""}
      </p>
      <WorkflowDiagram
        nodes={nodes}
        edges={edges}
        currentNodeName={flow.currentNodeName || fallbackName}
        visitedNodeNames={flow.visitedNodeNames}
        title="Approval flow"
        heightClass="h-96"
      />
    </div>
  );
}

function DocumentPanel({
  task,
  queueLabel,
  facts,
  detail,
  loading,
  history,
}: {
  task: WorkflowTask;
  queueLabel: string;
  facts?: DocumentFacts | null;
  detail?: Record<string, unknown> | null;
  loading: boolean;
  history: WorkflowEvent[];
}) {
  const fields = documentFields(task, queueLabel, facts, detail);
  return (
    <div className="space-y-4">
      <InquirySection title="Document" columns={3}>
        {fields.map((f) => (
          <ErpField key={f.label} label={f.label} value={f.value} />
        ))}
      </InquirySection>
      {loading ? <p className="text-xs text-slate-500">Loading document details…</p> : null}
      <InquirySection title="Previous comments" columns={1}>
        <PriorComments events={history} />
      </InquirySection>
    </div>
  );
}

function documentFields(
  task: WorkflowTask,
  queueLabel: string,
  facts?: DocumentFacts | null,
  detail?: Record<string, unknown> | null,
): { label: string; value: string }[] {
  const pick = (...vals: (string | undefined)[]) => vals.find((v) => (v || "").trim()) || "";
  const out: { label: string; value: string }[] = [
    { label: "Type", value: queueLabel },
    {
      label: "Reference",
      value: approvalDocumentLabel({
        ...task,
        documentNumber: pick(task.documentNumber, facts?.documentNumber),
      }),
    },
    { label: "Step", value: task.nodeName || "—" },
  ];
  const customer = pick(task.customerName, facts?.customerName);
  const customerCode = pick(task.customerCode, facts?.customerCode);
  const amount = pick(task.amount, facts?.amount);
  const currency = pick(task.currencyCode, facts?.currencyCode);
  const str = (k: string) => {
    const fromFacts = facts ? String((facts as Record<string, unknown>)[k] ?? "") : "";
    if (fromFacts) return fromFacts;
    const v = detail?.[k];
    return v == null || v === "" ? "" : String(v);
  };
  if (customer) out.push({ label: "Customer", value: customer });
  if (customerCode) out.push({ label: "Customer code", value: customerCode });
  if (str("product")) out.push({ label: "Product", value: str("product") });
  if (str("quantity")) out.push({ label: "Quantity", value: str("quantity") });
  if (str("vessel")) out.push({ label: "Vessel", value: str("vessel") });
  if (str("batchNumber")) out.push({ label: "Batch", value: str("batchNumber") });
  if (amount) out.push({ label: "Amount", value: `${currency} ${amount}`.trim() });
  const pair = [str("fromCurrency"), str("toCurrency")].filter(Boolean).join("/");
  if (pair) out.push({ label: "Pair", value: pair });
  if (str("rate")) out.push({ label: "Rate", value: str("rate") });
  if (str("effectiveFrom")) out.push({ label: "Effective from", value: str("effectiveFrom").slice(0, 10) });
  if (str("description")) out.push({ label: "Description", value: str("description") });
  if (str("status")) out.push({ label: "Status", value: str("status") });
  return out;
}
