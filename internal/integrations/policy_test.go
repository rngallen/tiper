package integrations

import (
	"testing"

	"dfms/apps/models"
	"dfms/pkg/config"
	"dfms/pkg/precision"
)

func TestPrecisionFromRowDefaults(t *testing.T) {
	got := precisionFromRow(models.IntegrationSetting{})
	if got != precision.Defaults {
		t.Fatalf("empty row: %+v", got)
	}
}

func TestPrecisionFromRowLegacyQuantity(t *testing.T) {
	got := precisionFromRow(models.IntegrationSetting{
		Config: models.JSONMap{"quantity": 2, "metricTonne": 1},
	})
	if got.Quantity != 2 || got.CubicMeter != 2 {
		t.Fatalf("legacy L/m³ share: %+v", got)
	}
	if got.MetricTonne != 1 || got.Density != precision.Defaults.Density {
		t.Fatalf("partial: %+v", got)
	}
}

func TestPrecisionZeroPlacesKept(t *testing.T) {
	got := precisionFromRow(models.IntegrationSetting{
		Config: models.JSONMap{
			"quantity": 0, "cubicMeter": 0, "metricTonne": 0,
			"density": 0, "price": 0, "miLoss": 0,
		},
	})
	if got.Quantity != 0 || got.CubicMeter != 0 || got.Density != 0 {
		t.Fatalf("explicit zeros: %+v", got)
	}
}

func TestPrecisionRoundTrip(t *testing.T) {
	in := precision.Settings{Quantity: 1, CubicMeter: 2, MetricTonne: 3, Density: 4, Price: 5, MiLoss: 6}
	got := precisionFromRow(models.IntegrationSetting{Config: models.JSONMap(precisionToConfig(in))})
	if got != in.Normalize() {
		t.Fatalf("round-trip: %+v", got)
	}
}

func TestOrdersFromRowDefaults(t *testing.T) {
	if got := ordersFromRow(models.IntegrationSetting{}); got.IloExpiryDays != 14 {
		t.Fatalf("empty: %+v", got)
	}
	if got := ordersFromRow(models.IntegrationSetting{Config: models.JSONMap{"iloExpiryDays": 14}}); got.IloExpiryDays != 14 {
		t.Fatalf("14: %+v", got)
	}
	if got := ordersFromRow(models.IntegrationSetting{Config: models.JSONMap{"iloExpiryDays": 200}}); got.IloExpiryDays != 90 {
		t.Fatalf("clamp: %+v", got)
	}
}

func TestOrdersRoundTrip(t *testing.T) {
	in := config.OrdersConfig{IloExpiryDays: 21}.Clamp()
	got := ordersFromRow(models.IntegrationSetting{Config: models.JSONMap(ordersToConfig(in))})
	if got.IloExpiryDays != 21 {
		t.Fatalf("round-trip: %+v", got)
	}
}

func TestLiveFallbacks(t *testing.T) {
	prev := Default
	Default = nil
	t.Cleanup(func() { Default = prev })
	if LivePrecision() != precision.Defaults {
		t.Fatal("LivePrecision")
	}
	if LiveOrders().IloExpiryDays != 14 {
		t.Fatal("LiveOrders")
	}
}
