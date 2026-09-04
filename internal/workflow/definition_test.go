package workflow

import (
	"errors"
	"strings"
	"testing"

	"dfms/apps/models"
)

func TestValidateApprovalSteps(t *testing.T) {
	err := validateApprovalSteps(nil)
	if !errors.Is(err, ErrInvalidDefinition) {
		t.Fatalf("expected ErrInvalidDefinition, got %v", err)
	}

	err = validateApprovalSteps([]ApprovalStepInput{
		{Name: "Reviewer", RoleID: 1},
		{Name: "reviewer", RoleID: 2},
	})
	if !errors.Is(err, ErrInvalidDefinition) {
		t.Fatalf("expected duplicate name error, got %v", err)
	}

	err = validateApprovalSteps([]ApprovalStepInput{
		{Name: "Stock Accountant", RoleID: 1, QuorumMode: "any"},
		{Name: "Finance Credit Controller", RoleID: 2, QuorumMode: "all"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = validateApprovalSteps([]ApprovalStepInput{
		{Name: "Review later"},
	})
	if err != nil {
		t.Fatalf("empty operator role should be allowed: %v", err)
	}
}

func TestSlugCode(t *testing.T) {
	if got := slugCode("Legal Review"); got != "legal-review" {
		t.Fatalf("got %q", got)
	}
	if got := slugCode("  "); got != "step" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildProcessNote(t *testing.T) {
	one := buildProcessNote("Receipt Clerk", []*models.Node{{Name: "Stock Accountant"}})
	if !strings.Contains(one, "Receipt Clerk submits") || !strings.Contains(one, "Stock Accountant must approve") {
		t.Fatalf("one-step note: %q", one)
	}
	three := buildProcessNote("Customer Care Supervisor", []*models.Node{{Name: "Stock Accountant"}, {Name: "Finance Credit Controller"}, {Name: "Managing Director"}})
	if !strings.Contains(three, "Customer Care Supervisor submits") || !strings.Contains(three, "must each approve") {
		t.Fatalf("three-step note: %q", three)
	}
}

func TestDefaultProcesses(t *testing.T) {
	var rcpt, varfee, zerol bool
	for _, raw := range defaultProcesses() {
		p := raw.withDefaults()
		if p.DraftName != initiatorNode {
			t.Fatalf("%s draft: %q", p.Code, p.DraftName)
		}
		if len(p.Steps) != 2 || p.Steps[0] != "Credit Controller" || p.Steps[1] != "Managing Director" {
			t.Fatalf("%s steps: %v", p.Code, p.Steps)
		}
		switch p.Code {
		case "RCPT":
			rcpt = true
		case "VARFEE":
			varfee = true
		case "ZEROL":
			zerol = true
		}
	}
	if !rcpt || !varfee || !zerol {
		t.Fatal("missing default DFMS process")
	}
}

func TestDraftNodeName(t *testing.T) {
	if got := draftNodeName(&models.Process{Code: "RCPT"}); got != "Initiator" {
		t.Fatalf("RCPT: %q", got)
	}
	if got := draftNodeName(&models.Process{Code: "custom"}); got != "Initiator" {
		t.Fatalf("custom: %q", got)
	}
}
