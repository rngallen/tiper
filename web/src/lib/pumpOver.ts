import { localISODate } from "@/lib/format";
import type { StockStatusRef } from "@/lib/status";

export type PartyRef = { id: string; code?: string; name?: string };

export type PumpOverVessel = {
  vessel?: PartyRef;
  vesselDate: string;
  quantity: string;
  stockStatus?: StockStatusRef;
};

export type PumpOverRequest = {
  id: string;
  documentNumber: string;
  orderDate: string;
  quantity: string;
  status: string;
  customerOrderNumber?: string;
  customer?: PartyRef;
  product?: PartyRef;
  depot?: PartyRef;
  stockStatus?: StockStatusRef;
  vessels?: PumpOverVessel[];
  createdBy?: { name?: string; email?: string };
};

export type PumpOverReport = {
  id: string;
  documentNumber: string;
  reportDate: string;
  actualDelivered: string;
  actualReceived: string;
  variance?: string;
  status: string;
  request?: PumpOverRequest;
  createdBy?: { name?: string; email?: string };
};

export function emptyPumpOverVessel(date = localISODate(new Date())) {
  return {
    vesselId: "",
    vesselLabel: "",
    vesselDate: date,
    stockStatusId: "",
    stockStatusLabel: "",
    quantity: "",
  };
}
