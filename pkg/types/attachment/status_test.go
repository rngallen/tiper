package attachment

import "testing"

func TestCanMutateStatus(t *testing.T) {
	if !CanMutateStatus("draft") || !CanMutateStatus("returned") {
		t.Fatal("draft and returned must allow files")
	}
	if CanMutateStatus("submitted") || CanMutateStatus("approved") {
		t.Fatal("submitted/approved must lock files")
	}
}
