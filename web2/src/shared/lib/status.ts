/** Document status vocabulary shared by every workflow-backed document. */
export type DocumentStatus =
  | "draft"
  | "submitted"
  | "approved"
  | "returned"
  | "rejected"
  | "posted"
  | "cancelled"
  | "expired"
  | "completed"
  | "closed"
  | "pending"
  | "running"
  | "provision"
  | "final"
  | string;

export type StatusTone = "muted" | "info" | "success" | "warning" | "destructive" | "secondary";

export function statusTone(status: string | null | undefined): StatusTone {
  switch ((status ?? "").toLowerCase()) {
    case "approved":
    case "completed":
    case "posted":
    case "final":
    case "active":
    case "effective":
    case "agree":
      return "success";
    case "submitted":
    case "running":
    case "pending":
    case "in_progress":
    case "provision":
      return "info";
    case "returned":
    case "expiring":
    case "on_hold":
    case "hold":
    case "warning":
      return "warning";
    case "rejected":
    case "cancelled":
    case "canceled":
    case "expired":
    case "inactive":
    case "locked":
    case "failed":
    case "reject":
      return "destructive";
    case "draft":
      return "secondary";
    default:
      return "muted";
  }
}

export function statusLabel(status: string | null | undefined): string {
  if (!status) return "—";
  return status.replace(/[_-]+/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

/** Editable = originator may still change header/lines/attachments. */
export function isEditableStatus(status: string | null | undefined): boolean {
  const s = (status ?? "").toLowerCase();
  return s === "draft" || s === "returned";
}

export function isSubmittableStatus(status: string | null | undefined): boolean {
  return isEditableStatus(status);
}
