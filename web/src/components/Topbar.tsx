"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { ChevronDown, LogOut, UserRound, Users } from "lucide-react";
import { usePathname } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { ApprovalBell } from "@/components/ApprovalBell";

export function Topbar() {
  const pathname = usePathname();
  const { user, logout, mustChangePassword } = useAuth();
  const isDashboard = pathname === "/dashboard";
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const initials = (
    [user?.firstName?.[0], user?.lastName?.[0]].filter(Boolean).join("") ||
    user?.email?.slice(0, 2) ||
    "?"
  ).toUpperCase();

  const displayName =
    [user?.firstName, user?.lastName].filter(Boolean).join(" ") || user?.email;

  useEffect(() => {
    function onDocClick(e: MouseEvent) {
      if (!menuRef.current?.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDocClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDocClick);
      document.removeEventListener("keydown", onKey);
    };
  }, []);

  return (
    <header className="pp-topbar flex h-16 shrink-0 items-center justify-between gap-4 px-6">
      <div className="min-w-0">
        {isDashboard ? (
          <>
            <div className="pp-topbar-title truncate text-sm font-semibold">
              Welcome back{user?.firstName ? `, ${user.firstName}` : ""}
            </div>
          </>
        ) : (
          <>
            <div className="pp-topbar-title truncate text-sm font-semibold">
              TIPER DFMS
            </div>
            <p className="pp-topbar-sub truncate text-xs">
              {new Date().toLocaleDateString("en-GB", {
                weekday: "short",
                day: "numeric",
                month: "short",
                year: "numeric",
              })}
            </p>
          </>
        )}
      </div>

      <div className="flex shrink-0 items-center gap-1">
        {!mustChangePassword && <ApprovalBell />}
        <div className="relative" ref={menuRef}>
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            className="flex items-center gap-2 rounded-lg px-2 py-1.5 transition hover:bg-slate-50"
            aria-expanded={open}
            aria-haspopup="menu"
          >
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-brand-800 text-xs font-semibold text-white">
            {initials}
          </div>
          <span className="pp-topbar-sub hidden max-w-[180px] truncate text-sm sm:block">
            {user?.email}
          </span>
          <ChevronDown
            size={16}
            className={`text-slate-400 transition ${open ? "rotate-180" : ""}`}
          />
        </button>

        {open && (
          <div
            role="menu"
            className="absolute right-0 z-40 mt-2 w-56 overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg"
          >
            <div className="border-b border-slate-100 px-3 py-2">
              <p className="truncate text-sm font-medium text-slate-900">
                {displayName}
              </p>
              <p className="truncate text-xs text-slate-500">{user?.email}</p>
            </div>
            {!mustChangePassword && (
              <>
            <Link
              href="/profile"
              role="menuitem"
              onClick={() => setOpen(false)}
              className="flex items-center gap-2 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
            >
              <UserRound size={16} /> My profile
            </Link>
            <Link
              href="/profile?tab=password"
              role="menuitem"
              onClick={() => setOpen(false)}
              className="block px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
            >
              Change password
            </Link>
            <Link
              href="/profile?tab=appearance"
              role="menuitem"
              onClick={() => setOpen(false)}
              className="block px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
            >
              Appearance
            </Link>
            <Link
              href="/profile?tab=delegation"
              role="menuitem"
              onClick={() => setOpen(false)}
              className="flex items-center gap-2 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
            >
              <Users size={16} /> Substitute
            </Link>
            <Link
              href="/profile?tab=roles"
              role="menuitem"
              onClick={() => setOpen(false)}
              className="block px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
            >
              My roles
            </Link>
              </>
            )}
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false);
                void logout();
              }}
              className="flex w-full items-center gap-2 border-t border-slate-100 px-3 py-2 text-sm text-slate-600 hover:bg-slate-50"
            >
              <LogOut size={16} /> Sign out
            </button>
          </div>
        )}
        </div>
      </div>
    </header>
  );
}
