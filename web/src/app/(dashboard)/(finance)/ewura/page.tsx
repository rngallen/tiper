"use client";

import { useMemo, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { ExportButton, ListSearch, useListQuery, useUrlFilter } from "@/lib/list";
import { DataTable } from "@/components/DataTable";
import { EditColumnsButton } from "@/components/EditColumns";
import { FilterField, ListShell } from "@/components/ListShell";
import { Modal } from "@/components/Modal";
import { Paginator } from "@/components/Paginator";
import { useColumnPrefs, type ColumnDef } from "@/lib/columnPrefs";
import {
  ErpField,
  InquiryHeader,
  InquirySection,
  ReviewTabBar,
  StatusPill,
} from "@/components/ApprovalReview";
import { ACTIVE_STATUS_PILLS } from "@/components/StatusPills";
import { ActiveIcon, Button, Select } from "@/components/ui";
import { formatDate } from "@/lib/format";
import type { Pagination } from "@/lib/types";
import type { EwuraLicense } from "@/lib/master";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";

export default function EwuraPage() {
  const list = useListQuery({ orderBy: "licensee", sortDirection: "ASC" });
  const [licenseClass, setClass] = useUrlFilter("licenseClass");
  const [active, setActive] = useUrlFilter("isActive", ["true", "false"]);
  const [tab, setTab] = useState<"license" | "particulars">("license");
  const [viewing, setViewing] = useState<EwuraLicense | null>(null);
  const [msg, setMsg] = useState("");
  const { hasPermission } = useAuth();
  const canSync = hasPermission(PERMISSIONS.ewuraManage);

  const query = useMemo(
    () => ({
      ...list.query,
      licenseClass: licenseClass || undefined,
      isActive: active === "" ? undefined : active === "true",
    }),
    [list.query, licenseClass, active],
  );

  const { data, loading, error, reload } = useAsync(
    () => api.get<Pagination<EwuraLicense>>("/ewura/licenses", query),
    [list.search, list.page, list.pageSize, list.orderBy, list.sortDirection, licenseClass, active],
  );
  const classes = useAsync(() => api.get<string[]>("/ewura/classes").catch(() => []), []);

  async function sync() {
    setMsg("");
    try {
      await api.post("/ewura/licenses/sync", {});
      setMsg("Licenses synced from EWURA");
      reload();
    } catch (e) {
      setMsg(formatApiError(e, "Sync failed"));
    }
  }

  const catalog: ColumnDef<EwuraLicense>[] = useMemo(
    () => [
      {
        id: "licenseNumber",
        header: "License",
        sortKey: "licenseNumber",
        defaultVisible: true,
        cell: (r) => (
          <span className="font-medium text-slate-800">{r.licenseNumber}</span>
        ),
      },
      {
        id: "licensee",
        header: "Licensee",
        sortKey: "licensee",
        defaultVisible: true,
        cell: (r) => <div className="font-medium text-slate-800">{r.licensee}</div>,
      },
      {
        id: "licenseClass",
        header: "Class",
        sortKey: "licenseClass",
        defaultVisible: true,
        cell: (r) => r.licenseClass || "—",
      },
      {
        id: "licenseType",
        header: "Type",
        sortKey: "licenseType",
        defaultVisible: false,
        cell: (r) => r.licenseType || "—",
      },
      {
        id: "regionName",
        header: "Region",
        sortKey: "regionName",
        defaultVisible: true,
        cell: (r) => r.regionName || "—",
      },
      {
        id: "districtName",
        header: "District",
        sortKey: "districtName",
        defaultVisible: false,
        cell: (r) => r.districtName || "—",
      },
      {
        id: "tinNumber",
        header: "TIN",
        sortKey: "tinNumber",
        defaultVisible: true,
        cell: (r) => r.tinNumber || "—",
      },
      {
        id: "expiryDate",
        header: "Expires",
        sortKey: "expiryDate",
        defaultVisible: true,
        className: "text-right",
        cell: (r) => formatDate(r.expiryDate),
      },
      {
        id: "isActive",
        header: "Active",
        sortKey: "isActive",
        defaultVisible: true,
        className: "w-24 text-center",
        cell: (r) => <ActiveIcon active={Boolean(r.isActive)} />,
      },
    ],
    [],
  );

  const { columns, pref, save, restoreDefaults } = useColumnPrefs(
    "ewura-licenses",
    catalog,
  );

  const classOptions = Array.isArray(classes.data) ? classes.data : [];

  return (
    <div>
      <ListShell
        title="EWURA petroleum licenses"
        subtitle="OMC, destination, and terminal licences. Inactive is expired (past expiry date). The 02:00 EWURA job syncs the register and marks expired rows; there is no edit modal because EWURA owns the data."
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <EditColumnsButton
              catalog={catalog}
              pref={pref}
              onApply={save}
              onRestore={restoreDefaults}
            />
            <ExportButton path="/ewura/licenses" query={query} />
            {canSync ? (
              <Button onClick={() => void sync()}>Sync from EWURA</Button>
            ) : null}
          </div>
        }
        lead={
          msg ? <p className="mb-3 text-sm text-slate-600">{msg}</p> : null
        }
        pills={{
          options: ACTIVE_STATUS_PILLS,
          value: active,
          onChange: (id) => {
            setActive(id);
            list.setPage(1);
          },
          allTitle: "All licenses",
          guideTitle: "Active = not expired. Inactive = expiry date has passed.",
        }}
        filters={
          <FilterField label="Class" className="w-56">
            <Select
              value={licenseClass}
              onChange={(e) => {
                setClass(e.target.value);
                list.setPage(1);
              }}
            >
              <option value="">All</option>
              {classOptions.filter(Boolean).map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </Select>
          </FilterField>
        }
        search={
          <ListSearch
            value={list.search}
            onChange={list.setSearch}
            placeholder="Number / licensee / TIN"
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
          columns={columns}
          rows={data?.items}
          loading={loading}
          error={error}
          emptyMessage={
            list.search || active || licenseClass
              ? "No licenses match the current filters"
              : "No licenses found"
          }
          sort={list.sort}
          onSort={list.onSort}
          onRowClick={(r) => {
            setTab("license");
            setViewing(r);
          }}
        />
      </ListShell>

      {viewing ? (
        <Modal
          open
          size="xl"
          title="License"
          onClose={() => setViewing(null)}
          footer={
            <Button variant="secondary" onClick={() => setViewing(null)}>
              Close
            </Button>
          }
        >
          <div className="space-y-4">
            <InquiryHeader
              crumb="Stock · Setups · EWURA · Inquiry"
              title={viewing.licensee || viewing.licenseNumber}
              pills={
                <StatusPill tone={viewing.isActive ? "navy" : "slate"}>
                  {viewing.isActive ? "Active" : "Inactive"}
                </StatusPill>
              }
            />
            <ReviewTabBar
              tab={tab}
              onChange={setTab}
              ids={[
                { id: "license", label: "License" },
                { id: "particulars", label: "Particulars" },
              ]}
              ariaLabel="License sections"
            />
            {tab === "license" ? (
              <InquirySection title="License">
                <ErpField label="Number" value={viewing.licenseNumber} />
                <ErpField label="Class" value={viewing.licenseClass} />
                <ErpField label="Type" value={viewing.licenseType} />
                <ErpField label="Sector" value={viewing.sector} />
                <ErpField label="Issued" value={formatDate(viewing.issueDate)} />
                <ErpField label="Expires" value={formatDate(viewing.expiryDate)} />
              </InquirySection>
            ) : null}
            {tab === "particulars" ? (
              <div className="grid gap-4 lg:grid-cols-2">
                <InquirySection title="Location" columns={1}>
                  <ErpField label="Zone" value={viewing.zoneName} />
                  <ErpField label="Region" value={viewing.regionName} />
                  <ErpField label="District" value={viewing.districtName} />
                </InquirySection>
                <InquirySection title="Contact" columns={1}>
                  <ErpField label="TIN" value={viewing.tinNumber} />
                  <ErpField label="Phone" value={viewing.phone} />
                  <ErpField label="Email" value={viewing.email} />
                </InquirySection>
              </div>
            ) : null}
          </div>
        </Modal>
      ) : null}
    </div>
  );
}
