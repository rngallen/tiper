"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Download } from "lucide-react";
import { Logo } from "@/components/Logo";
import { Button, ErrorBanner, Spinner } from "@/components/ui";

type PublicDoc = {
  kind?: string;
  label?: string;
  documentNumber?: string;
  status?: string;
  companyName?: string;
  customer?: string;
};

/**
 * Public document confirmation (no login).
 * URL: /verify/document/{kind}/{uid}/{sig} — sig is an HMAC issued by the API.
 */
export default function VerifyDocumentPage() {
  const params = useParams<{ kind: string; uid: string; sig: string }>();
  const kind = params?.kind || "";
  const uid = params?.uid || "";
  const sig = params?.sig || "";
  const [url, setUrl] = useState<string | null>(null);
  const [filename, setFilename] = useState("document.pdf");
  const [meta, setMeta] = useState<PublicDoc | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!kind || !uid || !sig) {
      setLoading(false);
      setError("Invalid confirmation link");
      return;
    }
    let revoked: string | null = null;
    let cancelled = false;
    setLoading(true);
    setError("");
    void (async () => {
      try {
        const base = `/api/v1/public/documents/${encodeURIComponent(kind)}/${encodeURIComponent(uid)}/${encodeURIComponent(sig)}`;
        const infoRes = await fetch(base, { credentials: "omit" });
        if (infoRes.ok) {
          const body = (await infoRes.json()) as { details?: PublicDoc };
          if (body.details) setMeta(body.details);
        }
        const res = await fetch(`${base}/pdf`, { credentials: "omit" });
        if (!res.ok) {
          throw new Error(
            res.status === 404
              ? "Document not found or link is invalid"
              : `Unable to load document (${res.status})`,
          );
        }
        const blob = await res.blob();
        const cd = res.headers.get("Content-Disposition") || "";
        const match = /filename\*?=(?:UTF-8''|")?([^\";]+)/i.exec(cd);
        if (match) setFilename(decodeURIComponent(match[1].replace(/"/g, "")));
        const objectUrl = URL.createObjectURL(blob);
        if (cancelled) {
          URL.revokeObjectURL(objectUrl);
          return;
        }
        revoked = objectUrl;
        setUrl(objectUrl);
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : "Unable to load document");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
      if (revoked) URL.revokeObjectURL(revoked);
    };
  }, [kind, uid, sig]);

  function download() {
    if (!url) return;
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
  }

  const company = meta?.companyName || "Depot Fuel Management System";
  const subtitle = [meta?.label, meta?.documentNumber].filter(Boolean).join(" · ") ||
    "Public document confirmation";

  return (
    <div className="min-h-screen bg-slate-100">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 py-3">
          <div className="flex items-center gap-3">
            <Logo height={36} />
            <div>
              <div className="text-sm font-semibold text-slate-900">{company}</div>
              <div className="text-xs text-slate-500">{subtitle}</div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              disabled={!url}
              onClick={download}
              className="!px-3 !py-1.5"
            >
              <Download className="mr-1.5 h-4 w-4" />
              Download
            </Button>
            <a
              href="/login"
              className="rounded-md bg-[#033860] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#022a48]"
            >
              Sign in
            </a>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-6">
        {error ? <ErrorBanner message={error} /> : null}
        {loading ? (
          <div className="flex h-80 items-center justify-center">
            <Spinner />
          </div>
        ) : url ? (
          <iframe
            title={meta?.label || "Document"}
            src={url}
            className="h-[80vh] w-full rounded-xl border border-slate-200 bg-white shadow-sm"
          />
        ) : null}
        <p className="mt-4 text-center text-xs text-slate-500">
          Signed confirmation link — no login required. Do not share unless
          intended for the recipient.
        </p>
      </main>
    </div>
  );
}
