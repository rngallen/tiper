"use client";

import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { asRows } from "@/lib/pagination";
import { AttachmentActions } from "@/components/AttachmentPreview";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { FileDrop } from "@/components/FileDrop";
import { Button, Card } from "@/components/ui";
import { Paginator } from "@/components/Paginator";
import type { Pagination } from "@/lib/types";

export type DocumentFile = {
  id: string;
  originalName: string;
  byteSize?: string;
  mime?: string;
  canPreview?: boolean;
  locked?: boolean;
};

function fileKind(name: string, mime?: string): string {
  const ext = name.includes(".") ? name.split(".").pop()?.toUpperCase() : "";
  if (ext) return ext;
  if ((mime || "").toLowerCase() === "application/pdf") return "PDF";
  if ((mime || "").toLowerCase().startsWith("image/")) return "IMAGE";
  return "—";
}

export function DocumentFiles({
  path,
  draft,
  hint = "Supporting files for this document (surveys, letters, rate sheets).",
  lockedCaption = "Copied",
}: {
  path: string;
  draft: boolean;
  hint?: string;
  /** Kept for callers; the register is always a drop well + ledger table. */
  layout?: "split" | "stack";
  lockedCaption?: string;
}) {
  const docs = useAsync(() => api.get<DocumentFile[] | Pagination<DocumentFile>>(`${path}/attachments`), [path]);
  const all = useMemo(() => asRows<DocumentFile>(docs.data), [docs.data]);
  const [pending, setPending] = useState<DocumentFile | null>(null);
  const [removing, setRemoving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const totalPages = all.length > 0 ? Math.ceil(all.length / pageSize) : 0;
  useEffect(() => {
    if (totalPages > 0 && page > totalPages) setPage(totalPages);
  }, [page, totalPages]);

  const files = useMemo(() => {
    const start = (page - 1) * pageSize;
    return all.slice(start, start + pageSize);
  }, [all, page, pageSize]);

  const pager: Pagination<DocumentFile> = {
    items: files,
    page,
    pageSize,
    itemsCount: all.length,
    totalPages,
  };

  async function upload(list: FileList | null) {
    if (!list?.length || !draft || uploading) return;
    const fd = new FormData();
    Array.from(list).forEach((f) => fd.append("files", f));
    setUploading(true);
    try {
      await api.upload(`${path}/attachments`, fd);
      docs.reload();
    } catch {
      /* message box from api */
    } finally {
      setUploading(false);
    }
  }

  async function confirmRemove() {
    if (!pending) return;
    setRemoving(true);
    try {
      await api.del(`${path}/attachments/${pending.id}`);
      setPending(null);
      docs.reload();
    } catch {
      /* message box from api */
    } finally {
      setRemoving(false);
    }
  }

  const emptyCopy = docs.loading && all.length === 0
    ? "Loading files…"
    : draft
      ? "No files yet. Drop them above or browse."
      : "No attachments";

  return (
    <div className="space-y-3">
      {hint ? <p className="text-sm text-slate-600">{hint}</p> : null}
      {draft ? (
        <FileDrop
          disabled={uploading}
          label={uploading ? "Uploading…" : "Drop files here or browse"}
          onFiles={(list) => void upload(list)}
        />
      ) : null}
      <p className="text-xs text-slate-500">
        {all.length === 0
          ? "None attached"
          : `${all.length} file${all.length === 1 ? "" : "s"}`}
        {uploading ? " · Uploading…" : null}
      </p>
      <div className="overflow-hidden rounded-lg border border-slate-200">
        <div className="overflow-x-auto">
          <table className="pp-ledger">
            <thead>
              <tr>
                <th>File</th>
                <th>Type</th>
                <th>Size</th>
                <th className="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {files.length === 0 ? (
                <tr>
                  <td className="!whitespace-normal py-10 text-center text-slate-500" colSpan={4}>
                    {emptyCopy}
                  </td>
                </tr>
              ) : (
                files.map((a) => (
                  <tr key={a.id}>
                    <td className="!whitespace-normal font-medium">
                      {a.originalName}
                      {a.locked ? (
                        <span className="ml-2 text-xs font-normal text-slate-500">{lockedCaption}</span>
                      ) : null}
                    </td>
                    <td>{fileKind(a.originalName, a.mime)}</td>
                    <td>{a.byteSize || "—"}</td>
                    <td className="text-right">
                      <div className="flex justify-end gap-1">
                        <AttachmentActions
                          downloadPath={`${path}/attachments/${a.id}`}
                          originalName={a.originalName}
                          mime={a.mime}
                          canPreview={a.canPreview}
                        />
                        {draft && !a.locked ? (
                          <Button variant="ghost" className="!px-2 !py-1" onClick={() => setPending(a)}>
                            Remove
                          </Button>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
        {all.length > 0 ? (
          <Paginator
            embedded
            data={pager}
            page={page}
            onPage={setPage}
            pageSize={pageSize}
            onPageSize={(size) => {
              setPageSize(size);
              setPage(1);
            }}
          />
        ) : null}
      </div>
      <ConfirmDialog
        open={Boolean(pending)}
        title="Remove file?"
        message={
          pending ? `Remove ${pending.originalName} from this document?` : "Remove this file?"
        }
        loading={removing}
        confirmLabel="Remove"
        onConfirm={() => void confirmRemove()}
        onCancel={() => {
          if (!removing) setPending(null);
        }}
      />
    </div>
  );
}

/** Document-page attachments: drop well + paginated ledger table. */
export function DocumentAttachmentsCard({
  path,
  draft,
  caption,
  lockedCaption,
}: {
  path: string;
  draft: boolean;
  caption: string;
  lockedCaption?: string;
}) {
  return (
    <Card className="space-y-3 p-5">
      <div>
        <h2 className="text-sm font-semibold text-slate-900">Attachments</h2>
        <p className="text-xs text-slate-500">{caption}</p>
      </div>
      <DocumentFiles path={path} draft={draft} hint="" lockedCaption={lockedCaption} />
    </Card>
  );
}
