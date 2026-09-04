"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { tabClass } from "@/components/ui";
import { useAuth } from "@/lib/auth";

export type ModuleTab = {
  href: string;
  label: string;
  /** When set, the tab is shown only if the user holds any of these codes. */
  perms?: string[];
};

/** Route tabs (Approvals, Access, Settings). In-record sections use ReviewTabBar. */
export function ModuleTabs({
  tabs,
  className = "border-b border-slate-200 pb-px",
}: {
  tabs: ModuleTab[];
  className?: string;
}) {
  const pathname = usePathname();
  const { hasPermission } = useAuth();
  const visible = tabs.filter(
    (t) => !t.perms?.length || hasPermission(...t.perms),
  );
  const sorted = [...visible].sort((a, b) => b.href.length - a.href.length);
  const current =
    sorted.find(
      (t) => pathname === t.href || pathname.startsWith(`${t.href}/`),
    )?.href ?? visible[0]?.href;

  if (visible.length === 0) return null;

  return (
    <div role="tablist" className={`flex flex-wrap gap-1 ${className}`}>
      {visible.map((t) => {
        const active = t.href === current;
        return (
          <Link
            key={t.href}
            role="tab"
            aria-selected={active}
            aria-current={active ? "page" : undefined}
            href={t.href}
            className={tabClass(active)}
          >
            {t.label}
          </Link>
        );
      })}
    </div>
  );
}
