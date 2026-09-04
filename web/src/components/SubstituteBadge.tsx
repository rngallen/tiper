"use client";

/** Marks inbox rows and history as work covering another operator. */
export function SubstituteBadge({ covering }: { covering?: string }) {
  const name = covering?.trim();
  if (!name) return null;
  return (
    <span
      title={`Acting as substitute for ${name}. Decisions are recorded on their behalf.`}
      className="mt-0.5 inline-flex max-w-full items-center gap-1 rounded-full bg-amber-50 px-2 py-0.5 text-[11px] font-medium text-amber-950 ring-1 ring-inset ring-amber-200"
    >
      <span className="shrink-0 text-[10px] font-bold uppercase tracking-wide text-amber-800">
        Acting for
      </span>
      <span className="truncate font-semibold">{name}</span>
    </span>
  );
}
