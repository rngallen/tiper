/**
 * Typed HTTP client for the DFMS Fiber API.
 *
 * - Secrets live in HttpOnly PASETO cookies set by the API (`dfms_access`,
 *   `dfms_refresh`, Path=/api). We always send `credentials: "include"`.
 * - Mutating requests carry `X-Csrf-Token` from the readable `dfms_csrf` cookie
 *   (double submit). `GET /auth/csrf` mints it when missing.
 * - A 401 triggers exactly one refresh (serialised across tabs with the Web
 *   Locks API) and a single retry; a second 401 raises `session:rejected`.
 * - Every response is an envelope `{ message?, details? }`; callers receive
 *   `details` (typed) and the envelope message is surfaced to the toast layer
 *   for mutations through `onSuccessMessage`.
 */
import { ApiError, parseFieldErrors, type ApiEnvelope } from "./errors";
import { env } from "@/shared/lib/env";

export const CSRF_COOKIE = "dfms_csrf";
export const CSRF_HEADER = "X-Csrf-Token";

export type QueryValue = string | number | boolean | null | undefined;
export type QueryParams = Record<string, QueryValue>;

export interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  query?: QueryParams;
  /** Explicit bearer (MFA challenge token). Skips cookie refresh on 401. */
  token?: string;
  /** Abort signal (TanStack Query passes one). */
  signal?: AbortSignal;
  /** When false, a 401 will not raise the session-rejected event (boot probe). */
  redirectOn401?: boolean;
  /** When false, success toast is suppressed even for mutations. */
  silent?: boolean;
  headers?: Record<string, string>;
}

type Listener<T> = (payload: T) => void;

/** Tiny event bus so the API layer stays UI-agnostic. */
class Emitter<Events extends Record<string, unknown>> {
  private map = new Map<keyof Events, Set<Listener<never>>>();
  on<K extends keyof Events>(key: K, fn: Listener<Events[K]>): () => void {
    const set = this.map.get(key) ?? new Set();
    set.add(fn as Listener<never>);
    this.map.set(key, set);
    return () => set.delete(fn as Listener<never>);
  }
  emit<K extends keyof Events>(key: K, payload: Events[K]): void {
    this.map.get(key)?.forEach((fn) => {
      try {
        (fn as Listener<Events[K]>)(payload);
      } catch (e) {
        console.error(e);
      }
    });
  }
}

export interface HttpEvents extends Record<string, unknown> {
  /** Cookies were rejected and refresh failed: the UI must go to /login. */
  "session:rejected": { reason: "unauthorized" };
  /** A mutation succeeded and the envelope carried a message. */
  "success:message": { message: string; method: string; path: string };
  /** A request failed; UI decides whether to show a dialog or inline error. */
  "request:error": { error: ApiError; method: string; path: string };
  /** Any authenticated request succeeded (idle-timer keep-alive). */
  activity: { method: string; path: string };
  /** Access-token expiry changed (login/refresh). */
  "session:expiry": { accessExpiresAt: number | null };
}

export const httpEvents = new Emitter<HttpEvents>();

function readCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const parts = `; ${document.cookie}`.split(`; ${name}=`);
  if (parts.length < 2) return null;
  const value = parts.pop()?.split(";").shift();
  return value ? decodeURIComponent(value) : null;
}

let csrfInFlight: Promise<string | null> | null = null;

/** Ensure a CSRF token exists (cookie or JSON) before an unsafe request. */
export async function ensureCsrf(force = false): Promise<string | null> {
  if (!force) {
    const existing = readCookie(CSRF_COOKIE);
    if (existing) return existing;
  }
  if (!csrfInFlight) {
    csrfInFlight = (async () => {
      try {
        const res = await fetch(`${env.apiBase}/auth/csrf`, {
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
    })().finally(() => {
      csrfInFlight = null;
    });
  }
  return csrfInFlight;
}

const UNSAFE = new Set(["POST", "PUT", "PATCH", "DELETE"]);

export function buildUrl(path: string, query?: QueryParams): string {
  const full = path.startsWith("http") ? path : env.apiBase + path;
  if (!query) return full;
  const usp = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v === undefined || v === null || v === "") continue;
    usp.set(k, String(v));
  }
  const qs = usp.toString();
  return qs ? `${full}${full.includes("?") ? "&" : "?"}${qs}` : full;
}

// ── Session refresh (single flight, cross-tab lock) ─────────────────────────

const ACCESS_EXP_KEY = "dfms.accessExp";
let refreshing: Promise<boolean> | null = null;
let sessionRejected = false;

export const sessionMeta = {
  get accessExpiresAt(): number | null {
    try {
      const raw = localStorage.getItem(ACCESS_EXP_KEY);
      return raw ? Number(raw) : null;
    } catch {
      return null;
    }
  },
  set(accessExpiresAt: string | null | undefined) {
    try {
      if (!accessExpiresAt) {
        localStorage.removeItem(ACCESS_EXP_KEY);
        httpEvents.emit("session:expiry", { accessExpiresAt: null });
        return;
      }
      const ms = Date.parse(accessExpiresAt);
      if (!Number.isNaN(ms)) {
        localStorage.setItem(ACCESS_EXP_KEY, String(ms));
        httpEvents.emit("session:expiry", { accessExpiresAt: ms });
      }
    } catch {
      /* storage unavailable */
    }
  },
  clear() {
    sessionMeta.set(null);
    sessionRejected = false;
  },
  /** Access token still valid for more than `leadMs` (another tab may have refreshed). */
  fresh(leadMs = 60_000): boolean {
    const exp = sessionMeta.accessExpiresAt;
    return exp !== null && exp - Date.now() > leadMs;
  },
};

async function doRefresh(): Promise<boolean> {
  if (sessionMeta.fresh()) return true;
  try {
    const csrf = await ensureCsrf();
    const headers: Record<string, string> = {};
    if (csrf) headers[CSRF_HEADER] = csrf;
    const res = await fetch(`${env.apiBase}/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers,
    });
    if (!res.ok) return false;
    try {
      const data = (await res.json()) as ApiEnvelope<{ tokenExpiresAt?: string }>;
      sessionMeta.set(data.details?.tokenExpiresAt);
    } catch {
      /* cookies rotated anyway */
    }
    sessionRejected = false;
    return true;
  } catch {
    return false;
  }
}

export async function refreshSession(): Promise<boolean> {
  const run = () => {
    if (!refreshing) {
      refreshing = doRefresh().finally(() => {
        refreshing = null;
      });
    }
    return refreshing;
  };
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

function rejectSession() {
  if (sessionRejected) return;
  sessionRejected = true;
  sessionMeta.set(null);
  httpEvents.emit("session:rejected", { reason: "unauthorized" });
}

export function markSessionAccepted() {
  sessionRejected = false;
}

// ── Core request ────────────────────────────────────────────────────────────

async function parseEnvelope<T>(res: Response): Promise<ApiEnvelope<T> | null> {
  const text = await res.text();
  if (!text) return null;
  try {
    return JSON.parse(text) as ApiEnvelope<T>;
  } catch {
    return null;
  }
}

function toApiError(res: Response, payload: ApiEnvelope<unknown> | null, fallback: string) {
  const details = payload?.details;
  const fields = parseFieldErrors(details);
  const message =
    fields.length > 0
      ? fields.map((f) => (f.field ? `${f.field}: ${f.error}` : f.error)).join("; ")
      : payload?.message || res.statusText || fallback;
  return new ApiError(message, res.status, details, res.headers.get("X-Request-ID"));
}

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const {
    method = "GET",
    body,
    query,
    token,
    signal,
    redirectOn401 = true,
    silent = false,
    headers: extraHeaders,
  } = opts;
  const unsafe = UNSAFE.has(method);
  const isForm = typeof FormData !== "undefined" && body instanceof FormData;

  const doFetch = async (retryCsrf = false): Promise<Response> => {
    const headers: Record<string, string> = { Accept: "application/json", ...extraHeaders };
    if (body !== undefined && !isForm) headers["Content-Type"] = "application/json";
    if (token) headers["Authorization"] = `Bearer ${token}`;
    if (unsafe) {
      const csrf = await ensureCsrf(retryCsrf);
      if (csrf) headers[CSRF_HEADER] = csrf;
    }
    return fetch(buildUrl(path, query), {
      method,
      headers,
      credentials: "include",
      body: body === undefined ? undefined : isForm ? (body as FormData) : JSON.stringify(body),
      signal,
    });
  };

  let res = await doFetch();

  // Stale CSRF cookie after a server restart → re-mint once and retry.
  if (res.status === 403 && unsafe) {
    const probe = await res.clone().json().catch(() => null);
    if (typeof probe?.message === "string" && /csrf/i.test(probe.message)) {
      res = await doFetch(true);
    }
  }

  if (res.status === 401 && !token) {
    const ok = await refreshSession();
    if (ok) res = await doFetch();
    if (!ok || res.status === 401) {
      if (redirectOn401) rejectSession();
      throw new ApiError("Your session has expired. Please sign in again.", 401);
    }
  }

  const payload = await parseEnvelope<T>(res);

  if (!res.ok) {
    const error = toApiError(res, payload, `Request failed (${res.status})`);
    httpEvents.emit("request:error", { error, method, path });
    throw error;
  }

  if (!token) httpEvents.emit("activity", { method, path });
  if (unsafe && !silent) {
    const message = payload?.message?.trim();
    if (message) httpEvents.emit("success:message", { message, method, path });
  }
  return (payload?.details as T) ?? (undefined as T);
}

// ── Binary responses (PDF / Excel) ──────────────────────────────────────────

export interface BlobResult {
  blob: Blob;
  filename: string;
  contentType: string;
}

function parseFilename(res: Response, fallback: string): string {
  const cd = res.headers.get("Content-Disposition") || "";
  const star = /filename\*=UTF-8''([^;]+)/i.exec(cd);
  if (star?.[1]) return decodeURIComponent(star[1]);
  const plain = /filename="?([^";]+)"?/i.exec(cd);
  return plain?.[1] ?? fallback;
}

export async function fetchBlob(
  path: string,
  opts: Pick<RequestOptions, "query" | "signal" | "method" | "body"> = {},
): Promise<BlobResult> {
  const method = opts.method ?? "GET";
  const unsafe = UNSAFE.has(method);
  const doFetch = async (): Promise<Response> => {
    const headers: Record<string, string> = {};
    if (unsafe) {
      const csrf = await ensureCsrf();
      if (csrf) headers[CSRF_HEADER] = csrf;
      if (opts.body !== undefined) headers["Content-Type"] = "application/json";
    }
    return fetch(buildUrl(path, opts.query), {
      method,
      headers,
      credentials: "include",
      signal: opts.signal,
      body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
    });
  };
  let res = await doFetch();
  if (res.status === 401) {
    const ok = await refreshSession();
    if (ok) res = await doFetch();
    if (!ok || res.status === 401) {
      rejectSession();
      throw new ApiError("Your session has expired. Please sign in again.", 401);
    }
  }
  if (!res.ok) {
    const payload = await parseEnvelope<unknown>(res);
    const error = toApiError(res, payload, `Download failed (${res.status})`);
    httpEvents.emit("request:error", { error, method, path });
    throw error;
  }
  httpEvents.emit("activity", { method, path });
  const blob = await res.blob();
  return {
    blob,
    filename: parseFilename(res, "download"),
    contentType: res.headers.get("Content-Type") || blob.type || "application/octet-stream",
  };
}

export function saveBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1_000);
}

export async function download(path: string, query?: QueryParams): Promise<void> {
  const { blob, filename } = await fetchBlob(path, { query });
  saveBlob(blob, filename);
}

/** Convenience verbs. `details` of the envelope is returned, typed as T. */
export const http = {
  get: <T>(path: string, query?: QueryParams, opts?: Omit<RequestOptions, "method" | "query">) =>
    request<T>(path, { ...opts, method: "GET", query }),
  post: <T>(path: string, body?: unknown, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "POST", body }),
  put: <T>(path: string, body?: unknown, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "PUT", body }),
  patch: <T>(path: string, body?: unknown, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "PATCH", body }),
  delete: <T>(path: string, opts?: Omit<RequestOptions, "method">) =>
    request<T>(path, { ...opts, method: "DELETE" }),
  upload: <T>(path: string, form: FormData, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "POST", body: form }),
  blob: fetchBlob,
  download,
  saveBlob,
};
