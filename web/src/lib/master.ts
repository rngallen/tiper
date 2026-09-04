import { api } from "./api";
import { formatQty } from "./format";
import { asRows } from "./pagination";

export type MasterRef = {
  id: string;
  code?: string;
  name?: string;
  isActive?: boolean;
  hasData?: boolean;
};

export type Customer = MasterRef & {
  email?: string;
  phone?: string;
  tinNumber?: string;
  kycNumber?: string;
  vrnNumber?: string;
  ewuraLicense: string;
  hasData?: boolean;
  billingAccounts?: BillingAccount[];
};

export type Supplier = MasterRef & {
  email?: string;
  phone?: string;
  mobile?: string;
  contactPerson?: string;
  tinNumber?: string;
  countryCode?: string;
  country?: { code: string; name?: string } | null;
  address?: string;
  address2?: string;
  hasData?: boolean;
  billingAccounts?: BillingAccount[];
};

export type BillingAccount = {
  id: string;
  feeCode: string;
  currencyCode: string;
  sageAccount: string;
  sageName?: string;
  billingUnit: string;
  isForeign?: boolean;
  isActive?: boolean;
};

export type Product = MasterRef & {
  unit: string;
  stockCategory?: MasterRef | null;
  hasData?: boolean;
};

export type Tank = MasterRef & {
  maximumCapacity?: string | number;
  deadStock?: string | number;
  product?: MasterRef | null;
  hasData?: boolean;
};

export type Vessel = MasterRef & {
  imoNumber?: string;
  hasData?: boolean;
};

export type Depot = MasterRef & {
  ewuraLicense?: string;
  isInternal?: boolean;
  customer?: MasterRef | null;
  hasData?: boolean;
};

export type Transporter = MasterRef & {
  license?: string;
  phone?: string;
  email?: string;
  contactPerson?: string;
  tinNumber?: string;
  vrnNumber?: string;
  countryCode?: string;
  country?: { code: string; name?: string } | null;
  address?: string;
  address2?: string;
  aeoEndDate?: string;
};

export type Truck = MasterRef & {
  plateNumber: string;
  trailer?: string;
  trailerTwo?: string;
  displayPlate?: string;
  vehicleType?: string;
  loadingType?: string;
  lngCng?: boolean;
  mplw?: string;
  gcwr?: string;
  tareWeight?: string;
  totalCapacity?: string;
  tanks?: TruckTank[];
};

export type TruckTank = {
  id: string;
  plateNumber: string;
  index: number;
  isActive?: boolean;
  capacity?: string;
  validTo?: string;
  compartments?: { index: number; capacity: string }[];
};

export type Driver = MasterRef & {
  licenseNumber?: string;
  licenseExpires?: string;
  phone?: string;
  email?: string;
};

export type Country = { code: string; name: string };

export type Destination = MasterRef & {
  isCountry?: boolean;
};

export type District = MasterRef & {
  destination?: Destination;
};

export type Category = MasterRef;

export type EwuraLicense = {
  licenseNumber: string;
  licensee: string;
  licenseClass?: string;
  licenseType?: string;
  sector?: string;
  zoneName?: string;
  regionName?: string;
  districtName?: string;
  tinNumber?: string;
  phone?: string;
  email?: string;
  issueDate?: string;
  expiryDate?: string;
  isActive?: boolean;
};

export function qty(v: string | number | undefined | null): string {
  return formatQty(v);
}

/** Combobox option from a master row (`code — name`). */
export function masterOption(r: { id: string; code?: string; name?: string }) {
  const label = r.code && r.name ? `${r.code} — ${r.name}` : r.name || r.code || r.id;
  return { value: r.id, label };
}

/** Search a catalogue or operational list. Sends page=1, pageSize=40; add allDates for dated lists. */
export async function searchMaster<T>(
  path: string,
  q = "",
  extra?: Record<string, string | number | boolean | undefined>,
): Promise<T[]> {
  const data = await api.get<unknown>(path, {
    search: q || undefined,
    page: 1,
    pageSize: 40,
    isActive: true,
    ...extra,
  });
  return asRows<T>(data);
}
