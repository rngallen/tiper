"use client";

import { useCallback } from "react";
import { ActiveCell, MasterCatalogue } from "@/components/MasterCatalogue";
import type { ColumnDef } from "@/lib/columnPrefs";
import { PERMISSIONS } from "@/lib/permissions";
import type { Customer } from "@/lib/master";
import { CustomerForm, CustomerInquiry } from "./CustomerRecord";

export default function CustomersPage() {
  const columns = useCallback(
    () =>
      [
        {
          id: "code",
          header: "Code",
          sortKey: "code",
          defaultVisible: true,
          cell: (r) => (
            <span className="font-mono text-xs text-slate-600">{r.code}</span>
          ),
        },
        {
          id: "name",
          header: "Customer",
          sortKey: "name",
          defaultVisible: true,
          cell: (r) => <div className="font-medium text-slate-800">{r.name}</div>,
        },
        {
          id: "kycNumber",
          header: "KYC",
          sortKey: "kycNumber",
          defaultVisible: true,
          cell: (r) => r.kycNumber || "—",
        },
        {
          id: "ewuraLicense",
          header: "EWURA license",
          sortKey: "ewuraLicense",
          defaultVisible: true,
          cell: (r) => r.ewuraLicense || "—",
        },
        {
          id: "vrnNumber",
          header: "VRN",
          sortKey: "vrnNumber",
          defaultVisible: false,
          cell: (r) => r.vrnNumber || "—",
        },
        {
          id: "email",
          header: "Email",
          sortKey: "email",
          defaultVisible: true,
          cell: (r) => r.email || "—",
        },
        {
          id: "phone",
          header: "Phone",
          defaultVisible: true,
          cell: (r) => r.phone || "—",
        },
        {
          id: "tinNumber",
          header: "TIN",
          defaultVisible: false,
          cell: (r) => r.tinNumber || "—",
        },
        {
          id: "isActive",
          header: "Active",
          defaultVisible: true,
          cell: (r) => <ActiveCell active={r.isActive} />,
        },
      ] satisfies ColumnDef<Customer>[],
    [],
  );

  return (
    <MasterCatalogue<Customer>
      tableId="master-customers"
      title="Customers"
      subtitle="Oil marketing companies that store product at TIPER"
      path="/master/customers"
      orderBy="name"
      searchPlaceholder="Search name, code, KYC, license"
      createLabel="Add customer"
      emptyMessage="No customers found — add one to start vessel receipts and gantry loading"
      columns={columns}
      inquiryTitle={(r) => r.name || r.code || "Customer"}
      inquirySize="xl"
      inquiry={(r) => <CustomerInquiry row={r} />}
      Form={CustomerForm}
      createPerms={[PERMISSIONS.customersCreate]}
      updatePerms={[PERMISSIONS.customersUpdate]}
      deletePerms={[PERMISSIONS.customersManage]}
      noun="customer"
    />
  );
}
