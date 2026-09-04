"use client";

import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { QtyInput } from "@/components/QtyInput";
import { DateInput, Field, Select, Textarea, ToggleField } from "@/components/ui";
import { masterOption, searchMaster } from "@/lib/master";
import { activeRows, catalogLabel, type BillingLookups } from "@/lib/lookups";
import type { ReceiptHeaderErrors, ReceiptHeaderValues, ReceiptKind } from "../_lib/header";

type Ref = { id: string; code?: string; name?: string };

export function ReceiptHeaderForm({
  kind,
  form,
  errors,
  catalogs,
  disabled,
  onChange,
}: {
  kind: ReceiptKind;
  form: ReceiptHeaderValues;
  errors: ReceiptHeaderErrors;
  catalogs?: BillingLookups;
  disabled?: boolean;
  onChange: (next: ReceiptHeaderValues) => void;
}) {
  const routes = activeRows(catalogs?.routes);
  const tenders = activeRows(catalogs?.tenders);
  const procurement = activeRows(catalogs?.procurementMethods);
  const kojRoute = form.routeCode.toUpperCase() === "KOJ";

  function patch(next: Partial<ReceiptHeaderValues>) {
    onChange({ ...form, ...next });
  }

  return (
    <div className="space-y-5">
      <section>
        <h3 className="mb-3 text-xs font-bold uppercase tracking-wide text-brand-800">
          Vessel and cargo
        </h3>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <Field label="Receipt date" required error={errors.date}>
            <DateInput
              value={form.date}
              disabled={disabled}
              onChange={(date) => patch({ date })}
            />
          </Field>
          <Field label="Vessel date" required error={errors.vesselDate}>
            <DateInput
              value={form.vesselDate}
              disabled={disabled}
              onChange={(vesselDate) => patch({ vesselDate })}
            />
          </Field>
          <Field label="Discharge route" required error={errors.routeCode}>
            <Select
              value={form.routeCode}
              disabled={disabled}
              onChange={(e) => {
                const routeCode = e.target.value;
                const stillKoj = routeCode.toUpperCase() === "KOJ";
                patch({
                  routeCode,
                  usesTiperPipeline: stillKoj && form.usesTiperPipeline,
                });
              }}
            >
              <option value="">Select…</option>
              {routes.map((r) => (
                <option key={r.code} value={r.code}>
                  {catalogLabel(r)}
                </option>
              ))}
            </Select>
          </Field>
          {kind === "internal" ? (
            <Field
              label="Density (MT / m³)"
              required
              error={errors.density}
              hint="Same number as kg/L — e.g. 0.84, not 840"
            >
              <QtyInput
                unit="density"
                value={form.density}
                disabled={disabled}
                onChange={(density) => patch({ density })}
              />
            </Field>
          ) : kojRoute ? (
            <div className="sm:col-span-2 xl:col-span-1">
              <ToggleField
                compact
                label="10-inch pipeline"
                description="Tick when cargo used the TIPER line that is billed. Leave off for the other KOJ pipe."
                checked={form.usesTiperPipeline}
                disabled={disabled}
                onChange={(usesTiperPipeline) => patch({ usesTiperPipeline })}
              />
            </div>
          ) : (
            <div className="hidden xl:block" />
          )}
          <Field label="Vessel" required error={errors.vesselId}>
            <AsyncSearchableSelect
              value={form.vesselId}
              selectedLabel={form.vesselLabel}
              placeholder="Search vessel"
              disabled={disabled}
              onChange={(value, option) =>
                patch({ vesselId: value, vesselLabel: option?.label || "" })
              }
              loadOptions={async (q) => (await searchMaster<Ref>("/master/vessels", q)).map(masterOption)}
            />
          </Field>
          <Field label="Product" required error={errors.productId}>
            <AsyncSearchableSelect
              value={form.productId}
              selectedLabel={form.productLabel}
              placeholder="Search product"
              disabled={disabled}
              onChange={(value, option) =>
                patch({ productId: value, productLabel: option?.label || "" })
              }
              loadOptions={async (q) => (await searchMaster<Ref>("/master/products", q)).map(masterOption)}
            />
          </Field>
          <div className="sm:col-span-2">
            <Field label="Notes" required error={errors.notes} hint="Voyage, survey, or operational remark">
              <Textarea
                value={form.notes}
                disabled={disabled}
                maxLength={500}
                rows={3}
                className="min-h-[4.25rem]"
                onChange={(e) => patch({ notes: e.target.value })}
              />
            </Field>
          </div>
        </div>
      </section>

      {kind === "internal" ? (
        <section>
          <h3 className="mb-3 text-xs font-bold uppercase tracking-wide text-brand-800">
            Commercial
          </h3>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <Field label="Supplier" required error={errors.supplierId}>
              <AsyncSearchableSelect
                value={form.supplierId}
                selectedLabel={form.supplierLabel}
                placeholder="Search supplier"
                disabled={disabled}
                onChange={(value, option) =>
                  patch({ supplierId: value, supplierLabel: option?.label || "" })
                }
                loadOptions={async (q) =>
                  (await searchMaster<Ref>("/master/suppliers", q)).map(masterOption)
                }
              />
            </Field>
            <Field label="Survey" required>
              <Select
                value={form.isProvision ? "provision" : "final"}
                disabled={disabled}
                onChange={(e) => patch({ isProvision: e.target.value === "provision" })}
              >
                <option value="provision">Provision (survey pending)</option>
                <option value="final">Final</option>
              </Select>
            </Field>
            <Field label="Nature of tender" required error={errors.tenderCode}>
              <Select
                value={form.tenderCode}
                disabled={disabled}
                onChange={(e) => patch({ tenderCode: e.target.value })}
              >
                <option value="">Select…</option>
                {tenders.map((r) => (
                  <option key={r.code} value={r.code}>
                    {catalogLabel(r)}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Method of tender" required error={errors.procurementMethodCode}>
              <Select
                value={form.procurementMethodCode}
                disabled={disabled}
                onChange={(e) => patch({ procurementMethodCode: e.target.value })}
              >
                <option value="">Select…</option>
                {procurement.map((r) => (
                  <option key={r.code} value={r.code}>
                    {catalogLabel(r)}
                  </option>
                ))}
              </Select>
            </Field>
          </div>
        </section>
      ) : null}
    </div>
  );
}
