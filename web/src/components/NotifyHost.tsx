"use client";

import { useEffect, useState } from "react";
import { Toaster } from "sonner";
import "sonner/dist/styles.css";
import { AlertTriangle, XCircle } from "lucide-react";
import { Button } from "@/components/ui";
import { useAppearance } from "@/lib/appearance";
import { DFMS_NOTIFY, DFMS_TOASTER_ID, type NotifyDialog } from "@/lib/notify";

/** Only one host may mount a toaster — Strict Mode / duplicate layouts stack otherwise. */
let hostCount = 0;

export function NotifyHost() {
  const { colorScheme } = useAppearance();
  const [primary, setPrimary] = useState(false);
  const [dialog, setDialog] = useState<NotifyDialog | null>(null);

  useEffect(() => {
    if (hostCount > 0) return;
    hostCount += 1;
    setPrimary(true);
    return () => {
      hostCount -= 1;
      setPrimary(false);
    };
  }, []);

  useEffect(() => {
    if (!primary) return;
    const onNotify = (e: Event) => {
      const payload = (e as CustomEvent<NotifyDialog>).detail;
      if (payload?.kind === "dialog") setDialog(payload);
    };
    window.addEventListener(DFMS_NOTIFY, onNotify);
    return () => window.removeEventListener(DFMS_NOTIFY, onNotify);
  }, [primary]);

  if (!primary) return null;

  return (
    <>
      <Toaster
        id={DFMS_TOASTER_ID}
        theme={colorScheme}
        position="top-right"
        closeButton
        duration={4000}
        visibleToasts={1}
        offset={{ top: 72, right: 20 }}
        gap={8}
        className="dfms-toaster print:hidden"
        toastOptions={{
          classNames: {
            toast: "dfms-toast",
            title: "dfms-toast-title",
            description: "dfms-toast-desc",
            closeButton: "dfms-toast-close",
          },
        }}
      />
      {dialog ? <MessageBox dialog={dialog} onClose={() => setDialog(null)} /> : null}
    </>
  );
}

function MessageBox({ dialog, onClose }: { dialog: NotifyDialog; onClose: () => void }) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" || e.key === "Enter") {
        e.preventDefault();
        onClose();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const error = dialog.tone === "error";

  return (
    <div
      className="fixed inset-0 z-[180] flex items-center justify-center p-4 print:hidden"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="dfms-msg-title"
      aria-describedby="dfms-msg-body"
    >
      <div className="absolute inset-0 bg-slate-950/50 backdrop-blur-[2px]" />
      <div className="relative z-10 w-full max-w-md overflow-hidden rounded-2xl border border-[var(--pp-border)] bg-[var(--pp-surface)] shadow-2xl">
        <div className={`h-1.5 ${error ? "bg-red-700" : "bg-amber-500"}`} />
        <div className="px-7 pb-5 pt-7">
          <div
            className={`flex h-12 w-12 items-center justify-center rounded-full ${
              error ? "bg-red-50 text-red-700 dark:bg-red-950/50 dark:text-red-300" : "bg-amber-50 text-amber-800 dark:bg-amber-950/40 dark:text-amber-200"
            }`}
          >
            {error ? <XCircle size={26} strokeWidth={2} /> : <AlertTriangle size={26} strokeWidth={2} />}
          </div>
          <h2 id="dfms-msg-title" className="mt-4 text-lg font-semibold tracking-tight text-[var(--pp-text)]">
            {dialog.title}
          </h2>
          <p id="dfms-msg-body" className="mt-2 text-sm leading-6 text-[var(--pp-muted)]">
            {dialog.message}
          </p>
          {dialog.detail ? (
            <p className="mt-3 rounded-lg border border-[var(--pp-border)] bg-[var(--pp-bg)] px-3 py-2 text-sm leading-6 text-[var(--pp-text)]">
              {dialog.detail}
            </p>
          ) : null}
        </div>
        <div className="flex items-center justify-end border-t border-[var(--pp-border)] bg-[var(--pp-bg)] px-6 py-3.5">
          <Button type="button" onClick={onClose} autoFocus>
            OK
          </Button>
        </div>
      </div>
    </div>
  );
}
