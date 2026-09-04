"use client";

import type { ChartPayload, DashboardWidget } from "@/lib/types";

const COLORS = ["#033860", "#0d7377", "#dc295c", "#c4a35a", "#3d5a80", "#ee6c4d", "#98c1d9", "#293241"];

function fmt(n: number): string {
  if (!Number.isFinite(n)) return "—";
  if (Math.abs(n) >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}m`;
  if (Math.abs(n) >= 1_000) return n.toLocaleString(undefined, { maximumFractionDigits: 0 });
  return n.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

function Empty({ label }: { label: string }) {
  return <p className="py-6 text-center text-sm text-slate-500">{label}</p>;
}

export function DashboardChart({
  widget,
  data,
}: {
  widget: DashboardWidget;
  data?: ChartPayload;
}) {
  const points = data?.points || [];
  if (widget.type === "kpi") {
    if (!points.length) return <Empty label="No figures yet." />;
    return (
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        {points.map((p) => (
          <div
            key={p.label}
            className="rounded-xl border border-slate-100 bg-gradient-to-br from-slate-50 to-white px-3 py-3"
          >
            <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">
              {p.label}
            </p>
            <p className="mt-1 text-2xl font-semibold tabular-nums tracking-tight text-[#033860]">
              {fmt(p.value)}
            </p>
          </div>
        ))}
      </div>
    );
  }
  if (widget.type === "table") {
    if (!points.length) return <Empty label="No rows." />;
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full text-sm">
          <thead>
            <tr className="text-left text-[11px] uppercase tracking-wide text-slate-500">
              <th className="pb-2 font-semibold">Label</th>
              <th className="pb-2 text-right font-semibold">Value</th>
            </tr>
          </thead>
          <tbody>
            {points.map((p) => (
              <tr key={p.label} className="border-t border-slate-100">
                <td className="py-1.5">{p.label}</td>
                <td className="py-1.5 text-right tabular-nums">{fmt(p.value)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }
  if (widget.type === "doughnut" || widget.type === "pie") {
    if (!points.length) return <Empty label="No breakdown yet." />;
    const total = points.reduce((s, p) => s + Math.max(0, p.value), 0);
    let acc = 0;
    const stops = points.map((p, i) => {
      const start = total ? (acc / total) * 360 : 0;
      acc += Math.max(0, p.value);
      const end = total ? (acc / total) * 360 : 0;
      return `${COLORS[i % COLORS.length]} ${start}deg ${end}deg`;
    });
    return (
      <div className="flex flex-wrap items-center gap-5">
        <div
          className="h-36 w-36 rounded-full shadow-inner"
          style={{
            background: total
              ? `conic-gradient(${stops.join(",")})`
              : "conic-gradient(#e2e8f0 0deg 360deg)",
            mask: widget.type === "pie" ? undefined : "radial-gradient(circle at center, transparent 48%, black 50%)",
            WebkitMask: widget.type === "pie" ? undefined : "radial-gradient(circle at center, transparent 48%, black 50%)",
          }}
        />
        <ul className="min-w-[11rem] space-y-1.5 text-xs">
          {points.map((p, i) => (
            <li key={p.label} className="flex items-center justify-between gap-3">
              <span className="flex items-center gap-2">
                <span className="h-2.5 w-2.5 rounded-full" style={{ background: COLORS[i % COLORS.length] }} />
                {p.label}
              </span>
              <span className="tabular-nums text-slate-600">{fmt(p.value)}</span>
            </li>
          ))}
        </ul>
      </div>
    );
  }

  if (!points.length) return <Empty label="No series yet." />;

  const max = Math.max(...points.map((p) => p.value), 1);
  const w = 520;
  const h = 168;
  const pad = 28;
  if (widget.type === "line") {
    const coords = points.map((p, i) => {
      const x = pad + (i * (w - pad * 2)) / Math.max(points.length - 1, 1);
      const y = h - pad - (p.value / max) * (h - pad * 2);
      return { x, y };
    });
    const line = coords.map((c) => `${c.x},${c.y}`).join(" ");
    const area = `${pad},${h - pad} ${line} ${coords[coords.length - 1]?.x ?? pad},${h - pad}`;
    return (
      <svg viewBox={`0 0 ${w} ${h}`} className="h-44 w-full">
        <defs>
          <linearGradient id="dfmsLineFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#033860" stopOpacity="0.22" />
            <stop offset="100%" stopColor="#033860" stopOpacity="0" />
          </linearGradient>
        </defs>
        <polygon fill="url(#dfmsLineFill)" points={area} />
        <polyline fill="none" stroke="#033860" strokeWidth="2.5" points={line} />
        {coords.map((c, i) => (
          <circle key={points[i].label + i} cx={c.x} cy={c.y} r="3" fill="#dc295c" />
        ))}
      </svg>
    );
  }

  const barW = (w - pad * 2) / points.length;
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="h-44 w-full">
      {points.map((p, i) => {
        const bh = (p.value / max) * (h - pad * 2);
        const x = pad + i * barW + 4;
        const y = h - pad - bh;
        return (
          <g key={p.label + i}>
            <rect
              x={x}
              y={y}
              width={Math.max(barW - 8, 4)}
              height={Math.max(bh, 1)}
              rx="3"
              fill={COLORS[i % COLORS.length]}
            />
            {points.length <= 12 ? (
              <text x={x + barW / 2 - 4} y={h - 8} fontSize="8" fill="#64748b" textAnchor="middle">
                {p.label.slice(0, 8)}
              </text>
            ) : null}
          </g>
        );
      })}
    </svg>
  );
}
