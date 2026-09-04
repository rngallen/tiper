"use client";

import { Modal } from "@/components/Modal";
import { Button } from "@/components/ui";

/** Enterprise confirm overlay. Replaces native window.confirm. */
export function ConfirmDialog({
  open,
  title,
  message,
  detail,
  confirmLabel = "Delete",
  cancelLabel = "Cancel",
  danger = true,
  loading,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  title: string;
  message: string;
  detail?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  if (!open) return null;
  return (
    <Modal
      open
      layer="nested"
      size="md"
      title={title}
      onClose={loading ? () => undefined : onCancel}
      footer={
        <>
          <Button variant="secondary" onClick={onCancel} disabled={loading}>
            {cancelLabel}
          </Button>
          <Button
            variant={danger ? "danger" : "primary"}
            loading={loading}
            onClick={onConfirm}
          >
            {confirmLabel}
          </Button>
        </>
      }
    >
      <p className="text-sm leading-6 text-slate-700">{message}</p>
      {detail ? (
        <p className="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm leading-6 text-amber-950">
          {detail}
        </p>
      ) : null}
    </Modal>
  );
}
