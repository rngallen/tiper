"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { Card, ErrorBanner, PageHeader } from "@/components/ui";
import { ChangeOfServiceForm, type CosPayload } from "../_components/ChangeOfServiceForm";

export default function NewChangeOfServicePage() {
  const router = useRouter();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save(body: CosPayload) {
    setError("");
    setSaving(true);
    try {
      const created = await api.post<{ id: string }>("/billing/change-of-service", body);
      router.push(`/billing/change-of-service/${created.id}`);
    } catch (e) {
      setError(formatApiError(e, "Could not create change of service"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="New change of service"
        subtitle="Switch delivery method on one approved vessel parcel. Fees stay mapped — only collection changes billing of that parcel."
        actions={
          <Link href="/billing/change-of-service" className="text-sm font-medium text-slate-600 hover:text-slate-900">
            Back to list
          </Link>
        }
      />
      {error ? <ErrorBanner message={error} /> : null}
      <Card className="p-6">
        <ChangeOfServiceForm
          saving={saving}
          submitLabel="Create"
          onCancel={() => router.push("/billing/change-of-service")}
          onSubmit={(body) => void save(body)}
          onError={setError}
        />
      </Card>
    </div>
  );
}
