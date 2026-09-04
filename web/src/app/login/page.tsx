"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Button, ErrorBanner, Field, Input, PasswordInput } from "@/components/ui";
import { Logo } from "@/components/Logo";
import { OtpInput } from "@/components/OtpInput";
import { PublicAuthShell } from "@/components/PublicAuthShell";
import { useAuth } from "@/lib/auth";
import { ApiError, ensureCsrf } from "@/lib/api";
import type { OtpPending } from "@/lib/types";

const YEAR = new Date().getFullYear();
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function safeNextPath(raw: string | null | undefined, fallback = "/dashboard"): string {
  if (!raw) return fallback;
  if (!raw.startsWith("/") || raw.startsWith("//")) return fallback;
  if (raw === "/login" || raw.startsWith("/login?")) return fallback;
  return raw;
}

function postLoginPath(mustChangePassword?: boolean): string {
  if (mustChangePassword) return "/profile?tab=password";
  if (typeof window === "undefined") return "/dashboard";
  return safeNextPath(new URLSearchParams(window.location.search).get("next"));
}

function formatCountdown(totalSec: number): string {
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

function secondsUntil(iso: string | undefined): number {
  if (!iso) return 0;
  const ms = Date.parse(iso);
  if (Number.isNaN(ms)) return 0;
  return Math.max(0, Math.ceil((ms - Date.now()) / 1000));
}

export default function LoginPage() {
  const router = useRouter();
  const { login, verifyOtp, resendOtp, user, loading: authLoading } = useAuth();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [otp, setOtp] = useState<OtpPending | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [resending, setResending] = useState(false);
  const [codeRemainingSec, setCodeRemainingSec] = useState(0);
  const [resendRemainingSec, setResendRemainingSec] = useState(0);
  const verifyingRef = useRef(false);

  useEffect(() => {
    void ensureCsrf();
  }, []);

  useEffect(() => {
    if (authLoading || !user) return;
    router.replace(postLoginPath());
  }, [authLoading, user, router]);

  useEffect(() => {
    if (!otp) {
      setCodeRemainingSec(0);
      setResendRemainingSec(0);
      return;
    }
    const tick = () => {
      setCodeRemainingSec(secondsUntil(otp.expiredAt));
      const resendAt =
        otp.resendAvailableAt ||
        new Date(Date.parse(otp.issuedAt) + 60_000).toISOString();
      setResendRemainingSec(secondsUntil(resendAt));
    };
    tick();
    const id = window.setInterval(tick, 1000);
    return () => window.clearInterval(id);
  }, [otp]);

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault();
    const workEmail = email.trim().toLowerCase();
    if (!EMAIL_RE.test(workEmail)) {
      setError("Enter a valid work email address.");
      return;
    }
    if (!password) {
      setError("Password is required.");
      return;
    }
    setError("");
    setLoading(true);
    try {
      const csrf = await ensureCsrf();
      if (!csrf) {
        setError("Could not start a secure session. Refresh the page and try again.");
        return;
      }
      const res = await login(workEmail, password);
      if (res.otp) {
        setEmail(workEmail);
        setPassword("");
        setOtp(res.otp);
        setCode("");
        return;
      }
      router.replace(postLoginPath(res.mustChangePassword));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  async function submitVerify(fullCode: string) {
    if (!otp || verifyingRef.current) return;
    if (fullCode.length !== 6) {
      setError("Enter the 6-digit verification code.");
      return;
    }
    if (secondsUntil(otp.expiredAt) === 0) {
      setError("This code has expired. Request a new one to continue.");
      return;
    }
    verifyingRef.current = true;
    setError("");
    setLoading(true);
    try {
      const res = await verifyOtp(otp.token, fullCode);
      router.replace(postLoginPath(res.mustChangePassword));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Verification failed");
      setCode("");
    } finally {
      verifyingRef.current = false;
      setLoading(false);
    }
  }

  async function handleVerify(e: React.FormEvent) {
    e.preventDefault();
    await submitVerify(code);
  }

  async function handleResend() {
    if (!otp || resendRemainingSec > 0 || resending) return;
    setError("");
    setResending(true);
    try {
      const next = await resendOtp(otp.token);
      setOtp(next);
      setCode("");
      verifyingRef.current = false;
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Could not resend code",
      );
    } finally {
      setResending(false);
    }
  }

  const emailOk = EMAIL_RE.test(email.trim());
  const canSignIn = emailOk && password.length > 0 && !loading;
  const codeExpired = Boolean(otp) && codeRemainingSec === 0;
  const canResend =
    Boolean(otp) && resendRemainingSec === 0 && !resending && !loading;

  return (
    <PublicAuthShell
      footer={
        <>
          <p>
            Need access? Contact{" "}
            <a
              href="mailto:info@tiper.co.tz"
              className="font-semibold text-brand-800 underline underline-offset-2 hover:text-brand-900"
            >
              info@tiper.co.tz
            </a>
          </p>
          <p className="mt-1.5">
            © {YEAR} TIPER. All rights reserved. Confidential — authorised
            personnel only.
          </p>
        </>
      }
    >
      <Logo height={64} priority />
      <h1 className="mt-5 text-2xl font-semibold tracking-tight text-brand-900">
        TIPER DFMS
      </h1>
      <p className="mt-2 text-sm text-slate-600">
        {otp
          ? "Enter the verification code sent to your registered channel."
          : "Sign in with your TIPER account."}
      </p>
      <p className="mt-2 text-xs text-slate-500">
        Secure enterprise login — authorised TIPER staff only.
      </p>

      {error && (
        <div className="mt-6 w-full">
          <ErrorBanner message={error} />
        </div>
      )}

      {!otp ? (
        <form onSubmit={handleLogin} className="mt-7 w-full space-y-5" noValidate>
          <Field label="Work email" required>
            <Input
              type="email"
              name="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="name@tiper.co.tz"
              required
              autoFocus
              autoComplete="username"
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              maxLength={120}
              disabled={loading}
              inputMode="email"
            />
          </Field>
          <Field label="Password" required>
            <PasswordInput
              name="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
              autoComplete="current-password"
              disabled={loading}
              maxLength={128}
            />
          </Field>

          <Button
            type="submit"
            loading={loading}
            disabled={!canSignIn}
            className="!rounded-md w-full hover:bg-brand-900"
          >
            Sign in
          </Button>
        </form>
      ) : (
        <form onSubmit={handleVerify} className="mt-7 w-full space-y-6">
          <p className="text-sm text-slate-600">{otp.message}</p>
          <div>
            <p className="mb-3 text-sm font-medium text-slate-900">
              Verification code
            </p>
            <OtpInput
              value={code}
              onChange={setCode}
              onComplete={(full) => void submitVerify(full)}
              autoFocus
              disabled={loading || codeExpired}
              error={Boolean(error)}
            />
          </div>
          <Button
            type="submit"
            loading={loading}
            disabled={code.length !== 6 || loading || codeExpired}
            className="!rounded-md w-full hover:bg-brand-900"
          >
            Verify
          </Button>

          <div className="space-y-1 border-t border-slate-200 pt-4 text-sm text-slate-600" aria-live="polite">
            {codeRemainingSec > 0 ? (
              <p>
                Code expires in{" "}
                <span className="font-semibold tabular-nums text-slate-950">
                  {formatCountdown(codeRemainingSec)}
                </span>
              </p>
            ) : (
              <p>Code expired. Request a new one to continue.</p>
            )}
            {resendRemainingSec > 0 ? (
              <p>
                Resend available in{" "}
                <span className="font-semibold tabular-nums text-slate-950">
                  {formatCountdown(resendRemainingSec)}
                </span>
              </p>
            ) : (
              <button
                type="button"
                disabled={!canResend}
                onClick={handleResend}
                className="min-h-11 font-semibold text-brand-800 underline underline-offset-2 hover:text-brand-900 disabled:cursor-not-allowed disabled:text-slate-500 disabled:no-underline"
              >
                {resending ? "Sending…" : "Resend code"}
              </button>
            )}
          </div>

          <button
            type="button"
            onClick={() => {
              setOtp(null);
              setCode("");
              setError("");
              verifyingRef.current = false;
            }}
            className="min-h-11 text-sm font-semibold text-slate-600 hover:text-slate-950"
          >
            Back to sign in
          </button>
        </form>
      )}
    </PublicAuthShell>
  );
}
