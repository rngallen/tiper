package permissions

import (
	"slices"
	"strings"
)

// ModuleManage builds a module-level wildcard code: {module}.manage
func ModuleManage(module string) string {
	return module + ".manage"
}

// Satisfies reports whether the held permission codes grant the required code.
//
// Hierarchy (most specific first):
//   - exact code match
//   - {module}.{resource}.manage (every action on that document)
//   - {module}.{action} for a three-part required code (legacy / admin convenience)
//   - {module}.manage (every action in the module)
//
// orders.pumpreport.submit does not grant orders.pumpover.submit.
// orders.read grants orders.ilr.read but not orders.ilr.submit.
func Satisfies(held []string, required string) bool {
	if required == "" {
		return false
	}
	if slices.Contains(held, required) {
		return true
	}

	parts := strings.Split(required, ".")
	switch len(parts) {
	case 2:
		return slices.Contains(held, ModuleManage(parts[0]))
	case 3:
		module, resource, action := parts[0], parts[1], parts[2]
		if slices.Contains(held, module+"."+resource+".manage") {
			return true
		}
		if slices.Contains(held, module+"."+action) {
			return true
		}
		return slices.Contains(held, ModuleManage(module))
	default:
		return false
	}
}

// SatisfiesAny reports whether the held codes satisfy at least one of the
// required codes (OR semantics used by route middleware).
func SatisfiesAny(held []string, required ...string) bool {
	for _, r := range required {
		if Satisfies(held, r) {
			return true
		}
	}
	return false
}
