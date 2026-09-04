"use client";

import { useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ExportButton, ListSearch, useListQuery } from "@/lib/list";
import { DataTable } from "@/components/DataTable";
import { Paginator } from "@/components/Paginator";
import { Button, StatusBadge } from "@/components/ui";
import { ListShell } from "@/components/ListShell";
import { EditColumnsButton } from "@/components/EditColumns";
import { useColumnPrefs, type ColumnDef } from "@/lib/columnPrefs";
import { formatDateTime, formatWait, waitOverdue } from "@/lib/format";
import { asPagination } from "@/lib/pagination";
import type { Pagination, WorkflowDecision, WorkflowTask } from "@/lib/types";
import {
  type ApprovalQueueId,
  approvalDocumentLabel,
  approvalQueue,
} from "@/lib/approvals";
import { ApprovalActModal, ApprovalViewModal } from "@/components/ApprovalActModal";

type Lane = "pending" | "approved" | "rejected";

export function ApprovalsQueue({ kind }: { kind: ApprovalQueueId }) {
  const queue = approvalQueue(kind);
  const list = useListQuery({ orderBy: "createdAt", sortDirection: "ASC" });
  const [lane, setLane] = useState<Lane>("pending");
  const [acting, setActing] = useState<WorkflowTask | null>(null);
  const [viewingId, setViewingId] = useState<string | null>(null);

  const query = {
    ...list.query,
    docContentType: queue.types,
    kind: lane === "pending" ? undefined : lane,
  };

  const path =
    lane === "pending" ? "/workflow/tasks/mine" : "/workflow/decisions/mine";

  const { data, loading, error, reload } = useAsync(
    () => api.get<Pagination<WorkflowTask | WorkflowDecision>>(path, query),
    [kind, lane, list.search, list.page, list.pageSize, list.orderBy, list.sortDirection],
  );
  const page = asPagination<WorkflowTask | WorkflowDecision>(data);

  const catalog = useMemo(() => {
    const cols: ColumnDef<WorkflowTask | WorkflowDecision>[] = [
      {
        id: "no",
        header: "Document",
        sortKey: "no",
        defaultVisible: true,
        cell: (r) => (
          <div>
            <div className="font-medium text-slate-800">{approvalDocumentLabel(r)}</div>
            <div className="text-xs text-slate-500">{queue.label}</div>
          </div>
        ),
      },
      {
        id: "step",
        header: lane === "pending" ? "Step" : "Decision",
        defaultVisible: true,
        cell: (r) =>
          lane === "pending" ? (
            (r as WorkflowTask).nodeName
          ) : (
            <StatusBadge
              status={(r as WorkflowDecision).actType === "agree" ? "approved" : "rejected"}
              label={(r as WorkflowDecision).actName || (r as WorkflowDecision).actType}
            />
          ),
      },
    ];
    if (lane === "pending") {
      cols.push({
        id: "waiting",
        header: "Waiting",
        defaultVisible: true,
        cell: (r) => (
          <span className={waitOverdue(r.createdAt) ? "font-semibold text-amber-800" : "tabular-nums text-slate-700"}>
            {formatWait(r.createdAt)}
          </span>
        ),
      });
    }
    cols.push({
      id: "createdAt",
      header: lane === "pending" ? "Received" : "When",
      sortKey: "createdAt",
      defaultVisible: true,
      cell: (r) => formatDateTime(r.createdAt),
    });
    if (lane === "pending") {
      cols.push({
        id: "onBehalf",
        header: "On behalf of",
        defaultVisible: false,
        cell: (r) => (r as WorkflowTask).onBehalfOfName || "You",
      });
      cols.push({
        id: "_actions",
        fixed: true,
        header: "",
        className: "text-right",
        cell: (r) => (
          <Button
            variant="secondary"
            className="!px-3 !py-1"
            onClick={(e) => {
              e.stopPropagation();
              setActing(r as WorkflowTask);
            }}
          >
            Review
          </Button>
        ),
      });
    }
    return cols;
  }, [lane, queue.label]);

  const { columns, pref, save, restoreDefaults } = useColumnPrefs(
    `approvals_${kind}_${lane}`,
    catalog,
  );

  return (
    <div>
      <ListShell
        title={queue.label}
        subtitle={queue.subtitle}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <EditColumnsButton
              catalog={catalog}
              pref={pref}
              onApply={save}
              onRestore={restoreDefaults}
            />
            <ExportButton path={path} query={query} />
          </div>
        }
        pills={{
          showAll: false,
          options: [
            { id: "pending", label: "Pending", meaning: "Waiting for your decision." },
            { id: "approved", label: "Approved", meaning: "You agreed this document." },
            { id: "rejected", label: "Rejected", meaning: "You rejected this document." },
          ],
          value: lane,
          guideTitle: "Queue lane",
          onChange: (id) => {
            setLane(id as Lane);
            list.setPage(1);
          },
        }}
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Search document or summary"
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
          emptyMessage={
            lane === "pending"
              ? `Nothing waiting for you in ${queue.label.toLowerCase()}. New work appears after the previous step agrees.`
              : lane === "approved"
                ? `You have not approved any ${queue.label.toLowerCase()} in this view.`
                : `You have not rejected any ${queue.label.toLowerCase()} in this view.`
          }
          sort={list.sort}
          onSort={list.onSort}
          onRowClick={(r) => {
            if (lane === "pending") {
              setActing(r as WorkflowTask);
            } else setViewingId(r.instanceId);
          }}
        />
      </ListShell>

      {acting ? (
        <ApprovalActModal
          queueLabel={queue.label}
          task={acting}
          onClose={() => setActing(null)}
          onDone={() => {
            setActing(null);
            reload();
          }}
        />
      ) : null}
      {viewingId ? (
        <ApprovalViewModal
          queueLabel={queue.label}
          instanceId={viewingId}
          onClose={() => setViewingId(null)}
        />
      ) : null}
    </div>
  );
}
