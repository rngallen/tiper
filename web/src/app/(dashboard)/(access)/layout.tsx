"use client";

import { usePathname } from "next/navigation";
import { ModuleTabs } from "@/components/ModuleTabs";
import { PERMISSIONS } from "@/lib/permissions";

const TABS = [
  {
    href: "/users",
    label: "Users",
    perms: [PERMISSIONS.usersRead, PERMISSIONS.usersManage],
  },
  {
    href: "/roles",
    label: "Roles",
    perms: [PERMISSIONS.rolesRead, PERMISSIONS.rolesManage],
  },
  {
    href: "/titles",
    label: "Titles",
    perms: [
      PERMISSIONS.titlesRead,
      PERMISSIONS.titlesCreate,
      PERMISSIONS.titlesUpdate,
      PERMISSIONS.titlesDelete,
      PERMISSIONS.titlesManage,
    ],
  },
  {
    href: "/workflows",
    label: "Workflows",
    perms: [PERMISSIONS.workflowRead, PERMISSIONS.workflowManage],
  },
];

const COPY: Record<string, { title: string; subtitle: string }> = {
  "/users": {
    title: "Users",
    subtitle: "Sign-in accounts. Job title is the position; roles are what they can do.",
  },
  "/roles": {
    title: "Roles",
    subtitle: "Permission catalogues. Assigned to users; selected as operator on workflow steps.",
  },
  "/titles": {
    title: "Titles",
    subtitle: "Job titles on user profiles. Not a workflow role and not a permission set.",
  },
  "/workflows": {
    title: "Workflows",
    subtitle:
      "One process per document type. Click a row to view the flow; Configure to change steps, pool, and watchers.",
  },
};

export default function AccessLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const copy =
    COPY[pathname] ??
    Object.entries(COPY).find(([href]) => pathname.startsWith(href))?.[1] ??
    COPY["/users"];

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-bold tracking-tight text-slate-900">Access</h1>
        <p className="mt-0.5 text-sm text-slate-600">
          <span className="font-semibold text-slate-800">{copy.title}</span>
          <span className="mx-1.5 text-slate-300">·</span>
          {copy.subtitle}
        </p>
      </div>
      <ModuleTabs tabs={TABS} />
      {children}
    </div>
  );
}
