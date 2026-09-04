"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api, ApiError, formatApiError } from "@/lib/api";
import { AsyncSearchableSelect } from "@/components/SearchableSelect";
import { Button, Card, ErrorBanner, Field, PageHeader } from "@/components/ui";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { searchMaster } from "@/lib/master";

type Ref = { id: string; code?: string; name?: string; documentNumber?: string; requestedQty?: string };

export default function NewCompartmentalizationPage() {
  const router = useRouter();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [nearExpiry, setNearExpiry] = useState("");
  const [iloId, setIloId] = useState("");
  const [iloLabel, setIloLabel] = useState("");
  const [badgeId, setBadgeId] = useState("");
  const [badgeLabel, setBadgeLabel] = useState("");

  async function save(confirmExpiry = false) {
    setError("");
    if (!iloId) {
      setError("Select an open GLO.");
      return;
    }
    setSaving(true);
    try {
      const created = await api.post<{ id: string }>("/orders/compartmentalizations", {
        iloId,
        badgeId,
        confirmExpiry,
      });
      router.push(`/stock/compartmentalization/${created.id}`);
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        const code =
          e.details && typeof e.details === "object" && "code" in e.details
            ? String((e.details as { code?: string }).code)
            : "";
        if (code === "nearExpiry") {
          setNearExpiry(e.message);
          setSaving(false);
          return;
        }
      }
      setError(formatApiError(e, "Could not create"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="New compartmentalization"
        subtitle="Build tank cells from the truck calibration chart"
        actions={
          <Link
            href="/stock/compartmentalization"
            className="text-sm font-medium text-slate-600 hover:text-slate-900"
          >
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="grid max-w-3xl gap-4 p-6 sm:grid-cols-2">
        <Field label="Open GLO">
          <AsyncSearchableSelect
            value={iloId}
            selectedLabel={iloLabel}
            placeholder="Search open GLO"
            onChange={(value, option) => {
              setIloId(value);
              setIloLabel(option?.label || "");
            }}
            loadOptions={async (q) =>
              (await searchMaster<Ref>("/orders/loading-lines", q, { status: "open", allDates: true })).map((l) => ({
                value: l.id,
                label: `${l.documentNumber || l.id} — ${l.requestedQty || ""}`,
              }))
            }
          />
        </Field>
        <Field label="RFID badge">
          <AsyncSearchableSelect
            value={badgeId}
            selectedLabel={badgeLabel}
            placeholder="Optional badge"
            onChange={(value, option) => {
              setBadgeId(value);
              setBadgeLabel(option?.label || "");
            }}
            loadOptions={async (q) =>
              (await searchMaster<Ref>("/orders/badges", q)).map((b) => ({
                value: b.id,
                label: b.code || b.name || b.id,
              }))
            }
          />
        </Field>
        <div className="sm:col-span-2 flex justify-end gap-2">
          <Button variant="secondary" onClick={() => router.push("/stock/compartmentalization")}>
            Cancel
          </Button>
          <Button onClick={() => void save()} loading={saving}>
            Create from calibration
          </Button>
        </div>
      </Card>
      <ConfirmDialog
        open={Boolean(nearExpiry)}
        title="GLO is near expiry"
        message={nearExpiry}
        detail="Continue only if gantry can still complete loading before the order expires."
        confirmLabel="Continue anyway"
        danger={false}
        loading={saving}
        onConfirm={() => {
          setNearExpiry("");
          void save(true);
        }}
        onCancel={() => setNearExpiry("")}
      />
    </div>
  );
}
