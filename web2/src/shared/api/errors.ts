/**
 * Error model for the DFMS API.
 *
 * Every response is `{ message?, details? }`. On 422 `details` is an array of
 * `{ field, error }`; on 409 it may carry a payload (e.g. `{ nearExpiry: true }`).
 */
export interface ApiEnvelope<T = unknown> {
  message?: string;
  details?: T;
}

export interface FieldError {
  field: string;
  error: string;
}

export class ApiError extends Error {
  readonly status: number;
  readonly details: unknown;
  readonly fieldErrors: FieldError[];
  readonly requestId: string | null;

  constructor(message: string, status: number, details?: unknown, requestId?: string | null) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.details = details;
    this.fieldErrors = parseFieldErrors(details);
    this.requestId = requestId ?? null;
  }

  get isValidation(): boolean {
    return this.status === 422 && this.fieldErrors.length > 0;
  }
  get isUnauthorized(): boolean {
    return this.status === 401;
  }
  get isForbidden(): boolean {
    return this.status === 403;
  }
  get isConflict(): boolean {
    return this.status === 409;
  }
  get isNotFound(): boolean {
    return this.status === 404;
  }
  get isServer(): boolean {
    return this.status >= 500;
  }

  /** `details` narrowed to an object, or null. Useful for 409 payloads. */
  detailObject<T extends object = Record<string, unknown>>(): T | null {
    const d = this.details;
    return d && typeof d === "object" && !Array.isArray(d) ? (d as T) : null;
  }
}

export function parseFieldErrors(details: unknown): FieldError[] {
  if (!Array.isArray(details)) return [];
  const out: FieldError[] = [];
  for (const item of details) {
    if (!item || typeof item !== "object") continue;
    const { field, error } = item as Partial<FieldError>;
    if (typeof error !== "string" || !error.trim()) continue;
    out.push({ field: typeof field === "string" ? field : "", error: error.trim() });
  }
  return out;
}

export function isApiError(err: unknown): err is ApiError {
  return err instanceof ApiError;
}

/** Human readable message for any thrown value. */
export function errorMessage(err: unknown, fallback = "Something went wrong"): string {
  if (isApiError(err)) {
    if (err.fieldErrors.length > 0) {
      return err.fieldErrors.map((f) => (f.field ? `${f.field}: ${f.error}` : f.error)).join("; ");
    }
    return err.message || fallback;
  }
  if (err instanceof Error) return err.message || fallback;
  if (typeof err === "string") return err;
  return fallback;
}

export function errorTitle(err: unknown): string {
  if (!isApiError(err)) return "Unable to complete";
  switch (true) {
    case err.status === 400:
      return "Invalid request";
    case err.status === 401:
      return "Session expired";
    case err.status === 403:
      return "Not allowed";
    case err.status === 404:
      return "Not found";
    case err.status === 409:
      return "Cannot save";
    case err.status === 422:
      return "Check the values";
    case err.status === 429:
      return "Too many requests";
    case err.status >= 500:
      return "Server error";
    default:
      return "Unable to complete";
  }
}
