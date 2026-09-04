"use client";

import { useState } from "react";
import {
  CircleCheck,
  CircleMinus,
  Clock,
  Eye,
  EyeOff,
  Lock,
  LockOpen,
  XCircle,
} from "lucide-react";
import { titleCase } from "@/lib/format";

type Variant = "primary" | "secondary" | "danger" | "ghost";

const btnStyles: Record<Variant, string> = {
  primary:
    "bg-brand-800 text-white hover:bg-brand-900 focus:ring-brand-700 shadow-sm",
  secondary:
    "pp-btn-secondary bg-white text-slate-800 border border-slate-300 hover:bg-slate-50 focus:ring-brand-300",
  danger: "bg-red-600 text-white hover:bg-red-700 focus:ring-red-500",
  ghost: "text-slate-900 hover:bg-slate-100 focus:ring-slate-300",
};

export function Button({
  variant = "primary",
  className = "",
  loading,
  children,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant;
  loading?: boolean;
}) {
  return (
    <button
      {...props}
      disabled={props.disabled || loading}
      className={`inline-flex items-center justify-center gap-2 rounded-xl px-4 py-2.5 text-sm font-semibold transition focus:outline-none focus:ring-2 focus:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-60 ${btnStyles[variant]} ${className}`}
    >
      {loading && (
        <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
      )}
      {children}
    </button>
  );
}

export function Card({
  children,
  className = "",
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`pp-panel rounded-2xl border border-slate-200/90 bg-white shadow-sm ${className}`}
    >
      {children}
    </div>
  );
}

export function PageHeader({
  title,
  subtitle,
  actions,
}: {
  title: string;
  subtitle?: string;
  actions?: React.ReactNode;
}) {
  return (
    <div className="mb-8 flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{title}</h1>
        {subtitle && (
          <p className="mt-1.5 max-w-2xl text-sm font-normal text-slate-600">
            {subtitle}
          </p>
        )}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  );
}

/** Filled chip used by ModuleTabs (page navigation), not in-record sections. */
export function tabClass(active: boolean) {
  return `rounded-md px-3.5 py-2 text-sm font-semibold transition ${
    active
      ? "bg-brand-800 text-white shadow-sm"
      : "text-slate-600 hover:bg-slate-100 hover:text-brand-800"
  }`;
}

const statusColors: Record<string, string> = {
  outstanding: "bg-slate-100 text-slate-700 dark:bg-white/10 dark:text-slate-200",
  pending_approval:
    "bg-amber-50 text-amber-800 dark:bg-amber-500/15 dark:text-amber-300",
  returned: "bg-orange-50 text-orange-800 dark:bg-orange-500/15 dark:text-orange-300",
  approved: "bg-emerald-50 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300",
  rejected: "bg-red-50 text-red-800 dark:bg-red-500/15 dark:text-red-300",
  paid: "bg-emerald-50 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300",
  pending: "bg-amber-50 text-amber-800 dark:bg-amber-500/15 dark:text-amber-300",
  submitted: "bg-sky-50 text-sky-800 dark:bg-sky-500/15 dark:text-sky-300",
  sent: "bg-emerald-50 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300",
  successful: "bg-emerald-50 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300",
  failed: "bg-red-50 text-red-800 dark:bg-red-500/15 dark:text-red-300",
  posted: "bg-emerald-50 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300",
  running: "bg-sky-50 text-sky-800 dark:bg-sky-500/15 dark:text-sky-300",
  completed: "bg-emerald-50 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300",
  unapproved: "bg-slate-100 text-slate-700 dark:bg-white/10 dark:text-slate-200",
  inprogress: "bg-amber-50 text-amber-800 dark:bg-amber-500/15 dark:text-amber-300",
  in_progress: "bg-amber-50 text-amber-800 dark:bg-amber-500/15 dark:text-amber-300",
  open: "bg-sky-50 text-sky-800 dark:bg-sky-500/15 dark:text-sky-300",
  loaded: "bg-emerald-50 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300",
  closed: "bg-slate-100 text-slate-700 dark:bg-white/10 dark:text-slate-200",
  cancelled: "bg-slate-100 text-slate-700 dark:bg-white/10 dark:text-slate-200",
  draft: "bg-slate-100 text-slate-700 dark:bg-white/10 dark:text-slate-200",
};

function StatusIcon({ status }: { status: string }) {
  const cls = "h-3 w-3 shrink-0";
  switch (status) {
    case "approved":
    case "paid":
    case "sent":
    case "posted":
    case "completed":
    case "successful":
      return <CircleCheck className={cls} />;
    case "rejected":
    case "failed":
      return <XCircle className={cls} />;
    case "pending":
    case "pending_approval":
    case "inprogress":
    case "in_progress":
    case "open":
    case "returned":
    case "submitted":
    case "running":
      return <Clock className={cls} />;
    default:
      return null;
  }
}

export function StatusBadge({
  status,
  label,
}: {
  status: string;
  label?: string;
}) {
  const cls =
    statusColors[status] ||
    "bg-slate-100 text-slate-700 dark:bg-white/10 dark:text-slate-200";
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-semibold ${cls}`}
    >
      <StatusIcon status={status} />
      {label ?? titleCase(status)}
    </span>
  );
}

/** On/off switch for settings flags and multi-select options. */
export function ToggleField({
  label,
  description,
  checked,
  disabled,
  onChange,
  compact = false,
}: {
  label: string;
  description?: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
  /** Tighter row for dense lists (role permissions). */
  compact?: boolean;
}) {
  return (
    <div
      className={`flex items-start justify-between gap-3 rounded-lg border transition-colors ${
        compact ? "px-2.5 py-2" : "px-3 py-2.5"
      } ${
        checked
          ? "border-brand-200 bg-brand-50/50 dark:border-brand-400/40 dark:bg-brand-900/45"
          : "border-slate-200 bg-[var(--pp-bg)] dark:border-[var(--pp-border)]"
      } ${disabled ? "opacity-60" : ""}`}
    >
      <div className="min-w-0">
        <div
          className={`font-medium text-slate-900 dark:text-[var(--pp-text)] ${compact ? "text-[13px]" : "text-sm"}`}
        >
          {label}
        </div>
        {description ? (
          <p className="mt-0.5 text-xs leading-snug text-slate-500">
            {description}
          </p>
        ) : null}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`relative shrink-0 rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-brand-200 focus:ring-offset-1 disabled:cursor-not-allowed ${
          compact ? "mt-0.5 h-5 w-9" : "mt-0.5 h-6 w-11"
        } ${checked ? "bg-brand-800" : "bg-slate-300"}`}
      >
        <span
          className={`absolute top-0.5 left-0.5 rounded-full bg-white shadow transition-transform ${
            compact ? "h-4 w-4" : "h-5 w-5"
          } ${checked ? (compact ? "translate-x-4" : "translate-x-5") : "translate-x-0"}`}
        />
      </button>
    </div>
  );
}

/** Active / inactive — use on every list that has an enablement flag. */
export function ActiveIcon({
  active,
  className = "",
}: {
  active: boolean;
  className?: string;
}) {
  if (active) {
    return (
      <span
        className={`inline-flex text-emerald-600 ${className}`}
        title="Active"
        aria-label="Active"
      >
        <CircleCheck className="h-5 w-5" aria-hidden />
      </span>
    );
  }
  return (
    <span
      className={`inline-flex text-slate-400 ${className}`}
      title="Inactive"
      aria-label="Inactive"
    >
      <CircleMinus className="h-5 w-5" aria-hidden />
    </span>
  );
}

/** Locked / unlocked account — closed vs open padlock. */
export function LockedIcon({
  locked,
  className = "",
}: {
  locked: boolean;
  className?: string;
}) {
  if (locked) {
    return (
      <span
        className={`inline-flex text-red-600 ${className}`}
        title="Locked — unlock from Manage"
        aria-label="Locked"
      >
        <Lock className="h-5 w-5" aria-hidden />
      </span>
    );
  }
  return (
    <span
      className={`inline-flex text-emerald-600 ${className}`}
      title="Unlocked"
      aria-label="Unlocked"
    >
      <LockOpen className="h-5 w-5" aria-hidden />
    </span>
  );
}

export function Field({
  label,
  children,
  required,
  error,
  hint,
}: {
  label: string;
  children: React.ReactNode;
  required?: boolean;
  error?: string;
  hint?: string;
}) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-[13px] font-semibold leading-5 text-slate-900">
        {label}
        {required ? (
          <span className="ml-0.5 text-red-600" aria-hidden>
            *
          </span>
        ) : null}
      </span>
      {children}
      {error ? (
        <span className="mt-1 block text-xs font-medium text-red-600">{error}</span>
      ) : hint ? (
        <span className="mt-1 block text-xs font-medium text-slate-700">{hint}</span>
      ) : null}
    </label>
  );
}

function numberAllowsDecimal(step: unknown): boolean {
  if (step == null || step === "" || step === "any") return true;
  const n = Number(step);
  return Number.isFinite(n) && n % 1 !== 0;
}

function numberAllowsSigned(min: unknown): boolean {
  return min != null && min !== "" && Number(min) < 0;
}

/** Partial values while typing: "", "12", "12.", "0.5". Reject letters and extra dots. */
function isNumericDraft(value: string, decimal: boolean, signed: boolean): boolean {
  if (value === "" || (signed && value === "-")) return true;
  const body = signed && value.startsWith("-") ? value.slice(1) : value;
  if (signed && value.startsWith("-") && body === "") return true;
  if (decimal) return /^\d*\.?\d*$/.test(body);
  return /^\d+$/.test(body);
}

export function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  if (props.type !== "number") {
    return (
      <input
        {...props}
        className={`pp-control ${props.className || ""}`}
      />
    );
  }

  const decimal = numberAllowsDecimal(props.step);
  const signed = numberAllowsSigned(props.min);

  return (
    <input
      {...props}
      type="text"
      inputMode={props.inputMode ?? (decimal ? "decimal" : "numeric")}
      className={`pp-control ${props.className || ""}`}
      onKeyDown={(e) => {
        if (!e.ctrlKey && !e.metaKey && !e.altKey && e.key.length === 1) {
          const ok =
            (e.key >= "0" && e.key <= "9") ||
            (decimal && e.key === ".") ||
            (signed && e.key === "-");
          if (!ok) e.preventDefault();
        }
        props.onKeyDown?.(e);
      }}
      onPaste={(e) => {
        const text = e.clipboardData.getData("text");
        if (!isNumericDraft(text.trim(), decimal, signed)) {
          e.preventDefault();
        }
        props.onPaste?.(e);
      }}
      onChange={(e) => {
        if (!isNumericDraft(e.target.value, decimal, signed)) return;
        props.onChange?.(e);
      }}
    />
  );
}

/** Password field with show/hide toggle. Used on login and change-password. */
export function PasswordInput({
  className = "",
  ...props
}: Omit<React.InputHTMLAttributes<HTMLInputElement>, "type">) {
  const [show, setShow] = useState(false);
  return (
    <div className="relative">
      <Input
        {...props}
        type={show ? "text" : "password"}
        className={`pr-11 ${className}`}
      />
      <button
        type="button"
        onClick={() => setShow((v) => !v)}
        className="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-slate-500 hover:text-brand-800"
        aria-label={show ? "Hide password" : "Show password"}
      >
        {show ? (
          <EyeOff className="h-4 w-4" aria-hidden />
        ) : (
          <Eye className="h-4 w-4" aria-hidden />
        )}
      </button>
    </div>
  );
}

export { DateInput } from "./DateInput";

export function Select(props: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={`pp-control ${props.className || ""}`}
    />
  );
}

export function Textarea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      {...props}
      className={`pp-control min-h-[5.5rem] ${props.className || ""}`}
    />
  );
}

export function Empty({ message }: { message: string }) {
  return (
    <div className="py-12 text-center text-sm font-semibold text-slate-700">
      {message}
    </div>
  );
}

export function Spinner() {
  return (
    <div className="flex justify-center py-12">
      <span className="h-8 w-8 animate-spin rounded-full border-4 border-brand-200 border-t-brand-700" />
    </div>
  );
}

export function ErrorBanner({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="flex gap-3 overflow-hidden rounded-lg border border-red-200 bg-red-50 dark:border-red-900/60 dark:bg-red-950/35"
    >
      <div className="w-1 shrink-0 bg-red-700" />
      <p className="px-3 py-2.5 text-sm font-medium leading-6 text-red-950 dark:text-red-100">{message}</p>
    </div>
  );
}

