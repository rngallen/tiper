"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api, formatApiError } from "@/lib/api";
import { Field, Input, Select } from "@/components/ui";
import { normalizePlate } from "@/lib/format";
import type { Truck } from "@/lib/master";
import { CatalogueForm } from "../form";

const typeOptions = [
  { value: "straight", label: "Straight" },
  { value: "semi", label: "Semi-trailer" },
  { value: "pulling", label: "Pulling" },
];

export function TruckForm({
  onClose,
  onSaved,
}: {
  row?: Truck;
  onClose: () => void;
  onSaved: () => void;
}) {
  const router = useRouter();
  const [plateNumber, setPlate] = useState("");
  const [trailer, setTrailer] = useState("");
  const [trailerTwo, setTrailerTwo] = useState("");
  const [vehicleType, setType] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  function onTypeChange(next: string) {
    setType(next);
    if (next === "straight") {
      setTrailer(plateNumber);
      setTrailerTwo("");
    } else if (next === "semi") {
      setTrailerTwo("");
    }
  }

  async function save() {
    setError("");
    const horse = normalizePlate(plateNumber);
    if (!horse) {
      setError("Horse plate number is required.");
      return;
    }
    const tankOne = vehicleType === "straight" ? horse : normalizePlate(trailer);
    const tankTwo = vehicleType === "pulling" || vehicleType === "" ? normalizePlate(trailerTwo) : "";
    if (vehicleType === "semi" && !tankOne) {
      setError("Semi-trailer requires tank one plate number.");
      return;
    }
    if (vehicleType === "pulling" && (!tankOne || !tankTwo)) {
      setError("Pulling requires tank one and tank two plate numbers.");
      return;
    }
    setSaving(true);
    try {
      const created = await api.post<Truck>("/master/trucks", {
        plateNumber: horse,
        trailer: tankOne,
        trailerTwo: vehicleType === "pulling" || !vehicleType ? tankTwo : "",
        vehicleType: vehicleType || undefined,
      });
      onSaved();
      if (created?.id) {
        router.push(`/stock/setups/trucks/${created.id}`);
        return;
      }
    } catch (err) {
      setError(formatApiError(err, "Failed to save truck"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <CatalogueForm
      title="Add truck"
      error={error}
      saving={saving}
      onClose={onClose}
      onSave={() => void save()}
      saveLabel="Create"
      size="md"
    >
      <Field label="Horse plate number">
        <Input
          value={plateNumber}
          onChange={(e) => {
            const plate = normalizePlate(e.target.value);
            setPlate(plate);
            if (vehicleType === "straight") setTrailer(plate);
          }}
          placeholder="T124ABC"
          autoFocus
        />
      </Field>
      <Field label="Vehicle type">
        <Select value={vehicleType} onChange={(e) => onTypeChange(e.target.value)}>
          <option value="">Not yet — horse only or plates from an order</option>
          {typeOptions.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </Select>
      </Field>
      {vehicleType === "straight" ? (
        <Field label="Tank one plate number">
          <Input value={plateNumber} disabled />
        </Field>
      ) : (
        <Field label="Tank one plate number">
          <Input
            value={trailer}
            onChange={(e) => setTrailer(normalizePlate(e.target.value))}
            placeholder={vehicleType === "semi" || vehicleType === "pulling" ? "Required" : "Optional until type is set"}
          />
        </Field>
      )}
      {vehicleType === "" || vehicleType === "pulling" ? (
        <Field label="Tank two plate number">
          <Input
            value={trailerTwo}
            onChange={(e) => setTrailerTwo(normalizePlate(e.target.value))}
            placeholder={vehicleType === "pulling" ? "Required" : "Optional until type is set"}
          />
        </Field>
      ) : null}
      <p className="text-xs leading-5 text-slate-500">
        Letters and digits only, stored uppercase. Leave type empty when the order only has a horse
        plate — the truck can still be used on an ILR line. Type, weights, and calibration must be
        completed before compartmentalization.
      </p>
    </CatalogueForm>
  );
}
