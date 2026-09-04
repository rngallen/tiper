package audit

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const maxDescriptionLen = 500

// EnrichDescription appends a human-readable list of changed fields to the
// base sentence. The full before/after values remain in Entry.Changes.
func EnrichDescription(base string, changes map[string]any) string {
	if len(changes) == 0 {
		return base
	}
	names := changedFieldNames(changes)
	summary := fmt.Sprintf("%s — changed: %s", base, strings.Join(names, ", "))
	if len(summary) > maxDescriptionLen {
		summary = summary[:maxDescriptionLen-3] + "..."
	}
	return summary
}

// changedFieldNames returns sorted, human-readable labels for the keys in changes.
func changedFieldNames(changes map[string]any) []string {
	names := make([]string, 0, len(changes))
	for k := range changes {
		names = append(names, humanizeField(k))
	}
	sort.Strings(names)
	return names
}

// humanizeField turns a JSON/camelCase key into a short label
// (e.g. tinNumber → Tin Number, appearanceSettings.theme → Appearance Settings Theme).
func humanizeField(name string) string {
	if name == "" {
		return name
	}
	parts := strings.Split(name, ".")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		var b strings.Builder
		for i, r := range part {
			if unicode.IsUpper(r) && i > 0 {
				b.WriteByte(' ')
			}
			if i == 0 {
				b.WriteRune(unicode.ToUpper(r))
			} else if r == '_' || r == '-' {
				b.WriteByte(' ')
			} else {
				b.WriteRune(r)
			}
		}
		labels = append(labels, b.String())
	}
	return strings.Join(labels, " · ")
}
