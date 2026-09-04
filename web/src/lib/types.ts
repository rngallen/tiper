// Shared API types mirroring the Go backend JSON shapes.

export interface ApiEnvelope<T> {
  message?: string;
  details?: T;
}

export interface Pagination<T> {
  items: T[];
  page: number;
  pageSize: number;
  totalPages: number;
  itemsCount: number;
  nextPage?: number;
  previousPage?: number;
}

export interface TokenResponse {
  email: string;
  name: string;
  issuedAt: string;
  /** Omitted for browser logins — access token is an HttpOnly cookie. */
  token?: string;
  tokenExpiresAt: string;
  /** Omitted for browser logins — refresh token is an HttpOnly cookie. */
  refreshToken?: string;
  refreshTokenExpiresAt: string;
  mustChangePassword: boolean;
  sessionIdleMinutes?: number;
  sessionIdleWarnSeconds?: number;
}

/** OTP challenge from login or /auth/mfa/resend. */
export interface OtpPending {
  message: string;
  /**
   * MFA bearer for /mfa/verify and /mfa/resend. JTI = challenge id — after
   * resend, replace this with the new token from the response.
   */
  token: string;
  issuedAt: string;
  /** When the 6-digit code expires (5 minutes). */
  expiredAt: string;
  /** Earliest time /mfa/resend is allowed (issuedAt + cooldown). */
  resendAvailableAt: string;
}

export interface TableColumnPref {
  order?: string[];
  hidden?: string[];
}

export interface AppearanceSettings {
  theme?: "light" | "dark" | "system" | string;
  compactMode?: boolean;
  largeText?: boolean;
  sidebarState?: boolean;
  /** Per-table column order / visibility. */
  tablePrefs?: Record<string, TableColumnPref>;
  dashboard?: DashboardWidget[];
}

export interface DashboardWidget {
  id: string;
  type: "kpi" | "bar" | "line" | "doughnut" | "pie" | "table";
  title: string;
  source: string;
  span?: 1 | 2;
}

export interface ChartCatalogItem {
  code: string;
  name: string;
  types: DashboardWidget["type"][];
  description: string;
}

export interface ChartPayload {
  source: string;
  title: string;
  unit?: string;
  points: { label: string; value: number }[];
}

export interface UserProfile {
  phoneNumber?: string;
  title?: string;
  appearanceSettings?: AppearanceSettings;
}

export interface User {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  isActive?: boolean;
  isLocked?: boolean;
  isSuperUser?: boolean;
  mustChangePassword?: boolean;
  twoFactorEnabled?: boolean;
  lastLogin?: string;
  /** True when the API allows hard-delete (never signed in, not self/super-user). */
  canDelete?: boolean;
  roles?: Role[];
  profile?: UserProfile;
}

export interface Permission {
  id: number;
  code: string;
  description: string;
  module: string;
}

export interface Role {
  id: number;
  name: string;
  description: string;
  /** system | terminal | gantry | finance | billing */
  category?: string;
  permissions?: Permission[];
  /** Count from list/options; detail payloads also set this. */
  permissionCount?: number;
  /** True when unused (not assigned to users, not a workflow step operator). */
  canDelete?: boolean;
}

/** Job title from the titles catalogue (`/auth/titles`). */
export interface Title {
  id: string;
  name: string;
  /** True once assigned to a user profile — cannot be deleted. */
  hasData?: boolean;
}

// MeResponse is the shape returned by GET /auth/profile.
export interface MeResponse {
  user: User;
  permissions: string[];
  isSuperUser: boolean;
  sessionIdleMinutes?: number;
  sessionIdleWarnSeconds?: number;
}

export interface Country {
  code: string;
  name: string;
  alpha3: string;
  numeric: string;
  isActive: boolean;
}

export interface Currency {
  code: string;
  name: string;
  symbol: string;
  isActive: boolean;
}

export interface Company {
  name: string;
  tinNumber: string;
  vrnNumber: string;
  isoNumber: string;
  address: string;
  address2: string;
  city: string;
  country: string;
  postalCode: string;
  phone: string;
  email: string;
  website: string;
  /**
   * Public SPA base URL for email CTAs. No trailing slash.
   */
  portalUrl?: string;
  /** ISO currency code; null/empty until set once. */
  currencyCode?: string | null;
}

/** GET /settings/integrations/mail — password never returned. */
export interface MailIntegrationSettings {
  enabled: boolean;
  host: string;
  port: number;
  user: string;
  fromName: string;
  fromEmail: string;
  useTLS: boolean;
  useSSL: boolean;
  hasPassword: boolean;
}

/** GET /settings/integrations/sms — API key never returned. */
export interface SMSIntegrationSettings {
  enabled: boolean;
  apiUrl: string;
  senderId: string;
  hasApiKey: boolean;
}

/** GET /settings/integrations/sage — password never returned. */
export interface SageIntegrationSettings {
  host: string;
  port: string;
  instance: string;
  user: string;
  name: string;
  encrypt: boolean;
  hasPassword: boolean;
  connected: boolean;
}

/** GET /settings/integrations/npgis — EWURA license register + NPGIS. */
export interface NpgisIntegrationSettings {
  enabled: boolean;
  licenseUrl: string;
  baseUrl: string;
  licenseNo: string;
  apiSourceId: string;
  depotName: string;
}

/** GET /settings/integrations/session */
export interface SessionIntegrationSettings {
  idleMinutes: number;
  warnMinutes: number;
  warnSeconds: number;
}

/** GET /settings/integrations/uploads */
export interface UploadsIntegrationSettings {
  directory: string;
  maxFileSizeMB: number;
  maxFilesPerRequest: number;
  processBodyLimitMB: number;
}

/** GET /settings/precision — rounding (key=precision) plus ILO days (key=orders). */
export interface DecimalPrecisionSettings {
  quantityPrecision: number;
  cubicMeterPrecision: number;
  metricTonnePrecision: number;
  densityPrecision: number;
  pricePrecision: number;
  miLossPrecision: number;
  iloExpiryDays: number;
}

/** GET /settings/schedules — robfig cron specs (with seconds). */
export interface ScheduleSettings {
  logRotation: string;
  ewuraLicenses: string;
  billingNth: string;
  billingTbs: string;
  billingVcf: string;
  ewuraNpgis: string;
  iloExpire: string;
  notifyOutbox: string;
}

export interface ReportMeta {
  code: string;
  name: string;
  group: string;
  description: string;
  filters: string[];
  href?: string;
}

export interface WorkflowTask {
  taskId: string;
  instanceId: string;
  no: string;
  summary: string;
  nodeName: string;
  /** draft = Initiator / soft-reject amendment step */
  nodeStatus?: string;
  docContentType: number;
  createdAt: string;
  onBehalfOfName?: string;
  docId?: string;
  documentNumber?: string;
  amount?: string;
  currencyCode?: string;
  customerCode?: string;
  customerName?: string;
}

/** Header-bell snapshot from GET /workflow/tasks/inbox. */
export interface WorkflowInbox {
  count: number;
  items: WorkflowTask[];
}

/** Past agree/reject decision by the signed-in user. */
export interface WorkflowDecision {
  instanceId: string;
  no: string;
  summary: string;
  actType: string;
  actName: string;
  comment: string;
  totalRejection: boolean;
  fromNode?: string;
  toNode?: string;
  instanceStatus: string;
  docContentType?: number;
  createdAt: string;
  docId?: string;
}

export interface WorkflowEvent {
  actType: string;
  actName: string;
  comment: string;
  totalRejection: boolean;
  oldNode?: string;
  newNode?: string;
  userEmail?: string;
  userName?: string;
  userTitle?: string;
  onBehalfOfName?: string;
  onBehalfOfEmail?: string;
  ipAddress?: string;
  createdAt: string;
}

export interface AuditFieldChange {
  before: unknown;
  after: unknown;
}

export interface AuditEntry {
  id: string;
  userName: string;
  module: string;
  action: string;
  description: string;
  changes?: Record<string, AuditFieldChange>;
  ipAddress: string;
  createdAt: string;
}

