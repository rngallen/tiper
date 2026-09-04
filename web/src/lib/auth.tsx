"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { useRouter } from "next/navigation";
import {
  api,
  apiRequest,
  ApiError,
  onApiActivity,
  refreshTokens,
  touchSession,
  tokenStore,
  clearSessionRejected,
  SESSION_REJECTED_EVENT,
} from "./api";
import { SessionTimeoutModal } from "@/components/SessionTimeoutModal";
import { applyServerIdlePolicy, idleLimitMs, idleWarnSeconds } from "@/lib/idle";
import { satisfiesAny } from "./permissions";
import type { MeResponse, OtpPending, TokenResponse, User } from "./types";

interface LoginResult {
  otp?: OtpPending;
  mustChangePassword?: boolean;
  ok?: boolean;
}

interface AuthState {
  user: User | null;
  permissions: string[];
  isSuperUser: boolean;
  loading: boolean;
  mustChangePassword: boolean;
  login: (email: string, password: string) => Promise<LoginResult>;
  verifyOtp: (otpToken: string, code: string) => Promise<LoginResult>;
  /** Request a new OTP after the previous code expires; returns a fresh challenge + bearer. */
  resendOtp: (otpToken: string) => Promise<OtpPending>;
  changePassword: (
    oldPassword: string,
    newPassword: string,
    confirmPassword: string,
  ) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
  // hasPermission returns true if the user holds ANY of the given codes (or is
  // a super-user), including manage / module wildcards (see satisfiesPermission).
  // With no codes it returns true.
  hasPermission: (...codes: string[]) => boolean;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

// Session tuning. Refresh the access token this many ms before it expires.
const REFRESH_LEAD_MS = 5 * 60 * 1000; // 5 minutes
const REFRESH_POLL_MS = 30 * 1000; // check every 30s
// Re-fetch /auth/profile (effective permissions) on this interval — aligned
// with backend PermsTTL (apps/auth/middleware/keys.go).
const PERMS_REFRESH_MS = 60 * 60 * 1000; // 1 hour
// Intentional input only. mousemove/scroll fire while the window sits open
// (cursor rest, trackpad drift, layout) and would never let the idle timer
// elapse. GET polls must not reset idle (api.ts defaults GET to countAsActivity false).
const ACTIVITY_EVENTS = ["pointerdown", "keydown"] as const;

function isOtpPending(d: TokenResponse | OtpPending): d is OtpPending {
  // Login OTP challenge includes a message and MFA bearer. Session tokens also
  // include `token` (access) but never the OTP `message` field.
  return (
    typeof (d as OtpPending).token === "string" &&
    typeof (d as OtpPending).message === "string" &&
    typeof (d as TokenResponse).tokenExpiresAt !== "string"
  );
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [permissions, setPermissions] = useState<string[]>([]);
  const [isSuperUser, setIsSuperUser] = useState(false);
  const [loading, setLoading] = useState(true);
  const [mustChangePassword, setMustChangePassword] = useState(false);

  const lastActivity = useRef(0);
  const warningOpen = useRef(false);
  const [idleLeftSec, setIdleLeftSec] = useState<number | null>(null);
  const signedIn = Boolean(user);

  const clearSession = useCallback(() => {
    tokenStore.clear();
    setUser(null);
    setPermissions([]);
    setIsSuperUser(false);
    setMustChangePassword(false);
  }, []);

  const refreshUser = useCallback(
    async (opts?: { allowStale?: boolean }) => {
      // Session is an HttpOnly cookie — always probe the profile. localStorage
      // only caches UI state / access expiry for proactive refresh.
      const allowStale = opts?.allowStale !== false;
      try {
        const me = await apiRequest<MeResponse>("/auth/profile", {
          redirectOn401: false,
          countAsActivity: false,
        });
        setUser(me.user);
        setPermissions(me.permissions ?? []);
        setIsSuperUser(Boolean(me.isSuperUser));
        applyServerIdlePolicy(me.sessionIdleMinutes, me.sessionIdleWarnSeconds);
        if (me.user?.mustChangePassword) {
          setMustChangePassword(true);
        } else {
          setMustChangePassword(false);
        }
        tokenStore.setUser(me.user);
        if (!tokenStore.accessExpiresAt) {
          // Unknown expiry after reload; schedule a conservative refresh window.
          tokenStore.setSession(
            new Date(Date.now() + 15 * 60 * 1000).toISOString(),
          );
        }
      } catch (err) {
        // 401: cookie session is gone. Boot: do not keep a localStorage ghost
        // user (empty permissions + app chrome, no login redirect). Later
        // hourly refresh still keeps the session on 5xx/network.
        if ((err instanceof ApiError && err.status === 401) || !allowStale) {
          clearSession();
        }
      }
    },
    [clearSession],
  );

  useEffect(() => {
    refreshUser({ allowStale: false }).finally(() => setLoading(false));
  }, [refreshUser]);

  // Keep UI permissions in sync with backend PermsTTL (safety net; role/user
  // changes still require a fresh profile fetch after the next interval).
  useEffect(() => {
    if (!signedIn) return;
    const timer = window.setInterval(() => {
      void refreshUser();
    }, PERMS_REFRESH_MS);
    return () => window.clearInterval(timer);
  }, [signedIn, refreshUser]);

  const logout = useCallback(async () => {
    try {
      await api.post("/auth/logout");
    } catch {
      /* ignore */
    }
    clearSession();
    router.push("/login");
  }, [router, clearSession]);

  const staySignedIn = useCallback(async () => {
    lastActivity.current = Date.now();
    warningOpen.current = false;
    setIdleLeftSec(null);
    const touched = await touchSession();
    const refreshed = touched === "ok" ? true : await refreshTokens();
    if (refreshed) clearSessionRejected();
  }, []);

  // ── Proactive token refresh + inactivity logout ──
  // Depend on signed-in flag, not the user object — hourly profile refresh
  // must not reset the idle clock.
  useEffect(() => {
    if (!signedIn) return;
    lastActivity.current = Date.now();
    warningOpen.current = false;
    setIdleLeftSec(null);
    let lastTouch = 0;
    const TOUCH_EVERY_MS = 90_000;
    let warnedThisIdle = false;

    const bump = () => {
      if (warningOpen.current) return;
      lastActivity.current = Date.now();
      warnedThisIdle = false;
      if (Date.now() - lastTouch < TOUCH_EVERY_MS) return;
      lastTouch = Date.now();
      // Best-effort: a failed touch must not bounce a working operator.
      void touchSession();
    };
    ACTIVITY_EVENTS.forEach((e) =>
      window.addEventListener(e, bump, { passive: true }),
    );
    const offApi = onApiActivity(bump);

    let idleLoggingOut = false;
    const startWarning = (seconds: number) => {
      warnedThisIdle = true;
      warningOpen.current = true;
      setIdleLeftSec(seconds);
    };
    const tick = () => {
      const remainingMs = idleLimitMs() - (Date.now() - lastActivity.current);
      const left = Math.max(0, Math.ceil(remainingMs / 1000));
      const warnSec = idleWarnSeconds();
      if (remainingMs > 0 && left <= warnSec) {
        startWarning(left);
        return;
      }
      if (remainingMs > 0) {
        warningOpen.current = false;
        setIdleLeftSec(null);
        return;
      }
      // Tab was backgrounded past the idle window — always show the warning
      // once before signing out.
      if (!warnedThisIdle) {
        lastActivity.current = Date.now() - idleLimitMs() + warnSec * 1000;
        startWarning(warnSec);
        return;
      }
      if (!idleLoggingOut) {
        idleLoggingOut = true;
        warningOpen.current = false;
        void logout();
      }
    };
    tick();
    const idleTimer = window.setInterval(tick, 250);
    const onVisible = () => {
      if (document.visibilityState === "visible") tick();
    };
    document.addEventListener("visibilitychange", onVisible);
    const onRejected = () => {
      if (!warnedThisIdle) {
        lastActivity.current = Date.now() - idleLimitMs() + idleWarnSeconds() * 1000;
        startWarning(idleWarnSeconds());
      }
    };
    window.addEventListener(SESSION_REJECTED_EVENT, onRejected);

    const refreshTimer = window.setInterval(async () => {
      const idle = Date.now() - lastActivity.current > idleLimitMs();
      const exp = tokenStore.accessExpiresAt;
      const due = exp > 0 && exp - Date.now() < REFRESH_LEAD_MS;
      if (!due) return;
      let ok = await refreshTokens();
      if (!ok) {
        await new Promise((r) => window.setTimeout(r, 1500));
        ok = await refreshTokens();
      }
      if (ok || !idle) return;
    }, REFRESH_POLL_MS);

    return () => {
      window.clearInterval(idleTimer);
      window.clearInterval(refreshTimer);
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener(SESSION_REJECTED_EVENT, onRejected);
      offApi();
      ACTIVITY_EVENTS.forEach((e) => window.removeEventListener(e, bump));
    };
  }, [signedIn, logout, clearSession, router]);

  const finishLogin = useCallback((tokens: TokenResponse) => {
    applyServerIdlePolicy(tokens.sessionIdleMinutes, tokens.sessionIdleWarnSeconds);
    // Refresh is HttpOnly-only; access also kept in memory for the first
    // authenticated calls after OTP (avoids bounce-to-login if cookies lag).
    tokenStore.setSession(tokens.tokenExpiresAt, tokens.token);
    const u: User = {
      id: "",
      email: tokens.email,
      firstName: tokens.name,
      mustChangePassword: tokens.mustChangePassword,
    };
    tokenStore.setUser(u);
    setUser(u);
    setMustChangePassword(Boolean(tokens.mustChangePassword));
    lastActivity.current = Date.now();
  }, []);

  const login = useCallback(
    async (email: string, password: string): Promise<LoginResult> => {
      const data = await apiRequest<TokenResponse | OtpPending>("/auth/login", {
        method: "POST",
        body: { email, password },
        auth: false,
      });
      if (isOtpPending(data)) {
        return { otp: data };
      }
      // Defensive: backend always challenges with OTP, but keep a direct path.
      finishLogin(data as TokenResponse);
      await refreshUser();
      return {
        ok: true,
        mustChangePassword: Boolean((data as TokenResponse).mustChangePassword),
      };
    },
    [finishLogin, refreshUser],
  );

  const verifyOtp = useCallback(
    async (otpToken: string, code: string): Promise<LoginResult> => {
      // OTP verify authenticates with the short-lived OTP token from /auth/login
      // (Bearer), not the challenge id in the body.
      const tokens = await apiRequest<TokenResponse>("/auth/mfa/verify", {
        method: "POST",
        body: { code },
        auth: false,
        token: otpToken,
      });
      finishLogin(tokens);
      await refreshUser();
      return {
        ok: true,
        mustChangePassword: Boolean(tokens.mustChangePassword),
      };
    },
    [finishLogin, refreshUser],
  );

  const resendOtp = useCallback(async (otpToken: string): Promise<OtpPending> => {
    return apiRequest<OtpPending>("/auth/mfa/resend", {
      method: "POST",
      auth: false,
      token: otpToken,
    });
  }, []);

  const changePassword = useCallback(
    async (
      oldPassword: string,
      newPassword: string,
      confirmPassword: string,
    ) => {
      // Backend bumps SessionVersion and returns a fresh token pair so this
      // client stays signed in while other sessions are revoked.
      const tokens = await api.post<TokenResponse>("/auth/change-password", {
        oldPassword,
        newPassword,
        confirmPassword,
      });
      if (tokens?.tokenExpiresAt) {
        finishLogin(tokens);
      }
      setMustChangePassword(false);
      await refreshUser();
    },
    [finishLogin, refreshUser],
  );

  const hasPermission = useCallback(
    (...codes: string[]) => {
      if (isSuperUser) return true;
      if (codes.length === 0) return true;
      return satisfiesAny(permissions, ...codes);
    },
    [isSuperUser, permissions],
  );

  return (
    <AuthContext.Provider
      value={{
        user,
        permissions,
        isSuperUser,
        loading,
        mustChangePassword,
        login,
        verifyOtp,
        resendOtp,
        changePassword,
        logout,
        refreshUser,
        hasPermission,
      }}
    >
      {children}
      {user && idleLeftSec != null ? (
        <SessionTimeoutModal
          secondsLeft={idleLeftSec}
          onContinue={staySignedIn}
          onSignOut={() => {
            warningOpen.current = false;
            void logout();
          }}
        />
      ) : null}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
