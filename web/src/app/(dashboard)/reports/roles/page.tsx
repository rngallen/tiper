"use client";

import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ListSearch, useListQuery } from "@/lib/list";
import { ReportExportActions } from "@/components/ReportExportActions";
import { ReportPageHeader } from "@/components/ReportPageHeader";
import { DataTable } from "@/components/DataTable";
import { Paginator } from "@/components/Paginator";
import { ListShell } from "@/components/ListShell";
import type { Pagination, Permission, Role } from "@/lib/types";

function groupByModule(perms: Permission[] | undefined): [string, string[]][] {
  const map = new Map<string, string[]>();
  for (const p of perms || []) {
    const mod = (p.module || "Other").trim() || "Other";
    const list = map.get(mod) || [];
    list.push(p.code);
    map.set(mod, list);
  }
  return Array.from(map.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([mod, codes]) => [mod, codes.sort()]);
}

export default function RolesReportPage() {
  const list = useListQuery({ orderBy: "name", sortDirection: "ASC" });

  const { data, loading, error } = useAsync(
    () => api.get<Pagination<Role>>("/reports/roles", list.query),
    [list.search, list.page, list.pageSize, list.orderBy, list.sortDirection],
  );

  return (
    <div>
      <ReportPageHeader
        title="Roles & permissions"
        subtitle="Permissions grouped by module for each role"
        actions={
          <ReportExportActions
            excelPath="/reports/roles"
            pdfPath="/reports/roles.pdf"
            query={list.query}
            pdfTitle="Roles & permissions"
          />
        }
      />
      <ListShell
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Search roles"
          />
        }
        footer={
          !loading && !error ? (
            <Paginator
              embedded
              data={data}
              page={list.page}
              onPage={list.setPage}
              pageSize={list.pageSize}
              onPageSize={list.setPageSize}
            />
          ) : null
        }
      >
        <DataTable
          plain
          loading={loading}
          error={error}
          rows={data?.items}
          emptyMessage="No roles found"
          columns={[
            { header: "Role", cell: (r) => r.name },
            { header: "Description", cell: (r) => r.description || "—" },
            {
              header: "Permissions by module",
              cell: (r) => {
                const groups = groupByModule(r.permissions);
                if (groups.length === 0) return "—";
                return (
                  <div className="space-y-1.5 py-0.5">
                    {groups.map(([mod, codes]) => (
                      <div key={mod}>
                        <span className="font-semibold text-slate-800">{mod}</span>
                        <span className="text-slate-500"> — </span>
                        <span className="text-slate-700">{codes.join(", ")}</span>
                      </div>
                    ))}
                  </div>
                );
              },
            },
            {
              header: "Count",
              cell: (r) => String((r.permissions || []).length),
            },
          ]}
        />
      </ListShell>
    </div>
  );
}
