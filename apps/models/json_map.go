package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONMap stores a JSON object in nvarchar/max columns. It implements
// driver.Valuer and sql.Scanner so database/sql never receives a raw
// map[string]any (which MSSQL rejects with "unsupported type map").
type JSONMap map[string]any

// Value marshals the map to a JSON string for the SQL driver.
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(map[string]any(m))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan unmarshals a JSON object from the database into the map.
func (m *JSONMap) Scan(src any) error {
	if m == nil {
		return fmt.Errorf("JSONMap: Scan on nil pointer")
	}
	if src == nil {
		*m = JSONMap{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("JSONMap: unsupported scan type %T", src)
	}
	if len(b) == 0 {
		*m = JSONMap{}
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	if out == nil {
		out = map[string]any{}
	}
	*m = JSONMap(out)
	return nil
}
