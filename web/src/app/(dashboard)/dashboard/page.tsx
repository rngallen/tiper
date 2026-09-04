"use client";

import { useMemo, useState } from "react";
import { GripVertical, LayoutDashboard, RotateCcw, X } from "lucide-react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useAsync } from "@/lib/hooks";
import { Button, Card, PageHeader, Select, Spinner } from "@/components/ui";
import { DashboardChart } from "@/components/DashboardChart";
import type {
  ChartCatalogItem,
  ChartPayload,
  DashboardWidget,
} from "@/lib/types";

const DEFAULT_WIDGETS: DashboardWidget[] = [
  { id: "kpis", type: "kpi", title: "Operations snapshot", source: "kpis", span: 2 },
  { id: "stock", type: "bar", title: "Book stock by product", source: "stock-by-product", span: 1 },
  { id: "routes", type: "pie", title: "Receipts by route", source: "receipts-by-route", span: 1 },
  { id: "flow", type: "line", title: "Daily throughput", source: "throughput-daily", span: 2 },
  { id: "wf", type: "bar", title: "Approvals waiting", source: "workflow-aging", span: 1 },
  { id: "orders", type: "table", title: "Open orders", source: "open-orders", span: 1 },
];

const CHART_LABEL: Record<DashboardWidget["type"], string> = {
  kpi: "Figures",
  bar: "Bar",
  line: "Line",
  doughnut: "Donut",
  pie: "Pie",
  table: "Table",
};

function ChartCard({
  widget,
  editing,
  onDragStart,
  onDrop,
}: {
  widget: DashboardWidget;
  editing: boolean;
  onDragStart: () => void;
  onDrop: () => void;
}) {
  const { data, loading } = useAsync(
    () => api.get<ChartPayload>(`/reports/charts/${widget.source}`),
    [widget.source],
  );
  return (
    <div
      className={widget.span === 2 ? "sm:col-span-2" : ""}
      onDragOver={(e) => {
        e.preventDefault();
      }}
      onDrop={(e) => {
        e.preventDefault();
        onDrop();
      }}
    >
      <Card className={`overflow-hidden p-5 ${editing ? "ring-1 ring-slate-400" : ""}`}>
        <div className="mb-3 flex items-start justify-between gap-3">
          <div>
            <h2 className="text-[15px] font-semibold text-slate-950">{widget.title}</h2>
            {data?.unit ? <p className="text-sm font-medium text-slate-700">{data.unit}</p> : null}
          </div>
          <button
            type="button"
            draggable
            onDragStart={onDragStart}
            className="cursor-grab rounded p-1 text-slate-600 hover:bg-slate-100 hover:text-slate-950"
            aria-label={`Drag ${widget.title}`}
          >
            <GripVertical className="h-5 w-5" />
          </button>
        </div>
        {loading ? <Spinner /> : <DashboardChart widget={widget} data={data ?? undefined} />}
      </Card>
    </div>
  );
}

export default function DashboardPage() {
  const { user, refreshUser } = useAuth();
  const catalog = useAsync(
    () => api.get<ChartCatalogItem[]>("/reports/charts/catalog"),
    [],
  );
  const saved = user?.profile?.appearanceSettings?.dashboard;
  const [widgets, setWidgets] = useState<DashboardWidget[] | null>(null);
  const [picker, setPicker] = useState(false);
  const [types, setTypes] = useState<Record<string, DashboardWidget["type"]>>({});
  const [saving, setSaving] = useState(false);
  const [dragging, setDragging] = useState<string | null>(null);

  const layout = widgets ?? (saved && saved.length ? saved : DEFAULT_WIDGETS);
  const firstName = user?.firstName || user?.email?.split("@")[0] || "there";
  const today = new Date().toLocaleDateString("en-GB", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  });
  const items = catalog.data || [];

  const onBoard = useMemo(() => new Set(layout.map((w) => w.source)), [layout]);

  async function persist(next: DashboardWidget[]) {
    setSaving(true);
    try {
      await api.put("/auth/profile/dashboard", { widgets: next });
      setWidgets(next);
      await refreshUser();
    } finally {
      setSaving(false);
    }
  }

  function chartType(item: ChartCatalogItem): DashboardWidget["type"] {
    const chosen = types[item.code];
    if (chosen && item.types.includes(chosen)) return chosen;
    return item.types[0] || "bar";
  }

  function toggle(item: ChartCatalogItem, on: boolean) {
    if (on) {
      if (onBoard.has(item.code) || layout.length >= 16) return;
      const type = chartType(item);
      void persist([
        ...layout,
        {
          id: `${item.code}-${Date.now().toString(36)}`,
          type,
          title: item.name,
          source: item.code,
          span: type === "kpi" || type === "line" ? 2 : 1,
        },
      ]);
      return;
    }
    void persist(layout.filter((w) => w.source !== item.code));
  }

  function setType(code: string, type: DashboardWidget["type"]) {
    setTypes((prev) => ({ ...prev, [code]: type }));
    const next = layout.map((w) => (w.source === code ? { ...w, type } : w));
    if (next.some((w, i) => w.type !== layout[i].type)) void persist(next);
  }

  function dropOn(targetId: string) {
    if (!dragging || dragging === targetId) return;
    const from = layout.findIndex((w) => w.id === dragging);
    const to = layout.findIndex((w) => w.id === targetId);
    if (from < 0 || to < 0) return;
    const next = [...layout];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    setDragging(null);
    void persist(next);
  }

  return (
    <div className="space-y-5">
      <div className="overflow-hidden rounded-2xl bg-[#033860] px-6 py-5 text-white shadow-sm">
        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-white/80">
          {today}
        </p>
        <h1 className="mt-1 text-2xl font-semibold tracking-tight">
          Good day, {firstName}
        </h1>
        <p className="mt-1 max-w-2xl text-[15px] font-medium text-white">
          Choose the charts for your work, pick bar, pie, or table, then drag them into order.
          The layout is saved on your profile only.
        </p>
      </div>

      <PageHeader
        title="My operations"
        subtitle={`${layout.length} chart${layout.length === 1 ? "" : "s"} on this board`}
        actions={
          <Button variant="secondary" onClick={() => setPicker((v) => !v)}>
            <LayoutDashboard className="h-4 w-4" />
            Widgets
          </Button>
        }
      />

      {picker ? (
        <div className="fixed inset-0 z-40 flex justify-end bg-slate-950/30" onClick={() => setPicker(false)}>
          <aside
            className="flex h-full w-full max-w-md flex-col border-l border-slate-400 bg-white shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b border-slate-300 px-5 py-4">
              <div>
                <h2 className="text-lg font-semibold text-slate-950">Widgets</h2>
                <p className="text-sm font-medium text-slate-700">Show, hide, and choose the chart type.</p>
              </div>
              <button type="button" className="rounded p-1 text-slate-600 hover:bg-slate-100" onClick={() => setPicker(false)}>
                <X className="h-5 w-5" />
              </button>
            </div>
            <ul className="flex-1 space-y-1 overflow-y-auto px-3 py-3">
              {items.map((item) => {
                const checked = onBoard.has(item.code);
                const type = chartType(item);
                return (
                  <li key={item.code} className="rounded-md border border-slate-300 px-3 py-3">
                    <label className="flex items-start gap-3">
                      <input
                        type="checkbox"
                        className="mt-1 h-4 w-4 accent-brand-800"
                        checked={checked}
                        disabled={saving}
                        onChange={(e) => toggle(item, e.target.checked)}
                      />
                      <span className="min-w-0 flex-1">
                        <span className="block text-[15px] font-semibold text-slate-950">{item.name}</span>
                        <span className="mt-0.5 block text-sm font-medium text-slate-700">{item.description}</span>
                      </span>
                    </label>
                    <div className="mt-2 pl-7">
                      <Select
                        value={type}
                        disabled={!item.types.length}
                        onChange={(e) => setType(item.code, e.target.value as DashboardWidget["type"])}
                      >
                        {item.types.map((t) => (
                          <option key={t} value={t}>
                            {CHART_LABEL[t] || t}
                          </option>
                        ))}
                      </Select>
                    </div>
                  </li>
                );
              })}
            </ul>
            <div className="border-t border-slate-300 p-4">
              <Button variant="secondary" onClick={() => void persist(DEFAULT_WIDGETS)} loading={saving}>
                <RotateCcw className="h-4 w-4" />
                Reset to default
              </Button>
            </div>
          </aside>
        </div>
      ) : null}

      <p className="text-sm font-medium text-slate-700">
        Drag the handle on a card to rearrange. Open Widgets to add customer outstanding, stock, or approvals.
      </p>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {layout.map((w) => (
          <ChartCard
            key={w.id}
            widget={w}
            editing={picker}
            onDragStart={() => setDragging(w.id)}
            onDrop={() => dropOn(w.id)}
          />
        ))}
      </div>
    </div>
  );
}
