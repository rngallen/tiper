import { useEffect, useMemo, useState } from "react";
import { Link, Outlet, useNavigate, useRouterState } from "@tanstack/react-router";
import { Toaster, toast } from "sonner";
import { LogOut, Moon, Search, Sun } from "lucide-react";
import { httpEvents } from "@/shared/api/http";
import { errorMessage, errorTitle } from "@/shared/api/errors";
import { Button } from "@/shared/ui/button";
import { ScrollArea } from "@/shared/ui/scroll-area";
import { CommandDialog, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/shared/ui/command";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/shared/ui/dropdown-menu";
import { Spinner } from "@/shared/ui/spinner";
import { fullName, initials } from "@/shared/lib/format";
import { useSession } from "@/features/auth/session";
import { NAV, NAV_SECTIONS } from "./nav";

function isActive(pathname: string, to: string) {
  if (to === "/dashboard") return pathname === "/dashboard";
  return pathname === to || pathname.startsWith(`${to}/`);
}

export function AppShell() {
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const { user, loading, ready, boot, logout, can } = useSession();
  const [palette, setPalette] = useState(false);
  const [dark, setDark] = useState(() => typeof document !== "undefined" && document.documentElement.classList.contains("dark"));

  useEffect(() => { if (!ready) void boot(); }, [ready, boot]);
  useEffect(() => { if (ready && !loading && !user) void navigate({ to: "/login" }); }, [ready, loading, user, navigate]);
  useEffect(() => {
    const off1 = httpEvents.on("success:message", ({ message }) => toast.success(message));
    const off2 = httpEvents.on("request:error", ({ error }) => { if (error.status !== 401) toast.error(errorTitle(error), { description: errorMessage(error) }); });
    const off3 = httpEvents.on("session:rejected", () => { toast.message("Session expired"); void navigate({ to: "/login" }); });
    return () => { off1(); off2(); off3(); };
  }, [navigate]);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") { e.preventDefault(); setPalette((v) => !v); }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const visible = useMemo(() => NAV.filter((n) => can(...n.perms)), [can, user, ready]);
  const current = visible.find((n) => isActive(pathname, n.to));

  if (!ready || loading || !user) {
    return <div className="flex min-h-full items-center justify-center"><Spinner /></div>;
  }

  return (
    <div className="flex min-h-full bg-background">
      <aside className="hidden w-64 shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground lg:flex">
        <div className="border-b border-sidebar-border px-5 py-5">
          <p className="text-[10px] uppercase tracking-[0.22em] text-sidebar-primary">TIPER DFMS</p>
          <p className="text-lg font-semibold text-sidebar-accent-foreground">Terminal OS</p>
          <p className="mt-1 text-xs text-sidebar-muted">Kigamboni bonded warehouse</p>
        </div>
        <ScrollArea className="flex-1">
          <nav className="space-y-4 px-3 py-4">
            {NAV_SECTIONS.map((section) => {
              const items = visible.filter((n) => n.section === section);
              if (!items.length) return null;
              return (
                <div key={section}>
                  <p className="mb-1 px-2 text-[10px] uppercase tracking-[0.16em] text-sidebar-muted">{section}</p>
                  <ul className="space-y-0.5">
                    {items.map((item) => {
                      const active = isActive(pathname, item.to);
                      const Icon = item.icon;
                      return (
                        <li key={item.to}>
                          <Link to={item.to} className={`flex items-center gap-2 rounded-md px-2 py-1.5 text-[13px] ${active ? "bg-sidebar-accent text-sidebar-accent-foreground" : "text-sidebar-foreground/80 hover:bg-sidebar-accent/70"}`}>
                            <Icon className="size-3.5 shrink-0 opacity-80" />
                            <span className="truncate">{item.label}</span>
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                </div>
              );
            })}
          </nav>
        </ScrollArea>
      </aside>
      <div className="min-w-0 flex-1">
        <header className="sticky top-0 z-20 border-b border-border bg-background/90 backdrop-blur">
          <div className="flex items-center justify-between gap-3 px-4 py-3 sm:px-6">
            <div className="min-w-0">
              <p className="text-[10px] uppercase tracking-[0.18em] text-muted-foreground">DFMS</p>
              <h1 className="truncate text-lg font-semibold">{current?.label ?? "TIPER"}</h1>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => setPalette(true)}>
                <Search className="size-3.5" /> Search
                <kbd className="ml-2 hidden rounded border border-border px-1.5 font-mono text-[10px] sm:inline">⌘K</kbd>
              </Button>
              <Button variant="ghost" size="icon-sm" onClick={() => { const next = !dark; setDark(next); document.documentElement.classList.toggle("dark", next); }}>
                {dark ? <Sun className="size-4" /> : <Moon className="size-4" />}
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm" className="gap-2">
                    <span className="flex size-6 items-center justify-center rounded-full bg-primary text-[10px] text-primary-foreground">{initials(user.firstName, user.lastName)}</span>
                    <span className="hidden max-w-[140px] truncate sm:inline">{fullName(user) || user.email}</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => navigate({ to: "/settings" })}>Settings</DropdownMenuItem>
                  <DropdownMenuItem onClick={async () => { await logout(); await navigate({ to: "/login" }); }}>
                    <LogOut className="size-4" /> Sign out
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </header>
        <main className="animate-in-fade px-4 py-5 sm:px-6">
          <Outlet />
        </main>
      </div>
      <CommandDialog open={palette} onOpenChange={setPalette}>
        <CommandInput placeholder="Jump to a workspace…" />
        <CommandList>
          <CommandEmpty>No match.</CommandEmpty>
          {NAV_SECTIONS.map((section) => {
            const items = visible.filter((n) => n.section === section);
            if (!items.length) return null;
            return (
              <CommandGroup key={section} heading={section}>
                {items.map((item) => (
                  <CommandItem key={item.to} value={`${item.label} ${item.to}`} onSelect={() => { setPalette(false); void navigate({ to: item.to }); }}>
                    <item.icon className="size-4" />
                    {item.label}
                  </CommandItem>
                ))}
              </CommandGroup>
            );
          })}
        </CommandList>
      </CommandDialog>
      <Toaster richColors position="top-right" />
    </div>
  );
}
