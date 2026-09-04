"use client";

import { usePathname } from "next/navigation";
import { ModuleTabs } from "@/components/ModuleTabs";
import { APPROVAL_QUEUES } from "@/lib/approvals";

export default function ApprovalsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  if (pathname === "/approvals") return <>{children}</>;
  return (
    <div className="space-y-4">
      <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">
        Approvals
      </p>
      <ModuleTabs
        tabs={APPROVAL_QUEUES.map((q) => ({
          href: q.href,
          label: q.label,
          perms: q.perms,
        }))}
      />
      {children}
    </div>
  );
}
