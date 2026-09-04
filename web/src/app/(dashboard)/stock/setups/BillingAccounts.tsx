"use client";

import { useMemo, useState } from "react";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { fetchLookups, type FeeRow } from "@/lib/lookups";
import { BILLING_UNITS } from "@/lib/billing";
import { searchMaster, type BillingAccount } from "@/lib/master";
import { AsyncSearchableSelect, SearchableSelect } from "@/components/SearchableSelect";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { Button, Field, Select, ToggleField } from "@/components/ui";
import { StatusPill } from "@/components/ApprovalReview";
import { SectionRow, SectionTable } from "@/components/SectionTable";

type SageClient = {
  account: string;
  name: string;
  onHold: boolean;
  currencyCode: string;
  available: boolean;
  inUseBy?: string;
};

type PartyKind = "customer" | "supplier";

const FALLBACK_FEES: FeeRow[] = [
  { code: "FSF", name: "Fixed storage fee", chargeTo: "both", isActive: true },
  { code: "VSF", name: "Variable storage fee", chargeTo: "both", isActive: true },
  { code: "KOJ", name: "KOJ facility fee", chargeTo: "customer", isActive: true },
  { code: "TBS", name: "TBS truck loading fee", chargeTo: "customer", isActive: true },
];

function feeAllowed(fee: FeeRow, party: PartyKind) {
  if (fee.isActive === false) return false;
  const to = fee.chargeTo || "customer";
  return to === "both" || to === party;
}

export function BillingAccountsPanel({
  partyKind,
  partyId,
  readOnly = false,
  showHelp = true,
}: {
  partyKind: PartyKind;
  partyId: string;
  readOnly?: boolean;
  showHelp?: boolean;
}) {
  const lookups = useAsync(fetchLookups, []);
  const fees = useMemo(() => {
    const rows = lookups.data?.fees?.length ? lookups.data.fees : FALLBACK_FEES;
    return rows.filter((f) => feeAllowed(f, partyKind));
  }, [lookups.data, partyKind]);

  const list = useAsync(
    () =>
      partyId
        ? api.get<BillingAccount[]>(`/master/${partyKind}s/${partyId}/billing-accounts`)
        : Promise.resolve([] as BillingAccount[]),
    [partyId, partyKind],
  );
  const sage = useAsync(
    () => api.get<{ connected: boolean }>("/sage/status"),
    [],
  );
  const rows = Array.isArray(list.data) ? list.data : [];

  const [feeCode, setFeeCode] = useState("");
  const [sageAccount, setSageAccount] = useState("");
  const [sageLabel, setSageLabel] = useState("");
  const [currencyCode, setCurrency] = useState("");
  const [billingUnit, setUnit] = useState("M3");
  const [isActive, setActive] = useState(true);
  const [editingId, setEditingId] = useState("");
  const [composer, setComposer] = useState(false);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState<BillingAccount | null>(null);

  function resetForm() {
    setFeeCode("");
    setSageAccount("");
    setSageLabel("");
    setCurrency("");
    setUnit("M3");
    setActive(true);
    setEditingId("");
    setComposer(false);
  }

  function startEdit(row: BillingAccount) {
    setComposer(true);
    setEditingId(row.id);
    setFeeCode(row.feeCode);
    setSageAccount(row.sageAccount);
    setSageLabel(
      row.sageName
        ? `${row.sageAccount} — ${row.sageName} (${row.currencyCode})`
        : `${row.sageAccount} (${row.currencyCode})`,
    );
    setCurrency(row.currencyCode);
    setUnit(row.billingUnit || "M3");
    setActive(row.isActive !== false);
    setMsg("");
  }

  async function save() {
    setMsg("");
    if (!feeCode) {
      setMsg("Select a fee.");
      return;
    }
    if (!sageAccount) {
      setMsg("Select a Sage account. Currency is taken from Sage, not typed.");
      return;
    }
    setSaving(true);
    try {
      const body = { feeCode, sageAccount, billingUnit, isActive };
      const path = `/master/${partyKind}s/${partyId}/billing-accounts`;
      if (editingId) await api.put(`${path}/${editingId}`, body);
      else await api.post(path, body);
      resetForm();
      list.reload();
    } catch (e) {
      setMsg(formatApiError(e, "Could not save billing account"));
    } finally {
      setSaving(false);
    }
  }

  async function setRowActive(row: BillingAccount, active: boolean) {
    setMsg("");
    try {
      await api.put(`/master/${partyKind}s/${partyId}/billing-accounts/${row.id}`, {
        feeCode: row.feeCode,
        sageAccount: row.sageAccount,
        billingUnit: row.billingUnit,
        isActive: active,
      });
      list.reload();
    } catch (e) {
      setMsg(formatApiError(e, "Could not update billing account"));
    }
  }

  async function confirmRemove() {
    if (!pending) return;
    setSaving(true);
    setMsg("");
    try {
      await api.del(`/master/${partyKind}s/${partyId}/billing-accounts/${pending.id}`);
      if (editingId === pending.id) resetForm();
      setPending(null);
      list.reload();
    } catch (e) {
      setMsg(formatApiError(e, "Could not delete billing account"));
    } finally {
      setSaving(false);
    }
  }

  if (!partyId) {
    return (
      <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
        Save the {partyKind} first, then map Sage AR accounts per fee. Currency
        comes from Sage Client (0 = TZS, 1 = USD, …).
      </p>
    );
  }

  function feeName(code: string) {
    return fees.find((f) => f.code === code)?.name || code;
  }

  return (
    <div className="space-y-4">
      {showHelp ? (
        <p className="text-xs leading-5 text-slate-500">
          One mapping per fee and currency (for example FSF in TZS and VSF in USD).
          The same Sage account can be reused for several fees on this {partyKind},
          but cannot be used by any other customer or supplier.
        </p>
      ) : null}
      {sage.data && sage.data.connected === false ? (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
          Sage 200 is not connected. Save the connection under Settings → Sage 200
          before mapping AR accounts.
        </p>
      ) : null}
      {list.error ? <p className="text-sm text-red-700">{list.error}</p> : null}
      {msg ? <p className="text-sm text-red-700">{msg}</p> : null}

      {readOnly ? null : composer ? (
        <div className="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 sm:grid-cols-2">
          <Field label="Fee" required>
            <SearchableSelect
              value={feeCode}
              placeholder="Search fee"
              emptyLabel="No matching fee"
              onChange={(value) => setFeeCode(value)}
              options={fees.map((f) => ({
                value: f.code,
                label: `${f.code} — ${f.name}`,
                keywords: f.name,
              }))}
            />
          </Field>
          <Field label="Sage account">
            <AsyncSearchableSelect
              value={sageAccount}
              selectedLabel={sageLabel}
              placeholder="Search Sage Client account or name"
              emptyLabel="No matching Sage account"
              onChange={(value, option) => {
                setSageAccount(value);
                setSageLabel(option?.label || value);
                setCurrency(option?.currencyCode || "");
              }}
              loadOptions={async (q) => {
                const clients = await searchMaster<SageClient>("/sage/clients", q, {
                  ownerType: partyKind,
                  ownerId: partyId,
                });
                return (Array.isArray(clients) ? clients : []).map((row) => {
                  const used = row.inUseBy ? ` · ${row.inUseBy}` : "";
                  const ccy = row.currencyCode || "?";
                  return {
                    value: row.account,
                    label: `${row.account} — ${row.name} (${ccy})${used}`,
                    keywords: row.name,
                    currencyCode: row.currencyCode,
                    disabled: !row.available,
                  };
                });
              }}
            />
          </Field>
          <Field label="Currency">
            <input
              className="pp-control pp-readonly w-full"
              value={currencyCode || "— from Sage —"}
              readOnly
              aria-readonly
            />
          </Field>
          <Field label="Billing unit">
            <Select
              value={billingUnit}
              onChange={(e) => setUnit(e.target.value)}
            >
              {BILLING_UNITS.map((u) => (
                <option key={u.value} value={u.value}>
                  {u.label}
                </option>
              ))}
            </Select>
          </Field>
          <div className="sm:col-span-2">
            <ToggleField
              compact
              label="Active"
              checked={isActive}
              onChange={setActive}
            />
          </div>
          <div className="flex gap-2 sm:col-span-2">
            <Button onClick={() => void save()} loading={saving}>
              {editingId ? "Save mapping" : "Add mapping"}
            </Button>
            <Button variant="secondary" onClick={resetForm} disabled={saving}>
              Cancel
            </Button>
          </div>
        </div>
      ) : (
        <div className="flex justify-end">
          <Button variant="secondary" onClick={() => setComposer(true)}>
            Add mapping
          </Button>
        </div>
      )}

      <SectionTable
        title="Sage mappings"
        columns={[
          { label: "Fee" },
          { label: "Currency" },
          { label: "Sage account" },
          { label: "Unit" },
          { label: "Status" },
          ...(readOnly ? [] : [{ label: "" }]),
        ]}
        empty={list.loading ? "Loading billing accounts…" : "No Sage accounts mapped yet"}
      >
        {list.loading
          ? null
          : rows.map((row) => (
              <SectionRow key={row.id}>
                <td>
                  <div className="font-semibold">{row.feeCode}</div>
                  <div className="text-[13px] font-medium text-slate-700">{feeName(row.feeCode)}</div>
                </td>
                <td className="font-semibold">{row.currencyCode}</td>
                <td>
                  <div className="font-semibold">{row.sageAccount}</div>
                  <div className="text-[13px] font-medium text-slate-700">{row.sageName || "—"}</div>
                </td>
                <td>{row.billingUnit}</td>
                <td>
                  <StatusPill tone={row.isActive === false ? "slate" : "navy"}>
                    {row.isActive === false ? "Inactive" : "Active"}
                  </StatusPill>
                </td>
                {readOnly ? null : (
                  <td className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button variant="ghost" className="!px-2 !py-1" onClick={() => startEdit(row)}>
                        Edit
                      </Button>
                      {row.isActive === false ? (
                        <Button variant="ghost" className="!px-2 !py-1" onClick={() => void setRowActive(row, true)}>
                          Activate
                        </Button>
                      ) : (
                        <Button variant="ghost" className="!px-2 !py-1" onClick={() => void setRowActive(row, false)}>
                          Deactivate
                        </Button>
                      )}
                      <Button variant="ghost" className="!px-2 !py-1 text-red-700" onClick={() => setPending(row)}>
                        Delete
                      </Button>
                    </div>
                  </td>
                )}
              </SectionRow>
            ))}
      </SectionTable>
      <ConfirmDialog
        open={Boolean(pending)}
        title="Delete billing account?"
        message={
          pending
            ? `Remove ${pending.feeCode} ${pending.currencyCode} (${pending.sageAccount})?`
            : "Remove this mapping?"
        }
        detail="The Sage account can then be mapped to a different customer or supplier if it is not used on another fee for this party."
        loading={saving}
        onConfirm={() => void confirmRemove()}
        onCancel={() => {
          if (!saving) setPending(null);
        }}
      />
    </div>
  );
}
