"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api, ApiError, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import {
  Button,
  ErrorBanner,
  Field,
  Input,
  Select,
  Spinner,
  ToggleField,
} from "@/components/ui";
import type {
  Company,
  Currency,
  MailIntegrationSettings,
  SageIntegrationSettings,
  ScheduleSettings,
  SMSIntegrationSettings,
  SessionIntegrationSettings,
  UploadsIntegrationSettings,
  NpgisIntegrationSettings,
  DecimalPrecisionSettings,
} from "@/lib/types";
import { applyServerIdlePolicy } from "@/lib/idle";
import { StatusPill } from "@/components/ApprovalReview";
import { reloadCurrencyCatalog } from "@/lib/currencies";
import { currencyFullLabel, hydrateDecimalPrecision } from "@/lib/format";
import { useSettings, useSettingsSave } from "./settings-context";

function localError(err: unknown, fallback: string): string {
  if (err instanceof ApiError) return "";
  return formatApiError(err, fallback);
}

const emptyCompany: Company = {
  name: "",
  tinNumber: "",
  vrnNumber: "",
  isoNumber: "",
  address: "",
  address2: "",
  city: "",
  country: "",
  postalCode: "",
  phone: "",
  email: "",
  website: "",
  portalUrl: "",
  currencyCode: "",
};

function companyCurrency(data: Company | null | undefined): string {
  if (!data) return "";
  return String(data.currencyCode || "").trim().toUpperCase();
}


function FormCard({ children }: { children: React.ReactNode }) {
  return <div className="w-full max-w-5xl space-y-3">{children}</div>;
}

function Wide({ children }: { children: React.ReactNode }) {
  return <div className="col-span-full">{children}</div>;
}

function SettingsSection({
  title,
  description,
  columns = 2,
  footer,
  children,
}: {
  title: string;
  description?: string;
  columns?: 1 | 2 | 3;
  footer?: React.ReactNode;
  children: React.ReactNode;
}) {
  const grid =
    columns === 1
      ? "grid grid-cols-1 gap-x-4 gap-y-3"
      : columns === 3
        ? "grid grid-cols-1 gap-x-4 gap-y-3 sm:grid-cols-2 xl:grid-cols-3"
        : "grid grid-cols-1 gap-x-4 gap-y-3 sm:grid-cols-2";
  return (
    <section className="overflow-hidden rounded-lg border border-slate-200 bg-white">
      <div className="border-b border-slate-100 px-4 py-2.5">
        <h4 className="text-xs font-bold uppercase tracking-wide text-brand-800">
          {title}
        </h4>
        {description ? (
          <p className="mt-0.5 text-xs leading-5 text-slate-500">{description}</p>
        ) : null}
      </div>
      <div className={`p-4 ${grid}`}>{children}</div>
      {footer ? (
        <div className="border-t border-slate-100 bg-slate-50 px-4 py-3">
          {footer}
        </div>
      ) : null}
    </section>
  );
}

function TestRow({
  title,
  hint,
  label,
  value,
  placeholder,
  type = "text",
  onChange,
  onSend,
  sending,
  disabled,
  sendLabel,
  error,
}: {
  title: string;
  hint: string;
  label: string;
  value: string;
  placeholder?: string;
  type?: string;
  onChange: (value: string) => void;
  onSend: () => void;
  sending: boolean;
  disabled?: boolean;
  sendLabel: string;
  error?: string;
}) {
  return (
    <div className="space-y-2">
      <div>
        <h3 className="text-sm font-semibold text-slate-900">{title}</h3>
        <p className="text-xs text-slate-500">{hint}</p>
      </div>
      {error ? <ErrorBanner message={error} /> : null}
      <div className="flex flex-wrap items-end gap-2">
        <div className="min-w-[14rem] max-w-sm flex-1">
          <Field label={label}>
            <Input
              type={type}
              value={value}
              placeholder={placeholder}
              onChange={(e) => onChange(e.target.value)}
            />
          </Field>
        </div>
        <Button variant="secondary" onClick={onSend} loading={sending} disabled={disabled}>
          {sendLabel}
        </Button>
      </div>
    </div>
  );
}

function SecretField({
  label,
  configured,
  value,
  cleared,
  disabled,
  onChange,
  onClear,
  onUndoClear,
}: {
  label: string;
  configured: boolean;
  value: string;
  cleared: boolean;
  disabled?: boolean;
  onChange: (value: string) => void;
  onClear: () => void;
  onUndoClear: () => void;
}) {
  // Status + actions sit on the label row so the input lines up with sibling fields.
  return (
    <div className="block">
      <div className="mb-1.5 flex min-h-[1.25rem] flex-wrap items-center gap-x-2 gap-y-1">
        <span className="text-sm font-medium text-slate-900">{label}</span>
        <span
          className={`rounded px-1.5 py-0.5 text-[11px] font-medium leading-none ${
            cleared
              ? "bg-amber-50 text-amber-800"
              : configured
                ? "bg-emerald-50 text-emerald-800"
                : "bg-slate-100 text-slate-600"
          }`}
        >
          {cleared
            ? "Will clear on save"
            : configured
              ? "Configured"
              : "Not set"}
        </span>
        {!disabled && configured && !cleared && (
          <button
            type="button"
            onClick={onClear}
            className="text-xs font-medium text-slate-600 underline-offset-2 hover:text-slate-900 hover:underline"
          >
            Clear
          </button>
        )}
        {!disabled && cleared && (
          <button
            type="button"
            onClick={onUndoClear}
            className="text-xs font-medium text-slate-600 underline-offset-2 hover:text-slate-900 hover:underline"
          >
            Undo clear
          </button>
        )}
      </div>
      {cleared ? (
        <p className="pp-control flex items-center text-sm text-slate-500">
          Secret will be removed when you save.
        </p>
      ) : (
        <Input
          type="password"
          value={value}
          disabled={disabled}
          placeholder={configured ? "•••• configured" : undefined}
          autoComplete="new-password"
          onChange={(e) => onChange(e.target.value)}
        />
      )}
    </div>
  );
}

export function CurrenciesPanel() {
  const { canManage } = useSettings();
  const { data, loading, error, reload } = useAsync(
    () => api.get<Currency[]>("/settings/currencies"),
    [],
  );
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [symbol, setSymbol] = useState("");
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");

  async function addCurrency() {
    if (!canManage) return;
    setSaveError("");
    setSaving(true);
    try {
      if (!/^[A-Za-z]{3}$/.test(code.trim())) {
        throw new Error("ISO code must be 3 letters");
      }
      if (!symbol.trim()) throw new Error("Symbol is required");
      await api.post("/settings/currencies", {
        code: code.trim().toUpperCase(),
        name: name.trim() || code.trim().toUpperCase(),
        symbol: symbol.trim(),
      });
      setCode("");
      setName("");
      setSymbol("");
      reload();
      void reloadCurrencyCatalog();
    } catch (err) {
      setSaveError(localError(err, "Failed to add currency"));
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <Spinner />;
  if (error) return <ErrorBanner message={error} />;

  return (
    <div className="space-y-4">
      {saveError && <ErrorBanner message={saveError} />}

      {canManage && (
        <FormCard>
          <SettingsSection title="Add currency">
            <Field label="ISO code">
              <Input
                value={code}
                maxLength={3}
                placeholder="USD"
                onChange={(e) => setCode(e.target.value.toUpperCase())}
              />
            </Field>
            <Field label="Name">
              <Input
                value={name}
                placeholder="US Dollar"
                onChange={(e) => setName(e.target.value)}
              />
            </Field>
            <Field label="Symbol">
              <Input
                value={symbol}
                maxLength={10}
                placeholder="$"
                onChange={(e) => setSymbol(e.target.value)}
              />
            </Field>
          </SettingsSection>
          <div className="flex justify-end">
            <Button onClick={addCurrency} loading={saving} disabled={!code}>
              Add currency
            </Button>
          </div>
        </FormCard>
      )}

      <section className="overflow-hidden rounded-lg border border-slate-200 bg-white">
        <table className="pp-ledger min-w-full text-left text-sm">
          <thead className="pp-table-head">
            <tr>
              <th>Currency</th>
              <th>Symbol</th>
            </tr>
          </thead>
          <tbody>
            {(data || []).map((c) => (
              <tr key={c.code} className="border-t border-slate-100">
                <td className="px-4 py-3">
                  <div className="font-medium text-slate-900">{c.code}</div>
                  <div className="text-xs text-slate-500">{c.name}</div>
                </td>
                <td className="px-4 py-3 text-lg text-slate-800">
                  {c.symbol || "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}

export function CompanyPanel() {
  const { canManage } = useSettings();
  const { data, loading, error, reload } = useAsync(
    () => api.get<Company>("/settings/company"),
    [],
  );

  const {
    data: currencies,
    loading: currenciesLoading,
    error: currenciesError,
  } = useAsync(() => api.get<Currency[]>("/settings/currencies"), []);

  const [form, setForm] = useState<Company>(emptyCompany);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");

  const currencyLocked = useMemo(
    () => Boolean(companyCurrency(data)),
    [data],
  );

  // Keep the form aligned with the latest server payload (initial load + reload).
  // Merge over emptyCompany so omitempty blanks from the API clear local fields.
  useEffect(() => {
    if (!data) return;
    setForm({
      ...emptyCompany,
      ...data,
      currencyCode: companyCurrency(data),
    });
  }, [data]);

  async function save() {
    setSaveError("");
    setSaving(true);
    try {
      const payload = { ...form };
      if (currencyLocked) {
        payload.currencyCode = companyCurrency(data);
      }
      const updated = await api.put<Company>("/settings/company", payload);
      setForm({
        ...emptyCompany,
        ...updated,
        currencyCode:
          companyCurrency(updated) ||
          payload.currencyCode ||
          companyCurrency(data),
      });
      reload();
    } catch (err) {
      setSaveError(localError(err, "Failed to save"));
    } finally {
      setSaving(false);
    }
  }

  const set = (k: keyof Company) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm({ ...form, [k]: e.target.value });

  useSettingsSave(() => void save(), saving, false);

  const currencyOptions = currencies ?? [];
  const lockedCurrency = currencyOptions.find(
    (c) => c.code === form.currencyCode,
  );

  return (
    <>
      {loading && !data ? (
        <Spinner />
      ) : error && !data ? (
        <ErrorBanner message={error} />
      ) : (
        <FormCard>
          {saveError && <ErrorBanner message={saveError} />}
          {currenciesError && <ErrorBanner message={currenciesError} />}
          <SettingsSection title="Identity">
            <Field label="Company name">
              <Input
                value={form.name}
                disabled={!canManage}
                onChange={set("name")}
              />
            </Field>
            <Field label="Default currency">
              {currencyLocked || !canManage ? (
                <Input
                  value={
                    lockedCurrency
                      ? currencyFullLabel(lockedCurrency)
                      : (form.currencyCode ?? "")
                  }
                  disabled
                  readOnly
                />
              ) : (
                <Select
                  value={form.currencyCode ?? ""}
                  disabled={currenciesLoading}
                  required
                  onChange={(e) =>
                    setForm({
                      ...form,
                      currencyCode: e.target.value.toUpperCase(),
                    })
                  }
                >
                  <option value="">
                    {currenciesLoading
                      ? "Loading currencies…"
                      : "Select currency"}
                  </option>
                  {currencyOptions.map((c) => (
                    <option key={c.code} value={c.code}>
                      {currencyFullLabel(c)}
                    </option>
                  ))}
                </Select>
              )}
              {currencyLocked && (
                <p className="mt-1 text-xs font-normal text-slate-500">
                  Currency is set and cannot be changed.
                </p>
              )}
            </Field>
            <Field label="TIN">
              <Input
                value={form.tinNumber}
                disabled={!canManage}
                onChange={set("tinNumber")}
              />
            </Field>
            <Field label="VRN">
              <Input
                value={form.vrnNumber}
                disabled={!canManage}
                onChange={set("vrnNumber")}
              />
            </Field>
            <Field label="ISO number">
              <Input
                value={form.isoNumber}
                disabled={!canManage}
                onChange={set("isoNumber")}
              />
            </Field>
          </SettingsSection>
          <SettingsSection title="Contact">
            <Field label="Phone">
              <Input
                value={form.phone}
                disabled={!canManage}
                onChange={set("phone")}
              />
            </Field>
            <Field label="Email">
              <Input
                value={form.email}
                disabled={!canManage}
                onChange={set("email")}
              />
            </Field>
            <Field label="Website">
              <Input
                value={form.website}
                disabled={!canManage}
                onChange={set("website")}
              />
            </Field>
            <Field label="Public site URL">
              <Input
                value={form.portalUrl || ""}
                disabled={!canManage}
                placeholder="https://dfms.tiper.co.tz"
                onChange={set("portalUrl")}
              />
              <p className="mt-1 text-xs text-slate-500">
                Public frontend base for email links and document QR
                confirmation. No trailing slash.
              </p>
            </Field>
          </SettingsSection>
          <SettingsSection title="Address">
            <Field label="Address">
              <Input
                value={form.address}
                disabled={!canManage}
                onChange={set("address")}
              />
            </Field>
            <Field label="Address line 2">
              <Input
                value={form.address2}
                disabled={!canManage}
                onChange={set("address2")}
              />
            </Field>
            <Field label="City">
              <Input
                value={form.city}
                disabled={!canManage}
                onChange={set("city")}
              />
            </Field>
            <Field label="Country">
              <Input
                value={form.country}
                disabled={!canManage}
                onChange={set("country")}
              />
            </Field>
            <Field label="Postal code">
              <Input
                value={form.postalCode}
                disabled={!canManage}
                onChange={set("postalCode")}
              />
            </Field>
          </SettingsSection>
        </FormCard>
      )}
    </>
  );
}

export function SagePanel() {
  const { canManage } = useSettings();
  const { data, loading, error, reload } = useAsync(
    () => api.get<SageIntegrationSettings>("/settings/integrations/sage"),
    [],
  );

  const [form, setForm] = useState<SageIntegrationSettings | null>(null);
  const [password, setPassword] = useState("");
  const [clearPassword, setClearPassword] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [testing, setTesting] = useState(false);
  const [testError, setTestError] = useState("");

  useEffect(() => {
    if (data) {
      setForm(data);
      setPassword("");
      setClearPassword(false);
    }
  }, [data]);

  async function save() {
    if (!form || !canManage) return;
    setSaveError("");
    setTestError("");
    setSaving(true);
    try {
      const body: Record<string, unknown> = {
        host: form.host,
        port: form.port,
        instance: form.instance,
        user: form.user,
        name: form.name,
        encrypt: form.encrypt,
      };
      if (clearPassword) body.clearPassword = true;
      else if (password.trim()) body.password = password.trim();
      await api.put("/settings/integrations/sage", body);
      setPassword("");
      setClearPassword(false);
      reload();
    } catch (err) {
      setSaveError(localError(err, "Failed to save Sage 200 settings"));
      reload();
    } finally {
      setSaving(false);
    }
  }

  async function sendTest() {
    if (!form || !canManage) return;
    setTestError("");
    setTesting(true);
    try {
      const body: Record<string, unknown> = {
        host: form.host,
        port: form.port,
        instance: form.instance,
        user: form.user,
        name: form.name,
        encrypt: form.encrypt,
      };
      if (password.trim()) body.password = password.trim();
      await api.post("/settings/integrations/sage/test", body);
    } catch (err) {
      setTestError(localError(err, "Sage 200 test failed"));
    } finally {
      setTesting(false);
    }
  }

  useSettingsSave(() => void save(), saving, !form);

  return (
    <>
      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner message={error} />
      ) : form ? (
        <FormCard>
          {saveError && <ErrorBanner message={saveError} />}
          <SettingsSection
            title="Connection"
            description="Changing host, user, or password reconnects immediately. The application database stays in the process environment file."
            footer={
              canManage ? (
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <p className="text-xs text-slate-500">
                    Test uses the fields above, including an unsaved password.
                    The live connection is not replaced until you save.
                  </p>
                  <div className="flex items-center gap-2">
                    {testError ? <span className="text-sm text-red-700">{testError}</span> : null}
                    <Button
                      variant="secondary"
                      onClick={() => void sendTest()}
                      loading={testing}
                    >
                      Test connection
                    </Button>
                  </div>
                </div>
              ) : undefined
            }
          >
            <Wide>
              <StatusPill tone={form.connected ? "emerald" : "slate"}>
                {form.connected ? "Connected" : "Not connected"}
              </StatusPill>
            </Wide>
            <Field label="Host">
              <Input
                value={form.host}
                disabled={!canManage}
                placeholder="sql.example.local"
                onChange={(e) => setForm({ ...form, host: e.target.value })}
              />
            </Field>
            <Field label="Port">
              <Input
                value={form.port}
                disabled={!canManage}
                placeholder="1433"
                onChange={(e) => setForm({ ...form, port: e.target.value })}
              />
            </Field>
            <Field label="Instance (optional)">
              <Input
                value={form.instance}
                disabled={!canManage}
                onChange={(e) =>
                  setForm({ ...form, instance: e.target.value })
                }
              />
            </Field>
            <Field label="Database">
              <Input
                value={form.name}
                disabled={!canManage}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </Field>
            <Field label="User">
              <Input
                value={form.user}
                disabled={!canManage}
                onChange={(e) => setForm({ ...form, user: e.target.value })}
              />
            </Field>
            <SecretField
              label="Password"
              configured={Boolean(form.hasPassword) && !clearPassword}
              value={password}
              cleared={clearPassword}
              disabled={!canManage}
              onChange={(v) => {
                setPassword(v);
                setClearPassword(false);
              }}
              onClear={() => {
                setPassword("");
                setClearPassword(true);
              }}
              onUndoClear={() => setClearPassword(false)}
            />
            <Wide>
              <ToggleField
                compact
                label="Encrypt (TLS)"
                description="SQL Server encrypt=true. Intranet instances often leave this off."
                checked={form.encrypt}
                disabled={!canManage}
                onChange={(encrypt) => setForm({ ...form, encrypt })}
              />
            </Wide>
          </SettingsSection>
        </FormCard>
      ) : null}
    </>
  );
}

export function AttachmentsPanel() {
  const { canManage } = useSettings();
  const { data, loading, error, reload } = useAsync(
    () =>
      api.get<UploadsIntegrationSettings>("/settings/integrations/uploads"),
    [],
  );

  const [form, setForm] = useState<UploadsIntegrationSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  async function save() {
    if (!form || !canManage) return;
    setSaveError("");
    setSaving(true);
    try {
      await api.put("/settings/integrations/uploads", {
        directory: form.directory,
        maxFileSizeMB: form.maxFileSizeMB,
        maxFilesPerRequest: form.maxFilesPerRequest,
      });
      reload();
    } catch (err) {
      setSaveError(localError(err, "Failed to save attachment settings"));
    } finally {
      setSaving(false);
    }
  }

  useSettingsSave(() => void save(), saving, !form);

  return (
    <>
      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner message={error} />
      ) : form ? (
        <FormCard>
          {saveError && <ErrorBanner message={saveError} />}
          <SettingsSection title="Storage">
            <Wide>
            <Field label="Directory">
              <Input
                value={form.directory ?? ""}
                disabled={!canManage}
                placeholder="./uploads"
                onChange={(e) => setForm({ ...form, directory: e.target.value })}
              />
              <p className="mt-1 text-xs text-slate-500">
                Folder on this server for new files (relative to the process working
                directory, or an absolute path). New files go under a type folder
                (ILR, ITT, Pump-over-request, Customers, Drivers, Tanks, …) then
                YYYY/MM/document-number-or-code. Changing this does not move files
                already saved — copy that folder here if downloads of older
                attachments should keep working.
              </p>
            </Field>
            </Wide>
          </SettingsSection>
          <SettingsSection title="Limits">
            <Field label="Max file size (MB)">
              <Input
                type="number"
                min={1}
                max={25}
                value={form.maxFileSizeMB}
                disabled={!canManage}
                onChange={(e) =>
                  setForm({
                    ...form,
                    maxFileSizeMB: Number(e.target.value) || 0,
                  })
                }
              />
              <p className="mt-1 text-xs text-slate-500">
                Ceiling per file (1–25 MB). Smaller documents (including under
                1 MB) are always accepted.
              </p>
            </Field>
            <Field label="Max files per request">
              <Input
                type="number"
                min={1}
                max={10}
                value={form.maxFilesPerRequest}
                disabled={!canManage}
                onChange={(e) =>
                  setForm({
                    ...form,
                    maxFilesPerRequest: Number(e.target.value) || 0,
                  })
                }
              />
              <p className="mt-1 text-xs text-slate-500">1–10 files.</p>
            </Field>
          </SettingsSection>
          <p className="mt-3 text-xs text-slate-500">
            The API request body is capped at {form.processBodyLimitMB} MB.
            Combined file size cannot exceed that without restarting the
            process — the server clamps values that would overflow it.
          </p>
        </FormCard>
      ) : null}
    </>
  );
}

export function SessionPanel() {
  const { canManage } = useSettings();
  const { data, loading, error, reload } = useAsync(
    () =>
      api.get<SessionIntegrationSettings>("/settings/integrations/session"),
    [],
  );

  const [form, setForm] = useState<SessionIntegrationSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  async function save() {
    if (!form || !canManage || saving) return;
    setSaveError("");
    setSaving(true);
    try {
      await api.put("/settings/integrations/session", {
        idleMinutes: form.idleMinutes,
        warnMinutes: form.warnMinutes,
      });
      applyServerIdlePolicy(form.idleMinutes, form.warnMinutes * 60);
      reload();
    } catch (err) {
      setSaveError(localError(err, "Failed to save session settings"));
    } finally {
      setSaving(false);
    }
  }

  useSettingsSave(() => void save(), saving, !form);

  return (
    <>
      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner message={error} />
      ) : form ? (
        <FormCard>
          {saveError && <ErrorBanner message={saveError} />}
          <SettingsSection
            title="Inactivity"
            description="Other people already signed in pick up the new times on their next page load. Shortening the timeout can end an unused session on the next token refresh."
          >
            <Field label="Sign out after (minutes)">
              <Input
                type="number"
                min={2}
                max={480}
                value={form.idleMinutes}
                disabled={!canManage}
                onChange={(e) =>
                  setForm({
                    ...form,
                    idleMinutes: Number(e.target.value) || 0,
                  })
                }
              />
              <p className="mt-1 text-xs text-slate-500">
                How long a signed-in user may sit unused before the next
                refresh signs them out (2–480 minutes). Default 10.
              </p>
            </Field>
            <Field label="Warn before sign-out (minutes)">
              <Input
                type="number"
                min={1}
                max={10}
                value={form.warnMinutes}
                disabled={!canManage}
                onChange={(e) =>
                  setForm({
                    ...form,
                    warnMinutes: Number(e.target.value) || 0,
                  })
                }
              />
              <p className="mt-1 text-xs text-slate-500">
                Countdown dialog length (1–10 minutes). Must be shorter than
                the sign-out time. Default 2.
              </p>
            </Field>
          </SettingsSection>
        </FormCard>
      ) : null}
    </>
  );
}

export function MailPanel() {
  const { canManage } = useSettings();
  const { data, loading, error, reload } = useAsync(
    () => api.get<MailIntegrationSettings>("/settings/integrations/mail"),
    [],
  );

  const [form, setForm] = useState<MailIntegrationSettings | null>(null);
  const [password, setPassword] = useState("");
  const [clearPassword, setClearPassword] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [testTo, setTestTo] = useState("");
  const [testing, setTesting] = useState(false);
  const [testError, setTestError] = useState("");

  useEffect(() => {
    if (data) {
      setForm(data);
      setPassword("");
      setClearPassword(false);
    }
  }, [data]);

  async function save() {
    if (!form || !canManage) return;
    setSaveError("");
    setTesting(false);
    setTestError("");
    setSaving(true);
    try {
      const body: Record<string, unknown> = {
        enabled: form.enabled,
        host: form.host,
        port: form.port,
        user: form.user,
        fromName: form.fromName,
        fromEmail: form.fromEmail,
        useTLS: form.useTLS,
        useSSL: form.useSSL,
      };
      if (clearPassword) body.clearPassword = true;
      else if (password.trim()) body.password = password.trim();
      await api.put("/settings/integrations/mail", body);
      setPassword("");
      setClearPassword(false);
      reload();
    } catch (err) {
      setSaveError(localError(err, "Failed to save mail settings"));
    } finally {
      setSaving(false);
    }
  }

  async function sendTest() {
    if (!canManage) return;
    setTestError("");
    setTesting(true);
    try {
      await api.post("/settings/integrations/mail/test", { to: testTo.trim() });
    } catch (err) {
      setTestError(localError(err, "Test email failed"));
    } finally {
      setTesting(false);
    }
  }

  const off = !form?.enabled;
  useSettingsSave(() => void save(), saving, !form);

  return (
    <>
      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner message={error} />
      ) : form ? (
        <FormCard>
          {saveError && <ErrorBanner message={saveError} />}
          <SettingsSection
            title="Gateway"
            footer={
              canManage ? (
                <TestRow
                  title="Send test email"
                  hint="Save credentials first, then send a sample message."
                  label="Recipient"
                  type="email"
                  value={testTo}
                  placeholder="you@example.com"
                  onChange={setTestTo}
                  onSend={() => void sendTest()}
                  sending={testing}
                  disabled={!testTo.trim() || !form.enabled}
                  sendLabel="Send test"
                  error={testError}
                />
              ) : undefined
            }
          >
            <Wide>
              <ToggleField
                compact
                label="Enable outbound mail"
                description="When off, DFMS will not send email (OTP, resets, alerts)."
                checked={form.enabled}
                disabled={!canManage}
                onChange={(enabled) => setForm({ ...form, enabled })}
              />
            </Wide>
            <Field label="SMTP host">
              <Input
                value={form.host}
                disabled={!canManage || off}
                onChange={(e) => setForm({ ...form, host: e.target.value })}
              />
            </Field>
            <Field label="Port">
              <Input
                type="number"
                value={form.port}
                disabled={!canManage || off}
                onChange={(e) =>
                  setForm({ ...form, port: Number(e.target.value) || 0 })
                }
              />
            </Field>
            <Field label="Username">
              <Input
                value={form.user}
                disabled={!canManage || off}
                onChange={(e) => setForm({ ...form, user: e.target.value })}
              />
            </Field>
            <SecretField
              label="Password"
              configured={Boolean(form.hasPassword) && !clearPassword}
              value={password}
              cleared={clearPassword}
              disabled={!canManage || off}
              onChange={(v) => {
                setPassword(v);
                setClearPassword(false);
              }}
              onClear={() => {
                setPassword("");
                setClearPassword(true);
              }}
              onUndoClear={() => setClearPassword(false)}
            />
            <Field
              label="From name"
              hint="Shown as the email From name and as the prefix on SMS."
            >
              <Input
                value={form.fromName}
                disabled={!canManage || off}
                placeholder="TIPER DFMS"
                onChange={(e) =>
                  setForm({ ...form, fromName: e.target.value })
                }
              />
            </Field>
            <Field label="From email">
              <Input
                type="email"
                value={form.fromEmail}
                disabled={!canManage || off}
                onChange={(e) =>
                  setForm({ ...form, fromEmail: e.target.value })
                }
              />
            </Field>
            <ToggleField
              compact
              label="Use STARTTLS"
              description="Upgrade a plain connection (typical ports 587 / 25)."
              checked={form.useTLS}
              disabled={!canManage || off}
              onChange={(useTLS) => setForm({ ...form, useTLS })}
            />
            <ToggleField
              compact
              label="Use SSL/TLS (implicit)"
              description="Encrypt from connect (typical port 465)."
              checked={form.useSSL}
              disabled={!canManage || off}
              onChange={(useSSL) => setForm({ ...form, useSSL })}
            />
          </SettingsSection>
        </FormCard>
      ) : null}
    </>
  );
}

export function SMSPanel() {
  const { canManage } = useSettings();
  const { data, loading, error, reload } = useAsync(
    () => api.get<SMSIntegrationSettings>("/settings/integrations/sms"),
    [],
  );

  const [form, setForm] = useState<SMSIntegrationSettings | null>(null);
  const [apiKey, setApiKey] = useState("");
  const [clearApiKey, setClearApiKey] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [testTo, setTestTo] = useState("");
  const [testing, setTesting] = useState(false);
  const [testError, setTestError] = useState("");

  useEffect(() => {
    if (data) {
      setForm(data);
      setApiKey("");
      setClearApiKey(false);
    }
  }, [data]);

  async function save() {
    if (!form || !canManage) return;
    setSaveError("");
    setTesting(false);
    setTestError("");
    setSaving(true);
    try {
      const body: Record<string, unknown> = {
        enabled: form.enabled,
        apiUrl: form.apiUrl,
        senderId: form.senderId,
      };
      if (clearApiKey) body.clearApiKey = true;
      else if (apiKey.trim()) body.apiKey = apiKey.trim();
      await api.put("/settings/integrations/sms", body);
      setApiKey("");
      setClearApiKey(false);
      reload();
    } catch (err) {
      setSaveError(localError(err, "Failed to save SMS settings"));
    } finally {
      setSaving(false);
    }
  }

  async function sendTest() {
    if (!canManage) return;
    setTestError("");
    setTesting(true);
    try {
      await api.post("/settings/integrations/sms/test", { to: testTo.trim() });
    } catch (err) {
      setTestError(localError(err, "Test SMS failed"));
    } finally {
      setTesting(false);
    }
  }

  const off = !form?.enabled;
  useSettingsSave(() => void save(), saving, !form);

  return (
    <>
      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner message={error} />
      ) : form ? (
        <FormCard>
          {saveError && <ErrorBanner message={saveError} />}
          <SettingsSection
            title="Gateway"
            footer={
              canManage ? (
                <TestRow
                  title="Send test SMS"
                  hint="Save credentials first, then send a sample message."
                  label="Phone number"
                  type="tel"
                  value={testTo}
                  placeholder="255711223344"
                  onChange={(v) => setTestTo(v.replace(/\D/g, "").slice(0, 15))}
                  onSend={() => void sendTest()}
                  sending={testing}
                  disabled={testTo.replace(/\D/g, "").length < 7 || !form.enabled}
                  sendLabel="Send test"
                  error={testError}
                />
              ) : undefined
            }
          >
            <Wide>
              <ToggleField
                compact
                label="Enable SMS gateway"
                description="When off, OTP and SMS alerts are not sent."
                checked={form.enabled}
                disabled={!canManage}
                onChange={(enabled) => setForm({ ...form, enabled })}
              />
            </Wide>
            <Wide>
              <Field label="API URL">
                <Input
                  value={form.apiUrl}
                  disabled={!canManage || off}
                  onChange={(e) => setForm({ ...form, apiUrl: e.target.value })}
                />
              </Field>
            </Wide>
            <Field
              label="Sender ID"
              hint="Notify Africa sender / company ID. Used as saved — not replaced by the product or mail From name."
            >
              <Input
                value={form.senderId}
                disabled={!canManage || off}
                placeholder="563"
                onChange={(e) =>
                  setForm({ ...form, senderId: e.target.value })
                }
              />
            </Field>
            <SecretField
              label="API key"
              configured={Boolean(form.hasApiKey) && !clearApiKey}
              value={apiKey}
              cleared={clearApiKey}
              disabled={!canManage || off}
              onChange={(v) => {
                setApiKey(v);
                setClearApiKey(false);
              }}
              onClear={() => {
                setApiKey("");
                setClearApiKey(true);
              }}
              onUndoClear={() => setClearApiKey(false)}
            />
          </SettingsSection>
        </FormCard>
      ) : null}
    </>
  );
}


export function EwuraPanel() {
  const { canManage } = useSettings();
  const { data, loading, error, reload } = useAsync(
    () => api.get<NpgisIntegrationSettings>("/settings/integrations/npgis"),
    [],
  );
  const [form, setForm] = useState<NpgisIntegrationSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  async function save() {
    if (!form || !canManage) return;
    setSaveError("");
    setSaving(true);
    try {
      await api.put("/settings/integrations/npgis", {
        enabled: form.enabled,
        licenseUrl: form.licenseUrl,
        baseUrl: form.baseUrl,
        licenseNo: form.licenseNo,
        apiSourceId: form.apiSourceId,
        depotName: form.depotName,
      });
      reload();
    } catch (err) {
      setSaveError(localError(err, "Failed to save EWURA settings"));
    } finally {
      setSaving(false);
    }
  }

  const npgisOff = !form?.enabled;
  useSettingsSave(() => void save(), saving, !form);

  return (
    <>
      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner message={error} />
      ) : form ? (
        <FormCard>
          {saveError && <ErrorBanner message={saveError} />}
          <SettingsSection title="Petroleum license register" columns={1}>
            <Field
              label="License register URL"
              hint="Used by Stock → Setups → EWURA licenses and the scheduled license sync. This is not the NPGIS retailer API."
            >
              <Input
                value={form.licenseUrl}
                disabled={!canManage}
                placeholder="https://… JSON endpoint that returns petroleum licenses"
                onChange={(e) =>
                  setForm({ ...form, licenseUrl: e.target.value })
                }
              />
            </Field>
          </SettingsSection>
          <SettingsSection title="NPGIS retailer API" columns={3}>
            <Wide>
              <ToggleField
                compact
                label="Enable NPGIS outbox"
                description="Post terminal loading and pump-over records to EWURA"
                checked={form.enabled}
                disabled={!canManage}
                onChange={(on) => setForm({ ...form, enabled: on })}
              />
            </Wide>
            <Wide>
              <Field label="Base URL">
                <Input
                  value={form.baseUrl}
                  disabled={!canManage || npgisOff}
                  placeholder="https://npgisretailer.ewura.go.tz:2990/api/"
                  onChange={(e) =>
                    setForm({ ...form, baseUrl: e.target.value })
                  }
                />
              </Field>
            </Wide>
            <Field label="TIPER license number">
              <Input
                value={form.licenseNo}
                disabled={!canManage || npgisOff}
                onChange={(e) =>
                  setForm({ ...form, licenseNo: e.target.value })
                }
              />
            </Field>
            <Field label="API source ID">
              <Input
                value={form.apiSourceId}
                disabled={!canManage || npgisOff}
                onChange={(e) =>
                  setForm({ ...form, apiSourceId: e.target.value })
                }
              />
            </Field>
            <Field label="Depot name">
              <Input
                value={form.depotName}
                disabled={!canManage || npgisOff}
                onChange={(e) =>
                  setForm({ ...form, depotName: e.target.value })
                }
              />
            </Field>
          </SettingsSection>
        </FormCard>
      ) : null}
    </>
  );
}

export function SchedulesPanel() {
  const { canManage } = useSettings();
  const { data, loading, error, reload } = useAsync(
    () => api.get<ScheduleSettings>("/settings/schedules"),
    [],
  );
  const [form, setForm] = useState<ScheduleSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  async function save() {
    if (!form || !canManage) return;
    setSaveError("");
    for (const f of fields) {
      const err = validateCronSpec(form[f.key]);
      if (err) {
        setSaveError(`${f.label}: ${err}`);
        return;
      }
    }
    setSaving(true);
    try {
      await api.put("/settings/schedules", form);
      reload();
    } catch (err) {
      setSaveError(localError(err, "Failed to save schedules"));
    } finally {
      setSaving(false);
    }
  }

  const fields: {
    key: keyof ScheduleSettings;
    label: string;
    purpose: string;
    hint: string;
  }[] = [
    {
      key: "logRotation",
      label: "Log rotation",
      purpose: "Archive and prune application log files",
      hint: "Default daily at midnight (0 0 0 * * *)",
    },
    {
      key: "ewuraLicenses",
      label: "EWURA license sync",
      purpose: "Refresh petroleum licenses from EWURA",
      hint: "Default daily at 02:00",
    },
    {
      key: "billingNth",
      label: "Nth-day billing",
      purpose: "Generate FSF / first-cycle billing runs",
      hint: "Default daily at 02:30",
    },
    {
      key: "billingTbs",
      label: "TBS billing",
      purpose: "Generate TBS fee billing runs",
      hint: "Default daily at 00:15",
    },
    {
      key: "billingVcf",
      label: "VSF billing",
      purpose: "Generate variable storage billing runs",
      hint: "Default daily at 03:30",
    },
    {
      key: "ewuraNpgis",
      label: "EWURA NPGIS outbox",
      purpose: "Flush pending NPGIS submissions",
      hint: "Default every minute",
    },
    {
      key: "iloExpire",
      label: "ILO expiry",
      purpose: "Mark expired loading orders and drop them from stock commitment",
      hint: "Default daily at 00:05",
    },
    {
      key: "notifyOutbox",
      label: "Notification outbox",
      purpose: "Retry welcome, password-reset, and workflow email/SMS after a provider outage",
      hint: "Default every minute",
    },
  ];

  useSettingsSave(() => void save(), saving, !form);

  return (
    <>
      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner message={error} />
      ) : form ? (
        <div className="space-y-3">
          {saveError && <ErrorBanner message={saveError} />}
          <div>
            <StatusPill tone="navy">Six-field cron</StatusPill>
          </div>
          <section className="overflow-hidden rounded-lg border border-slate-200 bg-white">
            <div className="border-b border-slate-100 px-4 py-3">
              <h4 className="text-xs font-bold uppercase tracking-wide text-brand-800">
                Job calendar
              </h4>
              <p className="mt-0.5 text-xs text-slate-500">
                Seconds first. Changes apply without restarting the API.
              </p>
            </div>
            <div className="overflow-x-auto">
              <table className="pp-ledger min-w-full text-left text-sm">
                <thead className="pp-table-head">
                  <tr>
                    <th>Job</th>
                    <th>Purpose</th>
                    <th>Schedule</th>
                    <th>Runs</th>
                  </tr>
                </thead>
                <tbody>
                  {fields.map((f) => {
                    const fieldErr = form[f.key]
                      ? validateCronSpec(form[f.key])
                      : null;
                    return (
                      <tr key={f.key}>
                        <td className="font-semibold text-slate-900">{f.label}</td>
                        <td className="max-w-xs text-slate-700">{f.purpose}</td>
                        <td className="min-w-[12rem]">
                          <Input
                            value={form[f.key]}
                            disabled={!canManage}
                            className={`font-mono text-xs ${fieldErr ? "border-red-400" : ""}`}
                            onChange={(e) =>
                              setForm({ ...form, [f.key]: e.target.value })
                            }
                          />
                          {fieldErr ? (
                            <p className="mt-1 text-xs text-red-600">{fieldErr}</p>
                          ) : (
                            <p className="mt-1 text-xs text-slate-500">{f.hint}</p>
                          )}
                        </td>
                        <td className="whitespace-nowrap font-semibold text-brand-800">
                          {fieldErr ? "—" : describeCron(form[f.key])}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </section>
          <CronSpecHelper />
        </div>
      ) : null}
    </>
  );
}

function describeCron(raw: string): string {
  const parts = raw.trim().split(/\s+/).filter(Boolean);
  if (parts.length !== 6) return "—";
  const [sec, min, hour, dom, month, dow] = parts;
  if (dom === "*" && month === "*" && dow === "*" && min.startsWith("*/") && hour === "*") {
    const n = min.slice(2);
    const off = sec === "0" ? "" : ` (offset ${sec}s)`;
    return `Every ${n} minute${n === "1" ? "" : "s"}${off}`;
  }
  if (dom === "*" && month === "*" && dow === "*" && !min.includes("*") && !hour.includes("*") && !min.includes("/") && !hour.includes("/")) {
    const hh = hour.padStart(2, "0");
    const mm = min.padStart(2, "0");
    return sec === "0" ? `Daily at ${hh}:${mm}` : `Daily at ${hh}:${mm}:${sec.padStart(2, "0")}`;
  }
  return raw.trim();
}

/** Frontend mirror of jobs.ValidateSpecCharset + 6-field shape check. */
function validateCronSpec(raw: string): string | null {
  const spec = raw.trim();
  if (!spec) return "cron spec is required";
  if (!/^[\d\s*/\-,]+$/.test(spec)) {
    return "only digits, spaces, *, /, -, and , are allowed";
  }
  const parts = spec.split(/\s+/).filter(Boolean);
  if (parts.length !== 6) {
    return "expected 6 fields (sec min hour day month weekday)";
  }
  return null;
}

const CRON_FIELDS = [
  { name: "Second", range: "0–59", tip: "Often 0" },
  { name: "Minute", range: "0–59", tip: "*/10 = every 10 min" },
  { name: "Hour", range: "0–23", tip: "* = every hour" },
  { name: "Day of month", range: "1–31", tip: "* = every day" },
  { name: "Month", range: "1–12", tip: "* = every month" },
  { name: "Day of week", range: "0–6", tip: "0 = Sunday" },
] as const;

function CronSpecHelper() {
  return (
    <section className="overflow-hidden rounded-lg border border-slate-200 bg-white">
      <div className="border-b border-slate-100 px-4 py-3">
        <h4 className="text-xs font-bold uppercase tracking-wide text-brand-800">
          Cron expression guide
        </h4>
        <p className="mt-0.5 text-xs text-slate-500">
          Six fields, left to right (seconds first). Allowed characters: digits,
          space, <code>*</code>, <code>/</code>, <code>-</code>, <code>,</code>.
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="pp-ledger min-w-full text-left text-sm">
          <thead className="pp-table-head">
            <tr>
              <th>#</th>
              <th>Field</th>
              <th>Values</th>
              <th>Notes</th>
            </tr>
          </thead>
          <tbody>
            {CRON_FIELDS.map((f, i) => (
              <tr key={f.name}>
                <td className="font-mono">{i + 1}</td>
                <td className="font-semibold">{f.name}</td>
                <td className="font-mono">{f.range}</td>
                <td>{f.tip}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex flex-wrap gap-x-6 gap-y-1 border-t border-slate-100 px-4 py-3 text-xs text-slate-600">
        <span>
          <code className="rounded bg-slate-50 px-1.5 py-0.5 font-mono">
            0 */10 * * * *
          </code>{" "}
          every 10 minutes
        </span>
        <span>
          <code className="rounded bg-slate-50 px-1.5 py-0.5 font-mono">
            0 0 2 * * *
          </code>{" "}
          daily at 02:00
        </span>
        <span>
          <code className="rounded bg-slate-50 px-1.5 py-0.5 font-mono">
            15 */2 * * * *
          </code>{" "}
          every 2 minutes (offset 15s)
        </span>
      </div>
    </section>
  );
}

export function PrecisionPanel() {
  const { canManage } = useSettings();
  const { data, loading, error, reload } = useAsync(
    () => api.get<DecimalPrecisionSettings>("/settings/precision"),
    [],
  );
  const [form, setForm] = useState<DecimalPrecisionSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  function patch(key: keyof DecimalPrecisionSettings, raw: string) {
    const n = Number(raw);
    if (!form || !Number.isFinite(n)) return;
    setForm({ ...form, [key]: n });
  }

  async function save() {
    if (!form || !canManage) return;
    setSaveError("");
    setSaving(true);
    try {
      const next = await api.put<DecimalPrecisionSettings>("/settings/precision", form);
      hydrateDecimalPrecision(next);
      setForm(next);
      reload();
    } catch (err) {
      setSaveError(localError(err, "Failed to save precision"));
    } finally {
      setSaving(false);
    }
  }

  const fields: {
    key: keyof DecimalPrecisionSettings;
    label: string;
    hint: string;
    min: number;
    max: number;
  }[] = [
    {
      key: "quantityPrecision",
      label: "Litres",
      hint: "L (default 2)",
      min: 0,
      max: 6,
    },
    {
      key: "cubicMeterPrecision",
      label: "Cubic metres",
      hint: "m³ (default 3)",
      min: 0,
      max: 6,
    },
    {
      key: "metricTonnePrecision",
      label: "Metric tonnes",
      hint: "MT (default 3)",
      min: 0,
      max: 6,
    },
    {
      key: "densityPrecision",
      label: "Density",
      hint: "MT / m³ (default 5)",
      min: 0,
      max: 6,
    },
    {
      key: "pricePrecision",
      label: "Price and amounts",
      hint: "Rates and money (default 4)",
      min: 0,
      max: 6,
    },
    {
      key: "miLossPrecision",
      label: "MI loss",
      hint: "Loss factor (default 4)",
      min: 0,
      max: 6,
    },
    {
      key: "iloExpiryDays",
      label: "ILO expiry (days)",
      hint: "Loading-order validity (default 14)",
      min: 1,
      max: 90,
    },
  ];

  useSettingsSave(() => void save(), saving, !form);

  return (
    <>
      {loading ? (
        <Spinner />
      ) : error ? (
        <ErrorBanner message={error} />
      ) : form ? (
        <FormCard>
          {saveError && <ErrorBanner message={saveError} />}
          <SettingsSection
            title="Decimal places"
            description="Rounding for stock and billing."
            columns={3}
          >
            {fields
              .filter((f) => f.key !== "iloExpiryDays")
              .map((f) => (
                <Field key={f.key} label={f.label} hint={f.hint} required>
                  <Input
                    type="number"
                    min={f.min}
                    max={f.max}
                    step={1}
                    value={form[f.key]}
                    disabled={!canManage}
                    onChange={(e) => patch(f.key, e.target.value)}
                  />
                </Field>
              ))}
          </SettingsSection>
          <SettingsSection title="Orders">
            {fields
              .filter((f) => f.key === "iloExpiryDays")
              .map((f) => (
                <Field key={f.key} label={f.label} hint={f.hint} required>
                  <Input
                    type="number"
                    min={f.min}
                    max={f.max}
                    step={1}
                    value={form[f.key]}
                    disabled={!canManage}
                    onChange={(e) => patch(f.key, e.target.value)}
                  />
                </Field>
              ))}
          </SettingsSection>
        </FormCard>
      ) : null}
    </>
  );
}
