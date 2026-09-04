import { create } from "zustand";
import { http, httpEvents, markSessionAccepted, sessionMeta } from "@/shared/api/http";
import { canAny } from "@/shared/config/permissions";
import { writeStorage, readStorage } from "@/shared/lib/storage";
import { isOtpPending, type MeResponse, type OtpPending, type TokenResponse, type User } from "./types";

const USER_KEY = "dfms.user";

export interface SessionState {
  user: User | null;
  permissions: string[];
  isSuperUser: boolean;
  loading: boolean;
  ready: boolean;
  mustChangePassword: boolean;
  boot: () => Promise<void>;
  login: (email: string, password: string) => Promise<{ otp?: OtpPending; mustChangePassword?: boolean }>;
  verifyOtp: (otpToken: string, code: string) => Promise<{ mustChangePassword?: boolean }>;
  resendOtp: (otpToken: string) => Promise<OtpPending>;
  changePassword: (oldPassword: string, newPassword: string, confirmPassword: string) => Promise<void>;
  logout: () => Promise<void>;
  can: (...codes: string[]) => boolean;
}

function applyTokens(tokens: TokenResponse) {
  sessionMeta.set(tokens.tokenExpiresAt);
  markSessionAccepted();
}

function applyMe(me: MeResponse) {
  writeStorage(USER_KEY, me.user);
  return {
    user: me.user,
    permissions: me.permissions ?? [],
    isSuperUser: Boolean(me.isSuperUser || me.user?.isSuperUser),
    mustChangePassword: Boolean(me.user?.mustChangePassword),
  };
}

export const useSession = create<SessionState>((set, get) => ({
  user: readStorage<User | null>(USER_KEY, null),
  permissions: [],
  isSuperUser: false,
  loading: true,
  ready: false,
  mustChangePassword: false,
  can: (...codes) => canAny(new Set(get().permissions), get().isSuperUser, codes),
  boot: async () => {
    set({ loading: true });
    try {
      const me = await http.get<MeResponse>("/auth/profile", undefined, { redirectOn401: false });
      set({ ...applyMe(me), loading: false, ready: true });
      if (!sessionMeta.accessExpiresAt) sessionMeta.set(new Date(Date.now() + 15 * 60_000).toISOString());
    } catch {
      writeStorage(USER_KEY, null);
      sessionMeta.clear();
      set({ user: null, permissions: [], isSuperUser: false, mustChangePassword: false, loading: false, ready: true });
    }
  },
  login: async (email, password) => {
    const data = await http.post<TokenResponse | OtpPending>("/auth/login", { email, password }, { silent: true });
    if (isOtpPending(data)) return { otp: data };
    applyTokens(data);
    await get().boot();
    return { mustChangePassword: data.mustChangePassword };
  },
  verifyOtp: async (otpToken, code) => {
    const tokens = await http.post<TokenResponse>("/auth/mfa/verify", { code }, { token: otpToken, silent: true });
    applyTokens(tokens);
    await get().boot();
    return { mustChangePassword: tokens.mustChangePassword };
  },
  resendOtp: (otpToken) => http.post<OtpPending>("/auth/mfa/resend", {}, { token: otpToken, silent: true }),
  changePassword: async (oldPassword, newPassword, confirmPassword) => {
    const tokens = await http.post<TokenResponse>("/auth/change-password", { oldPassword, newPassword, confirmPassword });
    applyTokens(tokens);
    await get().boot();
  },
  logout: async () => {
    try { await http.post("/auth/logout", {}, { silent: true, redirectOn401: false }); } catch { /* ignore */ }
    writeStorage(USER_KEY, null);
    sessionMeta.clear();
    set({ user: null, permissions: [], isSuperUser: false, mustChangePassword: false, loading: false, ready: true });
  },
}));

if (typeof window !== "undefined") {
  httpEvents.on("session:rejected", () => {
    writeStorage(USER_KEY, null);
    useSession.setState({ user: null, permissions: [], isSuperUser: false, mustChangePassword: false, loading: false, ready: true });
  });
}
