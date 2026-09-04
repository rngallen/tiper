// Package precision is the process-wide rounding policy (L, m³, MT, density, money).
// Values live in IntegrationSetting key=precision and are cached by internal/integrations.
package precision

import "github.com/shopspring/decimal"

// Settings is the live rounding policy. Defaults match the integrations seed.
type Settings struct {
	Quantity    int
	CubicMeter  int
	MetricTonne int
	Density     int
	Price       int
	MiLoss      int
}

// Defaults is 2 / 3 / 3 / 5 / 4 / 4 (litres two places; m³ stay three).
var Defaults = Settings{
	Quantity:    2,
	CubicMeter:  3,
	MetricTonne: 3,
	Density:     5,
	Price:       4,
	MiLoss:      4,
}

func ClampPlaces(n int) int {
	if n < 0 {
		return 0
	}
	if n > 6 {
		return 6
	}
	return n
}

func (s Settings) Normalize() Settings {
	out := s
	out.Quantity = ClampPlaces(s.Quantity)
	out.CubicMeter = ClampPlaces(s.CubicMeter)
	out.MetricTonne = ClampPlaces(s.MetricTonne)
	out.Density = ClampPlaces(s.Density)
	out.Price = ClampPlaces(s.Price)
	out.MiLoss = ClampPlaces(s.MiLoss)
	return out
}

func Round(d decimal.Decimal, places int) decimal.Decimal {
	return d.Round(int32(ClampPlaces(places)))
}
