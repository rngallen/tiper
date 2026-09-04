/** Mirrors pkg/types — keep string values in sync with the Go enums. */

export const OrderStatus = {
  Draft: "draft",
  Submitted: "submitted",
  Approved: "approved",
  Open: "open",
  Running: "running",
  InProgress: "inprogress",
  Loaded: "loaded",
  Closed: "closed",
  Completed: "completed",
  Rejected: "rejected",
  Cancelled: "cancelled",
} as const;

export type OrderStatus = (typeof OrderStatus)[keyof typeof OrderStatus];

export const AmendmentKind = {
  Normal: "normal",
  QtyIncrease: "qty_increase",
  QtyDecrease: "qty_decrease",
  Product: "product_change",
  Cancel: "cancel",
  BatchCancel: "batch_cancel",
  Extend: "extend",
} as const;

export type AmendmentKind = (typeof AmendmentKind)[keyof typeof AmendmentKind];

export const AMENDABLE_LINE_STATUSES: OrderStatus[] = [
  OrderStatus.Open,
  OrderStatus.Running,
  OrderStatus.InProgress,
];

export const StockStatusCode = {
  Local: "LOCAL",
  Transit: "TRANSIT",
  Mining: "MINING",
  Proration: "PRORATION",
} as const;
