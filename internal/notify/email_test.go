package notify

import (
	"strings"
	"testing"

	"dfms/apps/models"
	wfengine "dfms/internal/workflow"
	"dfms/pkg/types"
)

func TestOverlaySubstitute_MarksCovering(t *testing.T) {
	principal := &models.User{FirstName: "Jane", LastName: "Doe", Email: "jane@example.com"}
	intent := notifyIntentFor(nil, &models.Node{Name: "Gantry Supervisor"}, "")
	intro := "A document is awaiting your action at the Gantry Supervisor step."
	next := "Review the request in DFMS."
	sms := "Action required GLR-1"
	overlaySubstitute(principal, &intent, &intro, &next, &sms)
	if intent.badge != "Acting as substitute" {
		t.Fatalf("badge: %q", intent.badge)
	}
	if intent.kind != types.EmailBadgeInfo {
		t.Fatalf("kind: %q", intent.kind)
	}
	if !strings.HasPrefix(intent.subjectPrefix, "Substitute · ") {
		t.Fatalf("subject prefix: %q", intent.subjectPrefix)
	}
	if !strings.Contains(intro, "covering for") || !strings.Contains(intro, "Jane Doe") {
		t.Fatalf("intro: %q", intro)
	}
	if !strings.Contains(next, "acting for Jane Doe") {
		t.Fatalf("next: %q", next)
	}
	if !strings.Contains(sms, "Substitute for Jane Doe") {
		t.Fatalf("sms: %q", sms)
	}
}

func TestDocumentApprovalMessages_SubstituteOverlay(t *testing.T) {
	inst := &models.ProcessInstance{
		No:             "GLR-1",
		Summary:        "PUMA Energy AGO",
		DocContentType: types.GantryLoadingRequestContent,
		Status:         types.NodeRunning,
	}
	node := &models.Node{Name: "Gantry Supervisor", Status: types.NodeRunning}
	principal := &models.User{FirstName: "Asha", LastName: "Mwamba", Email: "asha@example.com"}
	subject, html, sms := documentApprovalMessages(inst, node, EmailBrand{FromName: "TIPER DFMS", PortalURL: "https://dfms.example.com"}, principal, "")
	if !strings.HasPrefix(subject, "Substitute · ") {
		t.Fatalf("subject: %q", subject)
	}
	for _, want := range []string{
		"ACTING AS SUBSTITUTE",
		"covering for",
		"Asha Mwamba",
		"Acting for",
		"Internal loading request",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
	if !strings.Contains(sms, "Substitute for Asha Mwamba") {
		t.Fatalf("sms: %q", sms)
	}
}

func TestNotifyIntentFor_CompleteAndSubmit(t *testing.T) {
	inst := &models.ProcessInstance{Status: types.NodeCompleted}
	node := &models.Node{Name: "Approved", Status: types.NodeCompleted}
	got := notifyIntentFor(inst, node, types.NotifyComplete)
	if got.subjectPrefix != "Approved" || got.kind != types.EmailBadgeInfo {
		t.Fatalf("complete: %+v", got)
	}
	got = notifyIntentFor(inst, node, types.NotifySubmit)
	if got.subjectPrefix != "Submitted" || got.introVerb != "has been submitted for approval" {
		t.Fatalf("submit: %+v", got)
	}
	got = notifyIntentFor(inst, node, "")
	if got.subjectPrefix != "Approved" {
		t.Fatalf("completed instance without event should be approved, got %+v", got)
	}
}

func TestReadableDocumentRef_SkipsULID(t *testing.T) {
	ulid := "01M1GDDVWMVSP29AZ2RS8R36ET"
	inst := &models.ProcessInstance{No: ulid, Summary: ulid, DocContentType: types.ExchangeRateContent}
	if got := readableDocumentRef(inst, nil, "Exchange rate"); got != "Exchange rate" {
		t.Fatalf("nil facts: %q", got)
	}
	facts := &wfengine.DocumentFacts{DocumentNumber: "USD/TZS · 2026-09-02", FromCurrency: "USD", ToCurrency: "TZS"}
	if got := readableDocumentRef(inst, facts, "Exchange rate"); got != "USD/TZS · 2026-09-02" {
		t.Fatalf("facts: %q", got)
	}
}

func TestDocumentApprovalMessages_UsesFactsNotULID(t *testing.T) {
	inst := &models.ProcessInstance{
		No:             "01M1GDDVWMVSP29AZ2RS8R36ET",
		Summary:        "01M1GDDVWMVSP29AZ2RS8R36ET",
		DocContentType: types.GantryLoadingRequestContent,
		Status:         types.NodeRunning,
	}
	node := &models.Node{Name: "Gantry Supervisor", Status: types.NodeRunning}
	facts := &wfengine.DocumentFacts{
		DocumentNumber: "ILR-1001",
		Description:    "September lifting",
		CustomerName:   "PUMA Energy",
		Product:        "AGO — Gasoil",
		Quantity:       "45000",
	}
	subject, html, _ := documentApprovalMessagesWithFacts(inst, node, EmailBrand{FromName: "TIPER DFMS", PortalURL: "https://dfms.example.com"}, nil, "", facts)
	if !strings.Contains(subject, "ILR-1001") {
		t.Fatalf("subject: %q", subject)
	}
	if strings.Contains(subject, "01M1GDDV") || strings.Contains(html, "01M1GDDV") {
		t.Fatalf("ULID leaked: subject=%q", subject)
	}
	for _, want := range []string{"ILR-1001", "PUMA Energy", "AGO — Gasoil", "45000", "September lifting", "/approvals/gantry"} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
}

func TestDocumentApprovalMessages_CompleteIsInfoOnly(t *testing.T) {
	inst := &models.ProcessInstance{
		No:             "GLR-1",
		Summary:        "PUMA Energy AGO",
		DocContentType: types.GantryLoadingRequestContent,
		Status:         types.NodeCompleted,
	}
	node := &models.Node{Name: "Approved", Status: types.NodeCompleted}
	subject, html, _ := documentApprovalMessages(inst, node, EmailBrand{FromName: "TIPER DFMS", PortalURL: "https://dfms.example.com"}, nil, types.NotifyComplete)
	if !strings.HasPrefix(subject, "Approved:") {
		t.Fatalf("subject: %q", subject)
	}
	for _, want := range []string{
		"APPROVED",
		"has been approved",
		"No further action is required",
		"listed for workflow notifications",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
	if strings.Contains(html, "Action required") {
		t.Error("complete mail must not say Action required")
	}
}
