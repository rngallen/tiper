package notify

import (
	"strings"
	"testing"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"
)

func TestNextBackoff(t *testing.T) {
	if NextBackoff(0) != 0 {
		t.Fatalf("attempt 0: %s", NextBackoff(0))
	}
	if NextBackoff(1) != time.Minute {
		t.Fatalf("attempt 1: %s", NextBackoff(1))
	}
	if NextBackoff(7) != time.Hour {
		t.Fatalf("attempt 7: %s", NextBackoff(7))
	}
	if NextBackoff(11) != 12*time.Hour {
		t.Fatalf("attempt 11: %s", NextBackoff(11))
	}
	if NextBackoff(99) != 12*time.Hour {
		t.Fatalf("attempt 99 should cap: %s", NextBackoff(99))
	}
}

func TestShouldDeadLetter(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	if ShouldDeadLetter(3, 12, now.Add(-time.Hour), now) {
		t.Fatal("young row with remaining attempts must retry")
	}
	if !ShouldDeadLetter(12, 12, now.Add(-time.Hour), now) {
		t.Fatal("max attempts must dead-letter")
	}
	if !ShouldDeadLetter(2, 12, now.Add(-73*time.Hour), now) {
		t.Fatal("older than 72h must dead-letter")
	}
	if ShouldDeadLetter(1, 0, now, now) {
		t.Fatal("zero maxAttempts defaults to 12")
	}
}

func TestBuildCredentialOutbox_EmailAndSMS(t *testing.T) {
	user := &models.User{
		ID:    9,
		Email: "ada@example.com",
		Profile: models.Profile{
			PhoneNumber: "255700000001",
		},
	}
	rows := buildCredentialOutbox(user, "Welcome", "<html>pwd</html>", "sms body", types.CredentialWelcome)
	if len(rows) != 2 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].Channel != types.NotifyChannelEmail || rows[0].Recipient != "ada@example.com" {
		t.Fatalf("email row: %+v", rows[0])
	}
	if rows[0].Kind != types.NotifyKindWelcome || rows[0].Subject != "Welcome" {
		t.Fatalf("email kind/subject: %+v", rows[0])
	}
	if rows[1].Channel != types.NotifyChannelSMS || rows[1].Recipient != "255700000001" {
		t.Fatalf("sms row: %+v", rows[1])
	}
	if rows[0].UserID == nil || *rows[0].UserID != 9 {
		t.Fatal("user id")
	}
}

func TestBuildCredentialOutbox_ResetEmailOnly(t *testing.T) {
	user := &models.User{Email: "ada@example.com"}
	rows := buildCredentialOutbox(user, "Reset", "body", "sms", types.CredentialReset)
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].Kind != types.NotifyKindReset || rows[0].Channel != types.NotifyChannelEmail {
		t.Fatalf("%+v", rows[0])
	}
}

func TestBuildWorkflowOutbox_DedupesAndPhones(t *testing.T) {
	users := []models.User{
		{ID: 1, Email: "A@example.com"},
		{ID: 2, Email: "a@example.com"},
		{ID: 3, Email: "b@example.com"},
		{ID: 4, Email: ""},
	}
	phones := map[uint]string{1: "2551", 3: "2553", 4: "2554"}
	rows := buildWorkflowOutbox(users, phones, "Action required", "<html>", "open DFMS")
	var emails, sms []string
	for _, r := range rows {
		if r.Kind != types.NotifyKindWorkflow {
			t.Fatalf("kind %q", r.Kind)
		}
		switch r.Channel {
		case types.NotifyChannelEmail:
			emails = append(emails, r.Recipient)
		case types.NotifyChannelSMS:
			sms = append(sms, r.Recipient)
		default:
			t.Fatalf("channel %q", r.Channel)
		}
	}
	if len(emails) != 2 {
		t.Fatalf("emails=%v", emails)
	}
	if !strings.EqualFold(emails[0], "A@example.com") {
		t.Fatalf("first email %q", emails[0])
	}
	if emails[1] != "b@example.com" {
		t.Fatalf("second email %q", emails[1])
	}
	if len(sms) != 3 {
		t.Fatalf("sms=%v", sms)
	}
}

func TestBuildWorkflowOutbox_ProfilePhoneWins(t *testing.T) {
	users := []models.User{{
		ID:      1,
		Email:   "a@example.com",
		Profile: models.Profile{PhoneNumber: "from-profile"},
	}}
	rows := buildWorkflowOutbox(users, map[uint]string{1: "from-lookup"}, "s", "e", "sms")
	if len(rows) != 2 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[1].Recipient != "from-profile" {
		t.Fatalf("phone %q", rows[1].Recipient)
	}
}

func TestCredentialMessages_WelcomeContainsPassword(t *testing.T) {
	user := &models.User{Email: "ada@example.com", FirstName: "Ada", LastName: "Lovelace"}
	subject, html, sms := credentialMessages(user, "Temp#1", EmailBrand{FromName: "TIPER DFMS", PortalURL: "https://dfms.example.com"}, types.CredentialWelcome)
	if subject != "Welcome to TIPER DFMS" {
		t.Fatalf("subject %q", subject)
	}
	if !strings.Contains(html, "Temp#1") || !strings.Contains(html, "ada@example.com") {
		t.Fatalf("html missing password or email")
	}
	if !strings.Contains(sms, "Temp#1") {
		t.Fatalf("sms %q", sms)
	}
}

func TestTruncateErr(t *testing.T) {
	if truncateErr("short") != "short" {
		t.Fatal("short")
	}
	long := strings.Repeat("x", outboxErrLimit+20)
	got := truncateErr(long)
	if n := len([]rune(got)); n != outboxErrLimit {
		t.Fatalf("runes=%d", n)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("suffix %q", got[len(got)-1:])
	}
}
