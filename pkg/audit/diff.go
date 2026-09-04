package audit

import (
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"
)

// FieldChange is a before/after pair recorded for one changed field.
type FieldChange struct {
	Before any `json:"before"`
	After  any `json:"after"`
}

// Diff compares two values of the same struct type and returns the fields that
// changed, keyed by the field's JSON name (falling back to the Go field name).
// Fields tagged `json:"-"` and unexported fields are ignored. Nested structs
// and map[string]any values (e.g. appearance settings JSON) are expanded into
// dotted keys so audits show theme: light → dark instead of object → object.
//
// before and after may be structs or pointers to structs; nil/zero inputs
// yield an empty map.
func Diff(before, after any) map[string]any {
	return diffValue("", before, after)
}

func diffValue(prefix string, before, after any) map[string]any {
	bv := indirect(reflect.ValueOf(before))
	av := indirect(reflect.ValueOf(after))
	out := map[string]any{}

	if !bv.IsValid() || !av.IsValid() {
		return out
	}
	if bv.Kind() != reflect.Struct || av.Kind() != reflect.Struct || bv.Type() != av.Type() {
		return out
	}

	t := bv.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" { // unexported
			continue
		}
		name, skip := jsonName(field)
		if skip {
			continue
		}
		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		bVal := bv.Field(i).Interface()
		aVal := av.Field(i).Interface()
		if equalValues(bVal, aVal) {
			continue
		}
		if expanded := diffMaps(key, bVal, aVal); expanded != nil {
			maps.Copy(out, expanded)
			continue
		}
		if nested := diffNestedStruct(key, bVal, aVal); nested != nil {
			maps.Copy(out, nested)
			continue
		}
		out[key] = FieldChange{Before: normalizeValue(bVal), After: normalizeValue(aVal)}
	}
	return out
}

// diffNestedStruct expands a nested struct field into dotted keys.
// Returns nil when the values are not comparable structs.
func diffNestedStruct(prefix string, before, after any) map[string]any {
	bv := indirect(reflect.ValueOf(before))
	av := indirect(reflect.ValueOf(after))
	if !bv.IsValid() || !av.IsValid() {
		return nil
	}
	if bv.Kind() != reflect.Struct || av.Kind() != reflect.Struct || bv.Type() != av.Type() {
		return nil
	}
	return diffValue(prefix, before, after)
}

// diffMaps expands map[string]any (or map[string]T) field diffs into dotted keys.
// Returns nil when the values are not maps.
func diffMaps(prefix string, before, after any) map[string]any {
	bm, bok := asStringMap(before)
	am, aok := asStringMap(after)
	if !bok && !aok {
		return nil
	}
	if !bok {
		bm = map[string]any{}
	}
	if !aok {
		am = map[string]any{}
	}

	keys := make(map[string]struct{}, len(bm)+len(am))
	for k := range bm {
		keys[k] = struct{}{}
	}
	for k := range am {
		keys[k] = struct{}{}
	}

	out := map[string]any{}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	for _, k := range sorted {
		bVal, bOK := bm[k]
		aVal, aOK := am[k]
		if !bOK {
			bVal = nil
		}
		if !aOK {
			aVal = nil
		}
		if equalValues(bVal, aVal) {
			continue
		}
		// Nested maps: recurse one level for readable nested JSON settings.
		if nested := diffMaps(prefix+"."+k, bVal, aVal); nested != nil {
			maps.Copy(out, nested)
			continue
		}
		out[prefix+"."+k] = FieldChange{
			Before: normalizeValue(bVal),
			After:  normalizeValue(aVal),
		}
	}
	if len(out) == 0 {
		return map[string]any{}
	}
	return out
}

func asStringMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[string]string:
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[k] = val
		}
		return out, true
	case map[string]bool:
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[k] = val
		}
		return out, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		out := make(map[string]any, rv.Len())
		for _, key := range rv.MapKeys() {
			out[key.String()] = rv.MapIndex(key).Interface()
		}
		return out, true
	}
	return nil, false
}

// normalizeValue flattens values so JSON audit payloads stay scalar when possible.
func normalizeValue(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
		v = rv.Interface()
	}
	return v
}

func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func jsonName(f reflect.StructField) (name string, skip bool) {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name, false
	}
	// tag may be "name,omitempty" or "-"
	for i := 0; i < len(tag); i++ {
		if tag[i] == ',' {
			tag = tag[:i]
			break
		}
	}
	if tag == "-" {
		return "", true
	}
	if tag == "" {
		return f.Name, false
	}
	return tag, false
}

// DropMetaKeys removes volatile identity/timestamp keys from a Diff result so
// audit UIs highlight business-field changes (email, quantity, …).
func DropMetaKeys(changes map[string]any) map[string]any {
	if len(changes) == 0 {
		return changes
	}
	for k := range changes {
		base := k
		if i := strings.LastIndex(k, "."); i >= 0 {
			base = k[i+1:]
		}
		switch base {
		case "id", "createdAt", "updatedAt", "contentType", "userId", "userID":
			delete(changes, k)
		}
	}
	return changes
}

// DropAppearanceSettingKeys removes appearance / table-pref JSON keys so that
// ordinary profile/name updates never audit UI preferences. Appearance changes
// are recorded only from the dedicated appearance-settings endpoint.
func DropAppearanceSettingKeys(changes map[string]any) map[string]any {
	if len(changes) == 0 {
		return changes
	}
	for k := range changes {
		if k == "appearanceSettings" ||
			strings.HasPrefix(k, "appearanceSettings.") ||
			strings.Contains(k, ".appearanceSettings.") ||
			strings.HasSuffix(k, ".appearanceSettings") {
			delete(changes, k)
		}
	}
	return changes
}

// equalValues compares two field values, using fmt for types that are not
// directly comparable (slices, maps) to keep the diff robust.
func equalValues(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	// Decimal and time types implement String(); compare textual forms as a
	// fallback so equivalent-but-not-DeepEqual values are not flagged.
	as, aok := a.(fmt.Stringer)
	bs, bok := b.(fmt.Stringer)
	if aok && bok {
		return as.String() == bs.String()
	}
	return false
}

// zeroOf returns a zero struct of the same type as v for create/delete snapshots.
func zeroOf(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return nil
	}
	return reflect.New(rv.Type()).Elem().Interface()
}

// AttachChanges fills Entry.Changes from before/after values:
// both set → field diffs (edit); after only → create snapshot; before only → delete snapshot.
func AttachChanges(e *Entry, before, after any) {
	if e == nil {
		return
	}
	var changes map[string]any
	switch {
	case before != nil && after != nil:
		changes = DropMetaKeys(Diff(before, after))
		if len(changes) > 0 {
			e.Description = EnrichDescription(e.Description, changes)
		}
	case after != nil:
		changes = DropMetaKeys(Diff(zeroOf(after), after))
	case before != nil:
		changes = DropMetaKeys(Diff(before, zeroOf(before)))
	}
	if len(changes) > 0 {
		e.Changes = changes
	}
}
