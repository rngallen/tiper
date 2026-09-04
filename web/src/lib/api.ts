// Typed API client for the TIPER DFMS backend.
//
// Auth secrets live in HttpOnly cookies (set by the API). This module only
// keeps non-secret session metadata (access expiry + cached user), sends
// credentials on every request, and attaches X-Csrf-Token on unsafe methods.

import type { ApiEnvelope, TokenResponse } from "./types";
import { notify } from "./notify";

// Same-origin /api/v1 via Next rewrite (dev) or Nginx (prod). Override only if
// you intentionally call the API on another origin.
const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE || "/api/v1";

const USER_KEY = "dfms.user";
const ACCESS_EXP_KEY = "dfms.accessExp";
/** Must match pkg/types/cookies.go CsrfCookieName. */
const CSRF_COOKIE = "dfms_csrf";

/** Short-lived access token in tab memory + sessionStorage (never localStorage). */
const ACCESS_MEM_KEY = "dfms.accessMem";
let accessTokenMemory: string | null = null;

function setAccessTokenMemory(token: string | null) {
  accessTokenMemory = token || null;
  if (typeof window === "undefined") return;
  try {
    if (token) sessionStorage.setItem(ACCESS_MEM_KEY, token);
    else sessionStorage.removeItem(ACCESS_MEM_KEY);
  } catch {
    /* private mode / quota */
  }
}

function getAccessTokenMemory(): string | null {
  if (accessTokenMemory) return accessTokenMemory;
  if (typeof window === "undefined") return null;
  try {
    accessTokenMemory = sessionStorage.getItem(ACCESS_MEM_KEY);
  } catch {
    accessTokenMemory = null;
  }
  return accessTokenMemory;
}

export const tokenStore = {
  /** True when a non-secret access expiry is still in the future (or a token is in memory). */
  get hasSession(): boolean {
    return this.accessExpiresAt > Date.now() || Boolean(accessTokenMemory);
  },
  get accessExpiresAt(): number {
    if (typeof window === "undefined") return 0;
    const raw = localStorage.getItem(ACCESS_EXP_KEY);
    return raw ? Number(raw) : 0;
  },
  /** Persist non-secret session metadata after login/refresh. */
  setSession(accessExpiresAt?: string, accessToken?: string) {
    if (accessToken) setAccessTokenMemory(accessToken);
    if (accessExpiresAt) {
      const ms = Date.parse(accessExpiresAt);
      if (!Number.isNaN(ms)) localStorage.setItem(ACCESS_EXP_KEY, String(ms));
    }
  },
  setUser(user: unknown) {
    if (user === undefined || user === null) {
      localStorage.removeItem(USER_KEY);
      return;
    }
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  },
  getUser<T>(): T | null {
    if (typeof window === "undefined") return null;
    const raw = localStorage.getItem(USER_KEY);
    if (!raw || raw === "undefined" || raw === "null") return null;
    try {
      return JSON.parse(raw) as T;
    } catch {
      localStorage.removeItem(USER_KEY);
      return null;
    }
  },
  clear() {
    setAccessTokenMemory(null);
    localStorage.removeItem(USER_KEY);
    localStorage.removeItem(ACCESS_EXP_KEY);
  },
};

export class ApiError extends Error {
  status: number;
  details?: unknown;
  constructor(message: string, status: number, details?: unknown) {
    super(message);
    this.status = status;
    this.details = details;
  }
}

/** Field-level validation item from a 422 Unprocessable Entity response. */
interface ValidationDetail {
  field?: string;
  error?: string;
}

export function formatApiError(err: unknown, fallback = "Request failed"): string {
  if (!(err instanceof ApiError)) {
    return err instanceof Error ? err.message : fallback;
  }
  const parts = validationMessages(err.details);
  if (parts.length > 0) return parts.join("; ");
  return err.message || fallback;
}

function validationMessages(details: unknown): string[] {
  if (!Array.isArray(details)) return [];
  const out: string[] = [];
  for (const item of details) {
    if (!item || typeof item !== "object") continue;
    const d = item as ValidationDetail;
    const msg = typeof d.error === "string" ? d.error.trim() : "";
    if (!msg) continue;
    const field = typeof d.field === "string" ? d.field.trim() : "";
    out.push(field ? `${field}: ${msg}` : msg);
  }
  return out;
}

function readCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const parts = `; ${document.cookie}`.split(`; ${name}=`);
  if (parts.length < 2) return null;
  const value = parts.pop()?.split(";").shift();
  return value ? decodeURIComponent(value) : null;
}

/** Ensure a CSRF cookie/token exists for unsafe requests (login, saves, etc.). */
export async function ensureCsrf(): Promise<string | null> {
  const token = readCookie(CSRF_COOKIE);
  if (token) return token;
  try {
    const res = await fetch(`${API_BASE}/auth/csrf`, {
      method: "GET",
      credentials: "include",
    });
    if (res.ok) {
      try {
        const data = (await res.json()) as ApiEnvelope<{ csrfToken?: string }>;
        if (data.details?.csrfToken) return data.details.csrfToken;
      } catch {
        /* fall through to cookie */
      }
    }
  } catch {
    return null;
  }
  return readCookie(CSRF_COOKIE);
}

function isUnsafeMethod(method: string): boolean {
  const m = method.toUpperCase();
  return m !== "GET" && m !== "HEAD" && m !== "OPTIONS" && m !== "TRACE";
}

function errorDialogTitle(status: number): string {
  if (status === 403) return "Not allowed";
  if (status === 404) return "Not found";
  if (status === 409) return "Cannot save";
  if (status === 422) return "Check the values";
  if (status >= 500) return "Server error";
  return "Unable to complete";
}

function shouldNotifyError(
  path: string,
  method: string,
  status: number,
  notifyOpt?: boolean,
): boolean {
  if (notifyOpt === false) return false;
  if (status === 401) return false;
  if (!isUnsafeMethod(method) && notifyOpt !== true) return false;
  const p = path.split("?")[0];
  if (
    p.includes("/auth/login") ||
    p.includes("/auth/mfa/") ||
    p.includes("/auth/session/") ||
    p.includes("/auth/csrf") ||
    p.includes("/auth/refresh") ||
    p.includes("/auth/logout") ||
    p.includes("/auth/profile/table-prefs") ||
    p.includes("/auth/profile/dashboard") ||
    p.endsWith("/refresh-stock")
  ) {
    return false;
  }
  return true;
}

function reportApiError(
  path: string,
  method: string,
  err: ApiError,
  notifyOpt?: boolean,
) {
  if (!shouldNotifyError(path, method, err.status, notifyOpt)) return;
  notify.error(err.message, { title: errorDialogTitle(err.status) });
}

function envelopeSuccessMessage(
  payload: ApiEnvelope<unknown> | null | undefined,
): string | undefined {
  const top = payload?.message?.trim();
  if (top) return top;
  const details = payload?.details;
  if (details && typeof details === "object" && details !== null && "message" in details) {
    const nested = (details as { message?: unknown }).message;
    if (typeof nested === "string" && nested.trim()) return nested.trim();
  }
  return undefined;
}

function shouldToastSuccess(
  path: string,
  method: string,
  envelopeMessage: string | undefined,
  success?: false,
): string | null {
  if (success === false) return null;
  if (!isUnsafeMethod(method)) return null;
  const p = path.split("?")[0];
  if (
    p.includes("/auth/login") ||
    p.includes("/auth/mfa/") ||
    p.includes("/auth/session/") ||
    p.includes("/auth/csrf") ||
    p.includes("/auth/refresh") ||
    p.includes("/auth/logout") ||
    p.includes("/auth/profile/table-prefs") ||
    p.includes("/auth/profile/dashboard") ||
    p.endsWith("/refresh-stock")
  ) {
    return null;
  }
  const message = envelopeMessage?.trim();
  return message || null;
}

function reportApiSuccess(
  path: string,
  method: string,
  envelopeMessage: string | undefined,
  success?: false,
) {
  const message = shouldToastSuccess(path, method, envelopeMessage, success);
  if (message) notify.success(message);
}

/** True while a forced login redirect is in flight (avoids redirect loops). */
let redirectingToLogin = false;
let sessionRejected = false;

export const SESSION_REJECTED_EVENT = "dfms:session-rejected";

function forceLoginRedirect(): void {
  if (typeof window === "undefined") return;
  const path = window.location.pathname;
  if (path === "/login" || path.startsWith("/login/")) return;
  if (sessionRejected || redirectingToLogin) return;
  sessionRejected = true;
  // Let AuthProvider show the idle warning. Hard navigation skipped the
  // countdown and looked like "session ended" while the operator was working.
  window.dispatchEvent(new CustomEvent(SESSION_REJECTED_EVENT));
}

export function clearSessionRejected(): void {
  sessionRejected = false;
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  auth?: boolean; // cookie session expected (default true)
  token?: string; // explicit bearer (OTP challenge)
  query?: Record<string, string | number | boolean | undefined>;
  /** When false, a successful call does not reset the idle-logout timer.
   *  Default: true for POST/PUT/PATCH/DELETE, false for GET (list polls). */
  countAsActivity?: boolean;
  /** When false, a 401 does not bounce to /login (session probe on boot). Default true. */
  redirectOn401?: boolean;
  /** When false, skip the error message box. Default: true for POST/PUT/PATCH/DELETE. */
  notify?: boolean;
  /** When false, skip the success toast. The toast text comes from the API envelope. */
  success?: false;
}

let refreshing: Promise<boolean> | null = null;

/** Listeners notified on successful API traffic (keeps idle timer honest). */
const activityListeners = new Set<() => void>();

export function onApiActivity(listener: () => void): () => void {
  activityListeners.add(listener);
  return () => {
    activityListeners.delete(listener);
  };
}

function bumpApiActivity() {
  activityListeners.forEach((fn) => {
    try {
      fn();
    } catch {
      /* ignore */
    }
  });
}

/** Access still usable for > leadMs — another tab may have just refreshed. */
function accessStillFresh(leadMs = 60_000): boolean {
  const exp = tokenStore.accessExpiresAt;
  return exp > 0 && exp - Date.now() > leadMs;
}

async function doRefreshRequest(): Promise<boolean> {
  // After waiting for a cross-tab lock, cookies/localStorage may already be new.
  if (accessStillFresh()) {
    setAccessTokenMemory(null);
    return true;
  }
  try {
    const csrf = await ensureCsrf();
    const headers: Record<string, string> = {};
    if (csrf) headers["X-Csrf-Token"] = csrf;
    const res = await fetch(`${API_BASE}/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers,
    });
    if (!res.ok) return false;
    const data = (await res.json()) as ApiEnvelope<TokenResponse>;
    if (data.details?.tokenExpiresAt) {
      tokenStore.setSession(data.details.tokenExpiresAt, data.details.token);
      sessionRejected = false;
      return true;
    }
    // Cookies rotated even if body shape differs — treat 200 as success.
    setAccessTokenMemory(null);
    sessionRejected = false;
    return true;
  } catch {
    return false;
  }
}

async function tryRefresh(): Promise<boolean> {
  const run = async (): Promise<boolean> => {
    if (!refreshing) {
      refreshing = doRefreshRequest().finally(() => {
        refreshing = null;
      });
    }
    return refreshing;
  };

  // Serialize refresh across tabs so two single-use rotations cannot race.
  const locks = typeof navigator !== "undefined" ? navigator.locks : undefined;
  if (locks?.request) {
    try {
      return await locks.request("dfms-auth-refresh", run);
    } catch {
      return run();
    }
  }
  return run();
}

export function refreshTokens(): Promise<boolean> {
  return tryRefresh();
}

/** Slide backend LastSeen so Continue / live use stay in sync with idle. */
export async function touchSession(): Promise<"ok" | "auth" | "error"> {
  try {
    await apiRequest("/auth/session/touch", {
      method: "POST",
      countAsActivity: false,
      redirectOn401: false,
      notify: false,
    });
    return "ok";
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return "auth";
    return "error";
  }
}

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  const full = path.startsWith("http") ? path : API_BASE + path;
  if (!query) return full;
  const usp = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== null && v !== "") usp.set(k, String(v));
  }
  const qs = usp.toString();
  return qs ? `${full}?${qs}` : full;
}

export async function apiRequest<T>(
  path: string,
  opts: RequestOptions = {},
): Promise<T> {
  const {
    method = "GET",
    body,
    auth = true,
    token,
    query,
    redirectOn401 = true,
    countAsActivity = isUnsafeMethod(method),
    notify: notifyOpt,
    success,
  } = opts;

  const doFetch = async (): Promise<Response> => {
    const headers: Record<string, string> = {};
    if (body !== undefined) headers["Content-Type"] = "application/json";
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    } else {
      // Prefer in-memory access (set at login) so the first /profile after OTP
      // succeeds even if HttpOnly cookies land a moment later. Cookies remain
      // the durable session via credentials: "include".
      const mem = auth ? getAccessTokenMemory() : null;
      if (mem) headers["Authorization"] = `Bearer ${mem}`;
    }
    if (isUnsafeMethod(method)) {
      const csrf = readCookie(CSRF_COOKIE) || (await ensureCsrf());
      if (csrf) headers["X-Csrf-Token"] = csrf;
    }
    return fetch(buildUrl(path, query), {
      method,
      headers,
      credentials: "include",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  };

  let res = await doFetch();

  // Session rejected — drop stale in-memory Bearer (other tabs may have
  // rotated cookies), refresh once, then retry with cookies/new access.
  if (res.status === 401 && auth && !token) {
    setAccessTokenMemory(null);
    const ok = await tryRefresh();
    if (ok) {
      res = await doFetch();
    }
    if (!ok || res.status === 401) {
      if (redirectOn401) forceLoginRedirect();
      throw new ApiError("Session expired. Please sign in again.", 401);
    }
  }

  let payload: ApiEnvelope<T> | null = null;
  const text = await res.text();
  if (text) {
    try {
      payload = JSON.parse(text) as ApiEnvelope<T>;
    } catch {
      payload = null;
    }
  }

  if (!res.ok) {
    const envelopeMessage =
      payload?.message || res.statusText || `Request failed (${res.status})`;
    const details = payload?.details;
    const parts = validationMessages(details);
    const err = new ApiError(
      parts.length > 0 ? parts.join("; ") : envelopeMessage,
      res.status,
      details,
    );
    reportApiError(path, method, err, notifyOpt);
    throw err;
  }

  if (auth && countAsActivity) bumpApiActivity();
  reportApiSuccess(path, method, envelopeSuccessMessage(payload), success);
  return (payload?.details as T) ?? (undefined as T);
}

type BlobResult = {
  blob: Blob;
  url: string;
  filename: string;
  contentType: string;
};

/** Fetch a binary response as a blob URL (caller must revokeObjectURL). */
async function apiFetchBlob(
  path: string,
  query?: RequestOptions["query"],
): Promise<BlobResult> {
  const headers: Record<string, string> = {};
  const mem = getAccessTokenMemory();
  if (mem) headers["Authorization"] = `Bearer ${mem}`;

  let res = await fetch(buildUrl(path, query), {
    method: "GET",
    headers,
    credentials: "include",
  });

  if (res.status === 401) {
    setAccessTokenMemory(null);
    delete headers["Authorization"];
    const ok = await tryRefresh();
    if (ok) {
      const mem2 = getAccessTokenMemory();
      if (mem2) headers["Authorization"] = `Bearer ${mem2}`;
      res = await fetch(buildUrl(path, query), {
        method: "GET",
        headers,
        credentials: "include",
      });
    }
    if (!ok || res.status === 401) {
      forceLoginRedirect();
      throw new ApiError("Session expired. Please sign in again.", 401);
    }
  }

  if (!res.ok) {
    let message = res.statusText || `Download failed (${res.status})`;
    try {
      const payload = (await res.json()) as ApiEnvelope<unknown>;
      if (payload?.message) message = payload.message;
    } catch {
      /* binary / empty */
    }
    const err = new ApiError(message, res.status);
    reportApiError(path, "GET", err, true);
    throw err;
  }

  bumpApiActivity();
  const blob = await res.blob();
  const cd = res.headers.get("Content-Disposition") || "";
  const match = /filename\*?=(?:UTF-8''|")?([^\";]+)/i.exec(cd);
  const filename = match
    ? decodeURIComponent(match[1].replace(/"/g, ""))
    : "download.bin";
  const contentType =
    res.headers.get("Content-Type") || blob.type || "application/octet-stream";
  return {
    blob,
    url: URL.createObjectURL(blob),
    filename,
    contentType,
  };
}

/** Download a binary response and trigger a browser save. */
async function apiDownload(
  path: string,
  query?: RequestOptions["query"],
): Promise<void> {
  const { url, filename } = await apiFetchBlob(path, query);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}


/** Multipart upload (FormData). Do not set Content-Type — browser sets the boundary. */
async function apiUpload<T>(
  path: string,
  formData: FormData,
  extra?: Pick<RequestOptions, "notify" | "success">,
): Promise<T> {
  const doFetch = async (): Promise<Response> => {
    const headers: Record<string, string> = {};
    const mem = getAccessTokenMemory();
    if (mem) headers["Authorization"] = `Bearer ${mem}`;
    const csrf = readCookie(CSRF_COOKIE) || (await ensureCsrf());
    if (csrf) headers["X-Csrf-Token"] = csrf;
    return fetch(buildUrl(path), {
      method: "POST",
      headers,
      credentials: "include",
      body: formData,
    });
  };

  let res = await doFetch();
  if (res.status === 401) {
    const ok = await tryRefresh();
    if (ok) res = await doFetch();
    if (!ok || res.status === 401) {
      forceLoginRedirect();
      throw new ApiError("Session expired. Please sign in again.", 401);
    }
  }

  let payload: ApiEnvelope<T> | null = null;
  const textBody = await res.text();
  if (textBody) {
    try {
      payload = JSON.parse(textBody) as ApiEnvelope<T>;
    } catch {
      payload = null;
    }
  }
  if (!res.ok) {
    const envelopeMessage =
      payload?.message || res.statusText || `Upload failed (${res.status})`;
    const details = payload?.details;
    const parts = validationMessages(details);
    const err = new ApiError(
      parts.length > 0 ? parts.join("; ") : envelopeMessage,
      res.status,
      details,
    );
    reportApiError(path, "POST", err, extra?.notify);
    throw err;
  }
  bumpApiActivity();
  reportApiSuccess(path, "POST", envelopeSuccessMessage(payload), extra?.success);
  return (payload?.details as T) ?? (undefined as T);
}

export const api = {
  get: <T>(
    path: string,
    query?: RequestOptions["query"],
    extra?: Omit<Partial<RequestOptions>, "method" | "query">,
  ) => apiRequest<T>(path, { method: "GET", query, ...extra }),
  post: <T>(path: string, body?: unknown, extra?: Partial<RequestOptions>) =>
    apiRequest<T>(path, { method: "POST", body, ...extra }),
  put: <T>(path: string, body?: unknown, extra?: Partial<RequestOptions>) =>
    apiRequest<T>(path, { method: "PUT", body, ...extra }),
  patch: <T>(path: string, body?: unknown, extra?: Partial<RequestOptions>) =>
    apiRequest<T>(path, { method: "PATCH", body, ...extra }),
  del: <T>(path: string, extra?: Partial<RequestOptions>) =>
    apiRequest<T>(path, { method: "DELETE", ...extra }),
  download: apiDownload,
  fetchBlob: apiFetchBlob,
  upload: apiUpload,
};
