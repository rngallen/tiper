package inventory

import (
	"context"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"

	"github.com/jellydator/validation"
	"github.com/shopspring/decimal"
)

func compact(s string) string { return strings.TrimSpace(s) }

func (in *receiptInput) Sanitize() {
	in.VesselID = compact(in.VesselID)
	in.SupplierID = compact(in.SupplierID)
	in.ProductID = compact(in.ProductID)
	in.RouteCode = strings.ToUpper(compact(in.RouteCode))
	in.TenderCode = strings.ToUpper(compact(in.TenderCode))
	in.ProcurementMethodCode = strings.ToUpper(compact(in.ProcurementMethodCode))
	in.ReceiptType = strings.ToLower(compact(in.ReceiptType))
	in.Density = compact(in.Density)
	in.Notes = compact(in.Notes)
}

func (in receiptInput) headerType() types.ReceiptType {
	if in.ReceiptType == "" {
		return types.ReceiptInternal
	}
	return types.ReceiptType(in.ReceiptType)
}

func (in receiptInput) Validate(ctx context.Context) error {
	kind := in.headerType()
	rules := []*validation.FieldRules{
		validation.Field(&in.Date, validation.Required.Error("date is required")),
		validation.Field(&in.VesselDate, validation.Required.Error("vessel date is required")),
		validation.Field(&in.VesselID, validation.Required.Error("vessel is required"), validation.Length(1, 26)),
		validation.Field(&in.ProductID, validation.Required.Error("product is required"), validation.Length(1, 26)),
		validation.Field(&in.RouteCode, validation.Required.Error("discharge route is required"), validation.Length(1, 20)),
		validation.Field(&in.Notes, validation.Required.Error("notes are required"), validation.Length(1, 500)),
	}
	if kind != types.ReceiptExternal {
		rules = append(rules,
			validation.Field(&in.SupplierID, validation.Required.Error("supplier is required"), validation.Length(1, 26)),
			validation.Field(&in.TenderCode, validation.Required.Error("nature of tender is required"), validation.Length(1, 20)),
			validation.Field(&in.ProcurementMethodCode, validation.Required.Error("method of tender is required"), validation.Length(1, 20)),
			validation.Field(&in.Density, validation.By(requireReceiptDensity)),
		)
	}
	return validation.ValidateStructWithContext(ctx, &in, rules...)
}

func requireReceiptDensity(value any) error {
	s, _ := value.(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return validation.NewError("validation_density", "density is required")
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return validation.NewError("validation_density", "density must be a number")
	}
	if d.LessThanOrEqual(decimal.Zero) {
		return validation.NewError("validation_density", "density must be greater than zero")
	}
	if d.GreaterThanOrEqual(decimal.NewFromInt(2)) {
		return validation.NewError("validation_density", "enter density as MT per m³ (e.g. 0.84), not kg/m³")
	}
	return nil
}

func assertReceiptHeader(row *models.Receipt) error {
	if row == nil {
		return validation.NewError("validation_required", "receipt is required")
	}
	errs := validation.Errors{}
	if row.Date.IsZero() {
		errs["date"] = validation.NewError("validation_required", "date is required")
	}
	if row.VesselDate.IsZero() {
		errs["vesselDate"] = validation.NewError("validation_required", "vessel date is required")
	}
	if row.VesselID == 0 {
		errs["vesselId"] = validation.NewError("validation_required", "vessel is required")
	}
	if row.ProductID == 0 {
		errs["productId"] = validation.NewError("validation_required", "product is required")
	}
	if strings.TrimSpace(string(row.RouteCode)) == "" {
		errs["routeCode"] = validation.NewError("validation_required", "discharge route is required")
	}
	if strings.TrimSpace(row.Notes) == "" {
		errs["notes"] = validation.NewError("validation_required", "notes are required")
	}
	if row.ReceiptType != types.ReceiptExternal {
		if row.SupplierID == nil || *row.SupplierID == 0 {
			errs["supplierId"] = validation.NewError("validation_required", "supplier is required")
		}
		if strings.TrimSpace(string(row.TenderCode)) == "" {
			errs["tenderCode"] = validation.NewError("validation_required", "nature of tender is required")
		}
		if strings.TrimSpace(string(row.ProcurementMethodCode)) == "" {
			errs["procurementMethodCode"] = validation.NewError("validation_required", "method of tender is required")
		}
		if err := requireReceiptDensity(row.Density.String()); err != nil {
			errs["density"] = err
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

type holdReleaseInput struct {
	ReleaseDate time.Time       `json:"releaseDate"`
	Description string          `json:"description"`
	Notes       string          `json:"notes"`
	Lines       []holdLineInput `json:"lines"`
}

type holdLineInput struct {
	CustomerID    string    `json:"customerId"`
	ProductID     string    `json:"productId"`
	VesselID      string    `json:"vesselId"`
	VesselDate    time.Time `json:"vesselDate"`
	StockStatusID string    `json:"stockStatusId"`
	Quantity      string    `json:"quantity"`
	CubicMeter    string    `json:"cubicMeter"`
	MetricTonne   string    `json:"metricTonne"`
}

func (in *holdReleaseInput) Sanitize() {
	in.Description = compact(in.Description)
	in.Notes = compact(in.Notes)
	for i := range in.Lines {
		in.Lines[i].CustomerID = compact(in.Lines[i].CustomerID)
		in.Lines[i].ProductID = compact(in.Lines[i].ProductID)
		in.Lines[i].VesselID = compact(in.Lines[i].VesselID)
		in.Lines[i].StockStatusID = compact(in.Lines[i].StockStatusID)
		in.Lines[i].Quantity = compact(in.Lines[i].Quantity)
		in.Lines[i].CubicMeter = compact(in.Lines[i].CubicMeter)
		in.Lines[i].MetricTonne = compact(in.Lines[i].MetricTonne)
	}
}

func (in holdReleaseInput) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &in,
		validation.Field(&in.ReleaseDate, validation.Required.Error("date is required")),
		validation.Field(&in.Description, validation.Required.Error("description is required"), validation.Length(2, 250)),
		validation.Field(&in.Lines, validation.Required.Error("add at least one parcel"), validation.Length(1, 200)),
	)
}
