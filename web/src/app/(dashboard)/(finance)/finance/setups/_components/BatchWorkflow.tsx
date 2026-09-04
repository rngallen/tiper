"use client";

import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ErrorBanner } from "@/components/ui";
import { WorkflowDiagram } from "@/components/WorkflowDiagram";
import type { WorkflowEvent } from "@/lib/types";

type Flow = {
  id?: string;
  no?: string;
  currentNodeName?: string;
  visitedNodeNames?: string[];
  process?: {
    process?: { name?: string; nodes?: { id: string; name: string; sequence?: number; status?: string }[] };
    diagram?: { from: string; to: string; actType: string; name?: string }[];
  };
  history?: WorkflowEvent[];
};

export function BatchWorkflow({ path }: { path: string }) {
  const { data, loading, error } = useAsync(() => api.get<Flow>(`${path}/workflow`), [path]);
  if (loading) return <p className="text-sm text-slate-500">Loading workflow…</p>;
  if (error) return <ErrorBanner message={error} />;
  const nodes = data?.process?.process?.nodes || [];
  const edges = data?.process?.diagram || [];
  if (!data?.id && !nodes.length) {
    return (
      <p className="text-sm text-slate-600">
        No approval has started. Submit the batch to open the workflow.
      </p>
    );
  }
  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-600">
        {data?.process?.process?.name || "Approval"}
        {data?.no ? ` · ${data.no}` : ""}
        {data?.currentNodeName ? ` · current step: ${data.currentNodeName}` : ""}
      </p>
      {nodes.length ? (
        <WorkflowDiagram
          nodes={nodes}
          edges={edges}
          currentNodeName={data?.currentNodeName}
          visitedNodeNames={data?.visitedNodeNames}
          title="Approval flow"
          heightClass="h-96"
        />
      ) : null}
      {data?.history?.length ? (
        <div>
          <h3 className="mb-2 text-sm font-semibold text-slate-900">Comments</h3>
          <ol className="space-y-2 text-sm text-slate-600">
            {data.history.map((ev, i) => (
              <li key={`${ev.createdAt}-${i}`} className="rounded-md border border-slate-200 px-3 py-2">
                <span className="font-semibold text-slate-800">{ev.actName || ev.actType}</span>
                {ev.userName ? ` · ${ev.userName}` : ""}
                {ev.createdAt ? (
                  <span className="ml-2 text-xs text-slate-400">{ev.createdAt}</span>
                ) : null}
                {ev.comment ? <p className="mt-1">{ev.comment}</p> : null}
              </li>
            ))}
          </ol>
        </div>
      ) : (
        <p className="text-sm text-slate-500">No comments yet.</p>
      )}
    </div>
  );
}
