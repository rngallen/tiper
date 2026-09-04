package integrations

import (
	"dfms/apps/models"
	"dfms/pkg/config"
	"dfms/pkg/precision"
	"dfms/pkg/types"
)

func precisionHas(cfg models.JSONMap, key string) bool {
	return cfg != nil && cfg[key] != nil
}

func precisionFromRow(row models.IntegrationSetting) precision.Settings {
	setting := &models.IntegrationSetting{Config: row.Config}
	out := precision.Defaults
	if precisionHas(row.Config, "quantity") {
		out.Quantity = setting.ConfigInt("quantity")
	}
	if precisionHas(row.Config, "cubicMeter") {
		out.CubicMeter = setting.ConfigInt("cubicMeter")
	} else if precisionHas(row.Config, "quantity") {
		// Legacy DecimalPrecision used one place count for both L and m³.
		out.CubicMeter = out.Quantity
	}
	if precisionHas(row.Config, "metricTonne") {
		out.MetricTonne = setting.ConfigInt("metricTonne")
	}
	if precisionHas(row.Config, "density") {
		out.Density = setting.ConfigInt("density")
	}
	if precisionHas(row.Config, "price") {
		out.Price = setting.ConfigInt("price")
	}
	if precisionHas(row.Config, "miLoss") {
		out.MiLoss = setting.ConfigInt("miLoss")
	}
	return out.Normalize()
}

func precisionToConfig(cfg precision.Settings) map[string]any {
	c := cfg.Normalize()
	return map[string]any{
		"quantity":    c.Quantity,
		"cubicMeter":  c.CubicMeter,
		"metricTonne": c.MetricTonne,
		"density":     c.Density,
		"price":       c.Price,
		"miLoss":      c.MiLoss,
	}
}

func defaultPrecisionConfig() map[string]any {
	return precisionToConfig(precision.Defaults)
}

func ordersFromRow(row models.IntegrationSetting) config.OrdersConfig {
	setting := &models.IntegrationSetting{Config: row.Config}
	days := setting.ConfigInt("iloExpiryDays")
	if days < 1 {
		return config.DefaultOrders()
	}
	return config.OrdersConfig{IloExpiryDays: days}.Clamp()
}

func ordersToConfig(cfg config.OrdersConfig) map[string]any {
	c := cfg.Clamp()
	return map[string]any{"iloExpiryDays": c.IloExpiryDays}
}

func defaultOrdersConfig() map[string]any {
	return ordersToConfig(config.DefaultOrders())
}

func (s *Store) Precision() precision.Settings {
	if s == nil {
		return precision.Defaults
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.precision.Normalize()
}

func (s *Store) Orders() config.OrdersConfig {
	if s == nil {
		return config.DefaultOrders()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.orders.IloExpiryDays < 1 {
		return config.DefaultOrders()
	}
	return s.orders.Clamp()
}

// LivePrecision is the process-wide rounding policy (no per-request DB read).
func LivePrecision() precision.Settings {
	if Default == nil {
		return precision.Defaults
	}
	return Default.Precision()
}

// LiveOrders is the process-wide order policy (ILO expiry days).
func LiveOrders() config.OrdersConfig {
	if Default == nil {
		return config.DefaultOrders()
	}
	return Default.Orders()
}

func (s *Store) SavePrecision(cfg precision.Settings) error {
	return s.persist(types.KeyPrecision, precisionToConfig(cfg))
}

func (s *Store) SaveOrders(cfg config.OrdersConfig) error {
	return s.persist(types.KeyOrders, ordersToConfig(cfg))
}
