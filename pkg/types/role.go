package types

import "strings"

// RoleFamily is the workplace bucket for a role (Users assignment, Roles list).
// Terminal is depot operations (reception, pump-over, ITT). Gantry is trucks.
type RoleFamily string

const (
	RoleSystem   RoleFamily = "system"   // access, settings, IT
	RoleTerminal RoleFamily = "terminal" // receipts, pump-over, ITT, stock book
	RoleGantry   RoleFamily = "gantry"   // ILR, trucks, compartmentalization
	RoleFinance  RoleFamily = "finance"  // credit, executive approval
	RoleBilling  RoleFamily = "billing"  // FCF / VCF / TBS / KOJ runs
)

func (f RoleFamily) Valid() bool {
	switch f {
	case RoleSystem, RoleTerminal, RoleGantry, RoleFinance, RoleBilling:
		return true
	}
	return false
}

func (f RoleFamily) Label() string {
	switch f {
	case RoleTerminal:
		return "Terminal"
	case RoleGantry:
		return "Gantry"
	case RoleFinance:
		return "Finance"
	case RoleBilling:
		return "Billing"
	default:
		return "System"
	}
}

func NormalizeRoleFamily(s string) RoleFamily {
	f := RoleFamily(strings.ToLower(strings.TrimSpace(s)))
	if f == "stock" || f == "reception" {
		return RoleTerminal
	}
	if f.Valid() {
		return f
	}
	return RoleSystem
}
