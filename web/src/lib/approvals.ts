import {
  FileText,
  Truck,
  CreditCard,
  ArrowLeftRight,
  Pencil,
  Layers,
  Droplets,
  Landmark,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { PERMISSIONS } from "@/lib/permissions";

/** DFMS document content types that have a workflow inbox. */
export const CONTENT_TYPE = {
  receipt: 51,
  hold: 53,
  itt: 55,
  zerolization: 56,
  fcfFee: 61,
  miLoss: 62,
  variableFee: 63,
  fxRate: 78,
  kojFee: 64,
  tbsFee: 65,
  billingRun: 66,
  gantryRequest: 80,
  pumpOverRequest: 82,
  pumpOverReport: 83,
  compartmentalization: 93,
  amendment: 96,
  changeOfService: 71,
} as const;

export type ApprovalQueueId =
  | "receipts"
  | "gantry"
  | "compartmentalization"
  | "amendments"
  | "pump-over"
  | "pump-over-reports"
  | "itt"
  | "zerolization"
  | "financial-hold"
  | "billing-runs"
  | "mi-loss"
  | "vsf-fees"
  | "koj-fees"
  | "tbs-fees"
  | "fsf-fees"
  | "fx-rates"
  | "change-of-service";

export type ApprovalQueue = {
  id: ApprovalQueueId;
  href: string;
  label: string;
  subtitle: string;
  icon: LucideIcon;
  /** Comma-separated content types for /workflow/tasks/mine. */
  types: string;
  perms: string[];
};

const tasks = [PERMISSIONS.workflowTasks];

/** One inbox per seeded process — do not group unrelated documents. */
export const APPROVAL_QUEUES: ApprovalQueue[] = [
  {
    id: "receipts",
    href: "/approvals/receipts",
    label: "Vessel receipts",
    subtitle: "Provision and final vessel receipts awaiting your step",
    icon: FileText,
    types: String(CONTENT_TYPE.receipt),
    perms: tasks,
  },
  {
    id: "gantry",
    href: "/approvals/gantry",
    label: "Internal loading",
    subtitle: "Internal loading requests (ILR) in your queue",
    icon: Truck,
    types: String(CONTENT_TYPE.gantryRequest),
    perms: tasks,
  },
  {
    id: "compartmentalization",
    href: "/approvals/compartmentalization",
    label: "Compartmentalization",
    subtitle: "Gantry dispatch files awaiting your step",
    icon: Layers,
    types: String(CONTENT_TYPE.compartmentalization),
    perms: tasks,
  },
  {
    id: "amendments",
    href: "/approvals/amendments",
    label: "Amendments",
    subtitle: "Quantity, product, and cancel amendments",
    icon: Pencil,
    types: String(CONTENT_TYPE.amendment),
    perms: tasks,
  },
  {
    id: "pump-over",
    href: "/approvals/pump-over",
    label: "Pump-over",
    subtitle: "Pipeline delivery requests awaiting your step",
    icon: Droplets,
    types: String(CONTENT_TYPE.pumpOverRequest),
    perms: tasks,
  },
  {
    id: "pump-over-reports",
    href: "/approvals/pump-over-reports",
    label: "Pump-over reports",
    subtitle: "Execution reports awaiting your step",
    icon: FileText,
    types: String(CONTENT_TYPE.pumpOverReport),
    perms: tasks,
  },
  {
    id: "itt",
    href: "/approvals/itt",
    label: "In-tank transfer",
    subtitle: "Ownership transfers between customers",
    icon: ArrowLeftRight,
    types: String(CONTENT_TYPE.itt),
    perms: tasks,
  },
  {
    id: "zerolization",
    href: "/approvals/zerolization",
    label: "Zerolization",
    subtitle: "Same-customer vessel consolidation",
    icon: ArrowLeftRight,
    types: String(CONTENT_TYPE.zerolization),
    perms: tasks,
  },
  {
    id: "financial-hold",
    href: "/approvals/financial-hold",
    label: "Financial hold",
    subtitle: "Hold releases awaiting your step",
    icon: Landmark,
    types: String(CONTENT_TYPE.hold),
    perms: tasks,
  },
  {
    id: "billing-runs",
    href: "/approvals/billing-runs",
    label: "Billing runs",
    subtitle: "Fixed storage (FSF) billing runs",
    icon: CreditCard,
    types: String(CONTENT_TYPE.billingRun),
    perms: tasks,
  },
  {
    id: "mi-loss",
    href: "/approvals/mi-loss",
    label: "MI loss",
    subtitle: "MI-loss rate batches",
    icon: CreditCard,
    types: String(CONTENT_TYPE.miLoss),
    perms: tasks,
  },
  {
    id: "fsf-fees",
    href: "/approvals/fsf-fees",
    label: "Fixed storage fees",
    subtitle: "Fixed storage price-list batches",
    icon: CreditCard,
    types: String(CONTENT_TYPE.fcfFee),
    perms: tasks,
  },
  {
    id: "fx-rates",
    href: "/approvals/fx-rates",
    label: "Exchange rates",
    subtitle: "TZS per USD quotes awaiting approval",
    icon: CreditCard,
    types: String(CONTENT_TYPE.fxRate),
    perms: tasks,
  },
  {
    id: "vsf-fees",
    href: "/approvals/vsf-fees",
    label: "Variable storage fees",
    subtitle: "EWURA-linked variable storage fee cards",
    icon: CreditCard,
    types: String(CONTENT_TYPE.variableFee),
    perms: tasks,
  },
  {
    id: "koj-fees",
    href: "/approvals/koj-fees",
    label: "KOJ fees",
    subtitle: "KOJ price-list batches",
    icon: CreditCard,
    types: String(CONTENT_TYPE.kojFee),
    perms: tasks,
  },
  {
    id: "tbs-fees",
    href: "/approvals/tbs-fees",
    label: "TBS fees",
    subtitle: "TBS truck-loading price lists",
    icon: CreditCard,
    types: String(CONTENT_TYPE.tbsFee),
    perms: tasks,
  },
  {
    id: "change-of-service",
    href: "/approvals/change-of-service",
    label: "Change of service",
    subtitle: "Customer billing-type switches",
    icon: CreditCard,
    types: String(CONTENT_TYPE.changeOfService),
    perms: tasks,
  },
];

export function approvalQueue(id: ApprovalQueueId): ApprovalQueue {
  return APPROVAL_QUEUES.find((q) => q.id === id) ?? APPROVAL_QUEUES[0];
}

const ULID = /^[0-9A-HJKMNP-TV-Z]{26}$/i;

function looksLikeUlid(value?: string): boolean {
  return Boolean(value && ULID.test(value.trim()));
}

/** Inbox queue for a workflow document content type. */
export function approvalQueueForType(ct?: number): ApprovalQueue | undefined {
  if (!ct) return undefined;
  return APPROVAL_QUEUES.find((q) =>
    q.types
      .split(",")
      .map((n) => Number(n.trim()))
      .includes(ct),
  );
}

export function approvalQueueHref(ct?: number): string {
  return approvalQueueForType(ct)?.href ?? "/approvals";
}

/** Operator-facing document ref — never a raw ULID. */
export function approvalDocumentLabel(row: {
  no?: string;
  documentNumber?: string;
  summary?: string;
  docContentType?: number;
}): string {
  for (const raw of [row.documentNumber, row.no, row.summary]) {
    const value = (raw || "").trim();
    if (value && !looksLikeUlid(value)) return value;
  }
  return approvalQueueForType(row.docContentType)?.label ?? "Document";
}

/** Open the source document from an approval task when the UID is known. */
export function approvalDocumentHref(ct?: number, docId?: string): string | null {
  if (!ct || !docId) return null;
  switch (ct) {
    case CONTENT_TYPE.receipt:
      return `/stock/receipts/${docId}/edit`;
    case CONTENT_TYPE.itt:
      return `/stock/itt/${docId}`;
    case CONTENT_TYPE.gantryRequest:
      return `/stock/loading/${docId}/edit`;
    case CONTENT_TYPE.compartmentalization:
      return `/stock/compartmentalization/${docId}`;
    case CONTENT_TYPE.pumpOverRequest:
      return `/stock/pump-over/${docId}`;
    case CONTENT_TYPE.pumpOverReport:
      return `/stock/pump-over-reports/${docId}`;
    case CONTENT_TYPE.fcfFee:
      return `/finance/setups/fsf/${docId}`;
    case CONTENT_TYPE.variableFee:
      return `/finance/setups/vsf/${docId}`;
    case CONTENT_TYPE.kojFee:
      return `/finance/setups/koj/${docId}`;
    case CONTENT_TYPE.tbsFee:
      return `/finance/setups/tbs/${docId}`;
    case CONTENT_TYPE.fxRate:
      return `/finance/setups/fx/${docId}`;
    case CONTENT_TYPE.miLoss:
      return `/finance/setups/miloss/${docId}`;
    case CONTENT_TYPE.billingRun:
      return `/billing/${docId}`;
    case CONTENT_TYPE.zerolization:
      return `/stock/zerolization/${docId}`;
    case CONTENT_TYPE.hold:
      return `/stock/financial-hold/${docId}`;
    case CONTENT_TYPE.amendment:
      return `/stock/amendments/${docId}`;
    case CONTENT_TYPE.changeOfService:
      return `/billing/change-of-service/${docId}`;
    default:
      return null;
  }
}

/** API path for document attachments shown in the approval review modal. */
export function approvalAttachPath(ct?: number, docId?: string): string | null {
  if (!ct || !docId) return null;
  switch (ct) {
    case CONTENT_TYPE.receipt:
      return `/ic/receipts/${docId}`;
    case CONTENT_TYPE.itt:
      return `/ic/itt/${docId}`;
    case CONTENT_TYPE.gantryRequest:
      return `/orders/loading/${docId}`;
    case CONTENT_TYPE.compartmentalization:
      return `/orders/compartmentalizations/${docId}`;
    case CONTENT_TYPE.pumpOverRequest:
      return `/orders/pump-over/${docId}`;
    case CONTENT_TYPE.pumpOverReport:
      return `/orders/pump-over-reports/${docId}`;
    case CONTENT_TYPE.amendment:
      return `/orders/amendments/${docId}`;
    case CONTENT_TYPE.zerolization:
      return `/ic/zerolization/${docId}`;
    case CONTENT_TYPE.hold:
      return `/ic/hold-releases/${docId}`;
    case CONTENT_TYPE.changeOfService:
      return `/billing/change-of-service/${docId}`;
    case CONTENT_TYPE.fxRate:
      return `/billing/fx/${docId}`;
    case CONTENT_TYPE.fcfFee:
      return `/billing/fsf/batches/${docId}`;
    case CONTENT_TYPE.variableFee:
      return `/billing/vsf/batches/${docId}`;
    case CONTENT_TYPE.kojFee:
      return `/billing/koj/batches/${docId}`;
    case CONTENT_TYPE.tbsFee:
      return `/billing/tbs/batches/${docId}`;
    case CONTENT_TYPE.miLoss:
      return `/billing/miloss/batches/${docId}`;
    case CONTENT_TYPE.billingRun:
      return `/billing/runs/${docId}`;
    default:
      return null;
  }
}

/** GET path for in-modal document facts (same resource as attachments). */
export function approvalApiPath(ct?: number, docId?: string): string | null {
  return approvalAttachPath(ct, docId);
}

export function approvalAttachLockedCaption(ct?: number): string {
  if (
    ct === CONTENT_TYPE.gantryRequest ||
    ct === CONTENT_TYPE.pumpOverRequest ||
    ct === CONTENT_TYPE.itt
  ) {
    return "From customer";
  }
  return "Copied";
}
