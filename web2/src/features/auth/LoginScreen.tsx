import { useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { ensureCsrf } from "@/shared/api/http";
import { errorMessage } from "@/shared/api/errors";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { useSession } from "./session";
import type { OtpPending } from "./types";

function secondsUntil(iso?: string) {
  if (!iso) return 0;
  const ms = Date.parse(iso);
  return Number.isNaN(ms) ? 0 : Math.max(0, Math.ceil((ms - Date.now()) / 1000));
}
function mmss(total: number) {
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, "0")}`;
}

export function LoginScreen() {
  const navigate = useNavigate();
  const { login, verifyOtp, resendOtp, user, loading: authLoading } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [otp, setOtp] = useState<OtpPending | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [left, setLeft] = useState(0);

  useEffect(() => { void ensureCsrf(); }, []);
  useEffect(() => { if (!authLoading && user) void navigate({ to: "/dashboard" }); }, [authLoading, user, navigate]);
  useEffect(() => {
    if (!otp) return;
    const tick = () => setLeft(secondsUntil(otp.expiredAt));
    tick();
    const id = window.setInterval(tick, 1000);
    return () => window.clearInterval(id);
  }, [otp]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      if (otp) {
        await verifyOtp(otp.token, code);
        await navigate({ to: "/dashboard" });
        return;
      }
      const result = await login(email.trim(), password);
      if (result.otp) { setOtp(result.otp); setCode(""); }
      else await navigate({ to: "/dashboard" });
    } catch (err) {
      setError(errorMessage(err, "Sign-in failed"));
    } finally {
      setBusy(false);
    }
  }

  async function onResend() {
    if (!otp) return;
    setBusy(true); setError("");
    try { setOtp(await resendOtp(otp.token)); setCode(""); }
    catch (err) { setError(errorMessage(err)); }
    finally { setBusy(false); }
  }

  return (
    <div className="grid min-h-full lg:grid-cols-2">
      <section className="relative hidden overflow-hidden bg-sidebar p-12 text-sidebar-foreground lg:flex lg:flex-col lg:justify-between">
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,oklch(0.45_0.12_240_/_0.35),transparent_50%)]" />
        <div className="relative">
          <p className="text-[11px] uppercase tracking-[0.28em] text-sidebar-primary">TIPER Tanzania</p>
          <h1 className="mt-5 max-w-md text-4xl font-semibold leading-tight text-sidebar-accent-foreground">Terminal operating system for the bonded farm.</h1>
          <p className="mt-5 max-w-sm text-sm text-sidebar-muted">Receipts, gantry, pump-over, billing and EWURA — one console, PASETO session, workflow inbox.</p>
        </div>
        <p className="relative text-sm text-sidebar-muted">Kigamboni · Dar es Salaam · 24/7 operations</p>
      </section>
      <section className="flex items-center justify-center p-6 sm:p-10">
        <form className="w-full max-w-sm space-y-4 rounded-2xl border border-border bg-card p-8 shadow-sm" onSubmit={onSubmit}>
          <div>
            <h2 className="text-2xl font-semibold">{otp ? "Enter the code" : "Sign in"}</h2>
            <p className="mt-1 text-sm text-muted-foreground">{otp ? otp.message : "Directory account. Cookies are HttpOnly PASETO tokens."}</p>
          </div>
          {otp ? (
            <>
              <div className="space-y-1.5">
                <Label htmlFor="otp">6-digit code</Label>
                <Input id="otp" inputMode="numeric" autoComplete="one-time-code" maxLength={6} value={code} onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))} className="text-center font-mono text-lg tracking-[0.4em]" />
                <p className="text-xs text-muted-foreground">Expires in {mmss(left)}</p>
              </div>
              <Button type="submit" className="w-full" loading={busy} disabled={code.length !== 6}>Verify</Button>
              <button type="button" className="w-full text-xs text-muted-foreground hover:text-foreground" onClick={() => void onResend()}>Resend code</button>
              <button type="button" className="w-full text-xs text-muted-foreground hover:text-foreground" onClick={() => { setOtp(null); setCode(""); }}>Use a different account</button>
            </>
          ) : (
            <>
              <div className="space-y-1.5">
                <Label htmlFor="email">Email</Label>
                <Input id="email" type="email" autoComplete="username" value={email} onChange={(e) => setEmail(e.target.value)} required />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="password">Password</Label>
                <Input id="password" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} required />
              </div>
              <Button type="submit" className="w-full" loading={busy}>Enter terminal</Button>
            </>
          )}
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
        </form>
      </section>
    </div>
  );
}
