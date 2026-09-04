"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, formatApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PRICE_WRITE_PERMS } from "@/lib/permissions";
import { EnterpriseTable, type EnterpriseColumn } from "@/components/EnterpriseTable";
import { ErpField, InquiryHeader, InquirySection, StatusPill } from "@/components/ApprovalReview";
import { Button, StatusBadge } from "@/components/ui";
import { DOC_STATUS_PILLS } from "@/components/StatusPills";
import { formatDate } from "@/lib/format";
import { qty } from "@/lib/master";
import { feeEditable, type PriceBatch } from "@/lib/billing";
import { creatorLabel } from "@/lib/creator";
import { CreateHeaderModal } from "./CreateHeaderModal";

const priceWrite = PRICE_WRITE_PERMS;

export function FeeBatchList({
  kind,
  title,
  subtitle,
  path,
  href,
  extra,
  withFx = true,
  requireMiLoss = false,
}: {
  kind: string;
  title: string;
  subtitle: string;
  path: string;
  href: string;
  extra?: Record<string, unknown>;
  withFx?: boolean;
  requireMiLoss?: boolean;
}) {
  const router = useRouter();
  const [tick, setTick] = useState(0);
  const [creating, setCreating] = useState(false);
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canWrite = hasPermission(...priceWrite);
  const printable = kind === "miloss";

  async function printPdf(id: string) {
    setMsg("");
    try {
      await api.download(`${path}/${id}/pdf`);
    } catch (e) {
      setMsg(formatApiError(e, "Could not print batch"));
    }
  }

  const columns: EnterpriseColumn<PriceBatch>[] = useMemo(
    () => [
      {
        id: "documentNumber",
        header: "Document",
        sortKey: "documentNumber",
        defaultVisible: true,
        cell: (r) => (
          <Link href={`${href}/${r.id}`} className="font-medium text-brand-800 hover:underline">
            {r.documentNumber}
          </Link>
        ),
      },
      {
        id: "date",
        header: "Date",
        sortKey: "date",
        defaultVisible: true,
        cell: (r) => formatDate(r.date),
      },
      {
        id: "effectiveFrom",
        header: "Effective",
        sortKey: "effectiveFrom",
        defaultVisible: true,
        cell: (r) => formatDate(r.effectiveFrom || r.date),
      },
      {
        id: "description",
        header: "Description",
        defaultVisible: true,
        cell: (r) => r.description || "—",
      },
      {
        id: "createdBy",
        header: "Created by",
        defaultVisible: true,
        exportValue: (r) => creatorLabel(r.createdBy),
        cell: (r) => creatorLabel(r.createdBy),
      },
      {
        id: "status",
        header: "Status",
        sortKey: "status",
        defaultVisible: true,
        cell: (r) => <StatusBadge status={r.status} />,
      },
      {
        id: "_actions",
        fixed: true,
        header: "",
        className: "text-right",
        cell: (r) => (
          <div className="flex justify-end gap-3">
            <Link
              href={`${href}/${r.id}`}
              className="text-sm font-semibold text-brand-800"
              onClick={(e) => e.stopPropagation()}
            >
              Open
            </Link>
            {canWrite && feeEditable(r.status) ? (
              <Link
                href={`${href}/${r.id}`}
                className="text-sm font-semibold text-brand-800"
                onClick={(e) => e.stopPropagation()}
              >
                Edit
              </Link>
            ) : null}
            {printable ? (
              <Button
                variant="secondary"
                className="!px-3 !py-1"
                onClick={(e) => {
                  e.stopPropagation();
                  void printPdf(r.id);
                }}
              >
                Print
              </Button>
            ) : null}
          </div>
        ),
      },
    ],
    [href, canWrite, printable, path],
  );

  return (
    <>
      <p className="mb-1 text-xs font-medium text-slate-500">Finance › Setups</p>
      {msg ? <p className="mb-2 text-sm text-slate-600">{msg}</p> : null}
      <EnterpriseTable
        tableId={`${kind}-fee-batches`}
        title={title}
        subtitle={subtitle}
        headerExtra={
          canWrite ? <Button onClick={() => setCreating(true)}>New batch</Button> : null
        }
        pills={{
          options: DOC_STATUS_PILLS,
          of: (r) => r.status,
          guideTitle: "Batch status",
          allTitle: "All batches",
        }}
        remote={{ path, dates: true, reloadKey: tick }}
        emptyMessage={`No ${title.toLowerCase()} yet. Create a draft batch, add lines, then submit for approval.`}
        columns={columns}
        searchPlaceholder="Document, description"
        rowSearch={(r) => `${r.documentNumber} ${r.status} ${r.description || ""} ${creatorLabel(r.createdBy)}`}
        defaultOrderBy="effectiveFrom"
        defaultSortDirection="DESC"
        dateOf={(r) => r.effectiveFrom || r.date}
        exportName={`${kind}_fees`}
        inquiryTitle={(r) => r.documentNumber}
        inquiry={(r) => (
          <div className="space-y-4">
            <InquiryHeader
              crumb={`Finance › Setups · ${title} · Inquiry`}
              title={r.documentNumber}
              pills={<StatusPill tone="navy">{r.status}</StatusPill>}
            />
            <InquirySection title="Header">
              <ErpField label="Date" value={formatDate(r.date)} />
              <ErpField label="Effective from" value={formatDate(r.effectiveFrom || r.date)} />
              <ErpField label="Description" value={r.description || "—"} />
              <ErpField label="Created by" value={creatorLabel(r.createdBy)} />
              {r.currencyCode ? <ErpField label="Home currency" value={r.currencyCode} /> : null}
              {r.exchangeRate ? <ErpField label="FX (TZS/USD)" value={qty(r.exchangeRate)} /> : null}
            </InquirySection>
            <div className="flex flex-wrap gap-2">
              <Link href={`${href}/${r.id}`}>
                <Button variant="secondary">Open</Button>
              </Link>
              {canWrite && feeEditable(r.status) ? (
                <Link href={`${href}/${r.id}`}>
                  <Button>Edit</Button>
                </Link>
              ) : null}
              {printable ? (
                <Button variant="secondary" onClick={() => void printPdf(r.id)}>
                  Print
                </Button>
              ) : null}
            </div>
          </div>
        )}
      />
      {creating ? (
        <CreateHeaderModal
          title={`New ${kind.toUpperCase()} batch`}
          path={path}
          extra={extra}
          withFx={withFx}
          requireMiLoss={requireMiLoss}
          onClose={() => setCreating(false)}
          onCreated={(id) => {
            setCreating(false);
            setTick((n) => n + 1);
            router.push(`${href}/${id}`);
          }}
        />
      ) : null}
    </>
  );
}
