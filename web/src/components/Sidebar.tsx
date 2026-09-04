"use client";

import Link from "next/link";
import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import { usePathname } from "next/navigation";
import { ChevronDown, ChevronLeft, ChevronRight, type LucideIcon } from "lucide-react";
import { Logo } from "@/components/Logo";
import { useAppearance } from "@/lib/appearance";
import { useAuth } from "@/lib/auth";
import {
  filterNav,
  isActive,
  NAV,
  nodeActive,
  type NavGroup,
  type NavNode,
} from "@/lib/navTree";

function linkClass(active: boolean, expanded: boolean, indent = false) {
  return `flex w-full items-center rounded-md text-sm font-medium text-white transition ${
    expanded ? `gap-3 py-2 ${indent ? "px-2.5" : "px-3 py-2.5"}` : "h-10 justify-center px-0"
  } ${
    active
      ? "bg-white/10 text-white ring-1 ring-inset ring-brand-300/80"
      : "text-white/95 hover:bg-white/10 hover:text-white"
  }`;
}

function NavIcon({
  icon: Icon,
  size = 18,
  compact,
}: {
  icon: LucideIcon;
  size?: number;
  compact?: boolean;
}) {
  if (compact) {
    return (
      <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center">
        <Icon size={size} className="block" strokeWidth={1.75} />
      </span>
    );
  }
  return <Icon size={size} className="block shrink-0" strokeWidth={1.75} />;
}

function FlyoutLinks({
  nodes,
  pathname,
  onPick,
}: {
  nodes: NavNode[];
  pathname: string;
  onPick: () => void;
}) {
  return (
    <>
      {nodes.map((node) => {
        if (node.kind === "link") {
          const active = isActive(pathname, node.href, node.exact);
          return (
            <Link
              key={node.href}
              href={node.href}
              className={`flex items-center gap-2 px-3 py-2 text-sm ${
                active ? "bg-slate-100 font-medium text-slate-900" : "hover:bg-slate-50"
              }`}
              onClick={onPick}
            >
              <NavIcon icon={node.icon} size={16} />
              {node.label}
            </Link>
          );
        }
        return (
          <div key={node.id} className="pt-1">
            <div className="px-3 py-1 text-[10px] font-semibold uppercase tracking-wide text-slate-400">
              {node.label}
            </div>
            <FlyoutLinks nodes={node.children} pathname={pathname} onPick={onPick} />
          </div>
        );
      })}
    </>
  );
}

function NestedGroup({
  group,
  pathname,
  openGroups,
  toggle,
  depth,
}: {
  group: NavGroup;
  pathname: string;
  openGroups: Record<string, boolean>;
  toggle: (id: string) => void;
  depth: number;
}) {
  const groupActive = nodeActive(pathname, group);
  const open = openGroups[group.id] ?? (group.defaultCollapsed ? false : groupActive);
  const nested = depth > 0;

  return (
    <div className="space-y-0.5">
      <button
        type="button"
        onClick={() => toggle(group.id)}
        className={`flex w-full items-center gap-3 rounded-md text-sm font-medium transition ${
          nested ? "px-2.5 py-1.5" : "px-3 py-2.5"
        } ${
          groupActive
            ? "bg-white/10 text-white ring-1 ring-inset ring-brand-300/80"
            : "text-white/95 hover:bg-white/10 hover:text-white"
        }`}
        aria-expanded={open}
      >
        <NavIcon icon={group.icon} size={nested ? 16 : 18} />
        <span className="flex-1 text-left">{group.label}</span>
        <ChevronDown
          size={16}
          className={`shrink-0 opacity-70 transition-transform ${
            open ? "rotate-0" : "-rotate-90"
          }`}
        />
      </button>
      {open && (
        <div
          className={`space-y-0.5 border-l border-white/15 ${
            nested ? "ml-2 pl-1.5" : "ml-3 pl-2"
          }`}
        >
          {group.children.map((child) =>
            child.kind === "group" ? (
              <NestedGroup
                key={child.id}
                group={child}
                pathname={pathname}
                openGroups={openGroups}
                toggle={toggle}
                depth={depth + 1}
              />
            ) : (
              <Link
                key={child.href}
                href={child.href}
                title={child.label}
                className={`flex items-center gap-3 rounded-md px-2.5 py-1.5 text-sm font-medium transition ${
                  isActive(pathname, child.href, child.exact)
                    ? "bg-white/10 text-white ring-1 ring-inset ring-brand-300/80"
                    : "text-white/90 hover:bg-white/10 hover:text-white"
                }`}
              >
                <NavIcon icon={child.icon} size={16} />
                <span>{child.label}</span>
              </Link>
            ),
          )}
        </div>
      )}
    </div>
  );
}

export function Sidebar() {
  const pathname = usePathname();
  const { user, hasPermission, mustChangePassword } = useAuth();
  const { sidebarExpanded, toggleSidebar } = useAppearance();
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({});
  const [flyout, setFlyout] = useState<string | null>(null);
  const flyoutRef = useRef<HTMLDivElement>(null);

  const visible = useMemo(() => {
    if (mustChangePassword) return [];
    return filterNav(NAV, hasPermission);
  }, [hasPermission, mustChangePassword]);

  useEffect(() => {
    const next: Record<string, boolean> = {};
    function walk(nodes: NavNode[]) {
      for (const node of nodes) {
        if (node.kind !== "group") continue;
        if (nodeActive(pathname, node) && !node.defaultCollapsed) {
          next[node.id] = true;
        }
        if (nodeActive(pathname, node)) {
          next[node.id] = true;
        }
        walk(node.children);
      }
    }
    walk(visible);
    setOpenGroups((prev) => ({ ...prev, ...next }));
  }, [pathname, visible]);

  useEffect(() => {
    if (!flyout) return;
    const onDoc = (e: MouseEvent) => {
      if (!flyoutRef.current?.contains(e.target as Node)) setFlyout(null);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [flyout]);

  useEffect(() => {
    setFlyout(null);
  }, [pathname, sidebarExpanded]);

  function toggleGroup(id: string) {
    setOpenGroups((prev) => ({ ...prev, [id]: !prev[id] }));
  }

  return (
    <aside
      className={`relative hidden shrink-0 flex-col bg-[#0f1419] text-white transition-[width] duration-200 md:flex ${
        sidebarExpanded ? "w-64" : "w-[4.5rem]"
      }`}
    >
      <div
        className={`flex h-14 items-center border-b border-white/10 ${
          sidebarExpanded ? "gap-3 px-4" : "justify-center px-2"
        }`}
      >
        <Logo variant="white" height={sidebarExpanded ? 36 : 28} />
        {sidebarExpanded && (
          <div className="min-w-0">
            <div className="truncate text-sm font-semibold leading-tight text-white">
              TIPER DFMS
            </div>
            <div className="truncate text-[10px] font-medium uppercase tracking-[0.14em] text-white/70">
              Kigamboni
            </div>
          </div>
        )}
      </div>

      <nav
        className={`flex-1 space-y-0.5 overflow-y-auto ${
          sidebarExpanded ? "p-2" : "flex flex-col items-stretch px-1.5 py-2"
        }`}
      >
        {visible.map((item, i) => {
          const prevSection = i > 0 ? visible[i - 1].section : undefined;
          const showSection =
            sidebarExpanded && Boolean(item.section) && item.section !== prevSection;
          const heading = showSection ? (
            <div
              className={`px-3 pb-1 text-[10px] font-semibold uppercase tracking-[0.14em] text-white/40 ${
                i === 0 ? "pt-1" : "pt-3"
              }`}
            >
              {item.section}
            </div>
          ) : null;

          if (item.kind === "link") {
            const active = isActive(pathname, item.href, item.exact);
            return (
              <Fragment key={item.href}>
                {heading}
                <Link
                  href={item.href}
                  title={item.label}
                  className={linkClass(active, sidebarExpanded)}
                >
                  <NavIcon icon={item.icon} compact={!sidebarExpanded} />
                  {sidebarExpanded && <span>{item.label}</span>}
                </Link>
              </Fragment>
            );
          }

          const groupActive = nodeActive(pathname, item);

          if (!sidebarExpanded) {
            return (
              <div
                key={item.id}
                className="relative w-full"
                ref={flyout === item.id ? flyoutRef : undefined}
              >
                <button
                  type="button"
                  title={item.label}
                  aria-expanded={flyout === item.id}
                  onClick={() =>
                    setFlyout((cur) => (cur === item.id ? null : item.id))
                  }
                  className={linkClass(groupActive || flyout === item.id, false)}
                >
                  <NavIcon icon={item.icon} compact />
                </button>
                {flyout === item.id && (
                  <div className="absolute left-full top-0 z-40 ml-2 max-h-[70vh] min-w-[12rem] overflow-y-auto rounded-md border border-slate-200 bg-white py-1 text-slate-800 shadow-lg">
                    <div className="px-3 py-1.5 text-xs font-semibold uppercase tracking-wide text-slate-400">
                      {item.label}
                    </div>
                    <FlyoutLinks
                      nodes={item.children}
                      pathname={pathname}
                      onPick={() => setFlyout(null)}
                    />
                  </div>
                )}
              </div>
            );
          }

          return (
            <Fragment key={item.id}>
              {heading}
              <NestedGroup
                group={item}
                pathname={pathname}
                openGroups={openGroups}
                toggle={toggleGroup}
                depth={0}
              />
            </Fragment>
          );
        })}
      </nav>

      <div className="border-t border-white/10 p-3">
        <Link
          href={mustChangePassword ? "/profile?tab=password" : "/profile"}
          title="My profile"
          className={`flex items-center rounded-lg transition hover:bg-white/10 ${
            sidebarExpanded ? "gap-3 px-2 py-2" : "justify-center px-0 py-2"
          }`}
        >
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-800 text-xs font-semibold text-white">
            {(
              [user?.firstName?.[0], user?.lastName?.[0]]
                .filter(Boolean)
                .join("") ||
              user?.email?.slice(0, 2) ||
              "?"
            ).toUpperCase()}
          </div>
          {sidebarExpanded && (
            <div className="min-w-0">
              <div className="truncate text-sm font-medium text-white">
                {[user?.firstName, user?.lastName].filter(Boolean).join(" ") ||
                  "Account"}
              </div>
              <div className="truncate text-xs text-white/60">{user?.email}</div>
            </div>
          )}
        </Link>
      </div>

      <button
        type="button"
        onClick={toggleSidebar}
        title={sidebarExpanded ? "Hide sidebar" : "Show sidebar"}
        aria-label={sidebarExpanded ? "Hide sidebar" : "Show sidebar"}
        className="absolute -right-3 top-16 z-20 flex h-6 w-6 items-center justify-center rounded-full border border-[var(--pp-border)] bg-[var(--pp-surface)] text-[var(--pp-text)] shadow-sm hover:bg-slate-50"
      >
        {sidebarExpanded ? (
          <ChevronLeft size={14} strokeWidth={2.5} />
        ) : (
          <ChevronRight size={14} strokeWidth={2.5} />
        )}
      </button>
    </aside>
  );
}
