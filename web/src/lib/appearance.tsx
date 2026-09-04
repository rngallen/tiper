"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useAuth } from "@/lib/auth";
import type { AppearanceSettings } from "@/lib/types";

type ResolvedAppearance = {
  theme: "light" | "dark" | "system";
  /** Effective color scheme after resolving "system". */
  colorScheme: "light" | "dark";
  compactMode: boolean;
  largeText: boolean;
  /** true = expanded sidebar (labels visible). */
  sidebarExpanded: boolean;
};

type AppearanceContextValue = ResolvedAppearance & {
  /** Apply settings immediately (e.g. while editing or after save). */
  applyLocal: (next: AppearanceSettings) => void;
  /** Drop local override and fall back to the profile (or defaults). */
  clearLocal: () => void;
  /** Instantly expand/collapse the sidebar (session preference; no profile save). */
  toggleSidebar: () => void;
};

const DEFAULTS: AppearanceSettings = {
  theme: "light",
  compactMode: true,
  largeText: false,
  sidebarState: true,
};

const SIDEBAR_SESSION_KEY = "dfms.sidebarExpanded";

const AppearanceContext = createContext<AppearanceContextValue | undefined>(
  undefined,
);

function normalize(raw?: AppearanceSettings | null): AppearanceSettings {
  return {
    theme: raw?.theme === "light" || raw?.theme === "dark" || raw?.theme === "system"
      ? raw.theme
      : "light",
    compactMode: raw?.compactMode === undefined ? true : Boolean(raw.compactMode),
    largeText: Boolean(raw?.largeText),
    sidebarState:
      raw?.sidebarState === undefined ? true : Boolean(raw.sidebarState),
  };
}

function systemPrefersDark(): boolean {
  if (typeof window === "undefined") return false;
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

function resolveColorScheme(theme: string): "light" | "dark" {
  if (theme === "dark") return "dark";
  if (theme === "light") return "light";
  return systemPrefersDark() ? "dark" : "light";
}

function applyDom(
  settings: AppearanceSettings,
  colorScheme: "light" | "dark",
  persist = true,
) {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  root.classList.toggle("dark", colorScheme === "dark");
  root.dataset.theme = colorScheme;
  root.dataset.compact = settings.compactMode ? "true" : "false";
  root.dataset.largeText = settings.largeText ? "true" : "false";
  root.style.colorScheme = colorScheme;
  if (!persist) return;
  try {
    localStorage.setItem("dfms.colorScheme", colorScheme);
  } catch {
    /* ignore */
  }
}

function readSidebarSession(): boolean | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = sessionStorage.getItem(SIDEBAR_SESSION_KEY);
    if (raw === "1") return true;
    if (raw === "0") return false;
  } catch {
    /* ignore */
  }
  return null;
}

export function AppearanceProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const { user } = useAuth();
  const [local, setLocal] = useState<AppearanceSettings | null>(null);
  const [systemDark, setSystemDark] = useState(systemPrefersDark);
  const [sidebarOverride, setSidebarOverride] = useState<boolean | null>(null);

  useEffect(() => {
    setSidebarOverride(readSidebarSession());
  }, []);

  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => setSystemDark(mq.matches);
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, []);

  useEffect(() => {
    if (!user) {
      setLocal(null);
      setSidebarOverride(null);
    }
  }, [user]);

  const merged = useMemo(
    () =>
      normalize({
        ...DEFAULTS,
        ...user?.profile?.appearanceSettings,
        ...local,
      }),
    [user?.profile?.appearanceSettings, local],
  );

  const sidebarExpanded =
    sidebarOverride !== null
      ? sidebarOverride
      : Boolean(merged.sidebarState);

  const colorScheme = useMemo(() => {
    if (!user) return "light" as const;
    if (merged.theme === "system") {
      return systemDark ? "dark" : "light";
    }
    return resolveColorScheme(merged.theme || "system");
  }, [user, merged.theme, systemDark]);

  useEffect(() => {
    if (!user) {
      applyDom({ ...DEFAULTS, theme: "light" }, "light", false);
      return;
    }
    applyDom(merged, colorScheme, true);
  }, [user, merged, colorScheme, sidebarExpanded]);

  const applyLocal = useCallback((next: AppearanceSettings) => {
    setLocal(normalize(next));
  }, []);

  const clearLocal = useCallback(() => {
    setLocal(null);
  }, []);

  const toggleSidebar = useCallback(() => {
    setSidebarOverride((prev) => {
      const current =
        prev !== null ? prev : Boolean(merged.sidebarState);
      const next = !current;
      try {
        sessionStorage.setItem(SIDEBAR_SESSION_KEY, next ? "1" : "0");
      } catch {
        /* ignore */
      }
      return next;
    });
  }, [merged.sidebarState]);

  const value = useMemo<AppearanceContextValue>(
    () => ({
      theme: (merged.theme as ResolvedAppearance["theme"]) || "system",
      colorScheme,
      compactMode: Boolean(merged.compactMode),
      largeText: Boolean(merged.largeText),
      sidebarExpanded,
      applyLocal,
      clearLocal,
      toggleSidebar,
    }),
    [
      merged,
      colorScheme,
      sidebarExpanded,
      applyLocal,
      clearLocal,
      toggleSidebar,
    ],
  );

  return (
    <AppearanceContext.Provider value={value}>
      {children}
    </AppearanceContext.Provider>
  );
}

export function useAppearance(): AppearanceContextValue {
  const ctx = useContext(AppearanceContext);
  if (!ctx) {
    throw new Error("useAppearance must be used within AppearanceProvider");
  }
  return ctx;
}
