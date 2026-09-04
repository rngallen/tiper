"use client";

import { useEffect, useState } from "react";
import { Download, Eye } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { Modal } from "@/components/Modal";
import { Button, ErrorBanner, Spinner } from "@/components/ui";

function isPreviewable(mime?: string, name?: string): boolean {
  const m = (mime || "").toLowerCase();
  const n = (name || "").toLowerCase();
  if (m.startsWith("image/") && m !== "image/svg+xml") return true;
  if (/\.(png|jpe?g|gif|webp|bmp|pdf)$/i.test(n)) return true;
  return false;
}

function isPdf(mime?: string, name?: string): boolean {
  const m = (mime || "").toLowerCase();
  const n = (name || "").toLowerCase();
  return m === "application/pdf" || n.endsWith(".pdf");
}

/** Preview PDF/images inline; other types download only. */
export function AttachmentActions({
  downloadPath,
  originalName,
  mime,
  canPreview: canPreviewFlag,
}: {
  downloadPath: string;
  originalName: string;
  mime?: string;
  canPreview?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const canPreview =
    canPreviewFlag !== undefined
      ? canPreviewFlag
      : isPreviewable(mime, originalName);

  return (
    <>
      <div className="flex shrink-0 gap-1">
        {canPreview ? (
          <Button
            variant="ghost"
            className="!px-2 !py-1"
            title="Preview"
            onClick={() => setOpen(true)}
          >
            <Eye className="h-4 w-4" />
            <span className="ml-1 hidden sm:inline">Preview</span>
          </Button>
        ) : null}
        <Button
          variant="ghost"
          className="!px-2 !py-1"
          title="Download"
          onClick={() => void api.download(downloadPath)}
        >
          <Download className="h-4 w-4" />
        </Button>
      </div>
      {open ? (
        <AttachmentPreviewModal
          downloadPath={downloadPath}
          originalName={originalName}
          mime={mime}
          onClose={() => setOpen(false)}
        />
      ) : null}
    </>
  );
}

function AttachmentPreviewModal({
  downloadPath,
  originalName,
  mime,
  onClose,
}: {
  downloadPath: string;
  originalName: string;
  mime?: string;
  onClose: () => void;
}) {
  const [url, setUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let revoked: string | null = null;
    let cancelled = false;
    setLoading(true);
    void api
      .fetchBlob(downloadPath)
      .then((r) => {
        if (cancelled) {
          URL.revokeObjectURL(r.url);
          return;
        }
        revoked = r.url;
        setUrl(r.url);
      })
      .catch((err) => {
        if (!cancelled) {
          setError(
            err instanceof ApiError ? err.message : "Failed to load file",
          );
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
      if (revoked) URL.revokeObjectURL(revoked);
    };
  }, [downloadPath]);

  const pdf = isPdf(mime, originalName);

  return (
    <Modal
      open
      size="2xl"
      onClose={onClose}
      title={originalName}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
          <Button onClick={() => void api.download(downloadPath)}>
            <Download className="mr-1.5 h-4 w-4" />
            Download
          </Button>
        </>
      }
    >
      {error ? <ErrorBanner message={error} /> : null}
      {loading ? (
        <div className="flex h-64 items-center justify-center">
          <Spinner />
        </div>
      ) : url ? (
        pdf ? (
          <iframe
            title={originalName}
            src={url}
            className="h-[75vh] w-full rounded-lg border border-slate-200 bg-slate-50"
          />
        ) : (
          <div className="flex max-h-[75vh] items-center justify-center overflow-auto rounded-lg border border-slate-200 bg-slate-50 p-4">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={url}
              alt={originalName}
              className="max-h-[70vh] max-w-full object-contain"
            />
          </div>
        )
      ) : null}
    </Modal>
  );
}
