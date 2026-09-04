package audit_test

import (
	"strings"
	"testing"

	"dfms/apps/models"
	"dfms/pkg/audit"
)

func TestSummarizeChangesStruct(t *testing.T) {
	t.Parallel()
	before := models.Company{Name: "Acme", Phone: "111"}
	after := models.Company{Name: "TIPER", Phone: "222"}
	got := audit.SummarizeChanges(audit.DiffValues(before, after))
	if !strings.Contains(got, "Name") || !strings.Contains(got, "Phone") {
		t.Fatalf("summary = %q", got)
	}
	if strings.Contains(got, "Acme") || strings.Contains(got, "→") {
		t.Fatalf("toast must not echo values: %q", got)
	}
}

func TestSummarizeChangesMapAndSecret(t *testing.T) {
	t.Parallel()
	before := map[string]any{"host": "old.example", "port": 25, "hasPassword": true}
	after := map[string]any{"host": "smtp.example", "port": 25, "hasPassword": true}
	changes := audit.DiffValues(before, after)
	delete(changes, "hasPassword")
	audit.MergeSecretChange(changes, "password", true, true, true)
	got := audit.SummarizeChanges(changes)
	if !strings.Contains(got, "Host") {
		t.Fatalf("host missing: %q", got)
	}
	if !strings.Contains(got, "Password") {
		t.Fatalf("password missing: %q", got)
	}
	if strings.Contains(got, "old.example") || strings.Contains(got, "→") {
		t.Fatalf("toast must not echo values: %q", got)
	}
}

func TestSummarizeChangesCapsManyFields(t *testing.T) {
	t.Parallel()
	before := map[string]any{"a": "1", "b": "1", "c": "1", "d": "1", "e": "1"}
	after := map[string]any{"a": "2", "b": "2", "c": "2", "d": "2", "e": "2"}
	got := audit.SummarizeChanges(audit.DiffValues(before, after))
	if !strings.Contains(got, "and 2 more") {
		t.Fatalf("expected cap, got %q", got)
	}
	if strings.Contains(got, "→") {
		t.Fatalf("toast must not echo values: %q", got)
	}
}

func TestSummarizeChangesUnchanged(t *testing.T) {
	t.Parallel()
	row := models.Company{Name: "TIPER"}
	if got := audit.SummarizeChanges(audit.DiffValues(row, row)); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestSummarizeSet(t *testing.T) {
	t.Parallel()
	got := audit.SummarizeSet("Roles", []string{"Clerk", "Finance"}, []string{"Finance", "Ops"})
	if got != "Roles: added Ops; removed Clerk" {
		t.Fatalf("got %q", got)
	}
	if got := audit.SummarizeSet("Roles", []string{"Finance"}, []string{"Finance"}); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestUpdateMessage(t *testing.T) {
	t.Parallel()
	type row struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	}
	if got := audit.UpdateMessage(row{Name: "A", Phone: "1"}, row{Name: "B", Phone: "1"}); got != "Updated successfully — Name" {
		t.Fatalf("got %q", got)
	}
	if got := audit.UpdateMessage(row{Name: "A"}, row{Name: "A"}); got != "Updated successfully" {
		t.Fatalf("unchanged got %q", got)
	}
	if got := audit.UpdateMessageFromSummary("Roles: added Ops; removed Clerk"); got != "Updated successfully — Roles: added Ops; removed Clerk" {
		t.Fatalf("summary got %q", got)
	}
}
