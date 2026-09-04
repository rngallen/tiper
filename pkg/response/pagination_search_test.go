package response

import (
	"testing"
	"time"
)

func TestHasSearch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"%%", false},
		{"  %%  ", false},
		{"%a%", true},
		{"%invoice%", true},
	}
	for _, tc := range cases {
		if got := (SearchOutput{Search: tc.in}).HasSearch(); got != tc.want {
			t.Errorf("HasSearch(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestDefaultFromForToAnchorsOnToDate(t *testing.T) {
	t.Parallel()
	to := time.Date(2008, 5, 8, 23, 59, 59, 0, time.UTC)
	from := defaultFromForTo(to, 0)
	if from.Year() != 2006 || from.Month() != time.May || from.Day() != 8 {
		t.Fatalf("report lookback from as-of 2008: got %v", from)
	}
	from90 := defaultFromForTo(to, 90)
	if from90.After(to) {
		t.Fatalf("ops lookback must stay before as-of: %v > %v", from90, to)
	}
}

func TestParseAsOfCutoff(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	got, err := ParseAsOf("", now)
	if err != nil || got.Year() != 2026 || got.Day() != 18 || got.Hour() != 23 {
		t.Fatalf("empty as-of: %v %v", got, err)
	}
	got, err = ParseAsOf("08/05/2008", now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Year() != 2008 || got.Month() != time.May || got.Day() != 8 {
		t.Fatalf("DD/MM/YYYY as-of: got %v", got)
	}
	if _, err := ParseAsOf("not-a-date", now); err == nil {
		t.Fatal("expected invalid toDate")
	}
}

func TestDefaultOpsLookbackDays(t *testing.T) {
	t.Parallel()
	if DefaultOpsLookbackDays != 90 {
		t.Fatalf("operational lists must default to 90 days, got %d", DefaultOpsLookbackDays)
	}
	if MinDateRangeYears != 2 {
		t.Fatalf("reports must keep a 2-year default, got %d", MinDateRangeYears)
	}
}

func TestStableOrderTie(t *testing.T) {
	t.Parallel()
	if got := StableOrder("DocumentNumber", SortDesc); got != "DocumentNumber DESC, ID ASC" {
		t.Fatalf("StableOrder: %q", got)
	}
	if got := StableOrderTie("[GantryLoadingRequest].OrderDate", SortDesc, "[GantryLoadingRequest].ID"); got != "[GantryLoadingRequest].OrderDate DESC, [GantryLoadingRequest].ID ASC" {
		t.Fatalf("joined tie-break: %q", got)
	}
	if got := StableOrderTie("[GantryLoadingRequest].ID", SortAsc, "[GantryLoadingRequest].ID"); got != "[GantryLoadingRequest].ID ASC" {
		t.Fatalf("id column should not double-sort: %q", got)
	}
	if got := StableOrderTie("SortOrder", SortAsc, "Code"); got != "SortOrder ASC, Code ASC" {
		t.Fatalf("code-keyed catalog: %q", got)
	}
	if got := StableOrderTie("SortOrder", SortAsc, "Days"); got != "SortOrder ASC, Days ASC" {
		t.Fatalf("billing cycle: %q", got)
	}
	if got := StableOrderTie("Code", SortAsc, "Code"); got != "Code ASC" {
		t.Fatalf("code-keyed catalog must not repeat Code: %q", got)
	}
	if got := StableOrderTie("Days", SortAsc, "Days"); got != "Days ASC" {
		t.Fatalf("billing cycle sort by Days must not repeat Days: %q", got)
	}
}
