"use client";

import Link from "next/link";
import { useEffect, useId, useRef, useState } from "react";
import { ChevronDown, X } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import type { Pagination, Title } from "@/lib/types";

/**
 * Searchable title picker (Select2-style) backed by /auth/titles.
 * Stores the title name on the user profile (same as the API contract).
 * Users with titles.create can add a new catalogue entry from the dropdown.
 */
export function TitleSelect({
  value,
  onChange,
  disabled,
  placeholder = "Select a title…",
}: {
  value: string;
  onChange: (name: string) => void;
  disabled?: boolean;
  placeholder?: string;
}) {
  const listId = useId();
  const rootRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const { hasPermission } = useAuth();
  const canCreate = hasPermission(
    PERMISSIONS.titlesCreate,
    PERMISSIONS.titlesManage,
  );
  const canManageCatalogue = hasPermission(
    PERMISSIONS.titlesRead,
    PERMISSIONS.titlesCreate,
    PERMISSIONS.titlesUpdate,
    PERMISSIONS.titlesDelete,
    PERMISSIONS.titlesManage,
  );

  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [titles, setTitles] = useState<Title[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    const handle = window.setTimeout(async () => {
      setLoading(true);
      setError("");
      try {
        const page = await api.get<Pagination<Title>>("/auth/titles", {
          search: query.trim(),
          page: 1,
          pageSize: 50,
          orderBy: "name",
          sortDirection: "ASC",
        });
        if (!cancelled) setTitles(page?.items ?? []);
      } catch (err) {
        if (!cancelled) {
          setTitles([]);
          setError(
            err instanceof ApiError ? err.message : "Failed to load titles",
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }, 200);
    return () => {
      cancelled = true;
      window.clearTimeout(handle);
    };
  }, [open, query]);

  function select(name: string) {
    onChange(name);
    setQuery("");
    setOpen(false);
  }

  function clear(e: React.MouseEvent) {
    e.stopPropagation();
    onChange("");
    setQuery("");
  }

  async function createTitle() {
    const name = query.trim();
    if (!name || !canCreate) return;
    setCreating(true);
    setError("");
    try {
      const created = await api.post<Title>("/auth/titles", { name });
      select(created?.name || name);
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Failed to create title",
      );
    } finally {
      setCreating(false);
    }
  }

  const q = query.trim().toLowerCase();
  const exactMatch = titles.some((t) => t.name.toLowerCase() === q);
  const showCreate =
    canCreate && q.length > 0 && !exactMatch && !loading && !creating;

  return (
    <div ref={rootRef} className="relative">
      <div
        className={`pp-control flex cursor-pointer items-center gap-1 ${
          disabled ? "cursor-not-allowed opacity-60" : ""
        }`}
        onClick={() => {
          if (disabled) return;
          setOpen(true);
          window.setTimeout(() => inputRef.current?.focus(), 0);
        }}
      >
        {open ? (
          <input
            ref={inputRef}
            className="min-w-0 flex-1 border-0 bg-transparent p-0 text-sm outline-none focus:ring-0"
            value={query}
            disabled={disabled}
            placeholder={value || placeholder}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Escape") setOpen(false);
              if (e.key === "Enter") {
                e.preventDefault();
                if (titles[0]) select(titles[0].name);
                else if (showCreate) void createTitle();
              }
            }}
            aria-autocomplete="list"
            aria-controls={listId}
            aria-expanded={open}
          />
        ) : (
          <span
            className={`min-w-0 flex-1 truncate text-sm ${
              value ? "text-slate-800" : "text-slate-400"
            }`}
          >
            {value || placeholder}
          </span>
        )}
        {value && !disabled && (
          <button
            type="button"
            className="rounded p-0.5 text-slate-400 hover:bg-slate-100 hover:text-slate-700"
            aria-label="Clear title"
            onClick={clear}
          >
            <X className="h-3.5 w-3.5" />
          </button>
        )}
        <ChevronDown className="h-4 w-4 shrink-0 text-slate-400" />
      </div>

      {open && !disabled && (
        <ul
          id={listId}
          role="listbox"
          className="absolute z-30 mt-1 max-h-56 w-full overflow-auto rounded-lg border border-slate-200 bg-white py-1 shadow-lg"
        >
          {loading && (
            <li className="px-3 py-2 text-sm text-slate-400">Loading…</li>
          )}
          {!loading && error && (
            <li className="px-3 py-2 text-sm text-red-700">{error}</li>
          )}
          {!loading && !error && titles.length === 0 && !showCreate && (
            <li className="px-3 py-2 text-sm text-slate-400">No titles found</li>
          )}
          {!loading &&
            titles.map((t) => (
              <li key={t.id}>
                <button
                  type="button"
                  role="option"
                  aria-selected={t.name === value}
                  className={`flex w-full px-3 py-2 text-left text-sm hover:bg-brand-50 ${
                    t.name === value
                      ? "bg-brand-50 font-medium text-brand-900"
                      : "text-slate-700"
                  }`}
                  onClick={() => select(t.name)}
                >
                  {t.name}
                </button>
              </li>
            ))}
          {showCreate && (
            <li className="border-t border-slate-100">
              <button
                type="button"
                className="flex w-full px-3 py-2 text-left text-sm font-medium text-brand-800 hover:bg-brand-50"
                disabled={creating}
                onClick={() => void createTitle()}
              >
                {creating ? "Creating…" : `Create “${query.trim()}”`}
              </button>
            </li>
          )}
          {canManageCatalogue && (
            <li className="border-t border-slate-100">
              <Link
                href="/titles"
                className="block px-3 py-2 text-left text-xs font-medium text-slate-500 hover:bg-slate-50 hover:text-slate-800"
                onClick={() => setOpen(false)}
              >
                Manage titles catalogue…
              </Link>
            </li>
          )}
        </ul>
      )}
    </div>
  );
}
