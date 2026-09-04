package orders

import (
	"testing"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"
)

func TestDocumentApprovalsUsesSnapshotWithoutDB(t *testing.T) {
	snap := models.ApprovalTrail{{
		ActedAt: time.Date(2024, 1, 2, 15, 4, 0, 0, time.UTC),
		ActType: "approve", ActName: "Approved", UserName: "Ada", Title: "CFO", Comment: "ok",
	}}
	got := DocumentApprovals(nil, types.GantryLoadingRequestContent, 9, snap)
	if len(got) != 1 || got[0].UserName != "Ada" {
		t.Fatalf("got %+v", got)
	}
	if DocumentApprovals(nil, types.GantryLoadingRequestContent, 0, snap)[0].UserName != "Ada" {
		t.Fatal("zero object id should still return snapshot")
	}
}

func TestApprovalTrailAsILRFormat(t *testing.T) {
	at := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	rows := models.ApprovalTrail{{
		ActedAt: at, ActType: "approve", ActName: "Approved",
		UserName: "Jane Doe", Title: "Stock Accountant", Comment: "cleared",
	}}.AsILR()
	if len(rows) != 1 {
		t.Fatalf("len=%d", len(rows))
	}
	if rows[0].ApprovedOn != "26/08/2026 10:00" {
		t.Fatalf("ApprovedOn=%q", rows[0].ApprovedOn)
	}
	if rows[0].ApprovedBy != "Jane Doe" || rows[0].Comment != "cleared" {
		t.Fatalf("%+v", rows[0])
	}
}
