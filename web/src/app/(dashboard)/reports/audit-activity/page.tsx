"use client";

import { useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ListSearch, useListQuery } from "@/lib/list";
import { ReportExportActions } from "@/components/ReportExportActions";
import { ReportPageHeader } from "@/components/ReportPageHeader";
import { DataTable } from "@/components/DataTable";
import { Paginator } from "@/components/Paginator";
import { Input, Select } from "@/components/ui";
import { DateRangeFields, FilterField, ListShell } from "@/components/ListShell";
import {
  defaultFromISO,
  formatDateTime,
  localISODate,
  titleCase,
  toApiDate,
} from "@/lib/format";
import type { AuditEntry, Pagination } from "@/lib/types";

const AUDIT_MODULES = [
  "Auth",
  "User",
  "Profile",
  "Role",
  "Title",
  "Customer",
  "Inventory",
  "Billing",
  "Ewura",
  "Orders",
  "Reports",
  "Workflow",
  "Settings",
  "System",
] as const;

const AUDIT_ACTIONS = [
  "login",
  "refresh_token",
  "logout",
  "create",
  "update",
  "delete",
  "approve",
  "reject",
  "initiate",
  "dispatch",
  "sync",
] as const;

export default function AuditActivityReportPage() {
  const list = useListQuery({
    orderBy: "createdAt",
    sortDirection: "DESC",
  });
  const [fromDate, setFromDate] = useState(() => defaultFromISO(30));
  const [toDate, setToDate] = useState(() => localISODate(new Date()));
  const [userName, setUserName] = useState("");
  const [action, setAction] = useState("");
  const [module, setModule] = useState("");

  const filters = useMemo(
    () => ({
      ...list.query,
      fromDate: toApiDate(fromDate),
      toDate: toApiDate(toDate),
      userName: userName.trim() || undefined,
      action: action || undefined,
      module: module || undefined,
    }),
    [list.query, fromDate, toDate, userName, action, module],
  );

  const { data, loading, error } = useAsync(
    () =>
      api.get<Pagination<AuditEntry>>("/reports/audit-activity", filters),
    [
      list.search,
      list.page,
      list.pageSize,
      list.orderBy,
      list.sortDirection,
      fromDate,
      toDate,
      userName,
      action,
      module,
    ],
  );

  return (
    <div>
      <ReportPageHeader
        title="Audit activity"
        subtitle="Who did what — filter by actor, action, or module"
        actions={
          <ReportExportActions
            excelPath="/reports/audit-activity"
            pdfPath="/reports/audit-activity.pdf"
            query={filters}
            pdfTitle="Audit activity"
          />
        }
      />
      <ListShell
        filters={
          <>
            <DateRangeFields
              from={fromDate}
              to={toDate}
              onFrom={(iso) => {
                setFromDate(iso);
                list.setPage(1);
              }}
              onTo={(iso) => {
                setToDate(iso);
                list.setPage(1);
              }}
            />
            <FilterField label="Actor" className="w-40">
              <Input
                value={userName}
                onChange={(e) => {
                  setUserName(e.target.value);
                  list.setPage(1);
                }}
                placeholder="All users"
              />
            </FilterField>
            <FilterField label="Action" className="w-40">
              <Select
                value={action}
                onChange={(e) => {
                  setAction(e.target.value);
                  list.setPage(1);
                }}
              >
                <option value="">All actions</option>
                {AUDIT_ACTIONS.map((a) => (
                  <option key={a} value={a}>
                    {titleCase(a)}
                  </option>
                ))}
              </Select>
            </FilterField>
            <FilterField label="Module" className="w-40">
              <Select
                value={module}
                onChange={(e) => {
                  setModule(e.target.value);
                  list.setPage(1);
                }}
              >
                <option value="">All modules</option>
                {AUDIT_MODULES.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </Select>
            </FilterField>
          </>
        }
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Actor or description"
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
          emptyMessage="No audit activity in this window"
          columns={[
            {
              header: "When",
              cell: (r) => formatDateTime(r.createdAt),
            },
            { header: "Actor", cell: (r) => r.userName || "—" },
            { header: "Module", cell: (r) => r.module },
            { header: "Action", cell: (r) => titleCase(r.action) },
            { header: "Description", cell: (r) => r.description || "—" },
            { header: "IP", cell: (r) => r.ipAddress || "—" },
          ]}
        />
      </ListShell>
    </div>
  );
}
