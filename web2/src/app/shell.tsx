import { Link, Outlet, useRouterState } from "@tanstack/react-router";
import { NAV } from "./nav";

function isActive(pathname: string, to: string) {
  return pathname === to || pathname.startsWith(`${to}/`);
}

export function AppShell() {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const sections = [...new Set(NAV.map((n) => n.section))];
  const current = NAV.find((n) => isActive(pathname, n.to));

  return (
    <div className="flex min-h-full bg-background">
      <aside className="hidden w-64 shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground lg:flex">
        <div className="border-b border-sidebar-border px-5 py-5">
          <p className="text-[10px] uppercase tracking-[0.22em] text-sidebar-primary">TIPER</p>
          <p className="text-lg font-semibold text-sidebar-accent-foreground">Terminal OS</p>
          <p className="mt-1 text-xs text-sidebar-muted">Kigamboni bonded warehouse</p>
        </div>
        <nav className="flex-1 space-y-5 overflow-y-auto px-3 py-4">
          {sections.map((section) => (
            <div key={section}>
              <p className="mb-1 px-2 text-[10px] uppercase tracking-[0.16em] text-sidebar-muted">{section}</p>
              <ul className="space-y-0.5">
                {NAV.filter((n) => n.section === section).map((item) => {
                  const active = isActive(pathname, item.to);
                  return (
                    <li key={item.to}>
                      <Link
                        to={item.to}
                        className={`flex items-center rounded-md px-2 py-1.5 text-sm ${
                          active
                            ? "bg-sidebar-accent text-sidebar-accent-foreground"
                            : "text-sidebar-foreground/80 hover:bg-sidebar-accent/70"
                        }`}
                      >
                        <span className={`mr-2 h-1.5 w-1.5 rounded-full ${active ? "bg-sidebar-primary" : "bg-sidebar-muted"}`} />
                        {item.label}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </nav>
      </aside>
      <div className="min-w-0 flex-1">
        <header className="sticky top-0 z-20 border-b border-border bg-background/90 backdrop-blur">
          <div className="flex items-center justify-between px-6 py-4">
            <div>
              <p className="text-[10px] uppercase tracking-[0.18em] text-muted-foreground">DFMS</p>
              <h1 className="text-xl font-semibold">{current?.label ?? "TIPER"}</h1>
            </div>
            <div className="flex items-center gap-3 text-sm text-muted-foreground">
              <span className="hidden sm:inline">Day shift · Gate open</span>
              <span className="rounded-full border border-border bg-card px-3 py-1 text-foreground">Ops</span>
            </div>
          </div>
        </header>
        <main className="animate-in-fade px-6 py-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
