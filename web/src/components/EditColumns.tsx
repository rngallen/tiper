"use client";

import { useEffect, useRef, useState } from "react";
import { Columns3, GripVertical } from "lucide-react";
import { Button } from "@/components/ui";
import type { ColumnDef } from "@/lib/columnPrefs";
import type { TableColumnPref } from "@/lib/types";

export function EditColumnsButton<T>({
  catalog,
  pref,
  onApply,
  onRestore,
}: {
  catalog: ColumnDef<T>[];
  pref: TableColumnPref;
  onApply: (next: TableColumnPref) => void | Promise<void>;
  onRestore: () => void | Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const [draftOrder, setDraftOrder] = useState<string[]>([]);
  const [draftHidden, setDraftHidden] = useState<Set<string>>(new Set());
  const [dragId, setDragId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const editable = catalog.filter((c) => !c.fixed);

  useEffect(() => {
    if (!open) return;
    const order =
      pref.order && pref.order.length > 0
        ? [
            ...pref.order.filter((id) => editable.some((c) => c.id === id)),
            ...editable
              .map((c) => c.id)
              .filter((id) => !(pref.order || []).includes(id)),
          ]
        : editable.map((c) => c.id);
    setDraftOrder(order);
    const hidden = new Set(
      pref.hidden ??
        editable.filter((c) => c.defaultVisible === false).map((c) => c.id),
    );
    setDraftHidden(hidden);
  }, [open, pref, catalog]);

  useEffect(() => {
    if (!open) return;
    function onDoc(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  function toggle(id: string) {
    setDraftHidden((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleAll(checked: boolean) {
    if (checked) setDraftHidden(new Set());
    else setDraftHidden(new Set(editable.map((c) => c.id)));
  }

  function onDrop(targetId: string) {
    if (!dragId || dragId === targetId) {
      setDragId(null);
      return;
    }
    setDraftOrder((prev) => {
      const next = prev.filter((id) => id !== dragId);
      const idx = next.indexOf(targetId);
      if (idx < 0) return prev;
      next.splice(idx, 0, dragId);
      return next;
    });
    setDragId(null);
  }

  async function apply() {
    setSaving(true);
    try {
      await onApply({
        order: draftOrder,
        hidden: Array.from(draftHidden),
      });
      setOpen(false);
    } finally {
      setSaving(false);
    }
  }

  const allChecked = editable.every((c) => !draftHidden.has(c.id));

  return (
    <div className="relative" ref={ref}>
      <Button
        variant="secondary"
        className="!gap-1.5"
        onClick={() => setOpen((v) => !v)}
      >
        <Columns3 className="h-4 w-4" />
        Edit columns
      </Button>
      {open && (
        <div className="absolute right-0 z-30 mt-2 w-80 overflow-hidden rounded-lg border border-slate-300 bg-white shadow-xl">
          <div className="bg-slate-800 px-3 py-2 text-sm font-semibold text-white">
            Edit columns
          </div>
          <div className="max-h-72 overflow-y-auto border-b border-slate-200">
            <label className="flex cursor-pointer items-center gap-2 border-b border-slate-100 px-3 py-2 text-sm font-medium text-slate-800 hover:bg-slate-50">
              <input
                type="checkbox"
                checked={allChecked}
                onChange={(e) => toggleAll(e.target.checked)}
              />
              All
            </label>
            {draftOrder.map((id) => {
              const col = editable.find((c) => c.id === id);
              if (!col) return null;
              return (
                <div
                  key={id}
                  draggable
                  onDragStart={() => setDragId(id)}
                  onDragOver={(e) => e.preventDefault()}
                  onDrop={() => onDrop(id)}
                  className={`flex cursor-grab items-center gap-2 border-b border-slate-50 px-3 py-2 text-sm text-slate-800 hover:bg-slate-50 active:cursor-grabbing ${
                    dragId === id ? "bg-brand-50" : ""
                  }`}
                >
                  <GripVertical className="h-4 w-4 shrink-0 text-slate-400" />
                  <input
                    type="checkbox"
                    checked={!draftHidden.has(id)}
                    onChange={() => toggle(id)}
                    onClick={(e) => e.stopPropagation()}
                  />
                  <span className="truncate">
                    {typeof col.header === "string" ? col.header : col.id}
                  </span>
                </div>
              );
            })}
          </div>
          <div className="flex items-center gap-2 px-3 py-3">
            <Button onClick={apply} loading={saving} className="!py-1.5">
              Apply
            </Button>
            <Button
              variant="secondary"
              className="!py-1.5"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
          </div>
          <button
            type="button"
            className="w-full border-t border-slate-200 px-3 py-2 text-left text-sm text-brand-800 hover:bg-slate-50"
            onClick={async () => {
              await onRestore();
              setOpen(false);
            }}
          >
            Restore table defaults
          </button>
        </div>
      )}
    </div>
  );
}
