import { PERMISSIONS } from "@/lib/permissions";

/** Settings tree: Settings → group → tab. Same shape as SAP SuccessFactors Admin Center. */

export type SettingsLeaf = {
  href: string;
  label: string;
  subtitle: string;
  /** Any of these codes. Default: settings.read / settings.manage. */
  perms?: string[];
};

export type SettingsGroup = {
  id: string;
  label: string;
  /** Path prefix that owns this group (`/settings/company`). */
  base: string;
  children: SettingsLeaf[];
};

export const SETTINGS_GROUPS: SettingsGroup[] = [
  {
    id: "company",
    label: "Company",
    base: "/settings/company",
    children: [
      {
        href: "/settings/company",
        label: "Profile",
        subtitle: "Letterhead and registration details used on documents",
      },
      {
        href: "/settings/company/attachments",
        label: "Attachments",
        subtitle: "Storage folder, upload size, and count caps (process limit 64 MB)",
      },
      {
        href: "/settings/company/currencies",
        label: "Currencies",
        subtitle: "ISO currencies used on billing and reports",
      },
    ],
  },
  // Add WhatsApp here when the channel ships.
  {
    id: "notifications",
    label: "Notifications",
    base: "/settings/notifications",
    children: [
      {
        href: "/settings/notifications/mail",
        label: "Mail",
        subtitle: "Outbound SMTP — save credentials, then send a test email",
      },
      {
        href: "/settings/notifications/sms",
        label: "SMS",
        subtitle: "SMS gateway for OTP and alerts",
      },
    ],
  },
  {
    id: "databases",
    label: "Databases",
    base: "/settings/databases",
    children: [
      {
        href: "/settings/databases/sage",
        label: "Sage 200",
        subtitle: "Sage 200 database — save and test without restarting the API",
      },
    ],
  },
  {
    id: "operations",
    label: "Operations",
    base: "/settings/operations",
    children: [
      {
        href: "/settings/operations/session",
        label: "Session",
        subtitle: "Sign-out after inactivity — applies without restarting the API",
      },
      {
        href: "/settings/operations/ewura",
        label: "EWURA",
        subtitle: "Petroleum license register URL and NPGIS retailer API",
      },
      {
        href: "/settings/operations/precision",
        label: "Precision",
        subtitle: "Decimal places for litres, m³, tonnes, density, and money",
      },
      {
        href: "/settings/operations/schedules",
        label: "Schedules",
        subtitle: "EWURA sync, nth/TBS/VSF billing, and log rotation",
        perms: [
          PERMISSIONS.settingsRead,
          PERMISSIONS.settingsManage,
          PERMISSIONS.jobsRun,
        ],
      },
    ],
  },
];

const SETTINGS_VIEW = [PERMISSIONS.settingsRead, PERMISSIONS.settingsManage];

export function settingsLeafPerms(leaf: SettingsLeaf): string[] {
  return leaf.perms ?? SETTINGS_VIEW;
}

export function settingsLeafVisible(
  leaf: SettingsLeaf,
  has: (...codes: string[]) => boolean,
): boolean {
  return has(...settingsLeafPerms(leaf));
}

export function visibleSettingsGroups(
  has: (...codes: string[]) => boolean,
): SettingsGroup[] {
  return SETTINGS_GROUPS.map((g) => ({
    ...g,
    children: g.children.filter((c) => settingsLeafVisible(c, has)),
  })).filter((g) => g.children.length > 0);
}

export function firstAllowedSettingsHref(
  has: (...codes: string[]) => boolean,
): string {
  return (
    visibleSettingsGroups(has)[0]?.children[0]?.href ??
    "/settings/operations/schedules"
  );
}

export function firstAllowedGroupHref(
  groupId: string,
  has: (...codes: string[]) => boolean,
): string {
  const g = visibleSettingsGroups(has).find((x) => x.id === groupId);
  return g?.children[0]?.href ?? firstAllowedSettingsHref(has);
}

export function matchSettings(
  pathname: string,
  groups: SettingsGroup[] = SETTINGS_GROUPS,
): {
  group: SettingsGroup;
  leaf: SettingsLeaf;
} {
  const group =
    groups.find(
      (g) => pathname === g.base || pathname.startsWith(`${g.base}/`),
    ) ??
    groups[0] ??
    SETTINGS_GROUPS[0];
  const byLength = [...group.children].sort(
    (a, b) => b.href.length - a.href.length,
  );
  const leaf =
    byLength.find(
      (c) => pathname === c.href || pathname.startsWith(`${c.href}/`),
    ) ?? group.children[0];
  return { group, leaf };
}
