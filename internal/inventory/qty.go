package inventory

import (
	"dfms/pkg/precision"

	"github.com/shopspring/decimal"
)

// QtyFromLitres derives tank units from litres and header density (MT/m³).
// m³ = L / 1000; MT = m³ × density. Uses seeded default places.
func QtyFromLitres(litres, density decimal.Decimal) (cm, mt decimal.Decimal) {
	return QtyFromLitresRounded(litres, density, precision.Defaults)
}

// QtyFromLitresRounded applies live places (L → m³, m³ × density → MT).
func QtyFromLitresRounded(litres, density decimal.Decimal, prec precision.Settings) (cm, mt decimal.Decimal) {
	if litres.IsZero() {
		return decimal.Zero, decimal.Zero
	}
	prec = prec.Normalize()
	cm = precision.Round(litres.Div(decimal.NewFromInt(1000)), prec.CubicMeter)
	mt = precision.Round(cm.Mul(density), prec.MetricTonne)
	return cm, mt
}
