"use client";

import { useEffect, useId, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";

export type SearchOption = {
  value: string;
  label: string;
  keywords?: string;
  email?: string;
  phone?: string;
  tinNumber?: string;
  disabled?: boolean;
  currencyCode?: string;
  isProration?: boolean;
};

type BaseProps = {
  value: string;
  onChange: (value: string, option?: SearchOption) => void;
  placeholder?: string;
  disabled?: boolean;
  emptyLabel?: string;
};

/** Client-side filter over a fixed option list (e.g. countries). */
export function SearchableSelect({
  options,
  value,
  onChange,
  placeholder = "Search…",
  disabled,
  emptyLabel = "No matches",
}: BaseProps & { options: SearchOption[] }) {
  const selected = options.find((o) => o.value === value);
  return (
    <Combobox
      value={value}
      display={selected?.label || ""}
      disabled={disabled}
      placeholder={placeholder}
      emptyLabel={emptyLabel}
      onChange={onChange}
      loadOptions={async (q) => {
        const needle = q.trim().toLowerCase();
        if (!needle) return options.slice(0, 80);
        return options
          .filter((o) => {
            const hay = `${o.label} ${o.value} ${o.keywords || ""}`.toLowerCase();
            return hay.includes(needle);
          })
          .slice(0, 80);
      }}
      staticOptions={options}
    />
  );
}

/** Debounced async search (e.g. Sage cashbook batches). */
export function AsyncSearchableSelect({
  value,
  onChange,
  loadOptions,
  placeholder = "Type to search…",
  disabled,
  emptyLabel = "No matches",
  selectedLabel,
}: BaseProps & {
  loadOptions: (query: string) => Promise<SearchOption[]>;
  /** Label for the currently selected value when it is not in the latest result set. */
  selectedLabel?: string;
}) {
  return (
    <Combobox
      value={value}
      display={selectedLabel || ""}
      disabled={disabled}
      placeholder={placeholder}
      emptyLabel={emptyLabel}
      onChange={onChange}
      loadOptions={loadOptions}
    />
  );
}

const LIST_MAX = 224;

function placeList(rect: DOMRect) {
  const gap = 4;
  const spaceBelow = window.innerHeight - rect.bottom - gap;
  const spaceAbove = rect.top - gap;
  const dropUp = spaceBelow < 180 && spaceAbove > spaceBelow;
  const maxHeight = Math.min(
    LIST_MAX,
    Math.max(120, dropUp ? spaceAbove : spaceBelow),
  );
  return {
    dropUp,
    maxHeight,
    top: dropUp ? undefined : rect.bottom + gap,
    bottom: dropUp ? window.innerHeight - rect.top + gap : undefined,
    left: rect.left,
    width: Math.max(rect.width, 220),
  };
}

function Combobox({
  value,
  display,
  onChange,
  loadOptions,
  placeholder,
  disabled,
  emptyLabel,
  staticOptions,
}: {
  value: string;
  display: string;
  onChange: (value: string, option?: SearchOption) => void;
  loadOptions: (query: string) => Promise<SearchOption[]>;
  placeholder: string;
  disabled?: boolean;
  emptyLabel: string;
  staticOptions?: SearchOption[];
}) {
  const listId = useId();
  const rootRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLUListElement>(null);
  const loadRef = useRef(loadOptions);
  useEffect(() => {
    loadRef.current = loadOptions;
  });
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState<ReturnType<typeof placeList> | null>(null);
  const [query, setQuery] = useState("");
  const [options, setOptions] = useState<SearchOption[]>([]);
  const [loading, setLoading] = useState(false);

  function updatePos() {
    const el = rootRef.current;
    if (!el) return;
    setPos(placeList(el.getBoundingClientRect()));
  }

  function openList() {
    updatePos();
    setOpen(true);
    setQuery("");
  }

  const shownLabel = useMemo(() => {
    if (open) return query;
    if (display) return display;
    if (value && staticOptions) {
      return staticOptions.find((o) => o.value === value)?.label || value;
    }
    return value ? display || value : "";
  }, [open, query, display, value, staticOptions]);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    const handle = window.setTimeout(async () => {
      setLoading(true);
      try {
        const rows = await loadRef.current(query);
        if (!cancelled) setOptions(rows);
      } catch {
        if (!cancelled) setOptions([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }, 220);
    return () => {
      cancelled = true;
      window.clearTimeout(handle);
    };
  }, [open, query]);

  useLayoutEffect(() => {
    if (!open) return;
    updatePos();
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function onDoc(e: MouseEvent) {
      const t = e.target as Node;
      if (rootRef.current?.contains(t) || listRef.current?.contains(t)) return;
      setOpen(false);
      setQuery("");
    }
    function onReposition() {
      updatePos();
    }
    document.addEventListener("mousedown", onDoc);
    window.addEventListener("resize", onReposition);
    window.addEventListener("scroll", onReposition, true);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      window.removeEventListener("resize", onReposition);
      window.removeEventListener("scroll", onReposition, true);
    };
  }, [open]);

  const list =
    open && !disabled && pos && typeof document !== "undefined"
      ? createPortal(
          <ul
            ref={listRef}
            id={listId}
            role="listbox"
            className="fixed z-[300] overflow-auto rounded-lg border border-slate-200 bg-white py-1 shadow-lg"
            style={{
              ...(pos.dropUp
                ? { bottom: pos.bottom }
                : { top: pos.top }),
              left: pos.left,
              width: pos.width,
              maxHeight: pos.maxHeight,
            }}
          >
            {loading && (
              <li className="px-3 py-2 text-sm text-slate-500">Searching…</li>
            )}
            {!loading && options.length === 0 && (
              <li className="px-3 py-2 text-sm text-slate-500">{emptyLabel}</li>
            )}
            {!loading &&
              options.map((o) => (
                <li
                  key={o.value}
                  role="option"
                  aria-selected={o.value === value}
                  aria-disabled={o.disabled}
                >
                  <button
                    type="button"
                    disabled={o.disabled}
                    className={`block w-full px-3 py-2 text-left text-sm ${
                      o.disabled
                        ? "cursor-not-allowed text-slate-400"
                        : o.value === value
                          ? "bg-brand-50 font-medium text-brand-900 hover:bg-brand-50"
                          : "text-slate-800 hover:bg-brand-50"
                    }`}
                    onClick={() => {
                      if (o.disabled) return;
                      onChange(o.value, o);
                      setOpen(false);
                      setQuery("");
                    }}
                  >
                    {o.label}
                  </button>
                </li>
              ))}
            {value && (
              <li className="border-t border-slate-100">
                <button
                  type="button"
                  className="block w-full px-3 py-2 text-left text-xs text-slate-500 hover:bg-slate-50"
                  onClick={() => {
                    onChange("");
                    setOpen(false);
                    setQuery("");
                  }}
                >
                  Clear selection
                </button>
              </li>
            )}
          </ul>,
          document.body,
        )
      : null;

  return (
    <div ref={rootRef} className="relative">
      <input
        className="pp-control w-full"
        role="combobox"
        aria-expanded={open}
        aria-controls={listId}
        aria-autocomplete="list"
        disabled={disabled}
        placeholder={placeholder}
        value={shownLabel}
        onFocus={openList}
        onChange={(e) => {
          if (!open) openList();
          else setOpen(true);
          setQuery(e.target.value);
        }}
        autoComplete="off"
        spellCheck={false}
      />
      {list}
    </div>
  );
}
