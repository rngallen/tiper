import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";

export const Route = createFileRoute("/login")({
  component: LoginPage,
});

function LoginPage() {
  const navigate = useNavigate();
  const [busy, setBusy] = useState(false);

  return (
    <div className="grid min-h-full lg:grid-cols-2">
      <section className="hidden flex-col justify-between bg-sidebar p-12 text-sidebar-foreground lg:flex">
        <div>
          <p className="text-[11px] uppercase tracking-[0.28em] text-sidebar-primary">TIPER Tanzania</p>
          <h1 className="mt-4 max-w-md text-4xl font-semibold leading-tight text-sidebar-accent-foreground">
            Command the terminal, not the spreadsheet.
          </h1>
          <p className="mt-5 max-w-sm text-sm text-sidebar-muted">
            Stock, gantry, pump-over, billing and EWURA in one bonded-warehouse console.
          </p>
        </div>
        <p className="text-sm text-sidebar-muted">Kigamboni · 24/7 operations</p>
      </section>
      <section className="flex items-center justify-center p-8">
        <form
          className="w-full max-w-sm space-y-4 rounded-xl border border-border bg-card p-8"
          onSubmit={(e) => {
            e.preventDefault();
            setBusy(true);
            setTimeout(() => navigate({ to: "/dashboard" }), 400);
          }}
        >
          <h2 className="text-2xl font-semibold">Sign in</h2>
          <p className="text-sm text-muted-foreground">Directory account. PASETO cookies via /api/v1/auth/login.</p>
          <label className="block text-sm">
            Email
            <input className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2" defaultValue="ngallen4@gmail.com" />
          </label>
          <label className="block text-sm">
            Password
            <input type="password" className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2" defaultValue="password" />
          </label>
          <button type="submit" className="w-full rounded-md bg-primary py-2 text-sm text-primary-foreground" disabled={busy}>
            {busy ? "Signing in…" : "Enter terminal"}
          </button>
        </form>
      </section>
    </div>
  );
}
