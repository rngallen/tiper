package types

// NpgisKind is an EWURA NPGIS outbox row.
type NpgisKind string

const (
	NpgisLoading  NpgisKind = "loading"
	NpgisPumpOver NpgisKind = "pump_over"
	NpgisReceipt  NpgisKind = "receipt"
	NpgisTank     NpgisKind = "tank"
)

func (k NpgisKind) Valid() bool {
	return k == NpgisLoading || k == NpgisPumpOver || k == NpgisReceipt || k == NpgisTank
}

// AlmaDirection is a SAP3C write or SAP3R read.
type AlmaDirection string

const (
	AlmaOut AlmaDirection = "out"
	AlmaIn  AlmaDirection = "in"
)
