"use client";

import { useState } from "react";
import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { useListQuery } from "@/lib/list";
import { asPagination } from "@/lib/pagination";
import type { Pagination } from "@/lib/types";
import { AttachmentActions } from "@/components/AttachmentPreview";
import { FileDrop } from "@/components/FileDrop";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { Paginator } from "@/components/Paginator";
import { StatusPills } from "@/components/StatusPills";
import { StatusPill } from "@/components/ApprovalReview";

type Attachment = {
  id: string;
  originalName: string;
  byteSize?: string;
  mime?: string;
  canPreview?: boolean;
  locked?: boolean;
  inUse?: boolean;
  isActive?: boolean;
  createdAt?: string;
};

export function PartyDocuments({
  path,
  readOnly = false,
  lifecycle = false,
  hint,
  deleteDetail,
}: {
  path: string;
  readOnly?: boolean;
  lifecycle?: boolean;
  hint?: string;
  deleteDetail?: string;
}) {
  const list = useListQuery({ orderBy: "createdAt", sortDirection: "DESC", pageSize: 10 });
  const [isActive, setIsActive] = useState(lifecycle ? "true" : "");
  const [saving, setSaving] = useState(false);
  const [pending, setPending] = useState<Attachment | null>(null);

  const docs = useAsync(
    () =>
      path
        ? api.get<Pagination<Attachment>>(`${path}/attachments`, {
            ...list.query,
            isActive: lifecycle ? isActive || undefined : undefined,
          })
        : Promise.resolve(null),
    [path, list.search, list.page, list.pageSize, list.orderBy, list.sortDirection, isActive, lifecycle],
  );
  const page = asPagination<Attachment>(docs.data);
  const files = page?.items ?? [];
  const cols = lifecycle ? 4 : 3;

  async function upload(listFiles: FileList) {
    setSaving(true);
    try {
      const fd = new FormData();
      Array.from(listFiles).forEach((f) => fd.append("files", f));
      await api.upload(`${path}/attachments`, fd);
      docs.reload();
    } catch {
      /* message box from api */
    } finally {
      setSaving(false);
    }
  }

  async function setActive(row: Attachment, active: boolean) {
    try {
      await api.patch(`${path}/attachments/${row.id}`, { isActive: active });
      docs.reload();
    } catch {
      /* message box from api */
    }
  }

  async function confirmRemove() {
    if (!pending) return;
    setSaving(true);
    try {
      await api.del(`${path}/attachments/${pending.id}`);
      setPending(null);
      docs.reload();
    } catch {
      /* message box from api */
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-3">
      {hint ? <p className="text-xs leading-5 text-slate-500">{hint}</p> : null}
      {docs.error ? <p className="text-sm text-red-700">{docs.error}</p> : null}
      {readOnly ? null : (
        <FileDrop
          disabled={saving}
          label={saving ? "Uploading…" : "Drop files here or browse"}
          onFiles={(listFiles) => void upload(listFiles)}
        />
      )}
      {lifecycle ? (
        <StatusPills
          options={[
            { id: "true", label: "Active", meaning: "Copied onto new ILR, pump-over, and ITT." },
            { id: "false", label: "Inactive", meaning: "Kept for history; not copied onto new work." },
          ]}
          value={isActive}
          onChange={(id) => {
            setIsActive(id);
            list.setPage(1);
          }}
          allTitle="All documents"
        />
      ) : null}
      <div className="overflow-hidden rounded-lg border border-slate-200">
        <div className="overflow-x-auto">
          <table className="pp-ledger">
            <thead>
              <tr>
                <th>File</th>
                <th>Size</th>
                {lifecycle ? <th>Status</th> : null}
                <th className="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {docs.loading ? (
                <tr>
                  <td className="!whitespace-normal py-8 text-center text-slate-500" colSpan={cols}>
                    Loading documents…
                  </td>
                </tr>
              ) : files.length === 0 ? (
                <tr>
                  <td className="!whitespace-normal py-8 text-center text-slate-500" colSpan={cols}>
                    No documents on file
                  </td>
                </tr>
              ) : (
                files.map((a) => (
                  <tr key={a.id}>
                    <td className="!whitespace-normal font-medium">{a.originalName}</td>
                    <td>{a.byteSize || "—"}</td>
                    {lifecycle ? (
                      <td>
                        <StatusPill tone={a.isActive === false ? "slate" : "navy"}>
                          {a.isActive === false ? "Inactive" : "Active"}
                        </StatusPill>
                      </td>
                    ) : null}
                    <td className="text-right">
                      <div className="flex justify-end gap-1">
                        <AttachmentActions
                          downloadPath={`${path}/attachments/${a.id}`}
                          originalName={a.originalName}
                          mime={a.mime}
                          canPreview={a.canPreview}
                        />
                        {readOnly ? null : (
                          <>
                            {lifecycle ? (
                              a.isActive === false ? (
                                <button
                                  type="button"
                                  className="px-2 text-xs font-semibold text-brand-800 hover:underline"
                                  onClick={() => void setActive(a, true)}
                                >
                                  Activate
                                </button>
                              ) : (
                                <button
                                  type="button"
                                  className="px-2 text-xs font-semibold text-amber-800 hover:underline"
                                  onClick={() => void setActive(a, false)}
                                >
                                  Deactivate
                                </button>
                              )
                            ) : null}
                            {a.inUse || a.locked ? null : (
                              <button
                                type="button"
                                className="px-2 text-xs font-semibold text-red-700 hover:underline"
                                onClick={() => setPending(a)}
                              >
                                Delete
                              </button>
                            )}
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
        <Paginator
          embedded
          data={page}
          page={list.page}
          onPage={list.setPage}
          pageSize={list.pageSize}
          onPageSize={list.setPageSize}
        />
      </div>
      <ConfirmDialog
        open={Boolean(pending)}
        title="Delete document?"
        message={
          pending
            ? `Remove ${pending.originalName}? This cannot be undone.`
            : "Remove this document?"
        }
        detail={deleteDetail}
        loading={saving}
        onConfirm={() => void confirmRemove()}
        onCancel={() => {
          if (!saving) setPending(null);
        }}
      />
    </div>
  );
}
