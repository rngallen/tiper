"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { notify } from "@/lib/notify";
import { CatalogueForm } from "@/app/(dashboard)/stock/setups/form";
import { Field, Input, Select } from "@/components/ui";
import { DateInput } from "@/components/ui";
import {
  fetchApprovedFx,
  fetchApprovedMiLoss,
  fetchHomeCurrency,
  clampEffectiveToDoc,
  effectiveBeforeDocError,
  isoDay,
  latestMiLossAsOf,
  miLossBatchLabel,
  miLossOnOrBefore,
  type MiLossBatch,
} from "@/lib/billing";
import { FxRateField } from "./FxRateField";

export function CreateHeaderModal({
  title,
  path,
  extra,
  withFx = true,
  requireMiLoss = false,
  onClose,
  onCreated,
}: {
  title: string;
  path: string;
  extra?: Record<string, unknown>;
  withFx?: boolean;
  requireMiLoss?: boolean;
  onClose: () => void;
  onCreated: (id: string) => void;
}) {
  const [date, setDate] = useState(isoDay());
  const [effectiveFrom, setEff] = useState(isoDay());
  const [description, setDescription] = useState("");
  const [homeCcy, setHomeCcy] = useState("");
  const [exchangeRate, setFx] = useState("");
  const [fxManual, setManual] = useState(false);
  const [miLossBatchId, setMiLossId] = useState("");
  const [miLossRows, setMiLossRows] = useState<MiLossBatch[] | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    void fetchHomeCurrency().then(setHomeCcy);
  }, []);

  useEffect(() => {
    if (!withFx || fxManual) return;
    void fetchApprovedFx(date).then((rate) => {
      if (rate) setFx(rate);
    });
  }, [date, fxManual, withFx]);

  useEffect(() => {
    if (!requireMiLoss) return;
    let cancelled = false;
    fetchApprovedMiLoss()
      .then((rows) => {
        if (!cancelled) setMiLossRows(rows);
      })
      .catch((e) => {
        if (!cancelled) {
          setMiLossRows([]);
          setError(formatApiError(e, "Could not load MI-loss batches"));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [requireMiLoss]);

  const asOf = (effectiveFrom || date).slice(0, 10);
  const asOfBatches = miLossOnOrBefore(miLossRows || [], asOf);

  useEffect(() => {
    if (!requireMiLoss || miLossRows === null) return;
    setMiLossId(latestMiLossAsOf(miLossRows, asOf)?.id || "");
  }, [requireMiLoss, miLossRows, asOf]);

  async function save() {
    setError("");
    if (!date) {
      setError("Date is required.");
      return;
    }
    if (!effectiveFrom) {
      setError("Effective from is required.");
      return;
    }
    const dateErr = effectiveBeforeDocError(date, effectiveFrom);
    if (dateErr) {
      setError(dateErr);
      return;
    }
    if (requireMiLoss && !miLossBatchId) {
      const msg = "Approve at least one MI-loss batch before creating a variable storage fee.";
      setError(msg);
      notify.error(msg, { title: "Cannot create" });
      return;
    }
    setSaving(true);
    try {
      const row = await api.post<{ id: string }>(path, {
        date,
        effectiveFrom,
        description: description.trim(),
        ...(withFx ? { exchangeRate: exchangeRate.trim() || "0", fxManual } : {}),
        ...(requireMiLoss ? { miLossBatchId } : {}),
        ...extra,
      });
      if (!row?.id) throw new Error("Batch created without an id");
      onCreated(row.id);
    } catch (e) {
      setError(formatApiError(e, "Could not create batch"));
    } finally {
      setSaving(false);
    }
  }

  const noMiLoss = requireMiLoss && miLossRows !== null && miLossRows.length === 0;

  return (
    <CatalogueForm
      title={title}
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel="Create header"
      size="lg"
    >
      <p className="text-sm text-slate-600">
        Save the header first. Add lines, attachments, and submit for approval on the next screen.
      </p>
      {noMiLoss ? (
        <p className="text-sm leading-6 text-slate-700">
          Variable storage fees snapshot MI-loss by product and contract.{" "}
          <Link href="/finance/setups/miloss" className="font-semibold text-brand-800 hover:underline">
            Create and approve an MI-loss batch
          </Link>{" "}
          first, then return here.
        </p>
      ) : null}
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="Date">
          <DateInput
            value={date}
            onChange={(v) => {
              setDate(v);
              setEff((e) => clampEffectiveToDoc(v, e));
            }}
          />
        </Field>
        <Field label="Effective from">
          <DateInput value={effectiveFrom} onChange={setEff} min={date} />
        </Field>
      </div>
      <Field label="Description">
        <Input
          value={description}
          maxLength={200}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Optional note for this list"
        />
      </Field>
      {requireMiLoss ? (
        <Field label="MI-loss batch">
          <Select
            value={miLossBatchId}
            disabled={noMiLoss || miLossRows === null}
            onChange={(e) => setMiLossId(e.target.value)}
          >
            {miLossRows === null ? (
              <option value="">Loading approved batches…</option>
            ) : asOfBatches.length === 0 ? (
              <option value="">No approved MI-loss effective on or before this date</option>
            ) : (
              asOfBatches.map((b) => (
                <option key={b.id} value={b.id}>
                  {miLossBatchLabel(b)}
                </option>
              ))
            )}
          </Select>
          <p className="mt-1 text-xs text-slate-500">
            The latest approved MI-loss effective on or before this date is selected. Only those
            batches are listed. Set Effective from later if the batch you need is not shown.
            Products, contract, and MI-loss % come from this batch.
          </p>
        </Field>
      ) : null}
      {withFx ? (
        <>
          {homeCcy ? (
            <p className="text-sm text-slate-600">
              Home currency is <span className="font-medium text-slate-800">{homeCcy}</span> from
              company setup.
            </p>
          ) : null}
          <FxRateField
            rate={exchangeRate}
            manual={fxManual}
            onRate={setFx}
            onManual={(on) => {
              setManual(on);
              if (!on) void fetchApprovedFx(date).then((r) => r && setFx(r));
            }}
          />
        </>
      ) : null}
    </CatalogueForm>
  );
}
