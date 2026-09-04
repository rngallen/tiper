package models

import "testing"

func TestClassifyStockStatus(t *testing.T) {
	cases := []struct {
		name                           string
		transit, local, mining, prorat bool
	}{
		{"Transit", true, false, false, false},
		{"Congo", false, true, false, false}, // country names are not guessed — UI sets Transit
		{"Local", false, true, false, false},
		{"Mining", false, true, true, false},
		{"Mines", false, true, true, false},
		{"Proration", false, false, false, true},
		{"Unknown domestic", false, true, false, false},
	}
	for _, tc := range cases {
		tr, loc, mine, pr := ClassifyStockStatus(tc.name)
		if tr != tc.transit || loc != tc.local || mine != tc.mining || pr != tc.prorat {
			t.Errorf("%q: got transit=%v local=%v mining=%v proration=%v",
				tc.name, tr, loc, mine, pr)
		}
	}
}

func TestSlugStatusCode(t *testing.T) {
	if got := SlugStatusCode("Congo DRC"); got != "CONGODRC" {
		t.Fatalf("slug: %q", got)
	}
}
