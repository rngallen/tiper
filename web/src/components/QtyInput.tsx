"use client";

import { Input } from "@/components/ui";
import { formatQtyDraft, parseQty, type QtyUnit } from "@/lib/format";

/** Thousand-grouped quantity input. Decimal places follow Settings → Precision. */
export function QtyInput({
  value,
  onChange,
  disabled,
  placeholder = "0",
  unit = "l",
}: {
  value: string;
  onChange: (raw: string) => void;
  disabled?: boolean;
  placeholder?: string;
  unit?: QtyUnit;
}) {
  return (
    <Input
      inputMode="decimal"
      placeholder={placeholder}
      disabled={disabled}
      value={value ? formatQtyDraft(value, unit) : ""}
      onChange={(e) => onChange(parseQty(e.target.value))}
    />
  );
}
