"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { QtyInput } from "@/components/QtyInput";
import {
  DocumentMeta,
  ErpField,
  InquirySection,
  StatusPill,
} from "@/components/ApprovalReview";
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { SectionRow, SectionTable } from "@/components/SectionTable";
import { Button, Card, ErrorBanner, Input, PageHeader, Spinner } from "@/components/ui";

type Cell = {
  id: string;
  tankPlate: string;
  index: number;
  capacity: string;
  quantity: string;
  topSeal?: string;
  productCode?: string;
  productName?: string;
};
type Comp = {
  id: string;
  documentNumber: string;
  status: string;
  almaFileName?: string;
  horsePlate?: string;
  trailerOnePlate?: string;
  ilo?: { documentNumber?: string };
  lines?: Cell[];
};

export default function CompartmentalizationDocumentPage() {
  const { id } = useParams<{ id: string }>();
  const loaded = useAsync(() => api.get<Comp>(`/orders/compartmentalizations/${id}`), [id]);
  const [row, setRow] = useState<Comp | null>(null);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const { hasPermission } = useAuth();
  const canUpdate = hasPermission(PERMISSIONS.compartmentalizationUpdate);
  const canSubmit = hasPermission(PERMISSIONS.compartmentalizationSubmit);
  const doc = row || loaded.data;

  useEffect(() => {
    if (loaded.data) setRow(loaded.data);
  }, [loaded.data]);

  async function saveLines() {
    if (!doc) return;
    setError("");
    setBusy("save");
    try {
      await api.post(`/orders/compartmentalizations/${doc.id}/lines`, {
        lines: (doc.lines || []).map((ln) => ({
          id: ln.id,
          quantity: ln.quantity,
          topSeal: ln.topSeal,
        })),
      });
      setMsg("Compartments saved");
    } catch (e) {
      setError(formatApiError(e, "Could not save"));
    } finally {
      setBusy("");
    }
  }

  async function submit() {
    if (!doc) return;
    setError("");
    setBusy("submit");
    try {
      await api.post(`/orders/compartmentalizations/${doc.id}/submit`, {});
      setMsg("Submitted — after approval DFMS writes the SAP3C file ALMA reads");
      loaded.reload();
      setRow(null);
    } catch (e) {
      setError(formatApiError(e, "Could not submit"));
    } finally {
      setBusy("");
    }
  }

  if (loaded.loading && !doc) return <Spinner />;
  if (!doc) {
    return (
      <div className="space-y-4">
        <ErrorBanner message={loaded.error || "Document not found"} />
        <Link href="/stock/compartmentalization" className="text-sm font-medium text-brand-800">
          Back to list
        </Link>
      </div>
    );
  }

  const draft = doc.status === "draft" || doc.status === "running";
  const canAttach = doc.status === "draft" || doc.status === "returned";
  const cells = doc.lines || [];

  return (
    <div className="space-y-6">
      <PageHeader
        title={doc.documentNumber}
        subtitle="Compartmentalization"
        actions={
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => void api.download("/reports/delivery-note.pdf", { id: doc.id })}>
              Print PDF
            </Button>
            <Link
              href="/stock/compartmentalization"
              className="self-center text-sm font-medium text-slate-600 hover:text-slate-900"
            >
              Back to list
            </Link>
          </div>
        }
      />
      {msg ? <p className="text-sm text-slate-600">{msg}</p> : null}
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="space-y-4 p-5">
        <DocumentMeta crumb="Stock · Gantry · Compartmentalization">
          <StatusPill tone="navy">{doc.status}</StatusPill>
        </DocumentMeta>
        <InquirySection title="Dispatch" columns={3}>
          <ErpField label="GLO" value={doc.ilo?.documentNumber || "—"} />
          <ErpField label="Horse" value={doc.horsePlate || "—"} />
          <ErpField label="Trailer" value={doc.trailerOnePlate || "—"} />
          <ErpField label="ALMA file" value={doc.almaFileName || "—"} />
        </InquirySection>
      </Card>
      <DocumentAttachmentsCard
        path={`/orders/compartmentalizations/${id}`}
        draft={canAttach}
        caption="Weighbridge slips and gantry photos. Drop on the left — the list stays on the right."
      />
      <Card className="space-y-3 p-5">
        <SectionTable
          title="Tank cells"
          columns={[
            { label: "Cell" },
            { label: "Capacity" },
            { label: "Litres" },
            { label: "Top seal" },
            { label: "Product" },
          ]}
          empty="No tank cells"
        >
          {cells.map((ln, i) => (
            <SectionRow key={ln.id}>
              <td className="font-semibold">
                {ln.tankPlate} / {ln.index}
              </td>
              <td className="tabular-nums">{ln.capacity}</td>
              <td>
                <QtyInput
                  value={ln.quantity}
                  disabled={!draft}
                  onChange={(quantity) => {
                    setRow({
                      ...doc,
                      lines: cells.map((c, j) => (j === i ? { ...c, quantity } : c)),
                    });
                  }}
                />
              </td>
              <td>
                <Input
                  value={ln.topSeal || ""}
                  disabled={!draft}
                  onChange={(e) => {
                    const topSeal = e.target.value;
                    setRow({
                      ...doc,
                      lines: cells.map((c, j) => (j === i ? { ...c, topSeal } : c)),
                    });
                  }}
                />
              </td>
              <td>{ln.productCode || ln.productName || "—"}</td>
            </SectionRow>
          ))}
        </SectionTable>
        <div className="flex flex-wrap gap-2">
          {canUpdate && draft ? (
            <Button onClick={() => void saveLines()} loading={busy === "save"}>
              Save cells
            </Button>
          ) : null}
          {canSubmit && draft ? (
            <Button variant="secondary" onClick={() => void submit()} loading={busy === "submit"}>
              Submit
            </Button>
          ) : null}
        </div>
      </Card>
    </div>
  );
}
