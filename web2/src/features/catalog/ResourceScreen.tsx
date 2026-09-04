import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Download, Plus, RefreshCw, Search, Trash2 } from "lucide-react";
import { http } from "@/shared/api/http";
import { toPage, DEFAULT_PAGE_SIZE, type ListQuery } from "@/shared/api/list";
import { errorMessage } from "@/shared/api/errors";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Dialog, DialogContent, DialogTitle } from "@/shared/ui/dialog";
import { EmptyState } from "@/shared/ui/empty-state";
import { Skeleton } from "@/shared/ui/skeleton";
import { useSession } from "@/features/auth/session";
import { Cell, pickId } from "./cells";
import type { FieldDef, ResourceDef } from "./registry";

export function ResourceScreen({ resource, selectedId }: { resource: ResourceDef; selectedId?: string }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const can = useSession((s) => s.can);
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [editing, setEditing] = useState<Record<string, unknown> | null>(null);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState<Record<string, unknown>>({});
  const [err, setErr] = useState("");

  const query: ListQuery = {
    page,
    pageSize: DEFAULT_PAGE_SIZE,
    search: search || undefined,
    allDates: resource.dated ? true : undefined,
    ...resource.extraQuery,
  };

  const list = useQuery({
    queryKey: ["list", resource.api, query],
    queryFn: async () => toPage<Record<string, unknown>>(await http.get(resource.api, query)),
  });

  const detail = useQuery({
    queryKey: ["detail", resource.api, selectedId],
    enabled: Boolean(selectedId),
    queryFn: () => http.get<Record<string, unknown>>(`${resource.api}/${selectedId}`),
  });

  const save = useMutation({
    mutationFn: async () => {
      const body: Record<string, unknown> = {};
      for (const f of resource.fields ?? []) {
        const v = form[f.key];
        if (v === "" || v === undefined) continue;
        body[f.key] = v;
      }
      if (editing?.id) return http.put(`${resource.api}/${editing.id}`, body);
      return http.post(resource.api, body);
    },
    onSuccess: async () => {
      setCreating(false);
      setEditing(null);
      setErr("");
      await qc.invalidateQueries({ queryKey: ["list", resource.api] });
    },
    onError: (e) => setErr(errorMessage(e)),
  });

  const remove = useMutation({
    mutationFn: (id: string) => http.delete(`${resource.api}/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["list", resource.api] }),
  });

  const items = list.data?.items ?? [];
  const pages = list.data?.totalPages ?? 1;
  const canCreate = can(...(resource.createPerms ?? resource.perms));
  const canUpdate = can(...(resource.updatePerms ?? resource.perms));
  const canDelete = can(...(resource.deletePerms ?? []));

  function openCreate() {
    const initial: Record<string, unknown> = {};
    for (const f of resource.fields ?? []) initial[f.key] = f.kind === "checkbox" ? true : "";
    setForm(initial);
    setEditing(null);
    setCreating(true);
    setErr("");
  }

  function openEdit(row: Record<string, unknown>) {
    setForm({ ...row });
    setEditing(row);
    setCreating(true);
    setErr("");
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">{resource.title}</h2>
          {resource.subtitle ? <p className="text-sm text-muted-foreground">{resource.subtitle}</p> : null}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute top-2.5 left-2.5 size-4 text-muted-foreground" />
            <Input className="w-56 pl-8" placeholder="Search…" value={search} onChange={(e) => { setSearch(e.target.value); setPage(1); }} />
          </div>
          <Button variant="outline" size="sm" onClick={() => list.refetch()} loading={list.isFetching}>
            <RefreshCw className="size-3.5" /> Refresh
          </Button>
          <Button variant="outline" size="sm" onClick={() => void http.download(resource.api, { ...query, export: true, page: undefined, pageSize: undefined })}>
            <Download className="size-3.5" /> Excel
          </Button>
          {canCreate && resource.fields?.length ? (
            <Button size="sm" onClick={openCreate}><Plus className="size-3.5" />{resource.createLabel ?? "New"}</Button>
          ) : null}
        </div>
      </div>

      <div className="overflow-hidden rounded-xl border border-border bg-card">
        {list.isLoading ? (
          <div className="space-y-2 p-4">{Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-9 w-full" />)}</div>
        ) : list.isError ? (
          <EmptyState title="Could not load" description={errorMessage(list.error)} />
        ) : items.length === 0 ? (
          <EmptyState title="Nothing here yet" description="Create a record or widen the search." />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-left text-[11px] uppercase tracking-wider text-muted-foreground">
                <tr>
                  {resource.columns.map((c) => <th key={c.key} className="px-4 py-2.5 font-medium">{c.label}</th>)}
                  <th className="px-4 py-2.5" />
                </tr>
              </thead>
              <tbody>
                {items.map((row) => {
                  const id = pickId(row);
                  return (
                    <tr key={id || JSON.stringify(row).slice(0, 32)} className="cursor-pointer border-t border-border hover:bg-muted/40" onClick={() => id && navigate({ to: `${resource.path}/${id}` })}>
                      {resource.columns.map((c) => <td key={c.key} className="px-4 py-2.5"><Cell row={row} col={c} /></td>)}
                      <td className="px-4 py-2 text-right" onClick={(e) => e.stopPropagation()}>
                        <div className="flex justify-end gap-1">
                          {canUpdate && resource.fields?.length ? <Button variant="ghost" size="xs" onClick={() => openEdit(row)}>Edit</Button> : null}
                          {canDelete && id ? <Button variant="ghost" size="icon-xs" onClick={() => confirm("Delete this record?") && remove.mutate(id)}><Trash2 className="size-3.5 text-destructive" /></Button> : null}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex items-center justify-between border-t border-border px-4 py-2 text-xs text-muted-foreground">
          <span>{list.data?.itemsCount ?? 0} rows · page {page} of {pages || 1}</span>
          <div className="flex gap-1">
            <Button variant="outline" size="xs" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>Prev</Button>
            <Button variant="outline" size="xs" disabled={page >= pages} onClick={() => setPage((p) => p + 1)}>Next</Button>
          </div>
        </div>
      </div>

      {selectedId ? (
        <section className="rounded-xl border border-border bg-card p-5">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="font-semibold">Record {selectedId}</h3>
            <Button variant="ghost" size="sm" onClick={() => navigate({ to: resource.path })}>Close</Button>
          </div>
          {detail.isLoading ? <Skeleton className="h-40 w-full" /> : detail.isError ? (
            <p className="text-sm text-destructive">{errorMessage(detail.error)}</p>
          ) : (
            <pre className="max-h-[420px] overflow-auto rounded-md bg-muted/50 p-3 font-mono text-xs leading-5">{JSON.stringify(detail.data, null, 2)}</pre>
          )}
        </section>
      ) : null}

      <Dialog open={creating} onOpenChange={setCreating}>
        <DialogContent size="lg">
          <DialogTitle>{editing ? "Edit record" : "New record"}</DialogTitle>
          <form className="mt-3 grid gap-3 sm:grid-cols-2" onSubmit={(e) => { e.preventDefault(); save.mutate(); }}>
            {(resource.fields ?? []).map((f) => (
              <FieldInput key={f.key} field={f} value={form[f.key]} onChange={(v) => setForm((s) => ({ ...s, [f.key]: v }))} />
            ))}
            {err ? <p className="sm:col-span-2 text-sm text-destructive">{err}</p> : null}
            <div className="sm:col-span-2 flex justify-end gap-2 pt-2">
              <Button type="button" variant="outline" onClick={() => setCreating(false)}>Cancel</Button>
              <Button type="submit" loading={save.isPending}>Save</Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function FieldInput({ field, value, onChange }: { field: FieldDef; value: unknown; onChange: (v: unknown) => void }) {
  const kind = field.kind ?? "text";
  if (kind === "checkbox") {
    return <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={Boolean(value)} onChange={(e) => onChange(e.target.checked)} />{field.label}</label>;
  }
  if (kind === "textarea") {
    return (
      <label className="sm:col-span-2 block text-sm">
        <span className="mb-1 block text-xs font-medium text-muted-foreground">{field.label}</span>
        <textarea className="min-h-20 w-full rounded-md border border-input bg-background px-3 py-2 text-sm" value={String(value ?? "")} placeholder={field.placeholder} onChange={(e) => onChange(e.target.value)} />
      </label>
    );
  }
  return (
    <label className="block text-sm">
      <span className="mb-1 block text-xs font-medium text-muted-foreground">{field.label}{field.required ? " *" : ""}</span>
      <Input
        type={kind === "number" ? "number" : kind === "date" ? "date" : kind === "email" ? "email" : kind === "tel" ? "tel" : "text"}
        required={field.required}
        placeholder={field.placeholder}
        value={value == null ? "" : String(value)}
        onChange={(e) => onChange(kind === "number" ? e.target.valueAsNumber : e.target.value)}
      />
    </label>
  );
}
