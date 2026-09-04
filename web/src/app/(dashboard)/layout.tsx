"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";
import { Sidebar } from "@/components/Sidebar";
import { Topbar } from "@/components/Topbar";
import { Spinner } from "@/components/ui";
import { useAuth } from "@/lib/auth";
import { useCurrencyCatalog } from "@/lib/currencies";
import { useDecimalPrecision } from "@/lib/precision";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const { user, loading, mustChangePassword } = useAuth();
  useCurrencyCatalog();
  useDecimalPrecision();

  useEffect(() => {
    if (!loading && !user) {
      const search = typeof window !== "undefined" ? window.location.search : "";
      const next = `${pathname}${search}`;
      router.replace(`/login?next=${encodeURIComponent(next)}`);
      return;
    }
    if (!loading && mustChangePassword && pathname !== "/profile" && user) {
      router.replace("/profile?tab=password");
    }
  }, [loading, user, router, mustChangePassword, pathname]);

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Spinner />
      </div>
    );
  }

  if (!user) return null;

  return (
    <div className="flex h-screen overflow-hidden bg-[var(--pp-bg)]">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <Topbar />
        <main className="pp-main flex-1 overflow-y-auto p-6 md:p-8">{children}</main>
      </div>
    </div>
  );
}
