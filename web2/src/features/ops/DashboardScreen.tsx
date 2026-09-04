import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { http } from "@/shared/api/http";
import { toPage } from "@/shared/api/list";
import { fmtQty, toNumber } from "@/shared/lib/format";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Skeleton } from "@/shared/ui/skeleton";
import { Badge } from "@/shared/ui/badge";

interface ChartPayload { title?: string; unit?: string; points?: { label: string; value: number }[] }
interface Tank { id: string; code?: string; name?: string; maximumCapacity?: string | number; product?: { code?: string } | null; bookQuantity?: string | number; quantity?: string | number }

export function DashboardScreen() {
  const kpis = useQuery({ queryKey: ["chart", "kpis"], queryFn: () => http.get<ChartPayload>("/reports/charts/kpis").catch(() => null) });
  const stock = useQuery({ queryKey: ["chart", "stock-by-product"], queryFn: () => http.get<ChartPayload>("/reports/charts/stock-by-product").catch(() => null) });
  const tanks = useQuery({ queryKey: ["tanks-farm"], queryFn: async () => toPage<Tank>(await http.get("/master/tanks", { page: 1, pageSize: 24, isActive: true })) });
  const inbox = useQuery({ queryKey: ["inbox-count"], queryFn: async () => toPage(await http.get("/workflow/tasks/mine", { page: 1, pageSize: 1 })).itemsCount });
  const kpiPoints = kpis.data?.points ?? [];
  const farm = tanks.data?.items ?? [];

  return (
    <div className="space-y-6">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {(kpiPoints.length ? kpiPoints.slice(0, 4) : [
          { label: "Stock on hand", value: NaN }, { label: "Open receipts", value: NaN },
          { label: "Gantry today", value: NaN }, { label: "Approvals", value: inbox.data ?? NaN },
        ]).map((p) => (
          <Card key={p.label}>
            <CardHeader><CardTitle className="text-xs font-medium uppercase tracking-wider text-muted-foreground">{p.label}</CardTitle></CardHeader>
            <CardContent><p className="text-2xl font-semibold tabular-nums">{Number.isFinite(p.value) ? fmtQty(p.value, p.value >= 1000 ? 0 : 2) : "—"}</p></CardContent>
          </Card>
        ))}
      </div>
      <div className="grid gap-4 xl:grid-cols-3">
        <Card className="xl:col-span-2">
          <CardHeader className="flex-row items-center justify-between">
            <CardTitle>Book stock by product</CardTitle>
            <Badge variant="outline">{stock.data?.unit || "m³"}</Badge>
          </CardHeader>
          <CardContent className="h-64">
            {stock.isLoading ? <Skeleton className="h-full w-full" /> : (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={(stock.data?.points ?? []).map((p) => ({ name: p.label, value: p.value }))}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} />
                  <XAxis dataKey="name" tick={{ fontSize: 11 }} />
                  <YAxis tick={{ fontSize: 11 }} />
                  <Tooltip />
                  <Bar dataKey="value" fill="var(--color-chart-1)" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Shift shortcuts</CardTitle></CardHeader>
          <CardContent className="grid gap-2">
            {[["/stock/receipts", "Internal receipts"], ["/stock/loading", "Gantry loadings"], ["/stock/pump-over", "Pump-over"], ["/approvals", "Approvals inbox"], ["/billing", "Billing runs"], ["/ewura", "EWURA licenses"]].map(([to, label]) => (
              <Link key={to} to={to} className="rounded-md border border-border px-3 py-2 text-sm hover:bg-accent">{label}</Link>
            ))}
          </CardContent>
        </Card>
      </div>
      <section>
        <div className="mb-3 flex items-end justify-between">
          <div>
            <h2 className="text-lg font-semibold">Tank farm</h2>
            <p className="text-sm text-muted-foreground">Live catalogue · capacity vs book where the API exposes it</p>
          </div>
          <Link to="/stock/setups/tanks" className="text-sm text-primary hover:underline">Manage tanks</Link>
        </div>
        {tanks.isLoading ? (
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4 xl:grid-cols-6">{Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-40" />)}</div>
        ) : (
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4 xl:grid-cols-6">
            {farm.map((t) => {
              const cap = toNumber(t.maximumCapacity) ?? 0;
              const qty = toNumber(t.bookQuantity ?? t.quantity) ?? 0;
              const pct = cap > 0 ? Math.max(0, Math.min(100, (qty / cap) * 100)) : 0;
              return (
                <div key={t.id} className="rounded-xl border border-border bg-card p-3 shadow-xs">
                  <p className="font-mono text-xs">{t.code}</p>
                  <div className="mt-2 h-24 overflow-hidden rounded-md bg-muted">
                    <div className="h-full origin-bottom bg-primary/80" style={{ transform: `translateY(${100 - pct}%)` }} />
                  </div>
                  <p className="mt-2 text-center text-sm font-medium tabular-nums">{pct.toFixed(0)}%</p>
                  <p className="text-center text-xs text-muted-foreground">{t.product?.code || t.name || "—"}</p>
                </div>
              );
            })}
          </div>
        )}
      </section>
    </div>
  );
}
