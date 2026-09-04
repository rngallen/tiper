"use client";

import { use, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api, formatApiError } from "@/lib/api";
import { useAsync } from "@/lib/hooks";
import { Button, Card, Field, Input, PageHeader, Select, ToggleField } from "@/components/ui";
import { DateInput } from "@/components/DateInput";
import { formatIntegerThousands, formatQty, normalizePlate, parseIntegerThousands, parseQty } from "@/lib/format";
import { QtyInput } from "@/components/QtyInput";
import { DocumentAttachmentsCard } from "@/components/DocumentFiles";
import type { Truck, TruckTank } from "@/lib/master";

const typeOptions = [
  { value: "straight", label: "Straight" },
  { value: "semi", label: "Semi-trailer" },
  { value: "pulling", label: "Pulling" },
];

function configuredType(vt?: string): string {
  if (vt === "straight" || vt === "semi" || vt === "pulling") return vt;
  return "";
}

function vehicleConfigured(vt?: string): boolean {
  return configuredType(vt) !== "";
}

function truckTotal(tanks?: TruckTank[]): string {
  const sum = (tanks || []).reduce((s, t) => s + (Number(parseQty(String(t.capacity || "0"))) || 0), 0);
  return String(sum);
}

export default function TruckConfigurePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const truck = useAsync(() => api.get<Truck>(`/master/trucks/${id}`), [id]);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");

  if (truck.loading && !truck.data) return <p className="text-sm text-slate-500">Loading…</p>;
  if (truck.error || !truck.data) return <p className="text-sm text-red-600">{truck.error || "Truck not found"}</p>;

  const configured = vehicleConfigured(truck.data.vehicleType);

  return (
    <div className="space-y-6">
      <PageHeader
        title={truck.data.displayPlate || truck.data.plateNumber}
        subtitle="Truck configuration — type, tanks, and calibration"
        actions={
          <Link href="/stock/setups/trucks" className="text-sm font-medium text-slate-600">
            Back to trucks
          </Link>
        }
      />
      {msg ? <p className="text-sm text-slate-700">{msg}</p> : null}
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      {!configured ? (
        <p className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950">
          Vehicle type is not set. Choose type and tank plates below. Calibration stays locked until
          type is saved.
        </p>
      ) : (
        <p className="rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-800">
          Total truck capacity:{" "}
          <span className="font-semibold tabular-nums">
            {formatQty(truck.data.totalCapacity || truckTotal(truck.data.tanks), "l")}
          </span>{" "}
          L
        </p>
      )}
      <ConfigCard
        row={truck.data}
        onSaved={() => {
          setMsg("Configuration saved");
          truck.reload();
        }}
        onError={setError}
      />
      <CalibrationCard
        truckId={id}
        tanks={truck.data.tanks || []}
        configured={configured}
        onSaved={() => {
          setMsg("Calibration saved");
          truck.reload();
        }}
      />
      <DocumentAttachmentsCard
        path={`/master/trucks/${id}`}
        draft
        caption="Calibration charts and certificates. Drop on the left — the list stays on the right."
      />
    </div>
  );
}

function ConfigCard({
  row,
  onSaved,
  onError,
}: {
  row: Truck;
  onSaved: () => void;
  onError: (s: string) => void;
}) {
  const horse = row.plateNumber;
  const [vehicleType, setType] = useState(configuredType(row.vehicleType));
  const [trailer, setTrailer] = useState(row.trailer ?? "");
  const [trailerTwo, setTrailerTwo] = useState(row.trailerTwo ?? "");
  const [loadingType, setLoading] = useState(row.loadingType || "top");
  const [lngCng, setLng] = useState(row.lngCng ?? false);
  const [isActive, setActive] = useState(row.isActive ?? true);
  const [mplw, setMplw] = useState(formatIntegerThousands(String(row.mplw ?? "")));
  const [gcwr, setGcwr] = useState(formatIntegerThousands(String(row.gcwr ?? "")));
  const [tareWeight, setTare] = useState(formatIntegerThousands(String(row.tareWeight ?? "")));
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setType(configuredType(row.vehicleType));
    setTrailer(row.trailer ?? "");
    setTrailerTwo(row.trailerTwo ?? "");
    setLoading(row.loadingType || "top");
    setLng(row.lngCng ?? false);
    setActive(row.isActive ?? true);
    setMplw(formatIntegerThousands(String(row.mplw ?? "")));
    setGcwr(formatIntegerThousands(String(row.gcwr ?? "")));
    setTare(formatIntegerThousands(String(row.tareWeight ?? "")));
  }, [row]);

  function onTypeChange(next: string) {
    setType(next);
    if (next === "straight") {
      setTrailer(horse);
      setTrailerTwo("");
    } else if (next === "semi") {
      setTrailerTwo("");
    }
  }

  async function save() {
    onError("");
    if (!vehicleType) {
      onError("Select vehicle type (straight, semi-trailer, or pulling).");
      return;
    }
    if (vehicleType === "semi" && !normalizePlate(trailer)) {
      onError("Semi-trailer requires tank one plate number.");
      return;
    }
    if (vehicleType === "pulling" && (!normalizePlate(trailer) || !normalizePlate(trailerTwo))) {
      onError("Pulling requires tank one and tank two plate numbers.");
      return;
    }
    setSaving(true);
    try {
      const tankOne = vehicleType === "straight" ? horse : normalizePlate(trailer);
      const tankTwo = vehicleType === "pulling" ? normalizePlate(trailerTwo) : "";
      await api.put(`/master/trucks/${row.id}`, {
        plateNumber: horse,
        trailer: tankOne,
        trailerTwo: tankTwo,
        vehicleType,
        loadingType,
        lngCng,
        mplw: parseIntegerThousands(mplw),
        gcwr: parseIntegerThousands(gcwr),
        tareWeight: parseIntegerThousands(tareWeight),
        isActive,
      });
      onSaved();
    } catch (e) {
      onError(formatApiError(e, "Could not save configuration"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card className="space-y-4 p-6">
      <h2 className="text-sm font-semibold text-slate-900">Vehicle type and plates</h2>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Horse plate">
          <Input value={horse} disabled />
        </Field>
        <Field label="Vehicle type">
          <Select value={vehicleType} onChange={(e) => onTypeChange(e.target.value)}>
            <option value="">Select type…</option>
            {typeOptions.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </Select>
        </Field>
        {vehicleType === "straight" ? (
          <Field label="Tank one plate number">
            <Input value={horse} disabled />
          </Field>
        ) : null}
        {vehicleType === "semi" || vehicleType === "pulling" ? (
          <Field label="Tank one plate number">
            <Input value={trailer} onChange={(e) => setTrailer(normalizePlate(e.target.value))} />
          </Field>
        ) : null}
        {vehicleType === "pulling" ? (
          <Field label="Tank two plate number">
            <Input value={trailerTwo} onChange={(e) => setTrailerTwo(normalizePlate(e.target.value))} />
          </Field>
        ) : null}
        <Field label="Loading">
          <Select value={loadingType} onChange={(e) => setLoading(e.target.value)}>
            <option value="top">Top</option>
            <option value="bottom">Bottom</option>
          </Select>
        </Field>
        <Field label="MPLW (kg)">
          <Input value={mplw} onChange={(e) => setMplw(formatIntegerThousands(e.target.value))} inputMode="numeric" />
        </Field>
        <Field label="GCWR (kg)">
          <Input value={gcwr} onChange={(e) => setGcwr(formatIntegerThousands(e.target.value))} inputMode="numeric" />
        </Field>
        <Field label="Tare weight (kg)">
          <Input value={tareWeight} onChange={(e) => setTare(formatIntegerThousands(e.target.value))} inputMode="numeric" />
        </Field>
        <ToggleField label="LNG / CNG" checked={lngCng} onChange={setLng} />
        <ToggleField label="Active" checked={isActive} onChange={setActive} />
      </div>
      <p className="text-xs leading-5 text-slate-500">
        Horse plate is fixed after registration. Straight: tank one is the horse, tank two empty.
        Semi-trailer: tank one required (may match the horse), tank two empty. Pulling: tank one and
        tank two required. Changing type unlinks extra tanks (they stay in the catalogue). A tank
        plate already in the system is assigned to this truck; otherwise a new tank is created.
      </p>
      <Button onClick={() => void save()} loading={saving}>
        Save configuration
      </Button>
    </Card>
  );
}

function CalibrationCard({
  truckId,
  tanks,
  configured,
  onSaved,
}: {
  truckId: string;
  tanks: TruckTank[];
  configured: boolean;
  onSaved: () => void;
}) {
  const [selected, setSelected] = useState(tanks[0]?.id ?? "");
  const tank = useMemo(
    () => tanks.find((t) => t.id === selected) || tanks[0],
    [tanks, selected],
  );
  const [validTo, setValidTo] = useState(tank?.validTo?.slice(0, 10) ?? "");
  const [caps, setCaps] = useState<string[]>(
    Array.from({ length: 10 }, (_, i) => tank?.compartments?.find((c) => c.index === i + 1)?.capacity ?? "0"),
  );
  const [newPlate, setNewPlate] = useState("");
  const [newIndex, setNewIndex] = useState("1");
  const [msg, setMsg] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!tank) return;
    setSelected(tank.id);
    setValidTo(tank.validTo?.slice(0, 10) ?? "");
    setCaps(Array.from({ length: 10 }, (_, i) => tank.compartments?.find((c) => c.index === i + 1)?.capacity ?? "0"));
  }, [tank]);

  const total = caps.reduce((s, v) => s + (Number(parseQty(v)) || 0), 0);

  async function saveCal() {
    if (!tank) return;
    setMsg("");
    setSaving(true);
    try {
      await api.put(`/master/trucks/${truckId}/tanks/${tank.id}/calibration`, {
        validTo,
        lines: caps.map((capacity, i) => ({ index: i + 1, capacity: parseQty(capacity) })),
      });
      onSaved();
      setMsg("Calibration saved");
    } catch (e) {
      setMsg(formatApiError(e, "Could not save calibration"));
    } finally {
      setSaving(false);
    }
  }

  async function addTank() {
    setMsg("");
    try {
      await api.post(`/master/trucks/${truckId}/tanks`, {
        plateNumber: normalizePlate(newPlate),
        index: Number(newIndex) || 1,
      });
      setNewPlate("");
      onSaved();
      setMsg("Tank added");
    } catch (e) {
      setMsg(formatApiError(e, "Could not add tank"));
    }
  }

  if (!configured) {
    return (
      <Card className="space-y-3 p-6">
        <h2 className="text-sm font-semibold text-slate-900">Tanks and calibration</h2>
        <p className="text-sm text-slate-500">
          Save vehicle type and tank plates first. Calibration cannot be set until the truck
          configuration is complete.
        </p>
      </Card>
    );
  }

  return (
    <Card className="space-y-4 p-6">
      <h2 className="text-sm font-semibold text-slate-900">Tanks and calibration</h2>
      <p className="text-xs text-slate-500">
        Select a tank to fill its compartment chart. Pulling trucks keep one tank at a time so each
        calibration stays readable. Total truck capacity is the sum of all tanks.
      </p>
      {tanks.length === 0 ? (
        <p className="text-sm text-slate-500">
          No tanks yet. Save vehicle type so tanks are linked for calibration, or add a tank below.
        </p>
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Tank">
              <Select value={tank?.id ?? ""} onChange={(e) => setSelected(e.target.value)}>
                {tanks.map((t) => (
                  <option key={t.id} value={t.id}>
                    {`Tank ${t.index} · ${t.plateNumber}`}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Certification end date">
              <DateInput value={validTo} onChange={setValidTo} />
            </Field>
          </div>
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
            {caps.map((v, i) => (
              <Field key={i} label={`Comp ${i + 1}`}>
                <QtyInput
                  unit="l"
                  value={v}
                  onChange={(raw) => setCaps((prev) => prev.map((p, idx) => (idx === i ? raw : p)))}
                />
              </Field>
            ))}
          </div>
          <p className="text-sm text-slate-700">
            Tank capacity: <span className="font-semibold tabular-nums">{total.toLocaleString()}</span> L
          </p>
          <Button variant="secondary" onClick={() => void saveCal()} loading={saving}>
            Save calibration
          </Button>
        </>
      )}
      <div className="grid gap-3 border-t border-slate-100 pt-4 sm:grid-cols-3">
        <Field label="New tank plate">
          <Input value={newPlate} onChange={(e) => setNewPlate(normalizePlate(e.target.value))} />
        </Field>
        <Field label="Index">
          <Select value={newIndex} onChange={(e) => setNewIndex(e.target.value)}>
            <option value="1">1 — tank one</option>
            <option value="2">2 — tank two</option>
          </Select>
        </Field>
        <div className="flex items-end">
          <Button variant="secondary" onClick={() => void addTank()} disabled={!newPlate.trim()}>
            Add tank
          </Button>
        </div>
      </div>
      {msg ? <p className="text-xs text-slate-600">{msg}</p> : null}
    </Card>
  );
}
