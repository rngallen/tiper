package types

// TxnType is a stock-ledger movement (StockMovement.TransactionType).
type TxnType string

const (
	TxnReceipt      TxnType = "receipt"
	TxnLoading      TxnType = "loading"
	TxnPumpOver     TxnType = "pump_over"
	TxnITT          TxnType = "itt"
	TxnZerolization TxnType = "zerolization"
	TxnAdjustment   TxnType = "adjustment"
	TxnReversal     TxnType = "reversal"
	TxnReservation  TxnType = "reservation"
	TxnHold         TxnType = "financial_hold"
	TxnHoldRelease  TxnType = "hold_release"
)

func (t TxnType) Valid() bool {
	switch t {
	case TxnReceipt, TxnLoading, TxnPumpOver, TxnITT, TxnZerolization,
		TxnAdjustment, TxnReversal, TxnReservation, TxnHold, TxnHoldRelease:
		return true
	}
	return false
}

// InventoryEventType is an inbound gantry / pipeline event (not a ledger txn).
// Pump-over uses a hyphen here; the ledger uses TxnPumpOver ("pump_over").
type InventoryEventType string

const (
	InvEventLoading  InventoryEventType = "loading"
	InvEventPumpOver InventoryEventType = "pump-over"
	InvEventITT      InventoryEventType = "itt"
)

// ReservationStatus is the stock-reservation row lifecycle.
type ReservationStatus string

const (
	ReservationOpen     ReservationStatus = "open"
	ReservationReleased ReservationStatus = "released"
	ReservationClosed   ReservationStatus = "closed"
)
