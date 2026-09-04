"use client";

import { Field, ToggleField } from "@/components/ui";
import { QtyInput } from "@/components/QtyInput";

export function FxRateField({
  rate,
  manual,
  onRate,
  onManual,
  disabled,
}: {
  rate: string;
  manual: boolean;
  onRate: (v: string) => void;
  onManual: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-2">
      <Field label="FX (TZS per 1 USD)">
        <QtyInput
          unit="fx"
          value={rate}
          onChange={onRate}
          disabled={disabled || !manual}
          placeholder={manual ? "Enter rate" : "Approved rate, or switch to manual"}
        />
      </Field>
      <p className="text-xs text-slate-500">
        Conversion: USD = TZS ÷ this rate. Approved quotes fill automatically; turn on manual to
        override.
      </p>
      <ToggleField
        label="Enter rate manually"
        checked={manual}
        onChange={onManual}
        disabled={disabled}
      />
    </div>
  );
}
