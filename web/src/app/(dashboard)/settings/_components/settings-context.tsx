"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { useAuth } from "@/lib/auth";
import { PERMISSIONS } from "@/lib/permissions";

export type SettingsSaveAction = {
  run: () => void;
  saving: boolean;
  disabled?: boolean;
};

const SettingsContext = createContext<{
  canManage: boolean;
  saveAction: SettingsSaveAction | null;
  registerSave: (action: SettingsSaveAction | null) => void;
}>({
  canManage: false,
  saveAction: null,
  registerSave: () => {},
});

export function SettingsProvider({ children }: { children: React.ReactNode }) {
  const { hasPermission } = useAuth();
  const [saveAction, setSaveAction] = useState<SettingsSaveAction | null>(null);
  const registerSave = useCallback((action: SettingsSaveAction | null) => {
    setSaveAction(action);
  }, []);

  return (
    <SettingsContext.Provider
      value={{
        canManage: hasPermission(PERMISSIONS.settingsManage),
        saveAction,
        registerSave,
      }}
    >
      {children}
    </SettingsContext.Provider>
  );
}

export function useSettings(): {
  canManage: boolean;
  saveAction: SettingsSaveAction | null;
} {
  const { canManage, saveAction } = useContext(SettingsContext);
  return { canManage, saveAction };
}

/** Puts Save on the Settings tab bar. Unregisters on unmount. */
export function useSettingsSave(
  run: () => void,
  saving: boolean,
  disabled?: boolean,
) {
  const { registerSave } = useContext(SettingsContext);
  const runRef = useRef(run);
  const lockRef = useRef(false);
  runRef.current = run;

  useEffect(() => {
    if (!saving) lockRef.current = false;
  }, [saving]);

  useEffect(() => {
    registerSave({
      run: () => {
        if (lockRef.current || saving || disabled) return;
        lockRef.current = true;
        runRef.current();
      },
      saving,
      disabled: Boolean(disabled),
    });
    return () => registerSave(null);
  }, [registerSave, saving, disabled]);
}
