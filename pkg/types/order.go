package types

import "strings"

// OrderStatus is the operational lifecycle of ILR, ILO, compartmentalization,
// pump-over, and order amendments.
type OrderStatus string

const (
	OrderDraft      OrderStatus = "draft"
	OrderSubmitted  OrderStatus = "submitted"
	OrderApproved   OrderStatus = "approved"
	OrderOpen       OrderStatus = "open"       // ILO ready for compartmentalization
	OrderRunning    OrderStatus = "running"    // compartments assigned, not yet sent to ALMA
	OrderInProgress OrderStatus = "inprogress" // SAP3C dropped; waiting for ATLAS NEO
	OrderLoaded     OrderStatus = "loaded"
	OrderClosed     OrderStatus = "closed" // posted to Sage / NPGIS
	OrderCompleted  OrderStatus = "completed"
	OrderRejected   OrderStatus = "rejected"
	OrderCancelled  OrderStatus = "cancelled"
	OrderExpired    OrderStatus = "expired" // midnight job; drops out of stock commitment
)

func (s OrderStatus) Valid() bool {
	switch s {
	case OrderDraft, OrderSubmitted, OrderApproved, OrderOpen, OrderRunning,
		OrderInProgress, OrderLoaded, OrderClosed, OrderCompleted, OrderRejected,
		OrderCancelled, OrderExpired:
		return true
	}
	return false
}

// AmendmentKind is the kind of change on an OrderAmendment (Django AmendementType).
type AmendmentKind string

const (
	AmendNormal      AmendmentKind = "normal"
	AmendQtyIncrease AmendmentKind = "qty_increase"
	AmendQtyDecrease AmendmentKind = "qty_decrease"
	AmendProduct     AmendmentKind = "product_change"
	AmendCancel      AmendmentKind = "cancel"
	AmendBatchCancel AmendmentKind = "batch_cancel"
	AmendExtend      AmendmentKind = "extend"
)

func (k AmendmentKind) Valid() bool {
	switch k {
	case AmendNormal, AmendQtyIncrease, AmendQtyDecrease, AmendProduct,
		AmendCancel, AmendBatchCancel, AmendExtend:
		return true
	}
	return false
}

// ImmediateAmendment reports kinds that apply without a workflow (Customer Care).
func ImmediateAmendment(kind AmendmentKind) bool {
	return kind == AmendExtend || kind == AmendBatchCancel
}

// LoadingType is how a truck is filled at the gantry (Django LoadingType).
type LoadingType string

const (
	LoadingTop    LoadingType = "top"
	LoadingBottom LoadingType = "bottom"
)

func (t LoadingType) Valid() bool {
	switch t {
	case "", LoadingTop, LoadingBottom:
		return true
	}
	return false
}

// VehicleType is the gantry truck configuration (Django VehicleType).
type VehicleType string

const (
	VehiclePending  VehicleType = "pending" // registered, type not configured yet
	VehicleStraight VehicleType = "straight"
	VehicleSemi     VehicleType = "semi"
	VehiclePulling  VehicleType = "pulling"
)

func NormalizeVehicleType(v string) VehicleType {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case string(VehiclePending):
		return VehiclePending
	case string(VehicleSemi):
		return VehicleSemi
	case string(VehiclePulling), "horse":
		return VehiclePulling
	case string(VehicleStraight):
		return VehicleStraight
	default:
		return VehicleStraight
	}
}

// VehicleTypeConfigured is true once an operator has chosen straight / semi / pulling.
func VehicleTypeConfigured(v VehicleType) bool {
	switch strings.ToLower(strings.TrimSpace(string(v))) {
	case string(VehicleStraight), string(VehicleSemi), string(VehiclePulling), "horse":
		return true
	default:
		return false
	}
}
