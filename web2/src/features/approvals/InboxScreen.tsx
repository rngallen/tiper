import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { http } from "@/shared/api/http";
import { toPage } from "@/shared/api/list";
import { errorMessage } from "@/shared/api/errors";
import { CONTENT_TYPES } from "@/shared/config/content-types";
import { Button } from "@/shared/ui/button";
import { EmptyState } from "@/shared/ui/empty-state";
import { Input } from "@/shared/ui/input";
import { Badge } from "@/shared/ui/badge";
import { fmtDateTime } from "@/shared/lib/format";

interface Task {
  taskId: string;
  instanceId: string;
  no: string;
  summary?: string;
  nodeName?: string;
  docContentType: number;
  createdAt?: string;
  docId?: string;
  documentNumber?: string;
}

export function InboxScreen() {
  const qc = useQueryClient();
  const [comment, setComment] = useState("");
  const [active, setActive] = useState<Task | null>(null);
  const list = useQuery({
    queryKey: ["inbox"],
    queryFn: async () => toPage<Task>(await http.get("/workflow/tasks/mine", { page: 1, pageSize: 50 })),
  });
  const act = useMutation({
    mutationFn: (body: { instanceId: string; action: "agree" | "back" | "reject"; comment?: string }) =>
      http.post(`/workflow/instances/${body.instanceId}/act`, {
        action: body.action,
        comment: body.comment,
        totalRejection: body.action === "reject",
      }),
    onSuccess: () => {
      setActive(null);
      setComment("");
      void qc.invalidateQueries({ queryKey: ["inbox"] });
    },
  });
  const items = list.data?.items ?? [];

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">Approvals inbox</h2>
        <p className="text-sm text-muted-foreground">Documents waiting on your workflow step</p>
      </div>
      <div className="grid gap-4 xl:grid-cols-[1fr_320px]">
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          {items.length === 0 && !list.isLoading ? (
            <EmptyState title="Inbox zero" description="Nothing is waiting on you." />
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-left text-[11px] uppercase tracking-wider text-muted-foreground">
                <tr>
                  <th className="px-4 py-2.5">Document</th>
                  <th className="px-4 py-2.5">Type</th>
                  <th className="px-4 py-2.5">Step</th>
                  <th className="px-4 py-2.5">Requested</th>
                </tr>
              </thead>
              <tbody>
                {items.map((t) => (
                  <tr key={t.taskId} className={`cursor-pointer border-t border-border hover:bg-muted/40 ${active?.taskId === t.taskId ? "bg-accent/60" : ""}`} onClick={() => setActive(t)}>
                    <td className="px-4 py-2.5">
                      <div className="font-mono text-xs">{t.documentNumber ?? t.no}</div>
                      {t.summary ? <div className="text-xs text-muted-foreground">{t.summary}</div> : null}
                    </td>
                    <td className="px-4 py-2.5">{CONTENT_TYPES[t.docContentType]?.label ?? "Document"}</td>
                    <td className="px-4 py-2.5">{t.nodeName ?? "—"}</td>
                    <td className="px-4 py-2.5 tabular-nums">{fmtDateTime(t.createdAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
        <aside className="rounded-xl border border-border bg-card p-4">
          {!active ? (
            <EmptyState compact title="Pick a task" description="Agree, return or reject with a comment." />
          ) : (
            <div className="space-y-3">
              <Badge variant="info">{CONTENT_TYPES[active.docContentType]?.label ?? "Task"}</Badge>
              <h3 className="font-semibold">{active.documentNumber ?? active.no}</h3>
              <p className="text-sm text-muted-foreground">{active.nodeName}</p>
              {active.docId && CONTENT_TYPES[active.docContentType] ? (
                <Link to={CONTENT_TYPES[active.docContentType].documentPath(active.docId)} className="block text-sm text-primary hover:underline">
                  Open source document
                </Link>
              ) : null}
              <Input placeholder="Comment (required to return / reject)" value={comment} onChange={(e) => setComment(e.target.value)} />
              <div className="flex flex-wrap gap-2">
                <Button size="sm" onClick={() => act.mutate({ instanceId: active.instanceId, action: "agree", comment })} loading={act.isPending}>Agree</Button>
                <Button size="sm" variant="warning" onClick={() => act.mutate({ instanceId: active.instanceId, action: "back", comment })} loading={act.isPending}>Return</Button>
                <Button size="sm" variant="destructive" onClick={() => act.mutate({ instanceId: active.instanceId, action: "reject", comment })} loading={act.isPending}>Reject</Button>
              </div>
              {act.isError ? <p className="text-xs text-destructive">{errorMessage(act.error)}</p> : null}
            </div>
          )}
        </aside>
      </div>
    </div>
  );
}
