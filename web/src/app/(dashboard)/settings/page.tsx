"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Spinner } from "@/components/ui";
import { useAuth } from "@/lib/auth";
import { firstAllowedSettingsHref } from "./_components/nav";

export default function SettingsIndexPage() {
  const { hasPermission, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading) return;
    router.replace(firstAllowedSettingsHref(hasPermission));
  }, [loading, hasPermission, router]);

  return <Spinner />;
}
