/**
 * Stable content-type numbers from `pkg/types/contentType.go`. Never renumber.
 * Approval inboxes are filtered with `GET /workflow/tasks/mine?docContentType=`.
 */
export const CT = {
  Receipt: 51,
  FinancialHold: 53,
  IttTransfer: 55,
  Zerolization: 56,
  BillingProfile: 61, // FCF / fixed storage fee batch
  MiLossBatch: 62,
  VariableFeeBatch: 63,
  KojFeeBatch: 64,
  TbsFeeBatch: 65,
  BillingRun: 66,
  EwuraLicense: 68,
  ChangeOfService: 71,
  ExchangeRate: 78,
  GantryLoadingRequest: 80, // ILR
  GantryLoadingLine: 81, // ILO
  PumpOverRequest: 82,
  PumpOverReport: 83,
  Compartmentalization: 93,
  OrderAmendment: 96,
} as const;

export type ContentType = (typeof CT)[keyof typeof CT];

export interface ContentTypeMeta {
  /** URL slug used for approvals inbox and mail links. */
  slug: string;
  label: string;
  /** Workflow process code (`internal/workflow/seed.go`). */
  process: string;
  /** Route of the source document, given its id. */
  documentPath: (id: string) => string;
}

export const CONTENT_TYPES: Record<number, ContentTypeMeta> = {
  [CT.Receipt]: {
    slug: "receipts",
    label: "Vessel receipt",
    process: "RCPT",
    documentPath: (id) => `/stock/receipts/${id}`,
  },
  [CT.FinancialHold]: {
    slug: "financial-hold",
    label: "Financial hold release",
    process: "HOLDREL",
    documentPath: (id) => `/stock/financial-hold/${id}`,
  },
  [CT.IttTransfer]: {
    slug: "itt",
    label: "In-tank transfer",
    process: "ITT",
    documentPath: (id) => `/stock/itt/${id}`,
  },
  [CT.Zerolization]: {
    slug: "zerolization",
    label: "Zerolization",
    process: "ZEROL",
    documentPath: (id) => `/stock/zerolization/${id}`,
  },
  [CT.BillingProfile]: {
    slug: "fsf-fees",
    label: "Fixed storage fee",
    process: "FCFEE",
    documentPath: (id) => `/billing/setups/fsf/${id}`,
  },
  [CT.MiLossBatch]: {
    slug: "mi-loss",
    label: "MI loss",
    process: "MILOSS",
    documentPath: (id) => `/billing/setups/miloss/${id}`,
  },
  [CT.VariableFeeBatch]: {
    slug: "vsf-fees",
    label: "Variable storage fee",
    process: "VARFEE",
    documentPath: (id) => `/billing/setups/vsf/${id}`,
  },
  [CT.KojFeeBatch]: {
    slug: "koj-fees",
    label: "KOJ fee",
    process: "KOJFEE",
    documentPath: (id) => `/billing/setups/koj/${id}`,
  },
  [CT.TbsFeeBatch]: {
    slug: "tbs-fees",
    label: "TBS fee",
    process: "TBSFEE",
    documentPath: (id) => `/billing/setups/tbs/${id}`,
  },
  [CT.BillingRun]: {
    slug: "billing-runs",
    label: "Billing run",
    process: "FCFBILL",
    documentPath: (id) => `/billing/runs/${id}`,
  },
  [CT.ChangeOfService]: {
    slug: "change-of-service",
    label: "Change of service",
    process: "COS",
    documentPath: (id) => `/billing/change-of-service/${id}`,
  },
  [CT.ExchangeRate]: {
    slug: "fx-rates",
    label: "Exchange rate",
    process: "FXRATE",
    documentPath: (id) => `/billing/setups/fx/${id}`,
  },
  [CT.GantryLoadingRequest]: {
    slug: "gantry",
    label: "Internal loading request",
    process: "ILR",
    documentPath: (id) => `/stock/loading/${id}`,
  },
  [CT.PumpOverRequest]: {
    slug: "pump-over",
    label: "Pump-over request",
    process: "PUMP",
    documentPath: (id) => `/stock/pump-over/${id}`,
  },
  [CT.PumpOverReport]: {
    slug: "pump-over-reports",
    label: "Pump-over report",
    process: "PUMPRPT",
    documentPath: (id) => `/stock/pump-over-reports/${id}`,
  },
  [CT.Compartmentalization]: {
    slug: "compartmentalization",
    label: "Compartmentalization",
    process: "COMP",
    documentPath: (id) => `/stock/compartmentalization/${id}`,
  },
  [CT.OrderAmendment]: {
    slug: "amendments",
    label: "Order amendment",
    process: "AMEND",
    documentPath: (id) => `/stock/amendments/${id}`,
  },
};

export function contentTypeLabel(ct: number): string {
  return CONTENT_TYPES[ct]?.label ?? "Document";
}

export function contentTypeBySlug(slug: string): { ct: number; meta: ContentTypeMeta } | null {
  for (const [k, meta] of Object.entries(CONTENT_TYPES)) {
    if (meta.slug === slug) return { ct: Number(k), meta };
  }
  return null;
}
