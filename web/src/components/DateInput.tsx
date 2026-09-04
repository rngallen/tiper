"use client";

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Calendar, ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from "lucide-react";
import { isoToDisplayDate, localISODate, parseTypedDate } from "@/lib/format";

const WEEKDAYS = ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"];
const MONTHS = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

const CAL_W = 280;
const CAL_H = 320;

function parseIso(iso: string): { y: number; m: number; d: number } | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  if (!m) return null;
  return { y: Number(m[1]), m: Number(m[2]), d: Number(m[3]) };
}

function inRange(iso: string, min?: string, max?: string): boolean {
  if (min && iso < min) return false;
  if (max && iso > max) return false;
  return true;
}

function placePopover(anchor: DOMRect): { top: number; left: number } {
  const pad = 8;
  const gap = 4;
  let top = anchor.bottom + gap;
  if (top + CAL_H > window.innerHeight - pad) {
    top = Math.max(pad, anchor.top - CAL_H - gap);
  }
  let left = anchor.left;
  if (left + CAL_W > window.innerWidth - pad) {
    left = anchor.right - CAL_W;
  }
  left = Math.min(Math.max(pad, left), window.innerWidth - CAL_W - pad);
  return { top, left };
}

/**
 * Day-first date field (DD/MM/YYYY) with a calendar. Native type=date follows
 * the OS locale (Safari/Firefox often MM/DD/YYYY even when html lang=en-GB).
 * The popover is portaled to document.body so dashboard overflow-hidden cannot
 * clip it, and it flips when the field sits on the right edge.
 * Click the month or year in the header to jump without paging 12×N times.
 * Value stays YYYY-MM-DD for page state; API conversion is still toApiDate.
 */
export function DateInput({
  value,
  onChange,
  min,
  max,
  className = "",
  disabled,
  ...props
}: Omit<React.InputHTMLAttributes<HTMLInputElement>, "type" | "value" | "onChange"> & {
  value: string;
  onChange?: (iso: string) => void;
  min?: string;
  max?: string;
}) {
  const [text, setText] = useState(() => isoToDisplayDate(value));
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const rootRef = useRef<HTMLDivElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setText(isoToDisplayDate(value));
  }, [value]);

  const updatePos = () => {
    const el = rootRef.current;
    if (!el) return;
    setPos(placePopover(el.getBoundingClientRect()));
  };

  useLayoutEffect(() => {
    if (!open) return;
    updatePos();
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      const t = e.target as Node;
      if (rootRef.current?.contains(t) || popoverRef.current?.contains(t)) return;
      setOpen(false);
    };
    const onReposition = () => updatePos();
    document.addEventListener("mousedown", onDoc);
    window.addEventListener("resize", onReposition);
    window.addEventListener("scroll", onReposition, true);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      window.removeEventListener("resize", onReposition);
      window.removeEventListener("scroll", onReposition, true);
    };
  }, [open]);

  function apply(iso: string) {
    if (disabled || !iso || !inRange(iso, min, max)) return false;
    onChange?.(iso);
    setText(isoToDisplayDate(iso));
    return true;
  }

  function commit(raw: string) {
    const iso = parseTypedDate(raw);
    if (!iso || !inRange(iso, min, max)) {
      setText(isoToDisplayDate(value));
      return;
    }
    apply(iso);
  }

  return (
    <div ref={rootRef} className={`relative w-full ${className}`}>
      <input
        {...props}
        type="text"
        inputMode="numeric"
        autoComplete="off"
        spellCheck={false}
        placeholder="DD/MM/YYYY"
        value={text}
        disabled={disabled}
        onChange={(e) => {
          const next = e.target.value;
          setText(next);
          const iso = parseTypedDate(next);
          if (iso && inRange(iso, min, max)) onChange?.(iso);
        }}
        onBlur={() => commit(text)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            commit(text);
          }
          if (e.key === "Escape") setOpen(false);
        }}
        className="pp-control pr-10"
      />
      <button
        type="button"
        tabIndex={-1}
        disabled={disabled}
        className="absolute inset-y-0 right-0 flex w-10 items-center justify-center text-slate-500 hover:text-brand-800 disabled:cursor-not-allowed disabled:hover:text-slate-500"
        aria-label="Open calendar"
        aria-expanded={open}
        onClick={() => {
          if (disabled) return;
          setOpen((v) => !v);
        }}
      >
        <Calendar className="h-4 w-4" aria-hidden />
      </button>
      {open && typeof document !== "undefined"
        ? createPortal(
            <CalendarPopover
              ref={popoverRef}
              top={pos.top}
              left={pos.left}
              value={value}
              min={min}
              max={max}
              onPick={(iso) => {
                apply(iso);
                setOpen(false);
              }}
            />,
            document.body,
          )
        : null}
    </div>
  );
}

function CalendarPopover({
  ref,
  top,
  left,
  value,
  min,
  max,
  onPick,
}: {
  ref: React.Ref<HTMLDivElement>;
  top: number;
  left: number;
  value: string;
  min?: string;
  max?: string;
  onPick: (iso: string) => void;
}) {
  const selected = parseIso(value);
  const todayIso = localISODate(new Date());
  const initial = selected || parseIso(todayIso)!;
  const [view, setView] = useState<"day" | "month" | "year">("day");
  const [viewY, setViewY] = useState(initial.y);
  const [viewM, setViewM] = useState(initial.m);
  const yearBlock = Math.floor(viewY / 12) * 12;

  const cells = useMemo(() => {
    const first = new Date(viewY, viewM - 1, 1);
    const pad = (first.getDay() + 6) % 7;
    const lastDay = new Date(viewY, viewM, 0).getDate();
    const out: { iso: string; day: number; inMonth: boolean }[] = [];
    for (let i = 0; i < pad; i++) out.push({ iso: "", day: 0, inMonth: false });
    for (let d = 1; d <= lastDay; d++) {
      const iso = `${viewY}-${String(viewM).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
      out.push({ iso, day: d, inMonth: true });
    }
    return out;
  }, [viewY, viewM]);

  function shiftMonth(delta: number) {
    const dt = new Date(viewY, viewM - 1 + delta, 1);
    setViewY(dt.getFullYear());
    setViewM(dt.getMonth() + 1);
  }

  function monthDisabled(year: number, month: number): boolean {
    const start = `${year}-${String(month).padStart(2, "0")}-01`;
    const end = `${year}-${String(month).padStart(2, "0")}-${String(new Date(year, month, 0).getDate()).padStart(2, "0")}`;
    if (max && start > max) return true;
    if (min && end < min) return true;
    return false;
  }

  function yearDisabled(year: number): boolean {
    if (max && `${year}-01-01` > max) return true;
    if (min && `${year}-12-31` < min) return true;
    return false;
  }

  const navBtn =
    "rounded-lg p-1 text-slate-600 hover:bg-slate-100 disabled:text-slate-300 disabled:hover:bg-transparent";
  const cellBtn = (active: boolean) =>
    `h-9 rounded-lg text-sm ${
      active
        ? "bg-brand-800 font-semibold text-white"
        : "text-slate-800 hover:bg-slate-100"
    } disabled:cursor-not-allowed disabled:text-slate-300 disabled:hover:bg-transparent`;

  return (
    <div
      ref={ref}
      style={{ top, left, width: CAL_W }}
      className="fixed z-[300] rounded-xl border border-slate-200 bg-white p-3 shadow-xl"
      role="dialog"
      aria-label="Choose date"
    >
      <div className="mb-2 flex items-center justify-between gap-1">
        <div className="flex items-center">
          <button
            type="button"
            className={navBtn}
            aria-label={view === "year" ? "Previous 12 years" : "Previous year"}
            onClick={() =>
              view === "year" ? setViewY(viewY - 12) : setViewY(viewY - 1)
            }
          >
            <ChevronsLeft className="h-4 w-4" />
          </button>
          {view === "day" ? (
            <button
              type="button"
              className={navBtn}
              aria-label="Previous month"
              onClick={() => shiftMonth(-1)}
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
          ) : null}
        </div>
        <div className="flex items-center gap-1">
          {view === "day" ? (
            <button
              type="button"
              className="rounded-md px-1.5 py-0.5 text-sm font-semibold text-slate-900 hover:bg-slate-100"
              onClick={() => setView("month")}
            >
              {MONTHS[viewM - 1]}
            </button>
          ) : null}
          <button
            type="button"
            className="rounded-md px-1.5 py-0.5 text-sm font-semibold text-slate-900 hover:bg-slate-100"
            onClick={() => setView(view === "year" ? "month" : "year")}
          >
            {view === "year" ? `${yearBlock} – ${yearBlock + 11}` : viewY}
          </button>
        </div>
        <div className="flex items-center">
          {view === "day" ? (
            <button
              type="button"
              className={navBtn}
              aria-label="Next month"
              onClick={() => shiftMonth(1)}
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          ) : null}
          <button
            type="button"
            className={navBtn}
            aria-label={view === "year" ? "Next 12 years" : "Next year"}
            onClick={() =>
              view === "year" ? setViewY(viewY + 12) : setViewY(viewY + 1)
            }
          >
            <ChevronsRight className="h-4 w-4" />
          </button>
        </div>
      </div>

      {view === "year" ? (
        <div className="grid grid-cols-3 gap-1">
          {Array.from({ length: 12 }, (_, i) => yearBlock + i).map((y) => {
            const disabled = yearDisabled(y);
            return (
              <button
                key={y}
                type="button"
                disabled={disabled}
                onClick={() => {
                  setViewY(y);
                  setView("month");
                }}
                className={cellBtn(y === initial.y)}
              >
                {y}
              </button>
            );
          })}
        </div>
      ) : null}

      {view === "month" ? (
        <div className="grid grid-cols-3 gap-1">
          {MONTHS.map((name, i) => {
            const month = i + 1;
            const disabled = monthDisabled(viewY, month);
            return (
              <button
                key={name}
                type="button"
                disabled={disabled}
                onClick={() => {
                  setViewM(month);
                  setView("day");
                }}
                className={cellBtn(viewY === initial.y && month === initial.m)}
              >
                {name.slice(0, 3)}
              </button>
            );
          })}
        </div>
      ) : null}

      {view === "day" ? (
        <>
          <div className="mb-1 grid grid-cols-7 gap-0.5 text-center text-[11px] font-semibold text-slate-500">
            {WEEKDAYS.map((d) => (
              <div key={d}>{d}</div>
            ))}
          </div>
          <div className="grid grid-cols-7 gap-0.5">
            {cells.map((c, i) => {
              if (!c.inMonth) return <div key={`e-${i}`} />;
              const disabled = !inRange(c.iso, min, max);
              const isSel = c.iso === value;
              const isToday = c.iso === todayIso;
              return (
                <button
                  key={c.iso}
                  type="button"
                  disabled={disabled}
                  onClick={() => onPick(c.iso)}
                  className={`h-8 rounded-lg text-sm ${
                    isSel
                      ? "bg-brand-800 font-semibold text-white"
                      : isToday
                        ? "font-semibold text-brand-800 ring-1 ring-brand-300"
                        : "text-slate-800 hover:bg-slate-100"
                  } disabled:cursor-not-allowed disabled:text-slate-300 disabled:hover:bg-transparent`}
                >
                  {c.day}
                </button>
              );
            })}
          </div>
        </>
      ) : null}
    </div>
  );
}
