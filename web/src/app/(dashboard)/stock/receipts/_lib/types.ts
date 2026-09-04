export type Party = { id?: string; code?: string; name?: string };

export type ReceiptDetail = {
  id: string;
  customer?: Party;
  stockStatus?: Party & { isProration?: boolean };
  depot?: Party | null;
  collectionMethod?: string;
  contractTypeCode?: string;
  pricingNature?: string;
  nextBillingDays?: number;
  financialHold?: boolean;
  density?: string;
  quantity?: string;
  cubicMeter?: string;
  metricTonne?: string;
  lineLoss?: string;
  lineLossCubicMeter?: string;
  lineLossMetricTonne?: string;
};

export type Receipt = {
  id: string;
  documentNumber: string;
  date: string;
  vesselDate?: string;
  receiptType: string;
  status: string;
  isProvision: boolean;
  isFinal?: boolean;
  routeCode?: string;
  tenderCode?: string;
  procurementMethodCode?: string;
  density?: string;
  tankQuantity?: string;
  tankCubicMeter?: string;
  tankMetricTonne?: string;
  lineLoss?: string;
  lineLossCubicMeter?: string;
  lineLossMetricTonne?: string;
  usesTiperPipeline?: boolean;
  notes?: string;
  vessel?: Party;
  product?: Party;
  supplier?: Party;
  details?: ReceiptDetail[];
  createdBy?: { id?: string; name?: string; email?: string };
};
