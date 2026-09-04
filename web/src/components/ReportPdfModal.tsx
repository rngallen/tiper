"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Download, ExternalLink, Printer } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { Modal } from "@/components/Modal";
import { Button, ErrorBanner, Spinner } from "@/components/ui";

type Query = Record<string, string | number | boolean | undefined>;

/** Preview / print / download a server-generated report PDF. */
export function ReportPdfModal({
  title,
  path,
  query,
  onClose,
}: {
  title: string;
  path: string;
  query?: Query;
  onClose: () => void;
}) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [downloading, setDownloading] = useState(false);
  const queryKey = useMemo(() => JSON.stringify(query ?? {}), [query]);

  useEffect(() => {
    let revoked: string | null = null;
    let cancelled = false;
    setLoading(true);
    setError("");
    const parsed = JSON.parse(queryKey) as Query;
    void api
      .fetchBlob(path, parsed)
      .then(({ url }) => {
        if (cancelled) {
          URL.revokeObjectURL(url);
          return;
        }
        revoked = url;
        setBlobUrl(url);
      })
      .catch((err) => {
        if (!cancelled) {
          setError(
            err instanceof ApiError ? err.message : "Failed to load PDF report",
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
  }, [path, queryKey]);

  async function download() {
    setDownloading(true);
    try {
      await api.download(path, query);
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Failed to download PDF",
      );
    } finally {
      setDownloading(false);
    }
  }

  function print() {
    const win = iframeRef.current?.contentWindow;
    if (win) {
      win.focus();
      win.print();
      return;
    }
    if (blobUrl) {
      const w = window.open(blobUrl, "_blank", "noopener,noreferrer");
      w?.addEventListener("load", () => w.print());
    }
  }

  return (
    <Modal
      open
      size="2xl"
      onClose={onClose}
      title={title}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
          {blobUrl ? (
            <>
              <Button
                variant="secondary"
                onClick={() =>
                  window.open(blobUrl, "_blank", "noopener,noreferrer")
                }
              >
                <ExternalLink className="mr-1.5 h-4 w-4" />
                Open
              </Button>
              <Button variant="secondary" onClick={print}>
                <Printer className="mr-1.5 h-4 w-4" />
                Print
              </Button>
            </>
          ) : null}
          <Button
            onClick={() => void download()}
            disabled={loading || downloading}
          >
            <Download className="mr-1.5 h-4 w-4" />
            Download PDF
          </Button>
        </>
      }
    >
      {error ? <ErrorBanner message={error} /> : null}
      {loading ? (
        <div className="flex h-64 items-center justify-center">
          <Spinner />
        </div>
      ) : blobUrl ? (
        <iframe
          ref={iframeRef}
          title={title}
          src={blobUrl}
          className="h-[75vh] w-full rounded-lg border border-slate-200 bg-slate-50"
        />
      ) : null}
    </Modal>
  );
}
