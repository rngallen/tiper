package response

import "strings"

// StableOrder builds `col DIR` with an ID tie-breaker so paging is deterministic.
// dir must already be ASC or DESC (ParseSearchRequest normalizes it).
func StableOrder(column, dir string) string {
	return StableOrderTie(column, dir, "ID")
}

// StableOrderTie is StableOrder with an explicit tie-break column (use
// `[Table].ID` when the query joins other tables).
func StableOrderTie(column, dir, tie string) string {
	col := strings.TrimSpace(column)
	if col == "" {
		col = "ID"
	}
	tie = strings.TrimSpace(tie)
	if tie == "" {
		tie = "ID"
	}
	if orderIdent(col) == orderIdent(tie) {
		return col + " " + dir
	}
	return col + " " + dir + ", " + tie + " ASC"
}

func orderIdent(col string) string {
	s := strings.ToUpper(strings.TrimSpace(col))
	if i := strings.LastIndex(s, "."); i >= 0 {
		s = s[i+1:]
	}
	return strings.Trim(s, "[]")
}
