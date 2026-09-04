package audit_test

import (
	"strings"
	"testing"
	"time"

	"dfms/apps/models"
	"dfms/pkg/audit"
)

func TestDiffExpandsAppearanceMap(t *testing.T) {
	before := models.Profile{
		AppearanceSetting: map[string]any{
			"theme":        "light",
			"compactMode":  false,
			"largeText":    false,
			"sidebarState": false,
		},
	}
	after := models.Profile{
		AppearanceSetting: map[string]any{
			"theme":        "dark",
			"compactMode":  true,
			"largeText":    false,
			"sidebarState": false,
		},
	}

	changes := audit.Diff(before, after)
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d: %#v", len(changes), changes)
	}

	theme, ok := changes["appearanceSettings.theme"].(audit.FieldChange)
	if !ok {
		t.Fatalf("missing theme change: %#v", changes)
	}
	if theme.Before != "light" || theme.After != "dark" {
		t.Fatalf("theme change = %#v", theme)
	}

	compact, ok := changes["appearanceSettings.compactMode"].(audit.FieldChange)
	if !ok {
		t.Fatalf("missing compactMode change: %#v", changes)
	}
	if compact.Before != false || compact.After != true {
		t.Fatalf("compactMode change = %#v", compact)
	}
}

func TestAttachChangesCreateEditDelete(t *testing.T) {
	created := models.CustomerBillingAccount{SageAccount: "4000", BillingUnit: "M3"}
	createEntry := audit.Entry{Description: "account created"}
	audit.AttachChanges(&createEntry, nil, created)
	got, ok := createEntry.Changes["sageAccount"].(audit.FieldChange)
	if !ok || got.After != "4000" {
		t.Fatalf("create snapshot = %#v", createEntry.Changes)
	}

	before := created
	after := created
	after.SageAccount = "4001"
	editEntry := audit.Entry{Description: "account updated"}
	audit.AttachChanges(&editEntry, before, after)
	got, ok = editEntry.Changes["sageAccount"].(audit.FieldChange)
	if !ok || got.Before != "4000" || got.After != "4001" {
		t.Fatalf("edit diff = %#v", editEntry.Changes)
	}
	if !strings.Contains(editEntry.Description, "Sage Account") {
		t.Fatalf("edit description missing field: %q", editEntry.Description)
	}

	deleteEntry := audit.Entry{Description: "account deleted"}
	audit.AttachChanges(&deleteEntry, before, nil)
	got, ok = deleteEntry.Changes["sageAccount"].(audit.FieldChange)
	if !ok || got.Before != "4000" {
		t.Fatalf("delete snapshot = %#v", deleteEntry.Changes)
	}
}

func TestDropAppearanceSettingKeysFromUserDiff(t *testing.T) {
	before := models.User{
		FirstName: "Ada",
		Profile: models.Profile{
			Title: "Clerk",
			AppearanceSetting: map[string]any{
				"theme": "light",
			},
		},
	}
	after := before
	after.FirstName = "Ada Lovelace"
	after.Profile.AppearanceSetting = map[string]any{
		"theme": "dark",
	}
	after.Profile.UpdatedAt = after.Profile.UpdatedAt.Add(time.Second)

	changes := audit.DropAppearanceSettingKeys(audit.DropMetaKeys(audit.Diff(before, after)))
	if len(changes) != 1 {
		t.Fatalf("expected only firstName, got %d: %#v", len(changes), changes)
	}
	name, ok := changes["firstName"].(audit.FieldChange)
	if !ok || name.Before != "Ada" || name.After != "Ada Lovelace" {
		t.Fatalf("firstName change = %#v", changes)
	}
}

func TestDiffCustomerBillingAccount(t *testing.T) {
	before := models.CustomerBillingAccount{
		SageAccount:  "4000",
		CurrencyCode: "TZS",
		BillingUnit:  "M3",
		FeeCode:      "FSF",
	}
	after := before
	after.SageAccount = "4001"
	after.BillingUnit = "MT"
	after.UpdatedAt = after.UpdatedAt.Add(time.Second)

	changes := audit.DropMetaKeys(audit.Diff(before, after))
	if len(changes) != 2 {
		t.Fatalf("expected 2 business changes, got %d: %#v", len(changes), changes)
	}
	bank, ok := changes["sageAccount"].(audit.FieldChange)
	if !ok || bank.Before != "4000" || bank.After != "4001" {
		t.Fatalf("sageAccount change = %#v", changes["sageAccount"])
	}
	name, ok := changes["billingUnit"].(audit.FieldChange)
	if !ok || name.Before != "M3" || name.After != "MT" {
		t.Fatalf("billingUnit change = %#v", changes["billingUnit"])
	}
}
