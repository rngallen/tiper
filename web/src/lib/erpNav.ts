import { PERMISSIONS } from "@/lib/permissions";

/** One screen in a module area (Stock › Gantry › Internal loading). */
export type WorkspaceLeaf = {
  href: string;
  label: string;
  perms: string[];
};

/** Second stage: Reception / Gantry / Pump-over / Terminal / Reports / Setups. */
export type WorkspaceArea = {
  id: string;
  label: string;
  children: WorkspaceLeaf[];
};

export type WorkspaceModule = {
  id: string;
  label: string;
  areas: WorkspaceArea[];
};

const masterRead = [PERMISSIONS.masterdataRead];
const reportRead = [PERMISSIONS.reportsRead];
const financeSetupRead = [PERMISSIONS.billingRead, PERMISSIONS.pricesRead];
const balanceRead = [
  PERMISSIONS.inventoryRead,
  PERMISSIONS.inventoryBalances,
  PERMISSIONS.reportsRead,
];

/** Stock work is split by where the operator stands: jetty, gantry, pump-over, or terminal. */
export const STOCK_MODULE: WorkspaceModule = {
  id: "stock",
  label: "Stock",
  areas: [
    {
      id: "reception",
      label: "Reception",
      children: [
        {
          href: "/stock/receipts",
          label: "Internal receipts",
          perms: [PERMISSIONS.receiptsRead],
        },
        {
          href: "/stock/receipts/external",
          label: "External receipts",
          perms: [PERMISSIONS.receiptsRead],
        },
      ],
    },
    {
      id: "gantry",
      label: "Gantry",
      children: [
        {
          href: "/stock/loading",
          label: "Internal loading",
          perms: [PERMISSIONS.ilrRead],
        },
        {
          href: "/stock/compartmentalization",
          label: "Compartmentalization",
          perms: [PERMISSIONS.compartmentalizationRead],
        },
        {
          href: "/stock/amendments",
          label: "Amendments",
          perms: [PERMISSIONS.amendmentRead],
        },
        {
          href: "/stock/deliveries",
          label: "Gantry completion",
          perms: [PERMISSIONS.ilrComplete],
        },
      ],
    },
    {
      id: "pump-over",
      label: "Pump-over",
      children: [
        {
          href: "/stock/pump-over",
          label: "Pump-over",
          perms: [PERMISSIONS.pumpOverRead],
        },
        {
          href: "/stock/pump-over-reports",
          label: "Pump-over reports",
          perms: [PERMISSIONS.pumpReportRead],
        },
      ],
    },
    {
      id: "terminal",
      label: "Terminal",
      children: [
        {
          href: "/stock/itt",
          label: "In-tank transfers",
          perms: [PERMISSIONS.ittRead],
        },
        {
          href: "/stock/zerolization",
          label: "Zerolization",
          perms: [PERMISSIONS.zerolizationRead],
        },
        {
          href: "/stock/financial-hold",
          label: "Financial hold",
          perms: [PERMISSIONS.holdRead],
        },
      ],
    },
    {
      id: "reports",
      label: "Reports",
      children: [
        {
          href: "/stock/balances",
          label: "Stock balances",
          perms: balanceRead,
        },
        {
          href: "/stock/movements",
          label: "Movements",
          perms: balanceRead,
        },
        {
          href: "/stock/reports",
          label: "Report catalog",
          perms: reportRead,
        },
      ],
    },
    {
      id: "setups",
      label: "Setups",
      children: [
        { href: "/stock/setups/customers", label: "Customers", perms: [PERMISSIONS.customersRead] },
        {
          href: "/ewura",
          label: "EWURA licenses",
          perms: [PERMISSIONS.ewuraRead, PERMISSIONS.ewuraManage],
        },
        { href: "/stock/setups/suppliers", label: "Suppliers", perms: masterRead },
        { href: "/stock/setups/categories", label: "Product categories", perms: masterRead },
        { href: "/stock/setups/products", label: "Products", perms: masterRead },
        { href: "/stock/setups/tanks", label: "Tanks", perms: masterRead },
        { href: "/stock/setups/vessels", label: "Vessels", perms: masterRead },
        { href: "/stock/setups/depots", label: "Depots", perms: masterRead },
        { href: "/stock/setups/trucks", label: "Trucks", perms: masterRead },
        { href: "/stock/setups/drivers", label: "Drivers", perms: masterRead },
        {
          href: "/stock/setups/transporters",
          label: "Haulers",
          perms: masterRead,
        },
        {
          href: "/stock/setups/destinations",
          label: "Destinations",
          perms: masterRead,
        },
        {
          href: "/stock/setups/districts",
          label: "Districts",
          perms: masterRead,
        },
        {
          href: "/stock/setups/statuses",
          label: "Stock statuses",
          perms: masterRead,
        },
      ],
    },
  ],
};

export const BILLING_MODULE: WorkspaceModule = {
  id: "billing",
  label: "Billing",
  areas: [
    {
      id: "transactions",
      label: "Transactions",
      children: [
        {
          href: "/billing",
          label: "Billing runs",
          perms: [PERMISSIONS.billingRead],
        },
        {
          href: "/billing/change-of-service",
          label: "Change of service",
          perms: [PERMISSIONS.billingRead, PERMISSIONS.changeOfServiceRead],
        },
      ],
    },
    {
      id: "reports",
      label: "Reports",
      children: [
        {
          href: "/finance/reports",
          label: "Billing reports",
          perms: reportRead,
        },
      ],
    },
    {
      id: "setups",
      label: "Setups",
      children: [
        {
          href: "/finance/setups/fsf",
          label: "Fixed storage fees",
          perms: financeSetupRead,
        },
        {
          href: "/finance/setups/vsf",
          label: "Variable storage fees",
          perms: financeSetupRead,
        },
        {
          href: "/finance/setups/koj",
          label: "KOJ fees",
          perms: financeSetupRead,
        },
        {
          href: "/finance/setups/tbs",
          label: "TBS fees",
          perms: financeSetupRead,
        },
        {
          href: "/finance/setups/fx",
          label: "Exchange rates",
          perms: financeSetupRead,
        },
        {
          href: "/finance/setups/miloss",
          label: "MI loss",
          perms: financeSetupRead,
        },
        {
          href: "/finance/setups/tenders",
          label: "Tenders",
          perms: [...masterRead, ...financeSetupRead],
        },
        {
          href: "/finance/setups/routes",
          label: "Discharge routes",
          perms: [...masterRead, ...financeSetupRead],
        },
        {
          href: "/finance/setups/delivery",
          label: "Delivery methods",
          perms: [...masterRead, ...financeSetupRead],
        },
        {
          href: "/finance/setups/procurement",
          label: "Procurement",
          perms: [...masterRead, ...financeSetupRead],
        },
        {
          href: "/finance/setups/contracts",
          label: "Contract types",
          perms: [...masterRead, ...financeSetupRead],
        },
        {
          href: "/finance/setups/pricing",
          label: "Pricing natures",
          perms: [...masterRead, ...financeSetupRead],
        },
        {
          href: "/finance/setups/cycles",
          label: "Billing cycles",
          perms: [...masterRead, ...financeSetupRead],
        },
      ],
    },
  ],
};
