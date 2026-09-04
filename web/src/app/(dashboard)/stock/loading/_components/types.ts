export type Party = { id: string; code?: string; name?: string; plateNumber?: string };

export type Charge = { charge: string; currencyCode: string; amount: string };

export type StockPos = {
  product?: Party;
  totalBalance: string;
  holdQty: string;
  freeQty: string;
  finalQty: string;
};

export type VesselRow = {
  id: string;
  vessel?: Party & { name?: string };
  product?: Party;
  vesselDate: string;
  quantity: string;
  financialHold: boolean;
};

export type LineRow = {
  id: string;
  documentNumber: string;
  customerOrderNumber?: string;
  product?: Party;
  requestedQty: string;
  transporterName?: string;
  driverName?: string;
  truckPlate?: string;
  horsePlate?: string;
  trailerOnePlate?: string;
  trailerTwoPlate?: string;
  destination?: string;
  district?: string;
  toDestination?: { id?: string; name?: string };
  toDistrict?: { id?: string; name?: string };
  ewuraLicense?: string;
  expirationDate?: string;
  transporter?: { id?: string; name?: string };
  driver?: { id?: string; name?: string };
  truck?: { id?: string; plateNumber?: string; trailer?: string; trailerTwo?: string; displayPlate?: string };
};

export type Approval = {
  approvedOn: string;
  approvedBy: string;
  title: string;
  comment: string;
  actName?: string;
};

export type GLR = {
  id: string;
  documentNumber: string;
  batchNumber?: string;
  orderDate: string;
  description?: string;
  status: string;
  quantity: string;
  byProductQuantity?: string;
  validContract?: boolean;
  loadingOrderAvailable?: boolean;
  iloExpiryDays?: number;
  customer?: Party & { ewuraLicense?: string };
  product?: Party;
  byProduct?: Party | null;
  stockStatus?: Party & { isTransit?: boolean; isLocal?: boolean };
  charges?: Charge[];
  outstanding?: Record<string, string>;
  stockPositions?: StockPos[];
  vessels?: VesselRow[];
  lines?: LineRow[];
  approvals?: Approval[];
};
