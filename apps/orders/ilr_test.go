package orders

import (
	"testing"

	"dfms/apps/models"

	"github.com/shopspring/decimal"
)

func TestGantryGradeOf(t *testing.T) {
	if gantryGradeOf(models.Product{Code: "1002", Name: "AGO"}) != gantryAGO {
		t.Fatal("AGO")
	}
	if gantryGradeOf(models.Product{Code: "1001", Name: "MOGAS"}) != gantryPMS {
		t.Fatal("MOGAS")
	}
	if gantryGradeOf(models.Product{Name: "PMS"}) != gantryPMS {
		t.Fatal("PMS")
	}
	if gantryGradeOf(models.Product{Name: "IK"}) != "" {
		t.Fatal("IK is not a gantry grade")
	}
}

func TestTruckComboPlate(t *testing.T) {
	if got := models.TruckComboPlate("T228EAP", "T600BTQ", "T600E"); got != "T228EAP/T600BTQ/T600E" {
		t.Fatalf("got %s", got)
	}
	if got := models.TruckComboPlate("T589EGM", "T747EAS", ""); got != "T589EGM/T747EAS" {
		t.Fatalf("got %s", got)
	}
	if got := models.TruckComboPlate("ZA11482", "", ""); got != "ZA11482" {
		t.Fatalf("got %s", got)
	}
}

func TestValidateByProduct(t *testing.T) {
	by := uint(2)
	same := uint(1)
	if err := validateByProduct(1, &same, decimal.NewFromInt(10), false); err == nil {
		t.Fatal("expected by-product to differ from product")
	}
	if err := validateByProduct(1, nil, decimal.NewFromInt(10), false); err == nil {
		t.Fatal("expected by-product when quantity is set")
	}
	if err := validateByProduct(1, &by, decimal.NewFromFloat(0.5), false); err == nil {
		t.Fatal("expected by-product quantity >= 1")
	}
	if err := validateByProduct(1, &by, decimal.NewFromInt(10), true); err == nil {
		t.Fatal("expected transit to reject by-product")
	}
	if err := validateByProduct(1, nil, decimal.Zero, false); err != nil {
		t.Fatal(err)
	}
	if err := validateByProduct(1, &by, decimal.NewFromInt(5), false); err != nil {
		t.Fatal(err)
	}
}

func TestMatchingOrderProduct(t *testing.T) {
	by := uint(2)
	req := models.GantryLoadingRequest{ProductID: 1, Quantity: decimal.NewFromInt(100), ByProductID: &by, ByProductQuantity: decimal.NewFromInt(40)}
	if err := matchingOrderProduct(req, 1); err != nil {
		t.Fatal(err)
	}
	if err := matchingOrderProduct(req, 2); err != nil {
		t.Fatal(err)
	}
	if err := matchingOrderProduct(req, 9); err == nil {
		t.Fatal("expected foreign product to fail")
	}
}

func TestValidateEqualTotals(t *testing.T) {
	by := uint(2)
	req := models.GantryLoadingRequest{
		ProductID: 1, Quantity: decimal.NewFromInt(100),
		ByProductID: &by, ByProductQuantity: decimal.NewFromInt(40),
		Vessels: []models.GantryRequestVessel{
			{ProductID: 1, Quantity: decimal.NewFromInt(100)},
			{ProductID: 2, Quantity: decimal.NewFromInt(40)},
		},
		Lines: []models.GantryLoadingLine{
			{ProductID: 1, RequestedQty: decimal.NewFromInt(60), IsActive: true},
			{ProductID: 1, RequestedQty: decimal.NewFromInt(40), IsActive: true},
			{ProductID: 2, RequestedQty: decimal.NewFromInt(40), IsActive: true},
		},
	}
	if err := validateEqualTotals(req); err != nil {
		t.Fatal(err)
	}
	req.Lines[0].RequestedQty = decimal.NewFromInt(50)
	if err := validateEqualTotals(req); err == nil {
		t.Fatal("expected truck totals to fail")
	}
}
