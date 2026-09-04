package sage

import "testing"

func TestCurrencyCodeMatchesSageClientIDs(t *testing.T) {
	cases := []struct {
		id   int
		want string
	}{
		{0, "TZS"},
		{1, "USD"},
		{2, "GBP"},
		{3, "EUR"},
		{4, "ZAR"},
		{5, "CHF"},
	}
	for _, tc := range cases {
		got, ok := CurrencyCode(tc.id)
		if !ok || got != tc.want {
			t.Errorf("CurrencyCode(%d) = %q, %v; want %q, true", tc.id, got, ok, tc.want)
		}
	}
	if _, ok := CurrencyCode(99); ok {
		t.Fatal("unknown Sage currency id should not map")
	}
}

func TestClientActivePredicate(t *testing.T) {
	if clientActive != "ISNULL(On_Hold, 0) = 0" {
		t.Fatalf("list must exclude on-hold clients: %q", clientActive)
	}
}

func TestEscapeLike(t *testing.T) {
	got := escapeLike(`100%_foo`)
	if got != `100foo` {
		t.Fatalf("escapeLike = %q", got)
	}
}

func TestClientFinishMapsHoldAndCurrency(t *testing.T) {
	c := Client{Account: " ABC ", Name: " Puma ", OnHold: true, CurrencyID: 1}
	c.Finish()
	if c.Account != "ABC" || c.Name != "Puma" {
		t.Fatalf("trim: %+v", c)
	}
	if !c.OnHold || c.CurrencyCode != "USD" {
		t.Fatalf("hold/currency: %+v", c)
	}
	home := Client{Account: "HOME", CurrencyID: 0}
	home.Finish()
	if home.OnHold || home.CurrencyCode != "TZS" {
		t.Fatalf("home currency: %+v", home)
	}
}
