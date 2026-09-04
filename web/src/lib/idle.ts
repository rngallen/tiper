/** Idle-session policy. Source of truth is the API (login + GET /auth/profile). */

function clampIdle(minutes: number, warnWanted: number): { minutes: number; warnSeconds: number } {
  const m = Math.min(480, Math.max(2, Math.round(minutes)));
  const warn = Math.min(600, Math.max(15, Math.round(warnWanted)));
  return { minutes: m, warnSeconds: Math.min(warn, Math.max(15, m * 60 - 15)) };
}

let policy = clampIdle(10, 120);

/** Apply idle times from login or GET /auth/profile. */
export function applyServerIdlePolicy(minutes?: number, warnSeconds?: number) {
  if (!Number.isFinite(minutes) && !Number.isFinite(warnSeconds)) return;
  policy = clampIdle(minutes ?? policy.minutes, warnSeconds ?? policy.warnSeconds);
}

export function idleMinutes(): number {
  return policy.minutes;
}

export function idleLimitMs(): number {
  return policy.minutes * 60 * 1000;
}

export function idleWarnSeconds(): number {
  return policy.warnSeconds;
}

export function formatIdleClock(totalSec: number): string {
  const s = Math.max(0, Math.floor(totalSec));
  const m = Math.floor(s / 60);
  const r = s % 60;
  return `${String(m).padStart(2, "0")}:${String(r).padStart(2, "0")}`;
}
