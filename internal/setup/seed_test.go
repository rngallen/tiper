package setup_test

import (
	"slices"
	"testing"

	"dfms/internal/setup"
)

func TestCurrencyCodesMatchSageCatalogue(t *testing.T) {
	got := setup.CurrencyCodes()
	for _, code := range []string{"USD", "GBP", "EUR", "ZAR", "CHF", "TZS"} {
		if !slices.Contains(got, code) {
			t.Errorf("missing %s in %v", code, got)
		}
	}
	for _, leftover := range []string{"AUD", "INR"} {
		if slices.Contains(got, leftover) {
			t.Errorf("retired %s should not be seeded", leftover)
		}
	}
}
