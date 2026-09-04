package orders

import (
	"context"
	"strings"
	"time"

	"dfms/pkg/types"

	"github.com/jellydator/validation"
	"github.com/shopspring/decimal"
)

func compact(s string) string { return strings.TrimSpace(s) }
func upper(s string) string   { return strings.ToUpper(strings.TrimSpace(s)) }

func optionalDecimal(value any) error {
	s, _ := value.(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if _, err := decimal.NewFromString(s); err != nil {
		return validation.NewError("validation_decimal", "must be a number")
	}
	return nil
}

func optionalDate(value any) error {
	s, _ := value.(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return nil
	}
	return validation.NewError("validation_date", "must be a date (YYYY-MM-DD)")
}

func distinctSeals(top, dip, bottom string) error {
	seen := map[string]int{}
	for _, v := range []string{top, dip, bottom} {
		if v == "" {
			continue
		}
		seen[v]++
		if seen[v] > 1 {
			return validation.NewError("validation_seals", "top, dip and bottom seals on a compartment must all be different")
		}
	}
	return nil
}

// createCompSchema is POST /orders/compartmentalizations.
// iloId is the open ILO (GantryLoadingLine). confirmExpiry is required when
// the ILO expires within three days (API returns 409 nearExpiry).
type createCompSchema struct {
	IloID         string `json:"iloId"`
	BadgeID       string `json:"badgeId"`
	ConfirmExpiry bool   `json:"confirmExpiry"`
}

func (s *createCompSchema) Sanitize() {
	s.IloID = compact(s.IloID)
	s.BadgeID = compact(s.BadgeID)
}

func (s createCompSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.IloID, validation.Required.Error("ILO is required"), validation.Length(1, 26)),
		validation.Field(&s.BadgeID, validation.Length(0, 26)),
	)
}

type saveCompLineSchema struct {
	ID         string `json:"id"`
	ProductID  string `json:"productId"`
	Quantity   string `json:"quantity"`
	TopSeal    string `json:"topSeal"`
	DipSeal    string `json:"dipSeal"`
	BottomSeal string `json:"bottomSeal"`
}

func (s *saveCompLineSchema) Sanitize() {
	s.ID = compact(s.ID)
	s.ProductID = compact(s.ProductID)
	s.Quantity = compact(s.Quantity)
	s.TopSeal = upper(s.TopSeal)
	s.DipSeal = upper(s.DipSeal)
	s.BottomSeal = upper(s.BottomSeal)
}

func (s saveCompLineSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.ID, validation.Required.Error("compartment id is required"), validation.Length(1, 26)),
		validation.Field(&s.ProductID, validation.Length(0, 26)),
		validation.Field(&s.Quantity, validation.By(optionalDecimal)),
		validation.Field(&s.TopSeal, validation.Length(0, 40)),
		validation.Field(&s.DipSeal, validation.Length(0, 40)),
		validation.Field(&s.BottomSeal, validation.Length(0, 40)),
		validation.Field(&s.TopSeal, validation.By(func(any) error {
			return distinctSeals(s.TopSeal, s.DipSeal, s.BottomSeal)
		})),
	)
}

type saveCompLinesSchema struct {
	Lines []saveCompLineSchema `json:"lines"`
}

func (s *saveCompLinesSchema) Sanitize() {
	for i := range s.Lines {
		s.Lines[i].Sanitize()
	}
}

func (s saveCompLinesSchema) Validate(ctx context.Context) error {
	if err := validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.Lines, validation.Required.Error("compartments are required"), validation.Length(1, 40)),
	); err != nil {
		return err
	}
	for i := range s.Lines {
		if err := s.Lines[i].Validate(ctx); err != nil {
			return err
		}
	}
	return nil
}

type createAmendmentSchema struct {
	Kind            string `json:"kind"`
	IloID           string `json:"iloId"`
	RequestedQty    string `json:"requestedQty"`
	ProductID       string `json:"productId"`
	ExpirationDate  string `json:"expirationDate"`
	Destination     string `json:"destination"`
	District        string `json:"district"`
	TruckPlate      string `json:"truckPlate"`
	TransporterName string `json:"transporterName"`
	DriverName      string `json:"driverName"`
	Notes           string `json:"notes"`
}

func (s *createAmendmentSchema) Sanitize() {
	s.Kind = compact(s.Kind)
	s.IloID = compact(s.IloID)
	s.RequestedQty = compact(s.RequestedQty)
	s.ProductID = compact(s.ProductID)
	s.ExpirationDate = compact(s.ExpirationDate)
	s.Destination = compact(s.Destination)
	s.District = compact(s.District)
	s.TruckPlate = upper(s.TruckPlate)
	s.TransporterName = compact(s.TransporterName)
	s.DriverName = compact(s.DriverName)
	s.Notes = compact(s.Notes)
}

func (s createAmendmentSchema) Validate(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &s,
		validation.Field(&s.Kind, validation.Required, validation.Length(1, 24), validation.By(func(any) error {
			if !types.AmendmentKind(s.Kind).Valid() {
				return validation.NewError("validation_kind", "unknown amendment kind")
			}
			return nil
		})),
		validation.Field(&s.IloID, validation.Required.Error("ILO is required"), validation.Length(1, 26)),
		validation.Field(&s.RequestedQty, validation.By(optionalDecimal)),
		validation.Field(&s.ProductID, validation.Length(0, 26)),
		validation.Field(&s.ExpirationDate, validation.By(optionalDate)),
		validation.Field(&s.Destination, validation.Length(0, 160)),
		validation.Field(&s.District, validation.Length(0, 80)),
		validation.Field(&s.TruckPlate, validation.Length(0, 80)),
		validation.Field(&s.TransporterName, validation.Length(0, 160)),
		validation.Field(&s.DriverName, validation.Length(0, 160)),
		validation.Field(&s.Notes, validation.Length(0, 400)),
	)
}
