package inventory

import (
	"testing"

	"dfms/apps/models"
	"dfms/pkg/types"
)

func TestFactDepotLabel(t *testing.T) {
	code, name := factDepotLabel(types.ReceiptInternal, nil)
	if code != "" || name != "TIPER" {
		t.Fatalf("internal without depot: %s %s", code, name)
	}
	code, name = factDepotLabel(types.ReceiptExternal, nil)
	if code != "" || name != "OTHERS" {
		t.Fatalf("external without depot: %s %s", code, name)
	}
	dep := &models.Depot{Code: "KPL", Name: "Kipawa"}
	code, name = factDepotLabel(types.ReceiptExternal, dep)
	if code != "KPL" || name != "Kipawa" {
		t.Fatalf("named depot: %s %s", code, name)
	}
}
