/** Download a table as an Excel-compatible SpreadsheetML workbook. */

function xmlEscape(value: unknown): string {
  return String(value ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function cell(value: string | number | boolean | null | undefined): string {
  if (typeof value === "number" && Number.isFinite(value)) {
    return `<Cell><Data ss:Type="Number">${value}</Data></Cell>`;
  }
  if (typeof value === "boolean") {
    return `<Cell><Data ss:Type="String">${value ? "Yes" : "No"}</Data></Cell>`;
  }
  return `<Cell><Data ss:Type="String">${xmlEscape(value)}</Data></Cell>`;
}

export function downloadExcel(
  filePrefix: string,
  headers: string[],
  rows: Array<Array<string | number | boolean | null | undefined>>,
) {
  const head = `<Row>${headers.map((h) => cell(h)).join("")}</Row>`;
  const body = rows.map((r) => `<Row>${r.map((c) => cell(c)).join("")}</Row>`).join("");
  const xml = `<?xml version="1.0"?>
<?mso-application progid="Excel.Sheet"?>
<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet">
 <Worksheet ss:Name="Export"><Table>
  ${head}
  ${body}
 </Table></Worksheet>
</Workbook>`;
  const blob = new Blob([xml], { type: "application/vnd.ms-excel" });
  const stamp = new Date()
    .toISOString()
    .replaceAll("-", "")
    .replaceAll(":", "")
    .replaceAll("T", "")
    .slice(0, 15);
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = `${filePrefix}_${stamp}.xls`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(a.href);
}
