"use client";

import { X } from "lucide-react";

export function Modal({
  open,
  onClose,
  title,
  children,
  footer,
  size = "md",
  layer = "base",
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  size?: "md" | "lg" | "xl" | "2xl";
  /** Nested confirms sit above an already-open modal. */
  layer?: "base" | "nested";
}) {
  if (!open) return null;
  const width =
    size === "2xl"
      ? "max-w-6xl"
      : size === "xl"
        ? "max-w-5xl"
        : size === "lg"
          ? "max-w-2xl"
          : "max-w-lg";
  const z = layer === "nested" ? "z-[80]" : "z-50";
  return (
    <div className={`fixed inset-0 ${z} flex items-center justify-center p-4 print:static print:inset-auto print:block print:p-0`}>
      <div
        className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm no-print"
        onClick={onClose}
      />
      <div
        className={`relative z-10 flex max-h-[90vh] w-full flex-col rounded-xl bg-white shadow-xl print:max-h-none print:w-full print:max-w-none print:rounded-none print:shadow-none ${width}`}
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slate-100 px-5 py-4 no-print">
          <h2 className="text-lg font-semibold text-slate-900">{title}</h2>
          <button
            onClick={onClose}
            className="rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
          >
            <X size={20} />
          </button>
        </div>
        <div className="overflow-y-auto px-5 py-4 print:overflow-visible print:p-0">
          {children}
        </div>
        {footer && (
          <div className="flex shrink-0 items-center justify-end gap-2 border-t border-slate-100 px-5 py-4 no-print">
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}
