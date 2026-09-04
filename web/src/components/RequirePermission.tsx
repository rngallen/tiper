"use client";

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { ShieldAlert } from "lucide-react";
import { Button } from "@/components/ui";
import { useAuth } from "@/lib/auth";

/** Gate a page. No session → login. Signed in without the code → leave. */
export function RequirePermission({
  codes,
  children,
  resource = "this page",
}: {
  codes: string[];
  children: React.ReactNode;
  resource?: string;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const { user, loading, hasPermission, logout } = useAuth();

  useEffect(() => {
    if (loading) return;
    if (!user) {
      const search = typeof window !== "undefined" ? window.location.search : "";
      router.replace(`/login?next=${encodeURIComponent(`${pathname}${search}`)}`);
    }
  }, [loading, user, router, pathname]);

  if (loading || !user) return null;
  if (!hasPermission(...codes)) {
    return (
      <div className="mx-auto max-w-lg rounded-xl border border-slate-200 bg-white px-6 py-8">
        <div className="flex items-start gap-3">
          <span className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-brand-800 text-white">
            <ShieldAlert className="h-5 w-5" aria-hidden />
          </span>
          <div>
            <p className="text-sm font-semibold text-slate-900">
              You do not have access to {resource}
            </p>
            <p className="mt-1 text-sm leading-relaxed text-slate-600">
              This account cannot open {resource}. Go to a page you are allowed
              to use, or sign in as someone who can.
            </p>
          </div>
        </div>
        <div className="mt-6 flex flex-wrap gap-3">
          <Link href="/dashboard">
            <Button type="button">Go to dashboard</Button>
          </Link>
          <Button
            type="button"
            variant="secondary"
            onClick={() => void logout()}
          >
            Sign in as another user
          </Button>
        </div>
      </div>
    );
  }
  return children;
}
