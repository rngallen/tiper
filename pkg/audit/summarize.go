package audit

import (
	"fmt"
	"sort"
	"strings"
)

const (
	maxNamedFields = 3
	updateOK       = "Updated successfully"
)

// UpdateMessage is "Updated successfully" plus field diffs when any exist.
func UpdateMessage(before, after any) string {
	return UpdateMessageFromSummary(SummarizeChanges(DiffValues(before, after)))
}

// UpdateMessageFromChanges is "Updated successfully" plus a Diff map summary.
func UpdateMessageFromChanges(changes map[string]any) string {
	return UpdateMessageFromSummary(SummarizeChanges(changes))
}

// UpdateMessageFromSummary is "Updated successfully" plus an already-built
// sentence (roles added/removed). Empty summary stays generic.
func UpdateMessageFromSummary(summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return updateOK
	}
	return updateOK + " — " + summary
}

// DiffValues diffs structs or string-keyed maps. Use this for API success copy.
func DiffValues(before, after any) map[string]any {
	var raw map[string]any
	if bm, bok := asStringMap(before); bok {
		am, _ := asStringMap(after)
		raw = diffMaps("", bm, am)
		if raw == nil {
			raw = map[string]any{}
		}
		stripped := make(map[string]any, len(raw))
		for k, v := range raw {
			stripped[strings.TrimPrefix(k, ".")] = v
		}
		raw = stripped
	} else {
		raw = Diff(before, after)
	}
	return DropMetaKeys(raw)
}

// MergeSecretChange records a password/API-key change without storing plaintext.
func MergeSecretChange(changes map[string]any, field string, touched, beforeSet, afterSet bool) {
	if changes == nil || !touched || strings.TrimSpace(field) == "" {
		return
	}
	switch {
	case beforeSet && afterSet:
		changes[field] = FieldChange{Before: "(set)", After: "(updated)"}
	case !beforeSet && afterSet:
		changes[field] = FieldChange{Before: nil, After: "(set)"}
	case beforeSet && !afterSet:
		changes[field] = FieldChange{Before: "(set)", After: "(cleared)"}
	}
}

// SummarizeChanges lists changed field names only — never before/after
// values. More than maxNamedFields becomes "Name, Phone and 4 more".
func SummarizeChanges(changes map[string]any) string {
	names := changedFieldNames(changes)
	if len(names) == 0 {
		return ""
	}
	if len(names) <= maxNamedFields {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(names[:maxNamedFields], ", "), len(names)-maxNamedFields)
}

// SummarizeSet describes added/removed names (roles, permissions).
func SummarizeSet(label string, before, after []string) string {
	added, removed := setDiff(before, after)
	if len(added) == 0 && len(removed) == 0 {
		return ""
	}
	var parts []string
	if len(added) > 0 {
		parts = append(parts, "added "+strings.Join(added, ", "))
	}
	if len(removed) > 0 {
		parts = append(parts, "removed "+strings.Join(removed, ", "))
	}
	return label + ": " + strings.Join(parts, "; ")
}

func setDiff(before, after []string) (added, removed []string) {
	b := map[string]struct{}{}
	a := map[string]struct{}{}
	for _, s := range before {
		s = strings.TrimSpace(s)
		if s != "" {
			b[s] = struct{}{}
		}
	}
	for _, s := range after {
		s = strings.TrimSpace(s)
		if s != "" {
			a[s] = struct{}{}
		}
	}
	for s := range a {
		if _, ok := b[s]; !ok {
			added = append(added, s)
		}
	}
	for s := range b {
		if _, ok := a[s]; !ok {
			removed = append(removed, s)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

func isSecretField(key string) bool {
	k := strings.ToLower(key)
	return strings.Contains(k, "password") ||
		strings.Contains(k, "apikey") ||
		strings.Contains(k, "api_key") ||
		strings.Contains(k, "secret") ||
		strings.Contains(k, "token")
}

func secretPhrase(fc FieldChange) string {
	b := displayValue(fc.Before)
	a := displayValue(fc.After)
	if (b == "—" || b == "(set)") && a != "—" && a != "(cleared)" {
		if a == "(updated)" {
			return "updated"
		}
		return "set"
	}
	if a == "—" || a == "(cleared)" {
		return "cleared"
	}
	return "updated"
}

func displayValue(v any) string {
	if v == nil {
		return "—"
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return "—"
		}
		return s
	case bool:
		if t {
			return "yes"
		}
		return "no"
	case fmt.Stringer:
		s := strings.TrimSpace(t.String())
		if s == "" {
			return "—"
		}
		return s
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return "—"
		}
		return s
	}
}
