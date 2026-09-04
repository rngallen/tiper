import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { CheckCircle2, Download, RefreshCw, Search, Send } from "lucide-react";
import { http } from "@/shared/api/http";
import { toPage, DEFAULT_PAGE_SIZE } from "@/shared/api/list";
import { errorMessage } from "@/shared/api/errors";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { EmptyState } from "@/shared/ui/empty-state";
import { Skeleton } from "@/shared/ui/skeleton";
import { Badge } from "@/shared/ui/badge";
import { isSubmittableStatus, statusLabel, statusTone } from "@/shared/lib/status";
import { Cell, pickId } from "./cells";
import type { ResourceDef } from "./registry";

export function DocumentScreen({ resource, selectedId }: { resource: ResourceDef; selectedId?: string }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const query = { page, pageSize: DEFAULT_PAGE_SIZE, search: search || undefined, allDates: true, ...resource.extraQuery };

  const list = useQuery({
    queryKey: ["list", resource.api, query],
    queryFn: async () => toPage<Record<string, unknown>>(await http.get(resource.api, query)),
  });
  const detail = useQuery({
    queryKey: ["detail", resource.api, selectedId],
    enabled: Boolean(selectedId),
    queryFn: () => http.get<Record<string, unknown>>(`${resource.api}/${selectedId}`),
  });
  const submit = useMutation({
    mutationFn: (id: string) => {
      const path = resource.submitPath?.(id);
      if (!path) throw new Error("This document cannot be submitted from the console yet.");
      return http.post(path, {});
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["list", resource.api] });
      void qc.invalidateQueries({ queryKey: ["detail", resource.api, selectedId] });
    },
  });

  const items = list.data?.items ?? [];
  const pages = list.data?.totalPages ?? 1;
  const status = String(detail.data?.status ?? "");

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">{resource.title}</h2>
          {resource.subtitle ? <p className="text-sm text-muted-foreground">{resource.subtitle}</p> : null}
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute top-2.5 left-2.5 size-4 text-muted-foreground" />
            <Input className="w-56 pl-8" placeholder="Search documents…" value={search} onChange={(e) => { setSearch(e.target.value); setPage(1); }} />
          </div>
          <Button variant="outline" size="sm" onClick={() => list.refetch()} loading={list.isFetching}><RefreshCw className="size-3.5" />Refresh</Button>
          <Button variant="outline" size="sm" onClick={() => void http.download(resource.api, { ...query, export: true, page: undefined, pageSize: undefined })}>
            <Download className="size-3.5" />Excel
          </Button>
        </div>
      </div>
      <div className="grid gap-4 xl:grid-cols-[1fr_360px]">
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          {list.isLoading ? <div className="space-y-2 p-4">{Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} className="h-9 w-full" />)}</div>
          : list.isError ? <EmptyState title="Could not load documents" description={errorMessage(list.error)} />
          : items.length === 0 ? <EmptyState title="No documents" description="Nothing matches the current filters." />
          : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 text-left text-[11px] uppercase tracking-wider text-muted-foreground">
                  <tr>{resource.columns.map((c) => <th key={c.key} className="px-4 py-2.5 font-medium">{c.label}</th>)}</tr>
                </thead>
                <tbody>
                  {items.map((row) => {
                    const id = pickId(row);
                    return (
                      <tr key={id} className={`cursor-pointer border-t border-border hover:bg-muted/40 ${id === selectedId ? "bg-accent/60" : ""}`} onClick={() => id && navigate({ to: `${resource.path}/${id}` })}>
                        {resource.columns.map((c) => <td key={c.key} className="px-4 py-2.5"><Cell row={row} col={c} /></td>)}
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
          <div className="flex items-center justify-between border-t border-border px-4 py-2 text-xs text-muted-foreground">
            <span>{list.data?.itemsCount ?? 0} documents · page {page}/{pages || 1}</span>
            <div className="flex gap-1">
              <Button variant="outline" size="xs" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>Prev</Button>
              <Button variant="outline" size="xs" disabled={page >= pages} onClick={() => setPage((p) => p + 1)}>Next</Button>
            </div>
          </div>
        </div>
        <aside className="rounded-xl border border-border bg-card p-4">
          {!selectedId ? <EmptyState compact title="Select a document" description="Open a row to review or submit." />
          : detail.isLoading ? <Skeleton className="h-64 w-full" />
          : detail.isError ? <p className="text-sm text-destructive">{errorMessage(detail.error)}</p>
          : (
            <div className="space-y-3">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <p className="font-mono text-xs text-muted-foreground">{selectedId}</p>
                  <h3 className="text-base font-semibold">{String(detail.data?.documentNumber ?? detail.data?.code ?? resource.title)}</h3>
                </div>
                <Badge variant={statusTone(status)}>{statusLabel(status)}</Badge>
              </div>
              <dl className="grid grid-cols-2 gap-2 text-xs">
                {Object.entries(detail.data ?? {}).filter(([k, v]) => ["string", "number", "boolean"].includes(typeof v) && k !== "id").slice(0, 16).map(([k, v]) => (
                  <div key={k} className="rounded-md bg-muted/50 px-2 py-1.5">
                    <dt className="text-[10px] uppercase tracking-wide text-muted-foreground">{k}</dt>
                    <dd className="truncate font-medium">{String(v)}</dd>
                  </div>
                ))}
              </dl>
              <div className="flex flex-wrap gap-2 pt-2">
                {resource.submitPath && isSubmittableStatus(status) ? (
                  <Button size="sm" onClick={() => selectedId && submit.mutate(selectedId)} loading={submit.isPending}><Send className="size-3.5" />Submit</Button>
                ) : (
                  <Button size="sm" variant="secondary" disabled><CheckCircle2 className="size-3.5" />{status ? statusLabel(status) : "Locked"}</Button>
                )}
                <Button size="sm" variant="outline" onClick={() => navigate({ to: resource.path })}>Close</Button>
              </div>
              {submit.isError ? <p className="text-xs text-destructive">{errorMessage(submit.error)}</p> : null}
            </div>
          )}
        </aside>
      </div>
    </div>
  );
}
