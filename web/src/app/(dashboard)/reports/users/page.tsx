"use client";

import { useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ListSearch, useListQuery } from "@/lib/list";
import { ReportExportActions } from "@/components/ReportExportActions";
import { ReportPageHeader } from "@/components/ReportPageHeader";
import { DataTable } from "@/components/DataTable";
import { Paginator } from "@/components/Paginator";
import { DateRangeFields, ListShell } from "@/components/ListShell";
import {
  defaultFromISO,
  formatDateTime,
  localISODate,
  toApiDate,
} from "@/lib/format";
import type { Pagination, User } from "@/lib/types";

export default function UsersReportPage() {
  const list = useListQuery({ orderBy: "email", sortDirection: "ASC" });
  const [fromDate, setFromDate] = useState(() => defaultFromISO(3650));
  const [toDate, setToDate] = useState(() => localISODate(new Date()));

  const filters = useMemo(
    () => ({
      ...list.query,
      fromDate: toApiDate(fromDate),
      toDate: toApiDate(toDate),
    }),
    [list.query, fromDate, toDate],
  );

  const { data, loading, error } = useAsync(
    () => api.get<Pagination<User>>("/reports/users", filters),
    [list.search, list.page, list.pageSize, list.orderBy, list.sortDirection, fromDate, toDate],
  );

  return (
    <div>
      <ReportPageHeader
        title="System users"
        subtitle="Access review listing — created within the selected window"
        actions={
          <ReportExportActions
            excelPath="/reports/users"
            pdfPath="/reports/users.pdf"
            query={filters}
            pdfTitle="System users"
          />
        }
      />
      <ListShell
        filters={
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
        }
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Email or name"
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
          sort={list.sort}
          onSort={list.onSort}
          emptyMessage="No users in this window"
          columns={[
            { header: "Email", sortKey: "email", cell: (r) => r.email },
            {
              header: "Name",
              cell: (r) => [r.firstName, r.lastName].filter(Boolean).join(" ") || "—",
            },
            {
              header: "Active",
              cell: (r) => (r.isActive ? "Yes" : "No"),
            },
            {
              header: "Locked",
              cell: (r) => (r.isLocked ? "Yes" : "No"),
            },
            {
              header: "Roles",
              cell: (r) => (r.roles || []).map((x) => x.name).join(", ") || "—",
            },
            {
              header: "Last login",
              cell: (r) => (r.lastLogin ? formatDateTime(r.lastLogin) : "Never"),
            },
          ]}
        />
      </ListShell>
    </div>
  );
}
