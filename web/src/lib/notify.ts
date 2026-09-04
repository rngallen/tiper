/** ERP feedback: one success/info toast at a time; errors are a blocking dialog. */

"use client";

import { toast } from "sonner";

export const DFMS_NOTIFY = "dfms:notify";
export const DFMS_TOASTER_ID = "dfms";

export type NotifyDialog = {
  kind: "dialog";
  tone: "error" | "warning";
  title: string;
  message: string;
  detail?: string;
};

const TOAST_MS = 4500;
const DETAIL_MS = 8000;
const BURST_MS = 2000;

const SLOT = {
  success: "dfms-success",
  info: "dfms-info",
  warning: "dfms-warning",
} as const;

type Tone = keyof typeof SLOT;

const lastBurst: Partial<Record<Tone, { text: string; at: number }>> = {};

const GENERIC_OK = new Set([
  "Saved",
  "Updated successfully",
  "Created successfully",
  "Deleted successfully",
]);

function coalesce(tone: Tone, text: string): string | null {
  const now = Date.now();
  const prev = lastBurst[tone];
  if (prev && now - prev.at < BURST_MS) {
    if (text === prev.text) return null;
    if (tone === "success" && GENERIC_OK.has(text) && !GENERIC_OK.has(prev.text)) {
      return null;
    }
  }
  lastBurst[tone] = { text, at: now };
  return text;
}

function splitUpdateMessage(message: string): { title: string; description?: string } {
  const sep = " — ";
  const i = message.indexOf(sep);
  if (i <= 0) return { title: message };
  const title = message.slice(0, i).trim();
  const description = message.slice(i + sep.length).trim();
  if (!title || !description) return { title: message };
  return { title, description };
}

function showToast(tone: Tone, message: string) {
  const text = coalesce(tone, message.trim());
  if (!text) return;
  const { title, description } = splitUpdateMessage(text);
  const opts = {
    id: SLOT[tone],
    toasterId: DFMS_TOASTER_ID,
    duration: description ? DETAIL_MS : TOAST_MS,
    description,
  };
  if (tone === "success") toast.success(title, opts);
  else if (tone === "warning") toast.warning(title, opts);
  else toast.info(title, opts);
}

function emitDialog(detail: NotifyDialog) {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent<NotifyDialog>(DFMS_NOTIFY, { detail }));
}

export const notify = {
  success(message: string) {
    showToast("success", message);
  },
  info(message: string) {
    showToast("info", message);
  },
  warning(message: string) {
    showToast("warning", message);
  },
  error(message: string, opts?: { title?: string; detail?: string }) {
    const text = message.trim();
    if (!text) return;
    emitDialog({
      kind: "dialog",
      tone: "error",
      title: opts?.title || "Unable to complete",
      message: text,
      detail: opts?.detail,
    });
  },
};
