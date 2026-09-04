"use client";

import { useState } from "react";
import { Download, FileText } from "lucide-react";
import { api, formatApiError } from "@/lib/api";
import { Button } from "@/components/ui";
import { ReportPdfModal } from "@/components/ReportPdfModal";

type Query = Record<string, string | number | boolean | undefined>;

/** Excel download + PDF preview/print/export for catalogue reports. */
export function ReportExportActions({
  excelPath,
  pdfPath,
  query,
  pdfTitle,
}: {
  excelPath: string;
  pdfPath: string;
  query?: Query;
  pdfTitle: string;
}) {
  const [excelLoading, setExcelLoading] = useState(false);
  const [excelErr, setExcelErr] = useState("");
  const [pdfOpen, setPdfOpen] = useState(false);

  async function exportExcel() {
    setExcelErr("");
    setExcelLoading(true);
    try {
      await api.download(excelPath, {
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
    <>
      <div className="flex flex-col items-end gap-1">
        <div className="flex flex-wrap items-center justify-end gap-2">
          <Button variant="secondary" onClick={() => setPdfOpen(true)}>
            <FileText className="h-4 w-4" />
            PDF
          </Button>
          <Button
            variant="secondary"
            onClick={() => void exportExcel()}
            loading={excelLoading}
          >
            <Download className="h-4 w-4" />
            Excel
          </Button>
        </div>
        {excelErr ? (
          <span className="text-xs font-medium text-red-700">{excelErr}</span>
        ) : null}
      </div>
      {pdfOpen ? (
        <ReportPdfModal
          title={pdfTitle}
          path={pdfPath}
          query={query}
          onClose={() => setPdfOpen(false)}
        />
      ) : null}
    </>
  );
}
