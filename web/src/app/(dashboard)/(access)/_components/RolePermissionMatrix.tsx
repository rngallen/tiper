"use client";

import { useEffect, useState } from "react";
import { ReviewTabBar } from "@/components/ApprovalReview";
import { ToggleField } from "@/components/ui";
import {
  groupModuleResources,
  permissionModuleLabel,
} from "@/lib/permissionGroups";
import type { Permission } from "@/lib/types";

export type AreaGroup<T extends { code: string; module?: string }> = {
  area: string;
  modules: [string, T[]][];
};

function areaCount<T>(modules: [string, T[]][]) {
  return modules.reduce((n, [, list]) => n + list.length, 0);
}

function areaSelected(modules: [string, Permission[]][], permIds: number[]) {
  return modules.reduce(
    (n, [, list]) => n + list.filter((p) => permIds.includes(p.id)).length,
    0,
  );
}

/** One tab per ERP area (Overview, Stock, Billing, …). Header fields stay above this strip. */
export function PermissionTabs({
  tab,
  onChange,
  byArea,
  permIds,
}: {
  tab: string;
  onChange: (t: string) => void;
  byArea: AreaGroup<Permission>[];
  /** When set, tab badges are selected counts on edit. */
  permIds?: number[];
}) {
  if (byArea.length === 0) return null;
  return (
    <ReviewTabBar
      ariaLabel="Permission areas"
      tab={tab}
      onChange={onChange}
      ids={byArea.map(({ area, modules }) => {
        const total = areaCount(modules);
        const selected = permIds ? areaSelected(modules, permIds) : total;
        return {
          id: area,
          label: area,
          count: permIds ? selected : total,
        };
      })}
    />
  );
}

export function usePermissionTab(byArea: AreaGroup<Permission>[], filter = "") {
  const [tab, setTab] = useState("");
  useEffect(() => {
    if (byArea.some((a) => a.area === tab)) return;
    setTab(byArea[0]?.area ?? "");
  }, [byArea, tab]);
  useEffect(() => {
    if (!filter.trim()) return;
    if (byArea.some((a) => a.area === tab)) return;
    if (byArea[0]) setTab(byArea[0].area);
  }, [filter, byArea, tab]);
  return { tab, setTab };
}

export function PermissionInquiryBody({
  group,
}: {
  group: AreaGroup<Permission> | undefined;
}) {
  if (!group || group.modules.length === 0) {
    return <p className="px-1 py-6 text-sm text-slate-500">No permissions in this area.</p>;
  }
  return (
    <div className="max-h-[min(24rem,48vh)] space-y-3 overflow-y-auto pr-1">
      {group.modules.map(([module, list]) => (
        <section key={module} className="overflow-hidden rounded-lg border border-slate-200">
          <div className="flex items-center justify-between bg-slate-50 px-3 py-2">
            <h5 className="text-xs font-bold uppercase tracking-wide text-brand-800">
              {permissionModuleLabel(module)}
            </h5>
            <span className="text-[11px] font-semibold tabular-nums text-slate-500">
              {list.length}
            </span>
          </div>
          {groupModuleResources(list).map((res) => (
            <div key={res.key}>
              {res.label ? (
                <div className="border-t border-slate-100 bg-white px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                  {res.label}
                </div>
              ) : null}
              <table className="pp-ledger min-w-full text-left text-sm">
                <tbody>
                  {res.perms.map((p) => (
                    <tr key={p.id}>
                      <td className="w-[40%] font-mono text-xs font-semibold text-slate-900">
                        {p.code}
                      </td>
                      <td className="text-slate-700">{p.description || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </section>
      ))}
    </div>
  );
}

function SelectAllSwitch({
  on,
  some,
  label,
  onToggle,
}: {
  on: boolean;
  some: boolean;
  label: string;
  onToggle: () => void;
}) {
  return (
    <div className="flex shrink-0 items-center gap-2">
      <span className="text-xs font-medium text-slate-600">Select all</span>
      <button
        type="button"
        role="switch"
        aria-checked={on}
        aria-label={label}
        onClick={onToggle}
        className={`relative h-5 w-9 rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-brand-200 focus:ring-offset-1 ${
          on ? "bg-brand-800" : some ? "bg-amber-400" : "bg-slate-300"
        }`}
      >
        <span
          className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform ${
            on
              ? "left-0.5 translate-x-4"
              : some
                ? "left-1/2 -translate-x-1/2"
                : "left-0.5 translate-x-0"
          }`}
        />
      </button>
    </div>
  );
}

export function PermissionEditBody({
  group,
  permIds,
  onToggle,
  onToggleMany,
}: {
  group: AreaGroup<Permission> | undefined;
  permIds: number[];
  onToggle: (id: number) => void;
  onToggleMany: (perms: Permission[], on: boolean) => void;
}) {
  if (!group || group.modules.length === 0) {
    return <p className="px-1 py-6 text-sm text-slate-500">No permissions match this area.</p>;
  }
  const areaPerms = group.modules.flatMap(([, list]) => list);
  const areaSelected = areaPerms.filter((p) => permIds.includes(p.id)).length;
  const areaAll = areaSelected === areaPerms.length && areaPerms.length > 0;
  const areaSome = areaSelected > 0 && !areaAll;

  return (
    <div className="max-h-[min(26rem,50vh)] space-y-3 overflow-y-auto pr-1">
      <div className="flex items-center justify-between gap-3 px-1">
        <p className="text-xs text-slate-500">
          {areaSelected} of {areaPerms.length} selected in {group.area}
        </p>
        <SelectAllSwitch
          on={areaAll}
          some={areaSome}
          label={`Select all ${group.area} permissions`}
          onToggle={() => onToggleMany(areaPerms, !areaAll)}
        />
      </div>
      {group.modules.map(([module, perms]) => {
        const selected = perms.filter((p) => permIds.includes(p.id)).length;
        const allOn = selected === perms.length && perms.length > 0;
        const someOn = selected > 0 && !allOn;
        return (
          <section key={module} className="overflow-hidden rounded-lg border border-slate-200">
            <div className="flex items-center justify-between gap-3 bg-slate-100 px-3 py-2">
              <div className="min-w-0">
                <div className="text-xs font-semibold uppercase tracking-wide text-slate-700">
                  {permissionModuleLabel(module)}
                </div>
                <p className="text-xs text-slate-500">
                  {someOn
                    ? `${selected} of ${perms.length} · partial`
                    : allOn
                      ? `${perms.length} · all on`
                      : `${perms.length} · none`}
                </p>
              </div>
              <SelectAllSwitch
                on={allOn}
                some={someOn}
                label={`Select all ${permissionModuleLabel(module)} permissions`}
                onToggle={() => onToggleMany(perms, !allOn)}
              />
            </div>
            <div className="space-y-3 bg-slate-50/50 p-2">
              {groupModuleResources(perms).map((res) => (
                <div key={res.key} className="space-y-1.5">
                  {res.label ? (
                    <div className="flex items-center justify-between gap-2 px-1 pt-1">
                      <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                        {res.label}
                      </span>
                      <button
                        type="button"
                        className="text-[11px] font-semibold text-brand-800"
                        onClick={() => {
                          const all = res.perms.every((p) => permIds.includes(p.id));
                          onToggleMany(res.perms, !all);
                        }}
                      >
                        {res.perms.every((p) => permIds.includes(p.id)) ? "Clear" : "Select"}
                      </button>
                    </div>
                  ) : null}
                  {res.perms.map((p) => (
                    <ToggleField
                      key={p.id}
                      compact
                      label={p.code}
                      description={p.description}
                      checked={permIds.includes(p.id)}
                      onChange={() => onToggle(p.id)}
                    />
                  ))}
                </div>
              ))}
            </div>
          </section>
        );
      })}
    </div>
  );
}
