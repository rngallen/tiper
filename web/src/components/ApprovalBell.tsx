"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Bell } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";
import { type WorkflowInbox, type WorkflowTask } from "@/lib/types";
import {
  approvalDocumentLabel,
  approvalQueueForType,
  approvalQueueHref,
} from "@/lib/approvals";
import { SubstituteBadge } from "@/components/SubstituteBadge";

/** Poll while the tab is visible; pause in the background to avoid idle load. */
const POLL_MS = 30_000;
const PREVIEW = 8;

function timeAgo(iso: string): string {
  const then = Date.parse(iso);
  if (Number.isNaN(then)) return "";
  const m = Math.floor((Date.now() - then) / 60_000);
  if (m < 1) return "Just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}

function useApprovalInbox(enabled: boolean) {
  const pathname = usePathname();
  const [count, setCount] = useState(0);
  const [items, setItems] = useState<WorkflowTask[]>([]);
  const prevCount = useRef<number | null>(null);
  const [arrived, setArrived] = useState(false);

  const load = useCallback(async () => {
    if (!enabled) return;
    try {
      const data = await api.get<WorkflowInbox>(
        "/workflow/tasks/inbox",
        undefined,
        { countAsActivity: false },
      );
      const next = data?.count ?? 0;
      setCount(next);
      setItems(data?.items || []);
      if (prevCount.current !== null && next > prevCount.current) {
        setArrived(true);
        window.setTimeout(() => setArrived(false), 2500);
      }
      prevCount.current = next;
    } catch (err) {
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
        setCount(0);
        setItems([]);
      }
    }
  }, [enabled]);

  useEffect(() => {
    if (!enabled) return;
    const id = window.setInterval(() => {
      if (document.visibilityState === "visible") void load();
    }, POLL_MS);
    const onVisible = () => {
      if (document.visibilityState === "visible") void load();
    };
    document.addEventListener("visibilitychange", onVisible);
    window.addEventListener("focus", onVisible);
    return () => {
      window.clearInterval(id);
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("focus", onVisible);
    };
  }, [enabled, load]);

  useEffect(() => {
    if (!enabled) return;
    void load();
  }, [pathname, enabled, load]);

  return { count, items, arrived, reload: load };
}

/** Header bell: pending count, preview list, links into Approvals. */
export function ApprovalBell() {
  const { hasPermission } = useAuth();
  const canSee = hasPermission(
    PERMISSIONS.workflowTasks,
    PERMISSIONS.workflowManage,
  );
  const { count, items, arrived, reload } = useApprovalInbox(canSee);
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    void reload();
    function onDoc(e: MouseEvent) {
      if (!wrapRef.current?.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open, reload]);

  if (!canSee) return null;

  const badge = count > 99 ? "99+" : String(count);
  const preview = items.slice(0, PREVIEW);
  const sameKind =
    preview.length > 0 &&
    preview.every((t) => t.docContentType === preview[0].docContentType);
  const footerQueue = approvalQueueForType(preview[0]?.docContentType);
  const footerHref = footerQueue?.href ?? "/approvals";
  const footerLabel = sameKind && footerQueue ? footerQueue.label : "approvals";

  return (
    <div className="relative" ref={wrapRef}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={`relative rounded-lg p-2 transition hover:bg-[var(--pp-row-hover)] ${
          arrived ? "ring-2 ring-accent/50" : ""
        }`}
        aria-label={
          count > 0
            ? `${count} pending approval${count === 1 ? "" : "s"}`
            : "No pending approvals"
        }
        aria-expanded={open}
        aria-haspopup="menu"
      >
        <Bell
          size={20}
          className={count > 0 ? "text-[var(--pp-text)]" : "text-[var(--pp-muted)]"}
        />
        {count > 0 ? (
          <span className="absolute right-0.5 top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-accent px-1 text-[10px] font-bold leading-none text-white">
            {badge}
          </span>
        ) : null}
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 z-40 mt-2 w-[24rem] overflow-hidden rounded-xl border border-[var(--pp-border)] bg-[var(--pp-surface)] shadow-lg"
        >
          <div className="flex items-center justify-between border-b border-[var(--pp-border)] px-3 py-2">
            <p className="text-sm font-semibold text-[var(--pp-text)]">
              Pending approvals
            </p>
            {count > 0 ? (
              <span className="rounded-full bg-accent/10 px-2 py-0.5 text-[11px] font-semibold text-accent">
                {count}
              </span>
            ) : null}
          </div>

          {preview.length === 0 ? (
            <p className="px-3 py-6 text-center text-sm text-[var(--pp-muted)]">
              Nothing waiting for you
            </p>
          ) : (
            <ul className="max-h-80 overflow-y-auto py-1">
              {preview.map((t) => {
                const kind = approvalQueueForType(t.docContentType)?.label || "Approval";
                const doc = approvalDocumentLabel(t);
                return (
                  <li key={t.taskId}>
                    <Link
                      href={approvalQueueHref(t.docContentType)}
                      role="menuitem"
                      onClick={() => setOpen(false)}
                      className="block px-3 py-2.5 hover:bg-[var(--pp-row-hover)]"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <span className="text-[11px] font-semibold uppercase tracking-wide text-brand-800">
                          {kind}
                        </span>
                        <span className="shrink-0 text-[11px] text-[var(--pp-muted)]">
                          {timeAgo(t.createdAt)}
                        </span>
                      </div>
                      <div className="mt-0.5 truncate text-sm font-semibold text-[var(--pp-text)]">
                        {doc}
                      </div>
                      {t.nodeName ? (
                        <div className="mt-0.5 truncate text-xs text-[var(--pp-muted)]">
                          Step: {t.nodeName}
                        </div>
                      ) : null}
                      {t.onBehalfOfName ? (
                        <div className="mt-1">
                          <SubstituteBadge covering={t.onBehalfOfName} />
                        </div>
                      ) : null}
                    </Link>
                  </li>
                );
              })}
            </ul>
          )}

          <div className="border-t border-[var(--pp-border)] px-3 py-2">
            <Link
              href={footerHref}
              onClick={() => setOpen(false)}
              className="text-sm font-semibold text-brand-800 hover:underline"
            >
              {count > preview.length
                ? `View all ${count} in ${footerLabel} →`
                : `Open ${footerLabel} →`}
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
