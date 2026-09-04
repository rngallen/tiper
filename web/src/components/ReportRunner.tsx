"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Download, ExternalLink, FileSpreadsheet, Printer } from "lucide-react";
import { api, ApiError, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { Modal } from "@/components/Modal";
import { Button, DateInput, ErrorBanner, Input, Select, Spinner } from "@/components/ui";
import type { ReportMeta } from "@/lib/types";
import {
  buildReportQuery,
  defaultReportParams,
  reportApiBase,
  reportCodeFromHref,
  reportParamsReady,
  type ReportParams,
} from "@/lib/reportRun";

type FilterOptions = {
  categories: { id: string; name: string }[];
  classes: string[];
};

const MONTHS = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

function yearOptions(): number[] {
  const y = new Date().getFullYear();
  return Array.from({ length: 8 }, (_, i) => y - 5 + i);
}

/**
 * Catalogue click → parameters → PDF preview → download / Excel / print.
 */
export function ReportRunner({
  href,
  title,
  description,
  onClose,
}: {
  href: string;
  title: string;
  description: string;
  onClose: () => void;
}) {
  const code = reportCodeFromHref(href);
  const apiBase = reportApiBase(href);
  const registry = useAsync(() => api.get<ReportMeta[]>("/reports/registry"), []);
  const meta = (registry.data || []).find((r) => r.code === code);
  const filters = meta?.filters || [];

  const [phase, setPhase] = useState<"params" | "preview">("params");
  const [params, setParams] = useState<ReportParams>(defaultReportParams);
  const [excelLoading, setExcelLoading] = useState(false);
  const [excelErr, setExcelErr] = useState("");
  const needOptions = filters.includes("category") || filters.includes("class");
  const options = useAsync(
    () =>
      needOptions
        ? api.get<FilterOptions>("/reports/filter-options")
        : Promise.resolve({ categories: [], classes: [] }),
    [needOptions],
  );

  const query = useMemo(
    () => buildReportQuery(filters, params, apiBase),
    [filters, params, apiBase],
  );
  const ready = reportParamsReady(filters, params);

  function patch(next: Partial<ReportParams>) {
    setParams((p) => ({ ...p, ...next }));
  }

  async function exportExcel() {
    setExcelErr("");
    setExcelLoading(true);
    try {
      await api.download(apiBase, {
        ...query,
        export: true,
        page: undefined,
        pageSize: undefined,
      });
    } catch (err) {
      setExcelErr(formatApiError(err, "Excel export failed"));
    } finally {
      setExcelLoading(false);
    }
  }

  return (
    <Modal
      open
      size={phase === "preview" ? "2xl" : "lg"}
      onClose={onClose}
      title={title}
      footer={
        phase === "params" ? (
          <>
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button
              onClick={() => setPhase("preview")}
              disabled={!ready || registry.loading}
            >
              Preview
            </Button>
          </>
        ) : (
          <>
            <Button variant="secondary" onClick={() => setPhase("params")}>
              Parameters
            </Button>
            <Button
              variant="secondary"
              onClick={() => void exportExcel()}
              loading={excelLoading}
            >
              <FileSpreadsheet className="h-4 w-4" />
              Excel
            </Button>
            <Button variant="secondary" onClick={onClose}>
              Close
            </Button>
          </>
        )
      }
    >
      {phase === "params" ? (
        <ParamsStep
          description={description || meta?.description || ""}
          filters={filters}
          loading={registry.loading || options.loading}
          error={registry.error || options.error}
          params={params}
          options={options.data}
          onChange={patch}
        />
      ) : (
        <PreviewStep
          title={title}
          path={`${apiBase}.pdf`}
          query={query}
          extraError={excelErr}
        />
      )}
    </Modal>
  );
}

function ParamsStep({
  description,
  filters,
  loading,
  error,
  params,
  options,
  onChange,
}: {
  description: string;
  filters: string[];
  loading: boolean;
  error?: string;
  params: ReportParams;
  options?: FilterOptions | null;
  onChange: (next: Partial<ReportParams>) => void;
}) {
  if (loading) {
    return <Spinner />;
  }
  return (
    <div className="space-y-4">
      {error ? <ErrorBanner message={error} /> : null}
      {description ? (
        <p className="text-sm leading-relaxed text-slate-600">{description}</p>
      ) : null}
      {filters.length === 0 ? (
        <p className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700">
          This report has no parameters. Click Preview to open the PDF.
        </p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {filters.includes("from") ? (
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-slate-900">
                From
              </span>
              <DateInput
                value={params.from}
                max={params.to}
                onChange={(from) => onChange({ from })}
              />
            </label>
          ) : null}
          {filters.includes("to") ? (
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-slate-900">
                To
              </span>
              <DateInput
                value={params.to}
                min={params.from}
                onChange={(to) => onChange({ to })}
              />
            </label>
          ) : null}
          {filters.includes("year") ? (
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-slate-900">
                Year
              </span>
              <Select
                value={params.year}
                onChange={(e) => onChange({ year: e.target.value })}
              >
                {yearOptions().map((y) => (
                  <option key={y} value={String(y)}>
                    {y}
                  </option>
                ))}
              </Select>
            </label>
          ) : null}
          {filters.includes("month") ? (
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-slate-900">
                Month
              </span>
              <Select
                value={params.month}
                onChange={(e) => onChange({ month: e.target.value })}
              >
                {MONTHS.map((name, i) => (
                  <option key={name} value={String(i + 1)}>
                    {name}
                  </option>
                ))}
              </Select>
            </label>
          ) : null}
          {filters.includes("date") ? (
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-slate-900">
                Date
              </span>
              <DateInput
                value={params.date}
                onChange={(date) => onChange({ date })}
              />
            </label>
          ) : null}
          {filters.includes("route") ? (
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-slate-900">
                Route
              </span>
              <Select
                value={params.route}
                onChange={(e) => onChange({ route: e.target.value })}
              >
                <option value="">SBM and KOJ</option>
                <option value="SBM">SBM — Single Buoy Mooring</option>
                <option value="KOJ">KOJ — Kurasini Oil Jetty</option>
              </Select>
            </label>
          ) : null}
          {filters.includes("customer") ? (
            <label className="block sm:col-span-2">
              <span className="mb-1.5 block text-sm font-medium text-slate-900">
                Customer code
              </span>
              <Input
                value={params.customer}
                onChange={(e) => onChange({ customer: e.target.value })}
                placeholder="Leave blank for all customers"
              />
            </label>
          ) : null}
          {filters.includes("id") ? (
            <label className="block sm:col-span-2">
              <span className="mb-1.5 block text-sm font-medium text-slate-900">
                Document UID or number
              </span>
              <Input
                value={params.id}
                onChange={(e) => onChange({ id: e.target.value })}
                placeholder="Required"
                autoFocus
              />
            </label>
          ) : null}
          {filters.includes("active") ? (
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-slate-900">
                Status
              </span>
              <Select
                value={params.active}
                onChange={(e) => onChange({ active: e.target.value })}
              >
                <option value="all">All</option>
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
              </Select>
            </label>
          ) : null}
          {filters.includes("category") ? (
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-slate-900">
                Category
              </span>
              <Select
                value={params.category}
                onChange={(e) => onChange({ category: e.target.value })}
              >
                <option value="">All categories</option>
                {(options?.categories || []).map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </Select>
            </label>
          ) : null}
          {filters.includes("class") ? (
            <fieldset className="block sm:col-span-2">
              <legend className="mb-1.5 block text-sm font-medium text-slate-900">
                License class
              </legend>
              <p className="mb-2 text-xs text-slate-500">
                Leave all unchecked to include every class. Select one or more to
                restrict the report.
              </p>
              <div className="max-h-48 overflow-auto rounded-lg border border-slate-200 bg-white p-2">
                {(options?.classes || []).length === 0 ? (
                  <p className="px-1 py-2 text-sm text-slate-500">
                    No classes on file.
                  </p>
                ) : (
                  <div className="grid gap-1 sm:grid-cols-2">
                    {(options?.classes || []).map((cls) => {
                      const checked = params.classes.includes(cls);
                      return (
                        <label
                          key={cls}
                          className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm text-slate-800 hover:bg-slate-50"
                        >
                          <input
                            type="checkbox"
                            className="h-4 w-4 rounded border-slate-300 text-slate-900"
                            checked={checked}
                            onChange={() =>
                              onChange({
                                classes: checked
                                  ? params.classes.filter((x) => x !== cls)
                                  : [...params.classes, cls],
                              })
                            }
                          />
                          {cls}
                        </label>
                      );
                    })}
                  </div>
                )}
              </div>
            </fieldset>
          ) : null}
        </div>
      )}
    </div>
  );
}

function PreviewStep({
  title,
  path,
  query,
  extraError,
}: {
  title: string;
  path: string;
  query: Record<string, string | number | boolean | undefined>;
  extraError: string;
}) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [downloading, setDownloading] = useState(false);
  const queryKey = useMemo(() => JSON.stringify(query), [query]);

  useEffect(() => {
    let revoked: string | null = null;
    let cancelled = false;
    setLoading(true);
    setError("");
    const parsed = JSON.parse(queryKey) as typeof query;
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
      setError(err instanceof ApiError ? err.message : "Failed to download PDF");
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
    <div className="space-y-3">
      {error ? <ErrorBanner message={error} /> : null}
      {extraError ? <ErrorBanner message={extraError} /> : null}
      {blobUrl ? (
        <div className="flex flex-wrap justify-end gap-2">
          <Button
            variant="secondary"
            onClick={() => window.open(blobUrl, "_blank", "noopener,noreferrer")}
          >
            <ExternalLink className="h-4 w-4" />
            Open
          </Button>
          <Button variant="secondary" onClick={print}>
            <Printer className="h-4 w-4" />
            Print
          </Button>
          <Button onClick={() => void download()} disabled={loading || downloading}>
            <Download className="h-4 w-4" />
            Download PDF
          </Button>
        </div>
      ) : null}
      {loading ? (
        <div className="flex h-64 items-center justify-center">
          <Spinner />
        </div>
      ) : blobUrl ? (
        <iframe
          ref={iframeRef}
          title={title}
          src={blobUrl}
          className="h-[70vh] w-full rounded-lg border border-slate-200 bg-slate-50"
        />
      ) : null}
    </div>
  );
}
