"use client";

import { useMemo, useState } from "react";
import { BarChart3, FileText, Search, Ship, Truck, Users, Warehouse } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { REPORTS, type ReportCard } from "@/lib/reportsCatalog";
import { ReportRunner } from "@/components/ReportRunner";
import { ListSearch } from "@/lib/list";

const ICONS: Record<string, LucideIcon> = {
  Stock: Warehouse,
  Reception: Ship,
  Master: FileText,
  Gantry: Truck,
  Documents: FileText,
  Access: Users,
  Billing: BarChart3,
};

export function ReportCatalog({
  groups,
  items = REPORTS,
}: {
  groups: readonly string[];
  items?: ReportCard[];
}) {
  const [open, setOpen] = useState<ReportCard | null>(null);
  const [search, setSearch] = useState("");
  const [group, setGroup] = useState("");

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    return items.filter((r) => {
      if (!groups.includes(r.group)) return false;
      if (group && r.group !== group) return false;
      if (!q) return true;
      return `${r.title} ${r.description} ${r.group}`.toLowerCase().includes(q);
    });
  }, [items, groups, group, search]);

  const byGroup = useMemo(() => {
    return groups
      .map((g) => ({ group: g, rows: visible.filter((r) => r.group === g) }))
      .filter((g) => g.rows.length > 0);
  }, [groups, visible]);

  return (
    <>
      <div className="mb-5 flex flex-wrap items-end gap-3">
        <div className="min-w-[16rem] flex-1">
          <ListSearch
            value={search}
            onChange={setSearch}
            placeholder="Search reports by name or topic"
          />
        </div>
        <p className="pb-2 text-xs font-medium text-slate-500">
          {visible.length} report{visible.length === 1 ? "" : "s"}
        </p>
      </div>
      <div className="mb-6 flex flex-wrap gap-2">
        <GroupChip
          label="All"
          active={!group}
          onClick={() => setGroup("")}
        />
        {groups.map((g) => (
          <GroupChip
            key={g}
            label={g}
            active={group === g}
            onClick={() => setGroup(group === g ? "" : g)}
          />
        ))}
      </div>

      {byGroup.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-slate-200 bg-white px-6 py-12 text-center">
          <Search className="mx-auto h-8 w-8 text-slate-300" aria-hidden />
          <p className="mt-3 text-sm font-semibold text-slate-800">No reports match</p>
          <p className="mt-1 text-sm text-slate-500">
            Try another word, or clear the group filter.
          </p>
        </div>
      ) : (
        byGroup.map((section) => {
          return (
            <section key={section.group} className="mb-8">
              <h2 className="mb-3 text-xs font-bold uppercase tracking-wide text-slate-500">
                {section.group}
                <span className="ml-2 font-medium text-slate-400">{section.rows.length}</span>
              </h2>
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                {section.rows.map((r) => {
                  const Icon = ICONS[r.group] || BarChart3;
                  return (
                    <button
                      key={r.href}
                      type="button"
                      onClick={() => setOpen(r)}
                      className="group block rounded-2xl border border-slate-200 bg-white p-5 text-left transition hover:border-brand-800/30 hover:shadow-sm focus:outline-none focus:ring-2 focus:ring-brand-300"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-800/10 text-brand-800">
                          <Icon size={20} strokeWidth={1.75} />
                        </div>
                        <span className="text-xs font-semibold text-brand-800 opacity-0 transition group-hover:opacity-100">
                          Open
                        </span>
                      </div>
                      <h3 className="mt-4 text-sm font-semibold text-slate-900">{r.title}</h3>
                      <p className="mt-1.5 text-xs leading-relaxed text-slate-500">
                        {r.description}
                      </p>
                    </button>
                  );
                })}
              </div>
            </section>
          );
        })
      )}
      {open ? (
        <ReportRunner
          href={open.href}
          title={open.title}
          description={open.description}
          onClose={() => setOpen(null)}
        />
      ) : null}
    </>
  );
}

function GroupChip({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-full px-3 py-1 text-xs font-semibold transition ${
        active
          ? "bg-brand-800 text-white"
          : "border border-slate-200 bg-white text-slate-600 hover:border-slate-300"
      }`}
    >
      {label}
    </button>
  );
}
