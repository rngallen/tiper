"use client";

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { ModuleTabs } from "@/components/ModuleTabs";
import { Button } from "@/components/ui";
import { useAuth } from "@/lib/auth";
import {
  firstAllowedSettingsHref,
  matchSettings,
  visibleSettingsGroups,
} from "./_components/nav";
import { SettingsProvider, useSettings } from "./_components/settings-context";

export default function SettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <SettingsProvider>
      <SettingsChrome>{children}</SettingsChrome>
    </SettingsProvider>
  );
}

function SettingsChrome({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { hasPermission, loading } = useAuth();
  const { canManage, saveAction } = useSettings();
  const groups = visibleSettingsGroups(hasPermission);
  const { group, leaf } = matchSettings(pathname, groups);

  useEffect(() => {
    if (loading) return;
    const visible = visibleSettingsGroups(hasPermission);
    if (visible.length === 0) return;
    const allowed =
      pathname === "/settings" ||
      visible.some(
        (g) =>
          pathname === g.base ||
          g.children.some(
            (c) => pathname === c.href || pathname.startsWith(`${c.href}/`),
          ),
      );
    if (!allowed) {
      router.replace(firstAllowedSettingsHref(hasPermission));
    }
  }, [loading, pathname, hasPermission, router]);

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-bold tracking-tight text-slate-900">
          Settings
        </h1>
        <nav aria-label="Breadcrumb" className="mt-1 text-xs font-medium text-slate-500">
          <ol className="flex flex-wrap items-center gap-1.5">
            <li>Settings</li>
            <li aria-hidden="true">›</li>
            <li>
              <Link
                href={group.children[0]?.href ?? group.base}
                className="hover:text-slate-800"
              >
                {group.label}
              </Link>
            </li>
            <li aria-hidden="true">›</li>
            <li className="text-slate-800">{leaf.label}</li>
          </ol>
        </nav>
      </div>
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start">
        <nav
          aria-label="Settings groups"
          className="flex shrink-0 gap-1 overflow-x-auto rounded-lg border border-slate-200 bg-white p-1.5 lg:w-44 lg:flex-col lg:overflow-visible"
        >
          {groups.map((g) => {
            const active = g.id === group.id;
            const href = g.children[0]?.href ?? g.base;
            return (
              <Link
                key={g.id}
                href={href}
                aria-current={active ? "page" : undefined}
                className={`whitespace-nowrap rounded-md px-3 py-2 text-sm font-semibold ${
                  active
                    ? "border-l-2 border-brand-800 bg-brand-50 text-brand-900"
                    : "border-l-2 border-transparent text-slate-700 hover:bg-slate-50"
                }`}
              >
                {g.label}
              </Link>
            );
          })}
        </nav>
        <div className="min-w-0 flex-1 space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200">
            <ModuleTabs
              tabs={group.children.map((c) => ({
                href: c.href,
                label: c.label,
              }))}
              className="min-w-0 flex-1"
            />
            {canManage && saveAction ? (
              <Button
                type="button"
                className="mb-1.5 shrink-0"
                onClick={saveAction.run}
                loading={saveAction.saving}
                disabled={saveAction.disabled}
              >
                Save changes
              </Button>
            ) : null}
          </div>
          {children}
        </div>
      </div>
    </div>
  );
}
