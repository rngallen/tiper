"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ShieldAlert } from "lucide-react";
import { Logo } from "@/components/Logo";
import { PublicAuthShell } from "@/components/PublicAuthShell";
import { useAuth } from "@/lib/auth";

const YEAR = new Date().getFullYear();

export function NotFoundView() {
  const pathname = usePathname();
  const { user, loading } = useAuth();
  const signedIn = Boolean(user);
  const homeHref = signedIn ? "/dashboard" : "/";

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
      <Link href={homeHref} aria-label="TIPER DFMS">
        <Logo height={64} />
      </Link>
      <h1 className="mt-5 text-2xl font-semibold tracking-tight text-brand-900">
        TIPER DFMS
      </h1>
      <p className="mt-2 text-sm text-slate-600">
        Depot fuel management — authorised staff only.
      </p>
      <p className="mt-2 text-xs text-slate-500">
        Secure enterprise login — authorised TIPER staff only.
      </p>

      <div className="mt-7 w-full rounded-lg border border-brand-100 bg-brand-50/70 px-4 py-4 text-left">
        <div className="flex items-start gap-3">
          <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-brand-800 text-white">
            <ShieldAlert className="h-5 w-5" aria-hidden />
          </span>
          <div className="min-w-0">
            <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-brand-700">
              404 — page not found
            </p>
            <p className="mt-1 text-sm font-semibold text-brand-900">
              This address is not a DFMS page
            </p>
            <p className="mt-1 text-sm leading-relaxed text-slate-600">
              The URL is not a valid TIPER DFMS route. Check the address, or
              continue from a page you are allowed to use.
            </p>
          </div>
        </div>
        {pathname && pathname !== "/" ? (
          <p className="mt-3 break-all rounded-md border border-slate-200 bg-white px-3 py-2 font-mono text-[11px] text-slate-700">
            <span className="mr-2 font-sans font-semibold uppercase tracking-wide text-slate-500">
              Requested
            </span>
            {pathname}
          </p>
        ) : null}
      </div>

      <div className="mt-7 flex w-full flex-col gap-3">
        {loading ? (
          <span
            className="h-8 w-8 animate-spin rounded-full border-4 border-brand-200 border-t-brand-700"
            aria-label="Loading"
          />
        ) : signedIn ? (
          <Link
            href="/dashboard"
            className="inline-flex items-center justify-center rounded-md bg-brand-800 px-5 py-2.5 text-sm font-semibold text-white hover:bg-brand-900"
          >
            Operations dashboard
          </Link>
        ) : (
          <>
            <Link
              href="/login"
              className="inline-flex items-center justify-center rounded-md bg-brand-800 px-5 py-2.5 text-sm font-semibold text-white hover:bg-brand-900"
            >
              Sign in
            </Link>
            <Link
              href="/"
              className="inline-flex items-center justify-center rounded-md border border-slate-300 bg-white px-5 py-2.5 text-sm font-semibold text-brand-900 hover:bg-slate-50"
            >
              TIPER home
            </Link>
          </>
        )}
      </div>
    </PublicAuthShell>
  );
}
