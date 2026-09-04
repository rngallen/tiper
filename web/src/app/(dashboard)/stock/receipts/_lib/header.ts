export type ReceiptKind = "internal" | "external";

export type ReceiptHeaderValues = {
  date: string;
  vesselDate: string;
  productId: string;
  productLabel: string;
  vesselId: string;
  vesselLabel: string;
  supplierId: string;
  supplierLabel: string;
  routeCode: string;
  tenderCode: string;
  procurementMethodCode: string;
  isProvision: boolean;
  density: string;
  notes: string;
  usesTiperPipeline: boolean;
};

export type ReceiptHeaderErrors = Partial<Record<keyof ReceiptHeaderValues, string>>;

export const emptyReceiptHeader = (): ReceiptHeaderValues => ({
  date: "",
  vesselDate: "",
  productId: "",
  productLabel: "",
  vesselId: "",
  vesselLabel: "",
  supplierId: "",
  supplierLabel: "",
  routeCode: "",
  tenderCode: "",
  procurementMethodCode: "",
  isProvision: true,
  density: "",
  notes: "",
  usesTiperPipeline: false,
});

export function validateReceiptHeader(
  form: ReceiptHeaderValues,
  kind: ReceiptKind,
): ReceiptHeaderErrors {
  const err: ReceiptHeaderErrors = {};
  if (!form.date) err.date = "Date is required";
  if (!form.vesselDate) err.vesselDate = "Vessel date is required";
  if (!form.vesselId) err.vesselId = "Vessel is required";
  if (!form.productId) err.productId = "Product is required";
  if (!form.routeCode) err.routeCode = "Discharge route is required";
  if (!form.notes.trim()) err.notes = "Notes are required";
  if (kind === "internal") {
    if (!form.supplierId) err.supplierId = "Supplier is required";
    if (!form.tenderCode) err.tenderCode = "Nature of tender is required";
    if (!form.procurementMethodCode) err.procurementMethodCode = "Method of tender is required";
    const densityErr = validateDensity(form.density);
    if (densityErr) err.density = densityErr;
  }
  return err;
}

export function validateDensity(raw: string): string {
  const s = raw.trim();
  if (!s) return "Density is required";
  const n = Number(s);
  if (!Number.isFinite(n)) return "Density must be a number";
  if (n <= 0) return "Density must be greater than zero";
  if (n >= 2) return "Enter density as MT per m³ (e.g. 0.84), not kg/m³";
  return "";
}

export function receiptHeaderPayload(form: ReceiptHeaderValues, kind: ReceiptKind) {
  const day = (d: string) => `${d}T00:00:00Z`;
  return {
    date: day(form.date),
    vesselDate: day(form.vesselDate),
    billingEffectiveDate: day(form.date),
    productId: form.productId,
    vesselId: form.vesselId,
    supplierId: kind === "internal" ? form.supplierId : "",
    routeCode: form.routeCode,
    tenderCode: kind === "internal" ? form.tenderCode : "",
    procurementMethodCode: kind === "internal" ? form.procurementMethodCode : "",
    receiptType: kind,
    isProvision: kind === "internal" ? form.isProvision : false,
    isFinal: kind === "internal" ? !form.isProvision : true,
    density: kind === "internal" ? form.density : "",
    notes: form.notes.trim(),
    usesTiperPipeline: kind === "external" ? form.usesTiperPipeline : false,
  };
}

export function firstHeaderError(errors: ReceiptHeaderErrors): string {
  const order: (keyof ReceiptHeaderValues)[] = [
    "date",
    "vesselDate",
    "vesselId",
    "productId",
    "routeCode",
    "supplierId",
    "tenderCode",
    "procurementMethodCode",
    "density",
    "notes",
  ];
  for (const key of order) {
    if (errors[key]) return errors[key] as string;
  }
  return "Complete all required header fields.";
}
