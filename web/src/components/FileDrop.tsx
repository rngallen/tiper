"use client";

import { useState } from "react";
import { Upload } from "lucide-react";

/** Drag-and-drop file well used on ILR (same pattern as the payment portal). */
export function FileDrop({
  onFiles,
  disabled,
  label = "Drop files here or browse",
  hint = "PDF and images preview in the browser. Other types download only.",
  className = "",
}: {
  onFiles: (files: FileList) => void;
  disabled?: boolean;
  label?: string;
  hint?: string;
  className?: string;
}) {
  const [over, setOver] = useState(false);

  function take(files: FileList | null) {
    if (!files?.length || disabled) return;
    onFiles(files);
  }

  return (
    <label
      className={`flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-4 py-6 text-center transition ${
        over ? "border-brand-600 bg-brand-50" : "border-slate-300 bg-slate-50/80 hover:border-brand-400"
      } ${disabled ? "pointer-events-none opacity-50" : ""} ${className}`}
      onDragOver={(e) => {
        e.preventDefault();
        setOver(true);
      }}
      onDragLeave={() => setOver(false)}
      onDrop={(e) => {
        e.preventDefault();
        setOver(false);
        take(e.dataTransfer.files);
      }}
    >
      <Upload className="mb-2 h-6 w-6 text-slate-400" />
      <span className="text-sm font-medium text-slate-800">{label}</span>
      <span className="mt-1 text-xs text-slate-500">{hint}</span>
      <input
        type="file"
        multiple
        className="hidden"
        disabled={disabled}
        onChange={(e) => {
          take(e.target.files);
          e.target.value = "";
        }}
      />
    </label>
  );
}
