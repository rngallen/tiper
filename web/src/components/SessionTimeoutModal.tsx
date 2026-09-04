"use client";

import { useEffect, useRef } from "react";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui";
import { formatIdleClock, idleMinutes } from "@/lib/idle";

/** Blocking session-expiry dialog (SAP/Oracle style). Backdrop does not dismiss. */
export function SessionTimeoutModal({
  secondsLeft,
  onContinue,
  onSignOut,
}: {
  secondsLeft: number;
  onContinue: () => void;
  onSignOut: () => void;
}) {
  const clock = formatIdleClock(secondsLeft);
  const urgent = secondsLeft <= 30;
  const continueRef = useRef(onContinue);
  continueRef.current = onContinue;

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        continueRef.current();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center p-4"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="session-timeout-title"
      aria-describedby="session-timeout-desc"
    >
      <div className="absolute inset-0 bg-slate-950/55 backdrop-blur-[2px]" />
      <div className="relative z-10 w-full max-w-md overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl">
        <div className="h-1.5 bg-brand-800" />
        <div className="px-8 pb-6 pt-8 text-center">
          <div
            className={`mx-auto flex h-14 w-14 items-center justify-center rounded-full ${
              urgent ? "bg-red-50 text-red-700" : "bg-amber-50 text-amber-800"
            }`}
          >
            <AlertTriangle size={28} strokeWidth={2.25} aria-hidden />
          </div>
          <h2
            id="session-timeout-title"
            className="mt-5 text-xl font-semibold tracking-tight text-slate-900"
          >
            Your session is about to expire
          </h2>
          <p id="session-timeout-desc" className="mt-2 text-sm leading-6 text-slate-600">
            No click or keystroke for a while. You will be signed out in
          </p>
          <p
            className={`mt-4 font-mono text-5xl font-semibold tabular-nums tracking-wide ${
              urgent ? "text-red-700" : "text-brand-800"
            }`}
          >
            {clock}
          </p>
          <p className="mt-3 text-xs text-slate-500">
            Sessions end after {idleMinutes()} minutes of inactivity. Continue
            tells the server you are still here and resets the idle clock.
            Unsaved form data may be lost if the session ends.
          </p>
        </div>
        <div className="flex items-center justify-end gap-3 border-t border-slate-100 bg-slate-50 px-6 py-4">
          <Button type="button" variant="ghost" onClick={onSignOut}>
            Sign out
          </Button>
          <Button type="button" onClick={onContinue} autoFocus>
            Continue
          </Button>
        </div>
      </div>
    </div>
  );
}
