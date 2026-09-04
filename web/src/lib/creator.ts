export type CreatedBy = { id?: string; name?: string; email?: string };

export function creatorLabel(c?: CreatedBy | null): string {
  if (!c) return "—";
  const name = (c.name || "").trim();
  return name || c.email || "—";
}
